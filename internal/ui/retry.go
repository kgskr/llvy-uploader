package ui

import (
	"context"
	"errors"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
)

type retryRequest struct {
	entry    *desktop.Entry
	file     discovery.File
	deviceID string
}

// Refresh only unattempted selections, retaining every uncertain reservation.
func selectRetry(entries map[string]*desktop.Entry, file discovery.File, deviceID string) *retryRequest {
	entry := entries[file.Path]
	if entry == nil || entry.Result.State == "completed" || entry.Result.State == "duplicate" {
		return nil
	}
	entry = desktop.SelectEntry(entries, file)
	return &retryRequest{entry: entry, file: file, deviceID: deviceID}
}

type retryClient interface {
	DeviceSessionID() string
	Upload(context.Context, *desktop.Entry, desktop.Progress) error
	Session(context.Context) (desktop.Session, error)
}

// A confirmation only authorizes a new attempt after recovery is checked again.
// Returning the replacement even on cancellation preserves any attempted key.
func retrySelected(ctx context.Context, client retryClient, request *retryRequest, confirmed bool, notify desktop.Progress) update {
	result := update{keepRows: true, queueDone: true}
	if err := ctx.Err(); err != nil {
		result.message = err.Error()
		return result
	}
	if request.deviceID == "" || request.deviceID != client.DeviceSessionID() {
		result.message = desktop.ErrDeviceChanged.Error()
		return result
	}
	checkQuota := func() bool {
		session, err := client.Session(ctx)
		if err != nil {
			result.message += "\r\n" + err.Error()
			return false
		}
		result.session = &session
		if request.deviceID != client.DeviceSessionID() || session.DeviceSessionID != request.deviceID {
			result.message = desktop.ErrDeviceChanged.Error()
			return false
		}
		if min(session.Quota.Account.Remaining, session.Quota.Global.Remaining) < 1 {
			result.message += "\r\n새 예약 한도가 없습니다. 한도 갱신 후 다시 시도하세요.\r\n" + quotaResetMessage(session)
			return false
		}
		if err := ctx.Err(); err != nil {
			result.message += "\r\n" + err.Error()
			return false
		}
		return true
	}
	// A local verification error may have happened before the first reservation.
	// Such an entry is still a fresh upload, unlike a zero-quota recovery.
	if request.entry.NeedsReservation() && !checkQuota() {
		return result
	}
	err := client.Upload(ctx, request.entry, notify)
	request.entry.Message = uploadMessage(request.entry, err)
	result.message = request.entry.Message
	var api *desktop.APIError
	if ctx.Err() == nil && errors.As(err, &api) && api.Status == 429 && !refreshThrottled(err) {
		if snapshot, lookupErr := client.Session(ctx); lookupErr == nil && snapshot.DeviceSessionID == request.deviceID && client.DeviceSessionID() == request.deviceID {
			result.session = &snapshot
			result.message += "\r\n" + quotaResetMessage(snapshot)
		}
	}
	if ctx.Err() != nil || !desktop.NewAttemptAllowed(err) {
		return result
	}
	if !confirmed {
		result.retryOffer = request
		return result
	}
	if !checkQuota() {
		return result
	}
	next := desktop.NewEntry(request.file)
	result.replacement = next
	err = client.Upload(ctx, next, notify)
	next.Message = uploadMessage(next, err)
	result.message = next.Message
	if !refreshThrottled(err) {
		if snapshot, lookupErr := client.Session(ctx); lookupErr == nil {
			result.session = &snapshot
		}
	}
	return result
}

func uploadMessage(entry *desktop.Entry, err error) string {
	if err != nil {
		return err.Error()
	}
	if entry.Result.Duplicate {
		return "중복 성공"
	}
	return "완료"
}

// A quota lookup can itself refresh, replaying the just-throttled credential.
// Daily reservation quota errors can still refresh their allowance snapshot.
func refreshThrottled(err error) bool {
	var api *desktop.APIError
	return errors.As(err, &api) && api.Status == 429 && api.Code == "rate_limited"
}
