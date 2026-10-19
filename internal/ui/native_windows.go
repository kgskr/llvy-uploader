// Package ui contains the native Win32 boundary. All control access stays on the UI thread.
package ui

import (
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"unsafe"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var shell32 = windows.NewLazySystemDLL("shell32.dll")
var ole32 = windows.NewLazySystemDLL("ole32.dll")
var gdi32 = windows.NewLazySystemDLL("gdi32.dll")
var moveMemory = windows.NewLazySystemDLL("ntdll.dll").NewProc("RtlMoveMemory")

//go:uintptrescapes
func proc(name string, args ...uintptr) uintptr {
	r, _, _ := user32.NewProc(name).Call(args...)
	return r
}
func wide(s string) *uint16 { p, _ := windows.UTF16PtrFromString(s); return p }
func text(h uintptr) string {
	n := proc("GetWindowTextLengthW", h)
	buf := make([]uint16, n+1)
	proc("GetWindowTextW", h, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return windows.UTF16ToString(buf)
}
func setText(h uintptr, s string) {
	p := wide(s)
	proc("SetWindowTextW", h, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

//go:uintptrescapes
func send(h, m, w, l uintptr) uintptr { return proc("SendMessageW", h, m, w, l) }
func Alert(message string) {
	p, title := wide(message), wide("LLVY Uploader")
	proc("MessageBoxW", 0, uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(title)), 0x40)
	runtime.KeepAlive(p)
	runtime.KeepAlive(title)
}

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct {
	Hwnd    uintptr
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}
type windowClass struct {
	Size, Style                        uint32
	Callback                           uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	SmallIcon                          uintptr
}

func pickFolder(owner uintptr) (string, error) {
	type browseInfo struct {
		Owner, Root     uintptr
		Display, Title  *uint16
		Flags           uint32
		Callback, Param uintptr
		Image           int32
	}
	buffer := make([]uint16, 32768)
	info := browseInfo{Owner: owner, Display: &buffer[0], Title: wide("리플레이 폴더 선택"), Flags: 0x1 | 0x40 | 0x10}
	pidl, _, _ := shell32.NewProc("SHBrowseForFolderW").Call(uintptr(unsafe.Pointer(&info)))
	if pidl == 0 {
		return "", nil
	}
	defer ole32.NewProc("CoTaskMemFree").Call(pidl)
	ok, _, _ := shell32.NewProc("SHGetPathFromIDListEx").Call(pidl, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), 0)
	if ok == 0 {
		return "", errors.New("선택한 위치의 로컬 폴더 경로를 읽을 수 없습니다")
	}
	return windows.UTF16ToString(buffer), nil
}
