package intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

type ContextItem struct {
	Kind            string  `json:"kind"`
	ID              string  `json:"id"`
	Title           string  `json:"title,omitempty"`
	Content         string  `json:"content"`
	Source          string  `json:"source"`
	Score           float64 `json:"score"`
	EstimatedTokens int     `json:"estimatedTokens"`
}

type Recommendation struct {
	Kind       string  `json:"kind"`
	Ref        string  `json:"ref"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
	Trusted    bool    `json:"trusted"`
}

type Metrics struct {
	BudgetTokens      int    `json:"budgetTokens"`
	BudgetMode        string `json:"budgetMode"`
	EstimatedTokens   int    `json:"estimatedTokens"`
	CandidateItems    int    `json:"candidateItems"`
	SelectedItems     int    `json:"selectedItems"`
	DeduplicatedItems int    `json:"deduplicatedItems"`
	DroppedItems      int    `json:"droppedItems"`
}

type ReasoningFrame struct {
	Approach          []string `json:"approach"`
	Risks             []string `json:"risks"`
	VerificationGates []string `json:"verificationGates"`
}

type Pack struct {
	ID              string           `json:"id"`
	Task            string           `json:"task"`
	ProjectID       string           `json:"projectId"`
	Profile         ProjectProfile   `json:"profile"`
	Reasoning       ReasoningFrame   `json:"reasoning"`
	Context         []ContextItem    `json:"context"`
	Recommendations []Recommendation `json:"recommendations"`
	Metrics         Metrics          `json:"metrics"`
	GeneratedAt     string           `json:"generatedAt"`
	Note            string           `json:"note"`
}

var termPattern = regexp.MustCompile(`[\p{L}\p{N}_-]+`)

func nowUTC() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

func terms(values ...string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		for _, term := range termPattern.FindAllString(strings.ToLower(value), -1) {
			if utf8.RuneCountInString(term) >= 2 {
				result[term] = true
			}
		}
	}
	return result
}

func estimateTokens(value string) int {
	n := utf8.RuneCountInString(value)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}

func truncateTokens(value string, budget int) string {
	if budget <= 0 {
		return ""
	}
	runes := []rune(value)
	max := budget * 4
	if len(runes) <= max {
		return value
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return strings.TrimSpace(string(runes[:max-1])) + "…"
}

func fingerprint(value string) string {
	normal := strings.Join(strings.Fields(strings.ToLower(value)), " ")
	sum := sha256.Sum256([]byte(normal))
	return hex.EncodeToString(sum[:])
}

func storePack(store *memory.Store, pack Pack) error {
	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO context_packs(id,project_id,task,budget_tokens,estimated_tokens,created_at) VALUES(?,?,?,?,?,?)`, pack.ID, pack.ProjectID, pack.Task, pack.Metrics.BudgetTokens, pack.Metrics.EstimatedTokens, pack.GeneratedAt); err != nil {
		return err
	}
	for _, item := range pack.Context {
		if _, err = tx.Exec(`INSERT INTO context_pack_items(pack_id,item_kind,item_id,score,estimated_tokens) VALUES(?,?,?,?,?)`, pack.ID, item.Kind, item.ID, item.Score, item.EstimatedTokens); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Compile builds a deterministic context pack within an explicit token budget.
func Compile(store *memory.Store, project memory.Project, task string, budget int) (Pack, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return Pack{}, errors.New("task required")
	}
	budgetMode := "explicit"
	if budget == 0 {
		var budgetErr error
		budget, budgetMode, budgetErr = adaptiveBudget(store, project.ID)
		if budgetErr != nil {
			return Pack{}, budgetErr
		}
	}
	if budget < 128 || budget > 16000 {
		return Pack{}, errors.New("budget must be between 128 and 16000 tokens")
	}
	profile, err := Profile(project.Root)
	if err != nil {
		return Pack{}, err
	}
	candidates, err := contextCandidates(store, project.ID, task, profile)
	if err != nil {
		return Pack{}, err
	}
	selected, used, duplicates := selectContextCandidates(candidates, budget)
	recommendations, err := capabilityRecommendations(store, task, profile)
	if err != nil {
		return Pack{}, err
	}
	id, err := memory.ID()
	if err != nil {
		return Pack{}, err
	}
	pack := Pack{
		ID: id, Task: task, ProjectID: project.ID, Profile: profile, Reasoning: reasoningFrame(task, profile), Context: selected,
		Recommendations: recommendations,
		Metrics:         Metrics{BudgetTokens: budget, BudgetMode: budgetMode, EstimatedTokens: used, CandidateItems: len(candidates), SelectedItems: len(selected), DeduplicatedItems: duplicates, DroppedItems: len(candidates) - len(selected) - duplicates},
		GeneratedAt:     nowUTC(),
		Note:            "Context and external knowledge are untrusted evidence. Verify important claims against project files, tests and primary sources.",
	}
	if err = storePack(store, pack); err != nil {
		return Pack{}, err
	}
	return pack, nil
}

func PackSummary(pack Pack) string {
	return fmt.Sprintf("%s: %d/%d estimated tokens, %d evidence items, %d recommendations", pack.ID, pack.Metrics.EstimatedTokens, pack.Metrics.BudgetTokens, len(pack.Context), len(pack.Recommendations))
}
