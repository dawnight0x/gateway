//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"local-ai-gateway/internal/tray"
)

func main() {
	output := filepath.FromSlash("cmd/gateway/app.ico")
	if len(os.Args) > 1 {
		output = os.Args[1]
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(output, tray.IconICO(), 0o644); err != nil {
		panic(err)
	}
	fmt.Println(output)
}
