package ui

import (
	"context"
	"llvy-uploader/internal/app"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
)

type update struct {
	folders         []discovery.Candidate
	rows            []app.Row
	message         string
	discovered      bool
	picker          bool
	queueDone       bool
	clearEntries    bool
	keepRows        bool
	ongoing         bool
	session         *desktop.Session
	signedOut       bool
	entryPath       string
	entryMessage    string
	selectionResult bool
	authChecked     bool
	hasCredentials  bool
	deviceID        string
	retryOffer      *retryRequest
	replacement     *desktop.Entry
}

func runWork(ctx context.Context, fn func(context.Context) update) update {
	result := fn(ctx)
	if ctx.Err() != nil {
		result = suppressCanceledNavigation(result)
		if !result.clearEntries && !result.discovered && result.rows == nil {
			result.keepRows = true
		}
		notice := "취소됨 — 전송 중이었다면 결과를 다시 확인하세요. 원본 파일은 보존했습니다"
		if result.message != "" {
			notice += "\r\n" + result.message
		}
		result.message = notice
	}
	return result
}

// Also applied by the UI when cancellation arrives after the worker returned.
// Committed uploads and credential changes still need their final update.
func suppressCanceledNavigation(result update) update {
	result.queueDone = true
	result.picker = false
	result.retryOffer = nil
	if result.selectionResult {
		result.folders = nil
		result.rows = nil
		result.discovered = false
		result.keepRows = true
	}
	return result
}

func runClientWork(ctx context.Context, client *desktop.Client, fn func(context.Context) update) update {
	result := runWork(ctx, fn)
	result.authChecked = true
	result.deviceID = client.DeviceSessionID()
	result.hasCredentials = client.HasCredentials()
	result.signedOut = !result.hasCredentials
	if result.retryOffer != nil && result.retryOffer.deviceID != result.deviceID {
		result.retryOffer = nil
	}
	if result.session != nil && result.session.DeviceSessionID != result.deviceID {
		result.session = nil
	}
	return result
}

func reconcileEntries(entries map[string]*desktop.Entry, result update) update {
	if !result.authChecked || result.clearEntries {
		return result
	}
	for _, entry := range entries {
		if !entry.NeedsReservation() && entry.DeviceSessionID != result.deviceID {
			result.clearEntries = true
			result.retryOffer = nil
			result.keepRows = false
			result.rows = nil
			result.message += "\r\n연결이 변경되어 이전 작업과 선택을 비웠습니다. 이전 서버 결과를 확인한 뒤 파일을 다시 찾으세요. 새 업로드는 예약 한도를 사용합니다."
			break
		}
	}
	return result
}

func connect(ctx context.Context, client *desktop.Client, persistent bool, open func(string) error) update {
	if !client.HasCredentials() {
		if err := client.Login(ctx, persistent, open); err != nil {
			return update{keepRows: true, message: err.Error()}
		}
	}
	session, err := client.Session(ctx)
	if err != nil {
		message := err.Error()
		if client.HasCredentials() {
			message += "\r\n연결 정보가 남아 있습니다. LLVY 로그인을 다시 눌러 상태를 확인하세요."
		}
		return update{keepRows: true, message: message}
	}
	return update{keepRows: true, session: &session, message: "연결 상태를 확인했습니다. 선택한 파일과 한도를 확인한 뒤 업로드하세요."}
}

// A failed restoration must not be retried immediately by session lookup.
func restoreSession(ctx context.Context, client *desktop.Client) (desktop.Session, error) {
	if err := client.Restore(ctx); err != nil {
		return desktop.Session{}, err
	}
	return client.Session(ctx)
}
