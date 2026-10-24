package ui

import (
	"fmt"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
	"time"
)

func quotaResetMessage(session desktop.Session) string {
	format := func(at *time.Time) string {
		if at == nil {
			return "구간 미시작"
		}
		return at.Local().Format("2006-01-02 15:04:05")
	}
	return "한도 갱신: 계정 " + format(session.Quota.Account.ResetAt) + " · 전체 " + format(session.Quota.Global.ResetAt)
}

func sessionSummary(session desktop.Session, selected int) string {
	return fmt.Sprintf("%s · 선택 %d개 · 잔여 %d개 (계정 %d/전체 %d) · 확인 %s\r\n%s", session.Account.Name, selected, min(session.Quota.Account.Remaining, session.Quota.Global.Remaining), session.Quota.Account.Remaining, session.Quota.Global.Remaining, session.Quota.AsOf.Local().Format("15:04:05"), quotaResetMessage(session))
}

// Completed snapshots are owned by the UI thread; rendering never reads entries
// that a running upload worker may be mutating.
func candidateStatus(file discovery.File, fallback, saved string, completed discovery.File, hasCompletion bool) string {
	if file.Problem != "" {
		return file.Problem
	}
	if hasCompletion && !desktop.SameFile(completed, file) {
		return "파일 변경됨 · 업로드하면 새 예약 사용"
	}
	if saved != "" {
		return saved
	}
	return fallback
}
