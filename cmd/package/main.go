// Command package creates a release archive from an already tested executable.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/WolfCanCode/personal-ai-brain/internal/release"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/package <binary> <version> <os> <arch> (from repository root)")
		os.Exit(1)
	}
	archive, err := release.Package(".", os.Args[1], os.Args[2], os.Args[3], os.Args[4])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(filepath.Base(archive))
}
