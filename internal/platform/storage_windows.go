package platform

import (
	"errors"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"runtime"
	"unsafe"
)

func lockFile(path string) (io.Closer, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		return nil, errors.New("이미 실행 중입니다")
	}
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), path), nil
}
func replaceFile(from, to string) error {
	f, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	t, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(f, t, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func LocalData() (string, error) { return windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0) }

// Protect/Unprotect are primitives only; no token protocol is invented here.
func Protect(data []byte) ([]byte, error)   { return crypt(data, true) }
func Unprotect(data []byte) ([]byte, error) { return crypt(data, false) }
func crypt(data []byte, protect bool) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("빈 자격 증명")
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	var err error
	if protect {
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	runtime.KeepAlive(data)
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...), nil
}
