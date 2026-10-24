// Package release creates the archives consumed by Brain's installers.
package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var versionPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.+-]*$`)

type archiveEntry struct {
	path string
	name string
	info os.FileInfo
}

// Package archives a tested executable and documentation and writes its SHA-256
// sidecar. root is the repository root; binary may be an absolute or relative path.
func Package(root, binary, version, goos, arch string) (string, error) {
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid version %q", version)
	}
	exe, ext, err := archiveFormat(goos, arch)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(binary) {
		binary = filepath.Join(root, binary)
	}
	entries := []archiveEntry{
		{path: binary, name: exe},
		{path: filepath.Join(root, "LICENSE"), name: "LICENSE"},
		{path: filepath.Join(root, "README.md"), name: "README.md"},
		{path: filepath.Join(root, "docs", "RELEASING.md"), name: "RELEASING.md"},
	}
	for i := range entries {
		info, err := os.Stat(entries[i].path)
		if err != nil {
			return "", fmt.Errorf("stat %s: %w", entries[i].path, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("not a regular file: %s", entries[i].path)
		}
		entries[i].info = info
	}
	out := filepath.Join(root, "dist")
	if err := os.MkdirAll(out, 0755); err != nil {
		return "", err
	}
	archive := filepath.Join(out, fmt.Sprintf("brain_%s_%s_%s%s", version, goos, arch, ext))
	f, err := os.CreateTemp(out, ".brain-archive-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	hash := sha256.New()
	if err := writeArchive(io.MultiWriter(f, hash), entries, goos == "windows"); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(f.Name(), 0644); err != nil {
		return "", err
	}
	if err := os.Rename(f.Name(), archive); err != nil {
		return "", err
	}
	checksum := fmt.Sprintf("%x  %s\n", hash.Sum(nil), filepath.Base(archive))
	if err := os.WriteFile(archive+".sha256", []byte(checksum), 0644); err != nil {
		os.Remove(archive)
		return "", err
	}
	return archive, nil
}

// archiveFormat validates the supported release matrix in one place.
func archiveFormat(goos, arch string) (string, string, error) {
	if goos == "windows" && arch == "amd64" {
		return "brain.exe", ".zip", nil
	}
	if (goos == "linux" || goos == "darwin") && (arch == "amd64" || arch == "arm64") {
		return "brain", ".tar.gz", nil
	}
	return "", "", fmt.Errorf("unsupported target %s/%s", goos, arch)
}

func writeArchive(out io.Writer, entries []archiveEntry, windows bool) error {
	if windows {
		return writeZIP(out, entries)
	}
	return writeTarGz(out, entries)
}

func writeZIP(out io.Writer, entries []archiveEntry) error {
	writer := zip.NewWriter(out)
	defer writer.Close()
	for _, entry := range entries {
		if err := writeZIPEntry(writer, entry); err != nil {
			return fmt.Errorf("archive %s: %w", entry.name, err)
		}
	}
	return writer.Close()
}

func writeZIPEntry(writer *zip.Writer, entry archiveEntry) error {
	header, err := zip.FileInfoHeader(entry.info)
	if err != nil {
		return err
	}
	header.Name, header.Method = entry.name, zip.Deflate
	out, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	return copyEntry(entry, out)
}

func writeTarGz(out io.Writer, entries []archiveEntry) error {
	gz := gzip.NewWriter(out)
	defer gz.Close()
	writer := tar.NewWriter(gz)
	defer writer.Close()
	for _, entry := range entries {
		if err := writeTarEntry(writer, entry); err != nil {
			return fmt.Errorf("archive %s: %w", entry.name, err)
		}
	}
	tarErr := writer.Close()
	gzipErr := gz.Close()
	if tarErr != nil {
		return tarErr
	}
	return gzipErr
}

func writeTarEntry(writer *tar.Writer, entry archiveEntry) error {
	header, err := tar.FileInfoHeader(entry.info, "")
	if err != nil {
		return err
	}
	header.Name, header.Mode = entry.name, 0644
	if entry.name == "brain" {
		header.Mode = 0755
	}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	return copyEntry(entry, writer)
}

func copyEntry(entry archiveEntry, out io.Writer) error {
	file, err := os.Open(entry.path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(out, file)
	return err
}
