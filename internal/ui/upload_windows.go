package ui

import (
	"context"
	"fmt"
	"llvy-uploader/internal/app"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/platform"
	"runtime"
	"time"
	"unsafe"
)

func openBrowser(url string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := ole32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		return fmt.Errorf("브라우저 초기화 실패")
	}
	defer ole32.NewProc("CoUninitialize").Call()
	verb, target := wide("open"), wide(url)
	result, _, _ := shell32.NewProc("ShellExecuteW").Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if result <= 32 {
		return fmt.Errorf("기본 브라우저를 열지 못했습니다")
	}
	return nil
}
func (w *window) progress(path, message string) {
	// Coalesce optional progress events; the final update is always delivered.
	select {
	case w.updates <- update{ongoing: true, keepRows: true, entryPath: path, entryMessage: message, message: message}:
		proc("PostMessageW", w.hwnd, wmDone, 0, 0)
	default:
	}
}
func (w *window) upload(retry bool) {
	files := app.Selected(w.rows)
	if retry {
		i := int(int32(send(w.controls[idResults], 0x188, 0, 0)))
		if i < 0 || i >= len(w.choices) {
			Alert("다시 시도할 파일을 선택하세요.")
			return
		}
		choice := w.choices[i]
		if choice.candidate < 0 {
			return
		}
		file := w.rows[choice.row].Candidates[choice.candidate]
		if request := selectRetry(w.entries, file, w.client.DeviceSessionID()); request != nil {
			w.work(func(ctx context.Context) update {
				return retrySelected(ctx, diagnosticClient{w.client, w.store}, request, false, w.retryProgress())
			})
		}
		return
	}
	if len(files) == 0 {
		Alert("업로드할 파일을 선택하세요.")
		return
	}
	// Capture entries before launching: only this worker mutates entries until done.
	entries := make([]*desktop.Entry, 0, len(files))
	for _, file := range files {
		previous := w.entries[file.Path]
		e := desktop.SelectEntry(w.entries, file)
		if e != previous {
			delete(w.states, file.Path)
			delete(w.completedFiles, file.Path)
		}
		entries = append(entries, e)
	}
	w.work(func(ctx context.Context) update {
		session, err := w.client.Session(ctx)
		if err != nil {
			recordResult(w.store, platform.EventAuthentication, err)
			return update{keepRows: true, message: err.Error(), signedOut: errorsIsLogin(err)}
		}
		remaining := min(session.Quota.Account.Remaining, session.Quota.Global.Remaining)
		ready, deferred := desktop.QuotaSelection(entries, remaining)
		var lastProgress time.Time
		refreshSnapshot := true
		message := "업로드 작업을 마쳤습니다. 중복과 실패도 예약 한도를 사용합니다."
		if deferred > 0 {
			message = fmt.Sprintf("새 예약 %d개는 현재 잔여 %d개를 초과하여 보류했습니다. 파일 선택 수를 줄이세요. 기존 예약은 복구를 진행합니다.", deferred, remaining)
		}
		for _, entry := range ready {
			if ctx.Err() != nil {
				break
			}
			err := (diagnosticClient{w.client, w.store}).Upload(ctx, entry, func(e *desktop.Entry, stage string, sent, total int64) {
				if time.Since(lastProgress) < 150*time.Millisecond && sent < total {
					return
				}
				lastProgress = time.Now()
				percent := int64(0)
				if total > 0 {
					percent = sent * 100 / total
				}
				w.progress(e.File.Path, fmt.Sprintf("%s · %s %d%%", e.File.Name, stage, percent))
			})
			state := "완료"
			if entry.Result.Duplicate {
				state = "중복 성공"
			}
			if err != nil {
				state = err.Error()
			}
			entry.Message = state
			w.progress(entry.File.Path, state)
			if desktop.PauseQueue(err) {
				refreshSnapshot = !refreshThrottled(err)
				message = "큐를 멈췄습니다. " + state
				break
			}
		}
		if refreshSnapshot {
			snapshot, lookupErr := w.client.Session(ctx)
			err = lookupErr
			if err == nil {
				session = snapshot
			}
		}
		// Final status map is copied by the UI thread via rows rendering after worker completion.
		message += "\r\n" + quotaResetMessage(session)
		return update{keepRows: true, session: &session, message: message, signedOut: errorsIsLogin(err), queueDone: true}
	})
}
func errorsIsLogin(err error) bool { return err == desktop.ErrLogin }

func (w *window) retryProgress() desktop.Progress {
	var last time.Time
	return func(e *desktop.Entry, stage string, sent, total int64) {
		if time.Since(last) < 150*time.Millisecond && sent < total {
			return
		}
		last = time.Now()
		percent := int64(0)
		if total > 0 {
			percent = sent * 100 / total
		}
		w.progress(e.File.Path, fmt.Sprintf("%s · %s %d%%", e.File.Name, stage, percent))
	}
}

// Called only by the UI thread after the recovery worker has stopped.
func (w *window) confirmRetry(request *retryRequest) {
	if w.client.DeviceSessionID() != request.deviceID || w.entries[request.file.Path] != request.entry {
		return
	}
	message := wide(request.file.Name + "\r\n" + request.entry.Message + "\r\n\r\n새 예약으로 다시 업로드하시겠습니까?\r\n새 예약 1회를 사용하며 중복·실패도 한도를 소비합니다. 결과가 불명확하면 기존 경기 등록 여부를 먼저 확인하세요.")
	title := wide("새 예약으로 다시 시도")
	w.setBusy(true)
	answer := proc("MessageBoxW", w.hwnd, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)), 0x134) // YESNO, warning, default No
	runtime.KeepAlive(message)
	runtime.KeepAlive(title)
	w.setBusy(false)
	if answer != 6 || w.ctx.Err() != nil {
		return
	}
	w.work(func(ctx context.Context) update {
		return retrySelected(ctx, diagnosticClient{w.client, w.store}, request, true, w.retryProgress())
	})
}

func (w *window) openGame() {
	i := int(int32(send(w.controls[idResults], 0x188, 0, 0)))
	if i < 0 || i >= len(w.choices) {
		return
	}
	choice := w.choices[i]
	if choice.candidate < 0 {
		return
	}
	file := w.rows[choice.row].Candidates[choice.candidate]
	entry := w.entries[file.Path]
	if entry == nil || !entry.Succeeded() || !desktop.SameFile(entry.File, file) {
		return
	}
	if err := openBrowser(w.client.Origin() + entry.Result.GamePath); err != nil {
		Alert(err.Error())
	}
}
