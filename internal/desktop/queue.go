package desktop

import (
	"context"
	"errors"
	"io"
	"llvy-uploader/internal/discovery"
	"llvy-uploader/internal/replay"
)

// Entry retains reservation identity across manual retries in this process.
// No retry silently creates a replacement for an expired or unknown reservation.
type Entry struct {
	File                 discovery.File
	Key                  string
	Binding              Reservation
	Metadata             Metadata
	Result               Result
	Message              string
	ReservationAttempted bool
	DeviceSessionID      string
}
type Progress func(entry *Entry, stage string, sent, total int64)

func NewEntry(file discovery.File) *Entry { return &Entry{File: file, Key: Random()} }

// SelectEntry refreshes an unreserved selection, but never changes an uncertain
// reservation's key or metadata after a request might have reached the server.
func SelectEntry(entries map[string]*Entry, file discovery.File) *Entry {
	e := entries[file.Path]
	if e == nil || e.NeedsReservation() && (e.File.Name != file.Name || e.File.Size != file.Size || !e.File.ModTime.Equal(file.ModTime) || e.File.Problem != file.Problem) {
		e = NewEntry(file)
		entries[file.Path] = e
	}
	return e
}

func (e *Entry) NeedsReservation() bool {
	return !e.ReservationAttempted && e.Binding.UploadID == ""
}

// QuotaSelection defers an oversized fresh batch without blocking recovery of
// already attempted reservations. The server remains authoritative for quota.
func QuotaSelection(entries []*Entry, remaining int) (ready []*Entry, deferred int) {
	fresh := 0
	for _, e := range entries {
		if e.NeedsReservation() {
			fresh++
		}
	}
	if fresh <= remaining {
		return entries, 0
	}
	for _, e := range entries {
		if !e.NeedsReservation() {
			ready = append(ready, e)
		}
	}
	return ready, fresh
}
func (c *Client) Upload(ctx context.Context, e *Entry, notify Progress) error {
	deviceID := c.DeviceSessionID()
	if deviceID == "" {
		return ErrLogin
	}
	if !e.NeedsReservation() && e.DeviceSessionID != deviceID {
		return ErrDeviceChanged
	}
	emit := func(stage string, sent, total int64) {
		if notify != nil {
			notify(e, stage, sent, total)
		}
	}
	if e.Result.State == "completed" || e.Result.State == "duplicate" {
		return nil
	}
	// Recover a known result without reopening or changing the original file.
	if e.Binding.UploadID != "" {
		result, err := c.Status(ctx, e.Binding.UploadID)
		if err != nil {
			return err
		}
		e.Result = result
		if result.State != "pending" {
			return resultError(result)
		}
	}
	emit("파일 확인", 0, e.File.Size)
	handle, err := replay.Open(e.File)
	if err != nil {
		return err
	}
	defer handle.Close()
	hash, err := handle.Hash(ctx)
	if err != nil {
		return err
	}
	metadata := Metadata{Filename: e.File.Name, Size: e.File.Size, LastModified: e.File.ModTime.UnixMilli(), SHA256: hash}
	if e.Metadata.Filename != "" && e.Metadata != metadata {
		return replay.ErrChanged
	}
	e.Metadata = metadata
	fresh := e.Binding.UploadID == ""
	if fresh {
		emit("예약", 0, e.File.Size)
		e.ReservationAttempted = true
		e.DeviceSessionID = deviceID
		e.Binding, err = c.Reserve(ctx, metadata, e.Key)
		if err != nil {
			return err
		}
	}
	// A previous/uncertain transfer is always probed through process first.
	shouldPut := fresh
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if shouldPut {
			if err := handle.Check(); err != nil {
				return err
			}
			if _, err := handle.Seek(0, io.SeekStart); err != nil {
				return err
			}
			descriptor, err := c.Transfer(ctx, e.Binding)
			if err != nil {
				return err
			}
			emit("전송", 0, e.File.Size)
			putErr := c.Put(ctx, descriptor, e.Binding.UploadID, handle, e.File.Size, func(n, total int64) { emit("전송", n, total) })
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Any PUT outcome is checked server-side, including no response bytes.
			if putErr != nil {
				emit("전송 결과 확인", e.File.Size, e.File.Size)
			}
		}
		emit("서버 처리", e.File.Size, e.File.Size)
		result, err := c.Process(ctx, e.Binding)
		if err == nil {
			e.Result = result
			return c.resolve(ctx, e)
		}
		var api *APIError
		if errors.As(err, &api) {
			if api.Code == "object_not_uploaded" {
				shouldPut = true
				continue
			}
			if api.Status == 401 || api.Status == 403 || api.Status == 429 {
				return err
			}
			if api.Status < 500 && api.Code != "outcome_unknown" {
				return err
			}
		}
		status, statusErr := c.Status(ctx, e.Binding.UploadID)
		if statusErr != nil {
			return err
		}
		e.Result = status
		if status.State != "pending" {
			return c.resolve(ctx, e)
		}
		shouldPut = false // A pending result is not proof that the object is absent.
	}
	return errors.New("결과 불명확 — 같은 예약의 상태를 다시 확인하세요")
}
func resultError(r Result) error {
	switch r.State {
	case "completed", "duplicate":
		return nil
	case "failed":
		return errors.New("처리 실패: " + r.ErrorCode)
	case "processing":
		return errors.New("서버 처리 중 — 상태를 다시 확인하세요")
	default:
		return errors.New("결과 불명확 — 새 예약을 만들지 마세요")
	}
}
func (c *Client) resolve(ctx context.Context, e *Entry) error {
	for attempt := 0; e.Result.State == "processing" && attempt < 3; attempt++ {
		if err := wait(ctx, 2e9); err != nil {
			return err
		}
		result, err := c.Status(ctx, e.Binding.UploadID)
		if err != nil {
			return err
		}
		e.Result = result
	}
	return resultError(e.Result)
}
func PauseQueue(err error) bool {
	var api *APIError
	return errors.Is(err, ErrLogin) || errors.Is(err, ErrDeviceChanged) || errors.As(err, &api) && (api.Status == 401 || api.Status == 403 || api.Status == 429 || api.Code == "desktop_disabled")
}
