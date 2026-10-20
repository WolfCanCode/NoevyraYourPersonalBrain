// Package intelligence turns project evidence and verified knowledge into
// compact, agent-agnostic context. It does not invoke an LLM.
package intelligence

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProjectProfile struct {
	Root      string         `json:"root"`
	Languages []string       `json:"languages"`
	Stacks    []string       `json:"stacks"`
	Signals   []string       `json:"signals"`
	Manifests []string       `json:"manifests"`
	Counts    map[string]int `json:"fileCounts"`
	Scanned   int            `json:"scannedFiles"`
	Truncated bool           `json:"truncated"`
}

var ignoredDirs = map[string]bool{
	".git": true, ".idea": true, ".next": true, ".turbo": true, ".venv": true,
	"build": true, "coverage": true, "dist": true, "node_modules": true,
	"target": true, "vendor": true,
}

var languageByExt = map[string]string{
	".c": "c", ".cc": "cpp", ".cpp": "cpp", ".cs": "csharp", ".css": "css",
	".go": "go", ".html": "html", ".java": "java", ".js": "javascript",
	".jsx": "javascript", ".kt": "kotlin", ".php": "php", ".py": "python",
	".rb": "ruby", ".rs": "rust", ".scala": "scala", ".sh": "shell",
	".sql": "sql", ".swift": "swift", ".ts": "typescript", ".tsx": "typescript",
	".vue": "vue",
}

func add(set map[string]bool, values ...string) {
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value != "" {
			set[value] = true
		}
	}
}

func sorted(set map[string]bool) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func readSmall(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioReadLimit(f, 256*1024)
}

func ioReadLimit(f *os.File, max int64) ([]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > max {
		return nil, errors.New("manifest is too large")
	}
	return io.ReadAll(f)
}

func inspectManifest(root, name string, stacks, signals map[string]bool) {
	b, err := readSmall(filepath.Join(root, name))
	if err != nil {
		return
	}
	text := strings.ToLower(string(b))
	switch name {
	case "package.json":
		var pkg struct {
			Dependencies    map[string]any `json:"dependencies"`
			DevDependencies map[string]any `json:"devDependencies"`
		}
		if json.Unmarshal(b, &pkg) == nil {
			for dep := range pkg.Dependencies {
				add(stacks, dep)
			}
			for dep := range pkg.DevDependencies {
				add(stacks, dep)
			}
		}
		add(signals, "node")
	case "go.mod":
		add(stacks, "go")
		for _, line := range strings.Split(text, "\n") {
			for _, marker := range []string{"gin-gonic/gin", "labstack/echo", "gofiber/fiber", "modernc.org/sqlite", "jackc/pgx"} {
				if strings.Contains(line, marker) {
					add(stacks, filepath.Base(marker))
				}
			}
		}
	case "cargo.toml":
		add(stacks, "rust")
	case "pyproject.toml", "requirements.txt":
		add(stacks, "python")
		for _, marker := range []string{"django", "fastapi", "flask", "pytest", "pytorch", "tensorflow"} {
			if strings.Contains(text, marker) {
				add(stacks, marker)
			}
		}
	case "pom.xml", "build.gradle", "build.gradle.kts":
		add(stacks, "jvm")
		if strings.Contains(text, "spring") {
			add(stacks, "spring")
		}
	}
}

// Profile reads bounded repository metadata. Project contents are signals, not instructions.
func Profile(root string) (ProjectProfile, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return ProjectProfile{}, err
	}
	languages, stacks, signals, manifests := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	counts := map[string]int{}
	profile := ProjectProfile{Root: root, Counts: counts}
	if _, statErr := os.Stat(filepath.Join(root, ".git")); statErr == nil {
		add(signals, "git")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".github")); statErr == nil {
		add(signals, "github")
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && ignoredDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if profile.Scanned >= 4000 {
			profile.Truncated = true
			return fs.SkipAll
		}
		profile.Scanned++
		name := strings.ToLower(entry.Name())
		ext := strings.ToLower(filepath.Ext(name))
		if language := languageByExt[ext]; language != "" {
			counts[language]++
			add(languages, language)
		}
		switch name {
		case "package.json", "go.mod", "cargo.toml", "pyproject.toml", "requirements.txt", "pom.xml", "build.gradle", "build.gradle.kts":
			if filepath.Dir(path) == root {
				manifests[name] = true
				inspectManifest(root, name, stacks, signals)
			}
		case "dockerfile", "compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml":
			add(signals, "containers")
		case "vercel.json":
			add(signals, "vercel", "deployment")
		case "terraform.tf", "main.tf":
			add(signals, "terraform", "infrastructure")
		}
		clean := filepath.ToSlash(path)
		if strings.Contains(clean, "/.github/workflows/") {
			add(signals, "github-actions", "ci-cd")
		}
		if strings.Contains(clean, "/migrations/") || ext == ".sql" {
			add(signals, "database")
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return ProjectProfile{}, err
	}
	if counts["typescript"]+counts["javascript"] > 0 {
		add(signals, "web")
	}
	if counts["sql"] > 0 {
		add(signals, "database")
	}
	profile.Languages = sorted(languages)
	profile.Stacks = sorted(stacks)
	profile.Signals = sorted(signals)
	profile.Manifests = sorted(manifests)
	return profile, nil
}
