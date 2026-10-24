package ui

import "unsafe"

type scrollInfo struct {
	Size, Mask    uint32
	Min, Max      int32
	Page          uint32
	Pos, TrackPos int32
}

// The content remains reachable on small/high-DPI displays. Native scrollbars
// supplement keyboard tab navigation instead of requiring a large desktop.
func (w *window) layout() {
	if w.hwnd == 0 || len(w.positions) == 0 {
		return
	}
	var bounds rect
	proc("GetClientRect", w.hwnd, uintptr(unsafe.Pointer(&bounds)))
	for axis, page := range []int{int(bounds.Right), int(bounds.Bottom)} {
		length, position := w.scale(1040), &w.scrollX
		if axis == 1 {
			length, position = w.scale(738), &w.scrollY
		}
		*position = max(0, min(*position, length-page))
		info := scrollInfo{Size: uint32(unsafe.Sizeof(scrollInfo{})), Mask: 1 | 2 | 4 | 8, Max: int32(length - 1), Page: uint32(max(0, page)), Pos: int32(*position)}
		proc("SetScrollInfo", w.hwnd, uintptr(axis), uintptr(unsafe.Pointer(&info)), 1)
	}
	for _, c := range w.positions {
		proc("MoveWindow", c.handle, uintptr(w.scale(c.x)-w.scrollX), uintptr(w.scale(c.y)-w.scrollY), uintptr(w.scale(c.w)), uintptr(w.scale(c.h)), 1)
	}
}
func (w *window) scroll(vertical bool, wp uintptr) {
	axis, position := uintptr(0), &w.scrollX
	if vertical {
		axis, position = 1, &w.scrollY
	}
	info := scrollInfo{Size: uint32(unsafe.Sizeof(scrollInfo{})), Mask: 0x17}
	proc("GetScrollInfo", w.hwnd, axis, uintptr(unsafe.Pointer(&info)))
	switch wp & 0xffff {
	case 0:
		*position -= w.scale(24)
	case 1:
		*position += w.scale(24)
	case 2:
		*position -= int(info.Page)
	case 3:
		*position += int(info.Page)
	case 4, 5:
		*position = int(info.TrackPos)
	case 6:
		*position = 0
	case 7:
		*position = int(info.Max)
	}
	w.layout()
}
