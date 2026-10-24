package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Root string `json:"root"`
}

func Home() (string, error) {
	h := os.Getenv("BRAIN_HOME")
	if h == "" {
		user, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		h = filepath.Join(user, ".brain")
	}
	return filepath.Abs(h)
}

func ProjectFor(cwd string) (Project, error) {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return Project{}, err
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = root
	if b, e := cmd.Output(); e == nil {
		root = strings.TrimSpace(string(b))
	}
	// Preserve the original Node project hash, including Git's slash-separated Windows path.
	sum := sha256.Sum256([]byte(root))
	return Project{ID: hex.EncodeToString(sum[:])[:16], Name: filepath.Base(root), Root: root}, nil
}
