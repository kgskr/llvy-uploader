package ui

import (
	"context"
	"strings"
	"testing"
)

func TestCanceledWorkPreservesCompletedLogoutAndDeletion(t *testing.T) {
	for _, clear := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		localSignedIn := true
		result := runWork(ctx, func(context.Context) update {
			localSignedIn = false
			cancel()
			return update{signedOut: true, clearEntries: clear, discovered: clear, keepRows: !clear, message: "서버 연결 해제를 확인하지 못했습니다"}
		})
		if localSignedIn || !result.signedOut || result.clearEntries != clear || result.discovered != clear || result.keepRows == clear {
			t.Fatalf("lost completed state: %+v", result)
		}
		if !strings.Contains(result.message, "취소됨") || !strings.Contains(result.message, "서버 연결 해제") {
			t.Fatalf("lost outcome notice: %s", result.message)
		}
	}
}
