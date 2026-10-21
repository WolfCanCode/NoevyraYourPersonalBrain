package intelligence

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
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

func capabilityRecommendations(store *memory.Store, task string, profile ProjectProfile) ([]Recommendation, error) {
	all := []string{task}
	all = append(all, profile.Languages...)
	all = append(all, profile.Stacks...)
	all = append(all, profile.Signals...)
	wanted := terms(all...)
	recommendations := []Recommendation{}
	seen := map[string]bool{}
	for _, item := range ProvisionCatalog {
		hits := []string{}
		for _, trigger := range item.TriggerSignals {
			if wanted[trigger] {
				hits = append(hits, trigger)
			}
		}
		if len(hits) == 0 {
			continue
		}
		confidence := math.Round(math.Min(0.95, 0.45+float64(len(hits))*0.12)*100) / 100
		recommendations = append(recommendations, Recommendation{item.Kind, item.Ref, item.Reason + " Signals: " + strings.Join(hits, ", ") + ".", confidence, false})
		seen[item.Kind+":"+item.Ref] = true
	}
	approved, err := store.Skills()
	if err != nil {
		return nil, err
	}
	for _, skill := range approved {
		keyword := strings.ToLower(strings.TrimSpace(skill.Keyword))
		if keyword == "" || (!wanted[keyword] && !strings.Contains(strings.ToLower(task), keyword)) {
			continue
		}
		key := "skill:" + skill.Ref
		if seen[key] {
			continue
		}
		recommendations = append(recommendations, Recommendation{"skill", skill.Ref, "Previously reviewed skill matches keyword " + keyword + ".", 0.95, true})
	}
	sort.SliceStable(recommendations, func(i, j int) bool { return recommendations[i].Confidence > recommendations[j].Confidence })
	if len(recommendations) > 6 {
		recommendations = recommendations[:6]
	}
	return recommendations, nil
}

func reasoningFrame(task string, profile ProjectProfile) ReasoningFrame {
	values := []string{task}
	values = append(values, profile.Languages...)
	values = append(values, profile.Stacks...)
	values = append(values, profile.Signals...)
	wanted := terms(values...)
	frame := ReasoningFrame{
		Approach: []string{
			"Inspect the smallest authoritative project evidence before choosing a change.",
			"Reuse existing conventions and define the minimal safe change boundary.",
			"Implement in reversible increments and preserve unrelated user work.",
		},
		Risks: []string{"Stale external guidance or memory may conflict with the current repository."},
		VerificationGates: []string{
			"Run the narrowest relevant checks, then the repository's required validation.",
			"Verify the exact artifact or user-visible path, not only helper functions.",
			"Record the outcome and any durable decision for the next agent.",
		},
	}
	if wanted["security"] || wanted["auth"] || wanted["payment"] || wanted["webhook"] {
		frame.Approach = append(frame.Approach, "Map trust boundaries and attacker-controlled inputs before implementation.")
		frame.Risks = append(frame.Risks, "Authentication, authorization, replay, secret handling and unsafe input paths require explicit checks.")
		frame.VerificationGates = append(frame.VerificationGates, "Exercise at least one negative or adversarial case.")
	}
	if wanted["database"] || wanted["sql"] || wanted["migration"] || wanted["sqlite"] || wanted["postgres"] {
		frame.Approach = append(frame.Approach, "Treat schema compatibility, transaction boundaries and rollback as part of the feature.")
		frame.Risks = append(frame.Risks, "Partial migrations, concurrent writers and irreversible data changes can invalidate a locally correct implementation.")
		frame.VerificationGates = append(frame.VerificationGates, "Test upgrade from the previous schema and reopen the resulting store.")
	}
	if wanted["api"] || wanted["mcp"] || wanted["contract"] {
		frame.Approach = append(frame.Approach, "Write the compatibility contract and error behavior before changing the adapter.")
		frame.Risks = append(frame.Risks, "Protocol drift and silently accepted invalid input create downstream agent failures.")
		frame.VerificationGates = append(frame.VerificationGates, "Test valid, invalid and backward-compatible protocol paths.")
	}
	if wanted["frontend"] || wanted["ui"] || wanted["web"] || wanted["browser"] {
		frame.Approach = append(frame.Approach, "Trace the user-visible flow across data, interaction, loading, error and empty states.")
		frame.VerificationGates = append(frame.VerificationGates, "Verify the final browser state at the intended viewport and interaction path.")
	}
	if wanted["release"] || wanted["deployment"] || wanted["ci-cd"] {
		frame.Risks = append(frame.Risks, "A source-level pass is insufficient if packaging, permissions or the deployed artifact differs.")
		frame.VerificationGates = append(frame.VerificationGates, "Validate checksums, packaged executables and publication state.")
	}
	return frame
}

func adaptiveBudget(store *memory.Store, project string) (int, string, error) {
	rows, err := store.DB.Query(`SELECT payload FROM events WHERE kind='context_feedback' AND project_id=? ORDER BY created_at DESC LIMIT 20`, project)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	used := []int{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return 0, "", err
		}
		var item struct {
			Outcome    string `json:"outcome"`
			TokensUsed int    `json:"tokensUsed"`
		}
		if json.Unmarshal([]byte(raw), &item) == nil && item.Outcome == "helpful" && item.TokensUsed >= 64 {
			used = append(used, item.TokensUsed)
		}
	}
	if err = rows.Err(); err != nil {
		return 0, "", err
	}
	if len(used) < 3 {
		return 1200, "default", nil
	}
	sort.Ints(used)
	budget := used[len(used)/2] * 5 / 4
	if budget < 256 {
		budget = 256
	}
	if budget > 1200 {
		budget = 1200
	}
	return budget, "adaptive", nil
}

