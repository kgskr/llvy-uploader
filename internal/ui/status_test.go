package ui

import (
	"context"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
	"strings"
	"testing"
	"time"
)

func TestChangedCompletedCandidateDoesNotShowOldSuccess(t *testing.T) {
	previous := discovery.File{Path: "replay.rofl", Name: "replay.rofl", Size: 1}
	current := previous
	current.Size++
	if got := candidateStatus(current, "선택됨", "완료", previous, true); !strings.Contains(got, "파일 변경됨") {
		t.Fatalf("stale result displayed: %s", got)
	}
	if got := candidateStatus(previous, "선택됨", "중복 성공", previous, true); got != "중복 성공" {
		t.Fatalf("unchanged result lost: %s", got)
	}
	if got := candidateStatus(current, "선택됨", "결과 확인 필요", previous, false); got != "결과 확인 필요" {
		t.Fatalf("uncertain reservation hidden: %s", got)
	}
	current.Problem = "로컬 다운로드 필요"
	if got := candidateStatus(current, "선택됨", "완료", previous, true); got != current.Problem {
		t.Fatalf("current file problem hidden: %s", got)
	}
}

func TestSessionAndQuotaBlockedRetryDisplayBothResetTimes(t *testing.T) {
	account := time.Date(2026, 10, 20, 10, 30, 0, 0, time.UTC)
	global := account.Add(3 * time.Hour)
	f, request := retryFixture()
	session := desktop.Session{DeviceSessionID: connectionID}
	session.Quota.Account.ResetAt = &account
	session.Quota.Global.ResetAt = &global
	f.lookup = func(context.Context) (desktop.Session, error) { return session, nil }
	result := retrySelected(context.Background(), f, request, true, nil)
	if result.replacement != nil || len(f.calls) != 1 {
		t.Fatal("quota-blocked retry created a new attempt")
	}
	for _, message := range []string{sessionSummary(session, 2), result.message} {
		for _, at := range []time.Time{account, global} {
			if !strings.Contains(message, at.Local().Format("2006-01-02 15:04:05")) {
				t.Fatalf("missing reset time: %s", message)
			}
		}
	}
	if got := quotaResetMessage(desktop.Session{}); strings.Count(got, "구간 미시작") != 2 {
		t.Fatalf("null resets not distinguished: %s", got)
	}
}
