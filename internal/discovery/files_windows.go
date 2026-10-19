package discovery

import (
	"golang.org/x/sys/windows"
	"time"
)

// FindFirstFile reads directory metadata, not placeholder content. Cloud tags
// are not name-surrogate tags and must not be excluded simply for being reparse points.
func (OSReader) Inspect(path string) (File, bool, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return File{}, false, err
	}
	var data windows.Win32finddata
	h, err := windows.FindFirstFile(p, &data)
	if err != nil {
		return File{}, false, err
	}
	defer windows.FindClose(h)
	return FromWindowsMetadata(path, data.FileAttributes, data.Reserved0,
		int64(data.FileSizeHigh)<<32|int64(data.FileSizeLow), time.Unix(0, data.LastWriteTime.Nanoseconds()))
}