func searchTokens(primary, secondary string, limit int) []string {
	tokens := []string{}
	seen := map[string]bool{}
	appendTerms := func(value string) {
		for _, term := range termPattern.FindAllString(strings.ToLower(value), -1) {
			if utf8.RuneCountInString(term) < 2 || seen[term] || len(tokens) >= limit {
				continue
			}
			seen[term] = true
			tokens = append(tokens, `"`+strings.ReplaceAll(term, `"`, ``)+`"`)
		}
	}
	// Task terms must survive the cap; project metadata only broadens retrieval.
	appendTerms(primary)
	appendTerms(secondary)
	return tokens
}

func knowledgeCandidates(store *memory.Store, task, projectSignals string, limit int) ([]ContextItem, error) {
	tokens := searchTokens(task, projectSignals, 20)
	if len(tokens) == 0 {
		return []ContextItem{}, nil
	}
	rows, err := store.DB.Query(`SELECT k.id,k.title,k.content,s.url,k.quality,bm25(knowledge_fts)
		FROM knowledge_fts JOIN knowledge_items k ON k.rowid=knowledge_fts.rowid
		JOIN knowledge_sources s ON s.id=k.source_id
		WHERE knowledge_fts MATCH ? AND k.status='active' AND s.status='active'
		ORDER BY bm25(knowledge_fts),k.quality DESC,k.updated_at DESC LIMIT ?`, strings.Join(tokens, " OR "), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ContextItem{}
	position := 0
	for rows.Next() {
		var item ContextItem
		var quality, rank float64
		if err = rows.Scan(&item.ID, &item.Title, &item.Content, &item.Source, &quality, &rank); err != nil {
			return nil, err
		}
		item.Kind = "knowledge"
		item.Score = 80 - float64(position) + quality*10 - rank
		item.EstimatedTokens = estimateTokens(item.Title + " " + item.Content)
		items = append(items, item)
		position++
	}
	return items, rows.Err()
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
	projectParts := append([]string{}, profile.Languages...)
	projectParts = append(projectParts, profile.Stacks...)
	projectParts = append(projectParts, profile.Signals...)
	projectSignals := strings.Join(projectParts, " ")
	query := strings.TrimSpace(task + " " + projectSignals)
	memories, err := store.Recall(query, project.ID, 24)
	if err != nil {
		return Pack{}, err
	}
	candidates := make([]ContextItem, 0, len(memories)+32)
	for index, item := range memories {
		candidates = append(candidates, ContextItem{Kind: "memory", ID: item.ID, Content: item.Content, Source: item.Source, Score: 100 - float64(index), EstimatedTokens: estimateTokens(item.Content)})
	}
	knowledge, err := knowledgeCandidates(store, task, projectSignals, 40)
	if err != nil {
		return Pack{}, err
	}
	candidates = append(candidates, knowledge...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	selected := []ContextItem{}
	seen := map[string]bool{}
	used, duplicates := 0, 0
	for _, item := range candidates {
		key := fingerprint(item.Content)
		if seen[key] {
			duplicates++
			continue
		}
		seen[key] = true
		remaining := budget - used
		if remaining < 40 {
			break
		}
		if item.EstimatedTokens > remaining {
			item.Content = truncateTokens(item.Content, remaining)
			item.EstimatedTokens = estimateTokens(item.Content)
		}
		if item.EstimatedTokens == 0 {
			continue
		}
		selected = append(selected, item)
		used += item.EstimatedTokens
	}
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

func Feedback(store *memory.Store, packID, outcome string, tokensUsed int, notes string) (map[string]any, error) {
	if outcome != "helpful" && outcome != "partial" && outcome != "harmful" {
		return nil, errors.New("outcome must be helpful, partial or harmful")
	}
	if tokensUsed < 0 || tokensUsed > 10_000_000 {
		return nil, errors.New("tokens used is outside the accepted range")
	}
	var exists int
	if err := store.DB.QueryRow("SELECT count(*) FROM context_packs WHERE id=?", packID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("context pack not found")
	}
	delta := 0.0
	if outcome == "helpful" {
		delta = 0.25
	} else if outcome == "harmful" {
		delta = -0.5
	}
	tx, err := store.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE knowledge_items SET quality=max(-2,min(2,quality+?))
		WHERE id IN (SELECT item_id FROM context_pack_items WHERE pack_id=? AND item_kind='knowledge')`, delta, packID)
	if err != nil {
		return nil, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{"packId": packID, "outcome": outcome, "tokensUsed": tokensUsed, "notes": strings.TrimSpace(notes), "knowledgeUpdated": updated})
	if err != nil {
		return nil, err
	}
	eventID, err := memory.ID()
	if err != nil {
		return nil, err
	}
	var project string
	if err = tx.QueryRow("SELECT project_id FROM context_packs WHERE id=?", packID).Scan(&project); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO events(id,kind,project_id,payload,created_at) VALUES(?,'context_feedback',?,?,?)", eventID, project, string(payload), nowUTC()); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"packId": packID, "outcome": outcome, "knowledgeUpdated": updated, "eventId": eventID}, nil
}

func PackSummary(pack Pack) string {
	return fmt.Sprintf("%s: %d/%d estimated tokens, %d evidence items, %d recommendations", pack.ID, pack.Metrics.EstimatedTokens, pack.Metrics.BudgetTokens, len(pack.Context), len(pack.Recommendations))
}
