//go:build !windows

package replay

import "os"

// Portable tests cannot assert Windows sharing semantics.
func openRead(path string) (*os.File, error) { return os.Open(path) }
func checkHandle(f *os.File) error           { return nil }
func regular(info os.FileInfo) bool          { return info.Mode().IsRegular() }
