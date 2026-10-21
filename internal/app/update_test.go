package app

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in             string
		maj, min, pat  int
		ok             bool
	}{
		{"v0.3.0", 0, 3, 0, true},
		{"v1.2.3", 1, 2, 3, true},
		{"0.3.0", 0, 3, 0, true},
		{"v0.3.0-dev", 0, 3, 0, true},  // pre-release stripped
		{"v0.3.0-rc.1", 0, 3, 0, true}, // rc suffix stripped
		{"unknown", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"v1", 0, 0, 0, false},
		{"v1.2", 0, 0, 0, false},
	}
	for _, tc := range cases {
		maj, min, pat, ok := parseSemver(tc.in)
		if ok != tc.ok {
			t.Errorf("parseSemver(%q) ok=%v, want %v", tc.in, ok, tc.ok)
			continue
		}
		if ok && (maj != tc.maj || min != tc.min || pat != tc.pat) {
			t.Errorf("parseSemver(%q) = %d.%d.%d, want %d.%d.%d",
				tc.in, maj, min, pat, tc.maj, tc.min, tc.pat)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		// strict version progression
		{"v0.3.0", "v0.2.1", true},
		{"v0.3.1", "v0.3.0", true},
		{"v1.0.0", "v0.9.9", true},
		// already up to date
		{"v0.3.0", "v0.3.0", false},
		{"v0.2.1", "v0.3.0", false},
		// dev builds are always outdated relative to any published release
		{"v0.3.0", "0.3.0-dev", true},
		{"v0.3.0", "0.2.0-dev", true},
		// unrecognised current version treated as outdated
		{"v0.3.0", "unknown", true},
	}
	for _, tc := range cases {
		got := isNewer(tc.latest, tc.current)
		if got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func testSHA256Hex(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

func TestVerifySHA256MissingEntry(t *testing.T) {
	data := []byte("hello")
	sumsContent := testSHA256Hex(data) + "  other_file.tar.gz\n"
	if err := verifySHA256(data, "brain_v0.3.0_linux_amd64.tar.gz", sumsContent); err == nil {
		t.Error("expected error for missing filename entry, got nil")
	}
}

func TestVerifySHA256RoundTrip(t *testing.T) {
	data := []byte("brain binary placeholder")
	filename := "brain_v0.3.0_linux_amd64.tar.gz"
	sumsContent := testSHA256Hex(data) + "  " + filename + "\n"
	if err := verifySHA256(data, filename, sumsContent); err != nil {
		t.Errorf("round-trip failed: %v", err)
	}
}

func TestVerifySHA256Mismatch(t *testing.T) {
	data := []byte("correct data")
	filename := "brain_v0.3.0_linux_amd64.tar.gz"
	badHash := testSHA256Hex([]byte("other data"))
	sumsContent := badHash + "  " + filename + "\n"
	if err := verifySHA256(data, filename, sumsContent); err == nil {
		t.Error("expected mismatch error, got nil")
	}
}
