package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type brokenWriter struct{ err error }

func (w brokenWriter) Write([]byte) (int, error) { return 0, w.err }

func TestArchivePropagatesWriterFailure(t *testing.T) {
	for _, windows := range []bool{false, true} {
		_, binary := fixture(t)
		info, err := os.Stat(binary)
		if err != nil {
			t.Fatal(err)
		}
		want := errors.New("disk full")
		err = writeArchive(brokenWriter{want}, []archiveEntry{{path: binary, name: "brain", info: info}}, windows)
		if !errors.Is(err, want) {
			t.Fatalf("windows=%v: got %v, want %v", windows, err, want)
		}
	}
}

func fixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"brain": "executable", "LICENSE": "license", "README.md": "readme", "docs/RELEASING.md": "releasing"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root, filepath.Join(root, "brain")
}

func TestPackageContentsAndChecksum(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}} {
		t.Run(target.os+"-"+target.arch, func(t *testing.T) {
			root, binary := fixture(t)
			archive, err := Package(root, binary, "v1.2.3-rc.1", target.os, target.arch)
			if err != nil {
				t.Fatal(err)
			}
			ext, exe := ".tar.gz", "brain"
			if target.os == "windows" {
				ext, exe = ".zip", "brain.exe"
			}
			if filepath.Base(archive) != "brain_v1.2.3-rc.1_"+target.os+"_"+target.arch+ext {
				t.Fatal(archive)
			}
			contents := map[string]string{}
			if target.os == "windows" {
				z, err := zip.OpenReader(archive)
				if err != nil {
					t.Fatal(err)
				}
				defer z.Close()
				for _, f := range z.File {
					r, err := f.Open()
					if err != nil {
						t.Fatal(err)
					}
					b, err := io.ReadAll(r)
					r.Close()
					if err != nil {
						t.Fatal(err)
					}
					contents[f.Name] = string(b)
				}
			} else {
				f, err := os.Open(archive)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				gz, err := gzip.NewReader(f)
				if err != nil {
					t.Fatal(err)
				}
				defer gz.Close()
				r := tar.NewReader(gz)
				for {
					h, err := r.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
					b, err := io.ReadAll(r)
					if err != nil {
						t.Fatal(err)
					}
					contents[h.Name] = string(b)
					if h.Name == exe && h.Mode&0111 == 0 {
						t.Fatal("binary is not executable")
					}
				}
			}
			want := map[string]string{exe: "executable", "LICENSE": "license", "README.md": "readme", "RELEASING.md": "releasing"}
			if len(contents) != len(want) {
				t.Fatalf("contents: %v", contents)
			}
			for name, body := range want {
				if contents[name] != body {
					t.Errorf("%s: got %q, want %q", name, contents[name], body)
				}
			}
			data, err := os.ReadFile(archive)
			if err != nil {
				t.Fatal(err)
			}
			checksum, err := os.ReadFile(archive + ".sha256")
			if err != nil {
				t.Fatal(err)
			}
			if string(checksum) != fmt.Sprintf("%x  %s\n", sha256.Sum256(data), filepath.Base(archive)) {
				t.Fatal("checksum mismatch")
			}
		})
	}
}

func TestPackageRejectsInvalidInput(t *testing.T) {
	for _, args := range [][3]string{{"../escape", "linux", "amd64"}, {"", "linux", "amd64"}, {"v1", "../linux", "amd64"}, {"v1", "linux", "../amd64"}, {"v1", "windows", "arm64"}} {
		root, binary := fixture(t)
		if _, err := Package(root, binary, args[0], args[1], args[2]); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestPackageMissingInputLeavesNoArtifacts(t *testing.T) {
	root, binary := fixture(t)
	if err := os.Remove(filepath.Join(root, "LICENSE")); err != nil {
		t.Fatal(err)
	}
	if _, err := Package(root, binary, "v1", "linux", "amd64"); err == nil {
		t.Fatal("missing input accepted")
	}
	entries, err := os.ReadDir(filepath.Join(root, "dist"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "brain_") {
			t.Errorf("partial artifact: %s", e.Name())
		}
	}
}
