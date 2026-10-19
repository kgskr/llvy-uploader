package main

import (
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/ui"
	"os"
)

var version = "dev"
var commit = "uncommitted"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--check-config" {
		origin, development := desktop.BuildOrigin()
		if desktop.ValidateOrigin(origin, development) != nil {
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--self-test" {
		if err := ui.SelfTest(); err != nil {
			os.Exit(1)
		}
		return
	}
	if err := ui.Run(version, commit); err != nil {
		ui.Alert(err.Error())
		os.Exit(1)
	}
}
