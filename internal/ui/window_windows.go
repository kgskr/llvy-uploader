package ui

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"llvy-uploader/internal/app"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
	"time"

	"llvy-uploader/internal/matching"
	"llvy-uploader/internal/platform"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

const (
	wmDone   = 0x8001
	idFolder = 101 + iota
	idBrowse
	idDiscover
	idIDs
	idSearch
	idResults
	idChoose
	idClear
	idCancel
)

type choice struct{ row, candidate int }

type control struct {
	handle     uintptr
	x, y, w, h int
}
type window struct {
	restored         bool
	client           *desktop.Client
	session          *desktop.Session
	deviceID         string
	hasCredentials   bool
	entries          map[string]*desktop.Entry
	states           map[string]string
	completedFiles   map[string]discovery.File
	hwnd, font       uintptr
	controls         map[int]uintptr
	positions        []control
	store            *platform.Store
	folders          []discovery.Candidate
	rows             []app.Row
	choices          []choice
	ctx              context.Context
	cancel           context.CancelFunc
	workCancel       context.CancelFunc
	wg               sync.WaitGroup
	updates          chan update
	busy             bool
	workCanceled     bool
	dpi              int
	scrollX, scrollY int
}

func Run(version, commit string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	proc("SetProcessDpiAwarenessContext", ^uintptr(3)) // PER_MONITOR_AWARE_V2
	hr, _, _ := ole32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		return fmt.Errorf("폴더 선택 초기화 실패: %x", hr)
	}
	defer ole32.NewProc("CoUninitialize").Call()
	base, err := platform.LocalData()
	if err != nil {
		return err
	}
	origin, development := desktop.BuildOrigin()
	if err := desktop.ValidateOrigin(origin, development); err != nil {
		return err
	}
	store, err := platform.OpenStore(base, origin)
	if err != nil {
		return err
	}
	defer store.Close()
	recordResult(store, platform.EventStartup, nil)
	client, err := desktop.New(origin, development, store)
	if err != nil {
		return err
	}
	config, configErr := store.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &window{client: client, entries: map[string]*desktop.Entry{}, states: map[string]string{}, completedFiles: map[string]discovery.File{}, store: store, controls: map[int]uintptr{}, ctx: ctx, cancel: cancel, updates: make(chan update, 1), dpi: int(proc("GetDpiForSystem"))}
	if w.dpi == 0 {
		w.dpi = 96
	}
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return err
	}
	className := wide("LLVYUploader")
	wc := windowClass{Size: uint32(unsafe.Sizeof(windowClass{})), Callback: windows.NewCallback(w.callback), Instance: uintptr(instance), Cursor: proc("LoadCursorW", 0, 32512), Background: 6, Name: className}
	if proc("RegisterClassExW", uintptr(unsafe.Pointer(&wc))) == 0 {
		return fmt.Errorf("창 등록 실패")
	}
	title := wide("LLVY Uploader (" + version + ", " + commit + ")")
	var area rect
	proc("SystemParametersInfoW", 0x30, 0, uintptr(unsafe.Pointer(&area)), 0)
	width, height := w.scale(1060), w.scale(760)
	if area.Right > area.Left {
		width = min(width, int(area.Right-area.Left))
		height = min(height, int(area.Bottom-area.Top))
	}
	w.hwnd = proc("CreateWindowExW", 0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), 0x00ff0000, 0x80000000, 0x80000000, uintptr(width), uintptr(height), 0, 0, uintptr(instance), 0)
	if w.hwnd == 0 {
		return fmt.Errorf("창 생성 실패")
	}
	w.createControls()
	w.layout()
	proc("ShowWindow", w.hwnd, 5)
	proc("UpdateWindow", w.hwnd)
	w.discover(config.ReplayFolder)
	if configErr != nil {
		Alert("저장한 설정을 읽지 못했습니다. 폴더를 다시 선택하세요.")
	}
	var message msg
	var loopErr error
	for {
		result := int32(proc("GetMessageW", uintptr(unsafe.Pointer(&message)), 0, 0, 0))
		if result == 0 {
			break
		}
		if result == -1 {
			loopErr = fmt.Errorf("창 메시지 처리 실패")
			break
		}
		if proc("IsDialogMessageW", w.hwnd, uintptr(unsafe.Pointer(&message))) == 0 {
			proc("TranslateMessage", uintptr(unsafe.Pointer(&message)))
			proc("DispatchMessageW", uintptr(unsafe.Pointer(&message)))
		}
	}
	cancel()
	w.wg.Wait()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	closeErr := client.Close(closeCtx)
	recordResult(store, platform.EventShutdown, closeErr)
	if err := closeErr; err != nil {
		Alert(err.Error())
	}
	closeCancel()
	if w.font != 0 {
		gdi32.NewProc("DeleteObject").Call(w.font)
	}
	return loopErr
}
func (w *window) scale(n int) int { return n * w.dpi / 96 }
func (w *window) add(id int, class, label string, style uint32, x, y, width, height int) uintptr {
	cp, lp := wide(class), wide(label)
	h := proc("CreateWindowExW", 0, uintptr(unsafe.Pointer(cp)), uintptr(unsafe.Pointer(lp)), uintptr(style|0x50000000), uintptr(w.scale(x)), uintptr(w.scale(y)), uintptr(w.scale(width)), uintptr(w.scale(height)), w.hwnd, uintptr(id), 0, 0)
	w.controls[id] = h
	w.positions = append(w.positions, control{h, x, y, width, height})
	send(h, 0x30, w.font, 1)
	return h
}
func (w *window) createFont() {
	face := wide("Malgun Gothic")
	font, _, _ := gdi32.NewProc("CreateFontW").Call(uintptr(int64(-w.scale(16))), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(face)))
	old := w.font
	w.font = font
	for _, c := range w.positions {
		send(c.handle, 0x30, font, 1)
	}
	if old != 0 {
		gdi32.NewProc("DeleteObject").Call(old)
	}
}
func (w *window) createControls() {
	w.createFont()
	w.add(1, "STATIC", "리플레이 폴더", 0, 16, 16, 990, 24)
	w.add(idFolder, "COMBOBOX", "", 0x00210003, 16, 44, 720, 220)
	w.add(idBrowse, "BUTTON", "폴더 선택", 0x10000, 750, 44, 120, 30)
	w.add(idDiscover, "BUTTON", "다시 찾기", 0x10000, 880, 44, 130, 30)
	w.add(2, "STATIC", "게임 ID (공백·줄바꿈·쉼표·세미콜론 구분, 최대 100개)", 0, 16, 86, 990, 24)
	h := w.add(idIDs, "EDIT", "", 0x00a110c4, 16, 114, 850, 90)
	send(h, 0xc5, 8192, 0)
	w.add(idSearch, "BUTTON", "파일 찾기", 0x10000, 880, 114, 130, 36)
	w.add(idCancel, "BUTTON", "취소", 0x10000, 880, 162, 130, 36)
	w.add(3, "STATIC", "입력 ID | 파일명 | 크기 | 수정 시각 | 상태/오류", 0, 16, 214, 990, 24)
	w.add(idResults, "LISTBOX", "", 0x00b10101, 16, 244, 994, 246)
	w.add(idChoose, "BUTTON", "선택한 후보 사용", 0x10000, 16, 502, 180, 32)
	w.add(idClear, "BUTTON", "로컬 기록 삭제", 0x10000, 210, 502, 170, 32)
	w.add(4, "STATIC", "", 0, 16, 540, 994, 52)
	w.add(5, "EDIT", "LLVY 로그인 후 선택한 파일을 업로드할 수 있습니다. 중복·실패도 예약 한도를 사용합니다.", 0x00a10844, 16, 596, 994, 72)
	for i, label := range []string{"LLVY 로그인", "로그인 유지", "업로드", "실패 항목 다시 시도"} {
		style := uint32(0x10000)
		if i == 1 {
			style |= 3
		}
		h := w.add(200+i, "BUTTON", label, style, 16+i*220, 680, 205, 32)
		proc("EnableWindow", h, 1)
	}
	w.add(204, "BUTTON", "로그아웃", 0x10000, 896, 680, 114, 32)
	proc("EnableWindow", w.controls[idCancel], 0)
}
func (w *window) callback(hwnd uintptr, message uint32, wp, lp uintptr) uintptr {
	switch message {
	case 0x24: // WM_GETMINMAXINFO
		type minmax struct{ Reserved, MaxSize, MaxPosition, MinTrack, MaxTrack point }
		// LPARAM is also used for packed integers in other messages. Keep it a
		// uintptr at the callback boundary, copying only documented OS buffers.
		var m minmax
		moveMemory.Call(uintptr(unsafe.Pointer(&m)), lp, unsafe.Sizeof(m))
		m.MinTrack = point{int32(w.scale(600)), int32(w.scale(420))}
		moveMemory.Call(lp, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		return 0
	case 0x2e0: // WM_DPICHANGED
		w.dpi = int(wp & 0xffff)
		var r rect
		moveMemory.Call(uintptr(unsafe.Pointer(&r)), lp, unsafe.Sizeof(r))
		proc("SetWindowPos", hwnd, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x14)
		w.createFont()
		w.layout()
		return 0
	case 0x5: // WM_SIZE
		w.layout()
		return 0
	case 0x114, 0x115: // WM_HSCROLL / WM_VSCROLL
		w.scroll(message == 0x115, wp)
		return 0
	case 0x111:
		id, notification := int(wp&0xffff), int(wp>>16)
		if id == idResults && notification == 2 && !w.busy {
			w.openGame()
			return 0
		}
		if id == idFolder && notification == 1 && !w.busy {
			w.rows = nil
			w.render()
			w.saveFolder()
		}
		if notification == 0 {
			w.command(id)
		}
		return 0
	case wmDone:
		select {
		case result := <-w.updates:
			if !result.ongoing && w.workCanceled {
				result = suppressCanceledNavigation(result)
			}
			if result.replacement != nil {
				w.entries[result.replacement.File.Path] = result.replacement
			}
			result = reconcileEntries(w.entries, result)
			if result.authChecked {
				w.deviceID = result.deviceID
				w.hasCredentials = result.hasCredentials
				if w.session != nil && w.session.DeviceSessionID != result.deviceID {
					w.session = nil
				}
			}
			if !result.ongoing {
				w.setBusy(false)
			}
			if result.clearEntries {
				w.entries = map[string]*desktop.Entry{}
				w.states = map[string]string{}
				w.completedFiles = map[string]discovery.File{}
			}
			if result.queueDone {
				for path, e := range w.entries {
					if e.Succeeded() {
						w.completedFiles[path] = e.File
					} else {
						delete(w.completedFiles, path)
					}
					if e.Message != "" {
						w.states[path] = e.Message
					}
				}
			}
			if result.session != nil {
				w.session = result.session
			}
			if result.signedOut {
				w.session = nil
			}
			if result.entryPath != "" {
				w.states[result.entryPath] = result.entryMessage
			}
			if result.discovered {
				w.folders = result.folders
				send(w.controls[idFolder], 0x14b, 0, 0)
				for _, f := range w.folders {
					p := wide(f.Path + " — " + f.Source)
					send(w.controls[idFolder], 0x143, 0, uintptr(unsafe.Pointer(p)))
				}
				if len(w.folders) == 1 {
					send(w.controls[idFolder], 0x14e, 0, 0)
				}
			}
			if !result.keepRows {
				w.rows = result.rows
			}
			w.render()
			setText(w.controls[5], result.message)
			if result.picker {
				w.browse()
			}
			if result.retryOffer != nil && !result.clearEntries && w.ctx.Err() == nil {
				w.confirmRetry(result.retryOffer)
			}
		default:
		}
		return 0
	case 0x10:
		w.cancel()
		proc("DestroyWindow", hwnd)
		return 0
	case 0x2:
		proc("PostQuitMessage", 0)
		return 0
	}
	return proc("DefWindowProcW", hwnd, uintptr(message), wp, lp)
}
func (w *window) setBusy(b bool) {
	w.busy = b
	enabled := uintptr(1)
	if b {
		enabled = 0
	}
	for _, id := range []int{idFolder, idBrowse, idDiscover, idSearch, idChoose, idClear, idIDs, 200, 201, 202, 203, 204} {
		proc("EnableWindow", w.controls[id], enabled)
	}
	proc("EnableWindow", w.controls[idCancel], 1-enabled)
}
func (w *window) work(fn func(context.Context) update) {
	w.workCanceled = false
	w.setBusy(true)
	ctx, cancel := context.WithCancel(w.ctx)
	w.workCancel = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer cancel()
		result := runClientWork(ctx, w.client, fn)
		select {
		case w.updates <- result:
			proc("PostMessageW", w.hwnd, wmDone, 0, 0)
		case <-w.ctx.Done():
		}
	}()
}
func (w *window) discover(saved string) {
	w.work(func(ctx context.Context) update {
		var session desktop.Session
		var sessionErr error
		if !w.restored {
			session, sessionErr = restoreSession(ctx, w.client)
			w.restored = true
		} else {
			session, sessionErr = w.client.Session(ctx)
		}
		candidates, err := platform.Candidates()
		result := discovery.Find(ctx, discovery.OSReader{}, saved, candidates)
		message := strings.Join(result.Warnings, "\r\n")
		if sessionErr != nil && !errors.Is(sessionErr, desktop.ErrLogin) {
			message += "\r\n" + sessionErr.Error()
		}
		if err != nil {
			message += "\r\nWindows 문서 폴더를 확인하지 못했습니다."
		}
		if len(result.Candidates) == 0 {
			message += "\r\n" + discovery.MissingFolder
		} else if len(result.Candidates) > 1 {
			message += "\r\n리플레이 폴더 후보 중 하나를 선택하세요."
		} else {
			message += "\r\n폴더를 찾았습니다. 게임 ID를 입력하세요."
		}
		u := update{selectionResult: true, folders: result.Candidates, discovered: true, picker: len(result.Candidates) == 0, message: message}
		if sessionErr == nil {
			u.session = &session
		}
		return u
	})
}
func (w *window) folder() string {
	i := int(int32(send(w.controls[idFolder], 0x147, 0, 0)))
	if i < 0 || i >= len(w.folders) {
		return ""
	}
	return w.folders[i].Path
}
func (w *window) saveFolder() {
	if folder := w.folder(); folder != "" {
		if err := w.store.Save(platform.Config{ReplayFolder: folder}); err != nil {
			Alert("폴더 설정을 저장하지 못했습니다.")
		}
	}
}
func (w *window) browse() {
	folder, err := pickFolder(w.hwnd)
	if err != nil {
		Alert(err.Error())
		return
	}
	if folder == "" {
		return
	}
	w.work(func(ctx context.Context) update {
		if _, err := os.ReadDir(folder); err != nil {
			return update{message: "선택한 폴더에 접근할 수 없습니다. 폴더를 다시 선택하세요."}
		}
		message := "폴더를 선택했습니다. 게임 ID를 입력하세요."
		if err := w.store.Save(platform.Config{ReplayFolder: folder}); err != nil {
			message += " 설정을 저장하지 못했습니다."
		}
		return update{folders: []discovery.Candidate{{Path: folder, Source: "직접 선택"}}, discovered: true, message: message}
	})
}
func (w *window) command(id int) {
	if id == idCancel {
		if w.busy {
			w.workCanceled = true
		}
		if w.workCancel != nil {
			w.workCancel()
		}
		return
	}
	if w.busy {
		return
	}
	switch id {
	case 200:
		persistent := send(w.controls[201], 0xf0, 0, 0) == 1
		w.work(func(ctx context.Context) update {
			return connect(ctx, w.client, persistent, openBrowser)
		})
	case 202:
		w.upload(false)
	case 203:
		w.upload(true)
	case 204:
		w.work(func(ctx context.Context) update {
			err := w.client.Logout(ctx)
			message := "로그아웃했습니다."
			if err != nil {
				message = err.Error()
			}
			return update{keepRows: true, signedOut: true, message: message}
		})
	case idBrowse:
		w.browse()
	case idDiscover:
		w.discover(w.folder())
	case idSearch:
		folder, input := w.folder(), text(w.controls[idIDs])
		if folder == "" {
			Alert("리플레이 폴더를 선택하세요.")
			return
		}
		w.work(func(ctx context.Context) update {
			ids, errs := matching.Parse(input)
			var messages []string
			for _, err := range errs {
				messages = append(messages, err.Error())
			}
			if len(ids) == 0 {
				messages = append(messages, "유효한 게임 ID를 입력하세요.")
				return update{message: strings.Join(messages, "\r\n")}
			}
			files, err := discovery.Scan(ctx, discovery.OSReader{}, folder)
			if err != nil {
				return update{message: "폴더를 읽을 수 없습니다. 폴더를 다시 선택하세요."}
			}
			if len(messages) == 0 {
				messages = append(messages, "검색 완료. 여러 파일이 있으면 후보를 고른 뒤 선택한 후보 사용을 누르세요.")
			}
			return update{selectionResult: true, rows: app.Preview(ids, files), message: strings.Join(messages, "\r\n")}
		})
	case idChoose:
		i := int(int32(send(w.controls[idResults], 0x188, 0, 0)))
		if i < 0 || i >= len(w.choices) {
			return
		}
		c := w.choices[i]
		if err := app.Select(w.rows, c.row, c.candidate); err != nil {
			Alert(err.Error())
			return
		}
		w.render()
	case idClear:
		w.work(func(ctx context.Context) update {
			err := w.client.Logout(ctx)
			clearErr := w.store.Clear()
			message := "로컬 기록을 삭제했습니다. 원본 리플레이는 보존했습니다."
			if err != nil {
				message += "\r\n" + err.Error()
			}
			if clearErr != nil {
				message += "\r\n로컬 기록 삭제에 실패했습니다."
			}
			return update{clearEntries: true, signedOut: true, discovered: true, message: message}
		})

	}
}
func (w *window) render() {
	send(w.controls[idResults], 0x18b, 0, 0)
	w.choices = nil
	add := func(line string, c choice) {
		p := wide(line)
		send(w.controls[idResults], 0x180, 0, uintptr(unsafe.Pointer(p)))
		w.choices = append(w.choices, c)
	}
	for i, row := range w.rows {
		if len(row.Candidates) == 0 {
			add(row.ID.String()+" | 파일 없음", choice{i, -1})
			continue
		}
		for j, f := range row.Candidates {
			status := row.Status()
			if f.Problem != "" {
				status = f.Problem
			} else if row.Selected == j {
				status = "선택됨"
			} else if row.Selected >= 0 {
				status = "선택 안 됨"
			}
			completed, hasCompletion := w.completedFiles[f.Path]
			status = candidateStatus(f, status, w.states[f.Path], completed, hasCompletion)
			add(fmt.Sprintf("%s | %s | %d 바이트 | %s | %s", row.ID, f.Name, f.Size, f.ModTime.Format("2006-01-02 15:04:05"), status), choice{i, j})
		}
	}
	send(w.controls[idResults], 0x194, uintptr(w.scale(2200)), 0)
	summary := fmt.Sprintf("선택 %d개 · 로그인 필요", len(app.Selected(w.rows)))
	if w.hasCredentials {
		summary = fmt.Sprintf("선택 %d개 · 연결 정보 있음 · LLVY 로그인으로 상태 확인", len(app.Selected(w.rows)))
	}
	if w.session != nil {
		summary = sessionSummary(*w.session, len(app.Selected(w.rows)))
	}
	setText(w.controls[4], summary)
}
