// Command inspect verifies the PE envelope without executing a Windows binary.
package main

import (
	"crypto/sha256"
	"debug/buildinfo"
	"debug/pe"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/inspect <exe>")
		os.Exit(1)
	}
	if err := inspect(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func inspect(path string) error {
	f, err := pe.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	optional, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || optional.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		return fmt.Errorf("expected Windows amd64 GUI executable")
	}
	if f.Section(".rsrc") == nil {
		return fmt.Errorf("missing Windows resources")
	}
	symbols, err := f.ImportedSymbols()
	if err != nil {
		return err
	}
	// ImportedLibraries is a stub in debug/pe; derive DLLs from named imports.
	seen := map[string]bool{}
	var dlls []string
	for _, symbol := range symbols {
		_, dll, ok := strings.Cut(symbol, ":")
		if !ok {
			return fmt.Errorf("invalid PE import")
		}
		dll = strings.ToLower(dll)
		if !seen[dll] {
			dlls = append(dlls, dll)
			seen[dll] = true
		}
	}
	if len(dlls) == 0 {
		return fmt.Errorf("no named PE imports found")
	}
	sort.Strings(dlls)
	for _, dll := range dlls {
		switch strings.ToLower(dll) {
		case "kernel32.dll", "user32.dll", "gdi32.dll", "shell32.dll", "ole32.dll", "advapi32.dll", "crypt32.dll", "ntdll.dll", "ws2_32.dll", "bcrypt.dll":
		default:
			return fmt.Errorf("unexpected imported DLL: %s", dll)
		}
	}
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Bytes        int
		SHA256       string
		ImportedDLLs []string
		GoVersion    string
		RuntimeDeps  any
	}{len(data), fmt.Sprintf("%x", sha256.Sum256(data)), dlls, info.GoVersion, info.Deps})
}
