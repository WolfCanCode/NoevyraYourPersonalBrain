package app

import (
	"fmt"
	"testing"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
)

func TestSetupKeywordsProjectTypes(t *testing.T) {
	cases := []struct {
		label        string
		profile      intelligence.ProjectProfile
		wantFirst    string   // most specific keyword must be first
		wantIncludes []string // all of these must appear
		wantAbsent   []string // none of these should appear
	}{
		{
			label: "Java/Spring backend",
			profile: intelligence.ProjectProfile{
				Languages: []string{"html", "java", "javascript", "shell"},
				Stacks:    []string{"jvm", "spring"},
				Signals:   []string{"containers", "git", "github", "web"},
			},
			wantFirst:    "spring",
			wantIncludes: []string{"spring", "java", "containers"},
			wantAbsent:   []string{"javascript"}, // excluded because java is present
		},
		{
			label: "Angular/TypeScript frontend",
			profile: intelligence.ProjectProfile{
				Languages: []string{"css", "html", "typescript"},
				Stacks:    []string{},
				Signals:   []string{"angular", "e2e", "git", "github", "web"},
			},
			wantFirst:    "angular",
			wantIncludes: []string{"angular", "typescript"},
			wantAbsent:   []string{"javascript"}, // excluded because typescript is present
		},
		{
			label: "Go backend with Gin",
			profile: intelligence.ProjectProfile{
				Languages: []string{"go"},
				Stacks:    []string{"gin"},
				Signals:   []string{"api", "database", "git"},
			},
			wantFirst:    "gin",
			wantIncludes: []string{"gin", "go", "api", "database"},
			wantAbsent:   []string{},
		},
		{
			label: "Python/Django",
			profile: intelligence.ProjectProfile{
				Languages: []string{"python"},
				Stacks:    []string{"django"},
				Signals:   []string{"database", "git", "github"},
			},
			wantFirst:    "django",
			wantIncludes: []string{"django", "python", "database"},
			wantAbsent:   []string{"javascript"},
		},
	}
	for _, tc := range cases {
		got := setupKeywords(tc.profile)
		gotSet := map[string]bool{}
		for _, k := range got {
			gotSet[k] = true
		}
		fmt.Printf("%-40s → %v\n", tc.label, got)
		if len(got) == 0 {
			t.Errorf("%s: got no keywords", tc.label)
			continue
		}
		if got[0] != tc.wantFirst {
			t.Errorf("%s: first keyword = %q, want %q", tc.label, got[0], tc.wantFirst)
		}
		for _, want := range tc.wantIncludes {
			if !gotSet[want] {
				t.Errorf("%s: expected %q in keywords %v", tc.label, want, got)
			}
		}
		for _, absent := range tc.wantAbsent {
			if gotSet[absent] {
				t.Errorf("%s: %q should not appear in keywords %v", tc.label, absent, got)
			}
		}
	}
}
