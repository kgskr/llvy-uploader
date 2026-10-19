package replay

import (
	"errors"
	"golang.org/x/sys/windows"
	"llvy-uploader/internal/discovery"
	"os"
	"time"
	"unsafe"
)

func openRead(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	// Permit other readers, but no writers or deletions until this handle closes.
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_OPEN_NO_RECALL|windows.FILE_FLAG_SEQUENTIAL_SCAN, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), path), nil
}

// Go marks hydrated Cloud Files irregular. checkHandle verifies the exact tag
// and availability before any read, so this exception is Windows-only.
func regular(info os.FileInfo) bool { return info.Mode()&(os.ModeType&^os.ModeIrregular) == 0 }

func checkHandle(f *os.File) error {
	var data struct{ Attributes, Tag uint32 }
	if err := windows.GetFileInformationByHandleEx(windows.Handle(f.Fd()), windows.FileAttributeTagInfo, (*byte)(unsafe.Pointer(&data)), uint32(unsafe.Sizeof(data))); err != nil {
		return err
	}
	metadata, include, err := discovery.FromWindowsMetadata(f.Name(), data.Attributes, data.Tag, 1, time.Time{})
	if err != nil {
		return err
	}
	if !include {
		return ErrChanged
	}
	if metadata.Problem != "" {
		return errors.New(metadata.Problem)
	}
	return nil
}
