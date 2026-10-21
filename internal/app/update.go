package app

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepo    = "WolfCanCode/personal-ai-brain"
	updateTimeout = 30 * time.Second
	updateMaxBody = 32 * 1024 * 1024 // 32 MiB
)

type releaseInfo struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// runUpdate implements "brain update" and "brain update --check".
//
//	--check   report whether an update is available without installing it
func runUpdate(currentVersion string, args []string, out, errOut io.Writer) error {
	_, f, err := flags(args, nil, []string{"--check"})
	if err != nil {
		return err
	}
	checkOnly := f["--check"] != ""

	fmt.Fprintln(out, "Checking for updates…")
	latest, err := fetchLatestTag()
	if err != nil {
		return fmt.Errorf("could not fetch release information: %w", err)
	}

	fmt.Fprintf(out, "Current version : %s\n", currentVersion)
	fmt.Fprintf(out, "Latest release  : %s\n", latest)

	if !isNewer(latest, currentVersion) {
		fmt.Fprintln(out, "Noevyra is up to date.")
		return nil
	}

	fmt.Fprintf(out, "Update available: %s → %s\n", currentVersion, latest)
	if checkOnly {
		fmt.Fprintln(out, "Run 'brain update' (without --check) to install.")
		return nil
	}

	return installUpdate(latest, out)
}

// fetchLatestTag returns the tag name of the latest non-draft, non-prerelease GitHub Release.
func fetchLatestTag() (string, error) {
	url := "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	client := &http.Client{Timeout: updateTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "personal-ai-brain/updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotFound:
		return "", errors.New("no releases have been published yet")
	case http.StatusOK:
		// ok
	default:
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	var info releaseInfo
	if err = json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&info); err != nil {
		return "", fmt.Errorf("parse release response: %w", err)
	}
	if info.Draft || info.Prerelease {
		return "", errors.New("latest release is a draft or pre-release; specify a tag with BRAIN_VERSION")
	}
	if info.TagName == "" {
		return "", errors.New("empty tag_name in release response")
	}
	return info.TagName, nil
}

// isNewer returns true when latestTag is strictly newer than currentVersion.
// A -dev build is always considered outdated relative to any published release.
func isNewer(latestTag, current string) bool {
	if strings.Contains(current, "-dev") || strings.Contains(current, "unknown") {
		_, _, _, ok := parseSemver(latestTag)
		return ok
	}
	lmaj, lmin, lpat, lok := parseSemver(latestTag)
	cmaj, cmin, cpat, cok := parseSemver(current)
	if !lok || !cok {
		return latestTag != current
	}
	if lmaj != cmaj {
		return lmaj > cmaj
	}
	if lmin != cmin {
		return lmin > cmin
	}
	return lpat > cpat
}

func parseSemver(v string) (major, minor, patch int, ok bool) {
	v = strings.TrimPrefix(v, "v")
	v = strings.SplitN(v, "-", 2)[0]
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	maj, e1 := strconv.Atoi(parts[0])
	min, e2 := strconv.Atoi(parts[1])
	pat, e3 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return 0, 0, 0, false
	}
	return maj, min, pat, true
}

// installUpdate downloads the release archive for the running OS/arch, verifies its
// SHA-256 checksum, extracts the brain binary, and atomically replaces the current
// executable on disk. It does not restart the process.
func installUpdate(tag string, out io.Writer) error {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	assetName := fmt.Sprintf("brain_%s_%s_%s%s", tag, goos, arch, ext)
	base := "https://github.com/" + updateRepo + "/releases/download/" + tag
	archiveURL := base + "/" + assetName
	sumsURL := base + "/SHA256SUMS"

	client := &http.Client{Timeout: 120 * time.Second}

	fmt.Fprintf(out, "Downloading %s…\n", assetName)
	archiveData, err := fetchBody(client, archiveURL)
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	fmt.Fprintln(out, "Verifying checksum…")
	sumsData, err := fetchBody(client, sumsURL)
	if err != nil {
		return fmt.Errorf("download SHA256SUMS: %w", err)
	}
	if err = verifySHA256(archiveData, assetName, string(sumsData)); err != nil {
		return fmt.Errorf("checksum mismatch: %w", err)
	}

	fmt.Fprintln(out, "Extracting binary…")
	binName := "brain"
	if goos == "windows" {
		binName = "brain.exe"
	}
	var binData []byte
	if goos == "windows" {
		binData, err = extractFromZip(archiveData, binName)
	} else {
		binData, err = extractFromTarGz(archiveData, binName)
	}
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}
	if len(binData) == 0 {
		return fmt.Errorf("%s not found in %s", binName, assetName)
	}

	return replaceBinary(binData, out)
}

func fetchBody(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, updateMaxBody))
}

func verifySHA256(data []byte, filename, sumsContent string) error {
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	for _, line := range strings.Split(sumsContent, "\n") {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) == 2 && parts[1] == filename {
			expected := strings.ToLower(parts[0])
			if actual != expected {
				return fmt.Errorf("got %s, want %s", actual, expected)
			}
			return nil
		}
	}
	return fmt.Errorf("no entry for %s in SHA256SUMS", filename)
}

func extractFromTarGz(data []byte, target string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == target {
			return io.ReadAll(io.LimitReader(tr, updateMaxBody))
		}
	}
	return nil, nil
}

func extractFromZip(data []byte, target string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == target {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, updateMaxBody))
		}
	}
	return nil, nil
}

// replaceBinary writes data to a temp file in the same directory as the current
// executable, sets permissions, then atomically renames it into place.
// On Windows, where renaming over a running executable fails, it first moves the
// current binary to a .old file before placing the new one.
func replaceBinary(data []byte, out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolve executable symlink: %w", err)
	}
	dir := filepath.Dir(exe)

	tmp, err := os.CreateTemp(dir, ".brain-update-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	// Ensure temp is cleaned up if we exit early.
	removeTemp := func() {
		if _, e := os.Stat(tmpPath); e == nil {
			os.Remove(tmpPath)
		}
	}
	defer removeTemp()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write update: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return err
	}

	// Set executable bit on Unix.
	if runtime.GOOS != "windows" {
		if err = os.Chmod(tmpPath, 0755); err != nil {
			return err
		}
	}

	// Atomic replace. On Windows, rename over a running exe is not permitted, so we
	// first move the current binary out of the way.
	if runtime.GOOS == "windows" {
		oldPath := exe + ".old"
		_ = os.Remove(oldPath) // best-effort cleanup of any leftover
		if err = os.Rename(exe, oldPath); err != nil {
			return fmt.Errorf("move current binary aside: %w", err)
		}
	}
	if err = os.Rename(tmpPath, exe); err != nil {
		return fmt.Errorf("install new binary: %w", err)
	}

	fmt.Fprintf(out, "Updated Noevyra to %s at %s\nRestart any running 'brain mcp' processes to use the new version.\n", exe, exe)
	return nil
}
