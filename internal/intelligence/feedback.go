package intelligence

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

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
