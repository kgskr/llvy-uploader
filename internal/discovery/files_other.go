//go:build !windows

package discovery

import "os"

func (OSReader) Inspect(path string) (File, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return File{}, false, err
	}
	if !info.Mode().IsRegular() {
		return File{}, false, nil
	}
	return File{Path: path, Name: info.Name(), Size: info.Size(), ModTime: info.ModTime()}, true, nil
}
