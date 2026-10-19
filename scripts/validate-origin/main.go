// Command validate-origin validates build configuration without making requests.
package main

import (
	"fmt"
	"llvy-uploader/internal/desktop"
	"os"
)

func main() {
	if err := desktop.ValidateOrigin(os.Getenv("LLVY_SERVER_ORIGIN"), false); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
