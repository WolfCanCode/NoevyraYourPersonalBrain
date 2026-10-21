package main

import (
	"errors"
	"fmt"
	"github.com/WolfCanCode/personal-ai-brain/internal/app"
	"os"
	"os/exec"
)

// Injected by release builds; local builds stay identifiable.
var version = "0.7.1-dev"
var commit = "unknown"
var builtAt = "unknown"

func main() {
	if err := app.Run(os.Args[1:], version, commit, builtAt, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "brain:", err)
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			os.Exit(exit.ExitCode())
		}
		os.Exit(1)
	}
}
