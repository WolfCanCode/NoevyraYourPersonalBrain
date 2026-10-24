package app

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// maxExternalOutput caps subprocess output to avoid flooding the terminal or memory.
const maxExternalOutput = 64_000

// limitedBuffer is a bytes.Buffer capped at maxExternalOutput bytes.
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := maxExternalOutput - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}

// external runs bin with args, caps combined stdout+stderr, and enforces a timeout.
// No shell interpolation — args are passed directly to exec.
func external(bin string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	var buf limitedBuffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
