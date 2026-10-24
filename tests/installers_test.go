package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPOSIXInstaller(t *testing.T) {
	script, err := os.ReadFile("../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if sh, err := exec.LookPath("sh"); err == nil {
		cmd := exec.Command(sh, "-n")
		cmd.Stdin = strings.NewReader(string(script))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("shell syntax: %v\n%s", err, out)
		}
	} else {
		t.Log("sh unavailable; POSIX syntax check runs on Unix CI runners")
	}
	for _, text := range []string{"SHA256SUMS", "releases/download"} {
		if !strings.Contains(string(script), text) {
			t.Errorf("missing %q", text)
		}
	}
	if !strings.Contains(string(script), "sha256sum") && !strings.Contains(string(script), "shasum") {
		t.Error("checksum verification missing")
	}
	for _, word := range strings.Fields(string(script)) {
		if word == "sudo" {
			t.Error("installer requires sudo")
		}
	}
}

func TestPowerShellInstaller(t *testing.T) {
	script, err := os.ReadFile("../install.ps1")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Get-FileHash -Algorithm SHA256", "brain_${Version}_windows_amd64.zip", "releases/download"} {
		if !strings.Contains(string(script), text) {
			t.Errorf("missing %q", text)
		}
	}
}
