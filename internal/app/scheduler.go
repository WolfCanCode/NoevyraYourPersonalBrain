package app

import (
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func schedulerFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "LaunchAgents", "dev.personal-ai-brain.daily.plist"), nil
	case "linux":
		return filepath.Join(home, ".config", "systemd", "user", "personal-ai-brain.timer"), nil
	}
	return "", nil
}
func schedulerStatus() map[string]any {
	file, err := schedulerFile()
	configured := false
	if file != "" {
		_, e := os.Stat(file)
		configured = e == nil
	}
	v := map[string]any{"platform": runtime.GOOS, "definition": file, "configured": configured, "note": "Definition presence does not prove the timer is active."}
	if err != nil {
		v["error"] = err.Error()
	}
	if runtime.GOOS == "windows" {
		v["note"] = "Configure Task Scheduler manually with the native binary and sync argument."
	}
	return v
}
func unitQuote(s string) (string, error) {
	if strings.ContainsAny(s, "\x00\r\n") {
		return "", errors.New("scheduler paths must not contain NUL or newlines")
	}
	// systemd specifiers and ExecStart environment substitution use separate escaping.
	s = strings.ReplaceAll(s, "%", "%%")
	s = strings.ReplaceAll(s, "$", "$$")
	return strconv.Quote(s), nil
}
func installScheduler(brainHome string) (map[string]any, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, err
	}
	file, err := schedulerFile()
	if err != nil {
		return nil, err
	}
	if file == "" {
		return map[string]any{"installed": false, "message": "Configure Windows Task Scheduler manually: brain.exe sync (set BRAIN_HOME if needed)."}, nil
	}
	if err = os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return nil, err
	}
	var command string
	var args []string
	if runtime.GOOS == "darwin" {
		esc := html.EscapeString
		plist := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>dev.personal-ai-brain.daily</string><key>ProgramArguments</key><array><string>` + esc(exe) + `</string><string>sync</string></array><key>EnvironmentVariables</key><dict><key>BRAIN_HOME</key><string>` + esc(brainHome) + `</string></dict><key>StartCalendarInterval</key><dict><key>Hour</key><integer>8</integer><key>Minute</key><integer>0</integer></dict><key>StandardOutPath</key><string>` + esc(filepath.Join(brainHome, "sync.log")) + `</string><key>StandardErrorPath</key><string>` + esc(filepath.Join(brainHome, "sync.error.log")) + `</string></dict></plist>`
		if err = os.WriteFile(file, []byte(plist), 0600); err != nil {
			return nil, err
		}
		command = "launchctl"
		args = []string{"bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), file}
	} else {
		quotedExe, e := unitQuote(exe)
		if e != nil {
			return nil, e
		}
		// Environment= does not perform ExecStart's $ substitution.
		if strings.ContainsAny(brainHome, "\x00\r\n") {
			return nil, errors.New("invalid BRAIN_HOME for systemd")
		}
		env := strconv.Quote("BRAIN_HOME=" + strings.ReplaceAll(brainHome, "%", "%%"))
		service := "[Unit]\nDescription=Personal AI Brain local maintenance\n[Service]\nType=oneshot\nEnvironment=" + env + "\nExecStart=" + quotedExe + " sync\n"
		if err = os.WriteFile(filepath.Join(filepath.Dir(file), "personal-ai-brain.service"), []byte(service), 0600); err != nil {
			return nil, err
		}
		timer := "[Unit]\nDescription=Daily Personal AI Brain maintenance\n[Timer]\nOnCalendar=*-*-* 08:00:00\nPersistent=true\n[Install]\nWantedBy=timers.target\n"
		if err = os.WriteFile(file, []byte(timer), 0600); err != nil {
			return nil, err
		}
		if text, e := external("systemctl", []string{"--user", "daemon-reload"}); e != nil {
			return map[string]any{"installed": false, "definition": file, "message": text, "error": e.Error()}, nil
		}
		command = "systemctl"
		args = []string{"--user", "enable", "--now", "personal-ai-brain.timer"}
	}
	// Only invoked by explicit `init --schedule`. Existing schedules are never silently replaced on startup.
	if _, err = exec.LookPath(command); err != nil {
		return map[string]any{"installed": false, "definition": file, "error": err.Error()}, nil
	}
	text, err := external(command, args)
	result := map[string]any{"installed": err == nil, "definition": file, "message": "Daily local maintenance at 08:00 local time.", "output": text}
	if err != nil {
		result["error"] = err.Error()
	}
	return result, nil
}
