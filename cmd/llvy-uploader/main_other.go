//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "LLVY Uploader 화면은 Windows x64에서 실행하세요. 이 호스트에서는 go test ./...를 사용할 수 있습니다.")
	os.Exit(1)
}
