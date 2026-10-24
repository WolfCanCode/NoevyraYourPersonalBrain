package memory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (s *Store) Log(kind, project string, payload any) (string, error) {
	id, err := ID()
	if err != nil {
		return "", err
	}
	p, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	var pid *string
	if project != "" {
		pid = &project
	}
	_, err = s.DB.Exec("INSERT INTO events(id,kind,project_id,payload,created_at) VALUES(?,?,?,?,?)", id, kind, pid, string(p), nowUTC())
	return id, err
}

func (s *Store) Stats() (map[string]any, error) {
	result := map[string]any{"dbPath": s.Path}
	for key, q := range map[string]string{"total": "SELECT count(*) FROM memory", "verified": "SELECT count(*) FROM memory WHERE status='verified'", "candidates": "SELECT count(*) FROM memory WHERE status='candidate'", "events": "SELECT count(*) FROM events"} {
		var n int
		if err := s.DB.QueryRow(q).Scan(&n); err != nil {
			return nil, err
		}
		result[key] = n
	}
	var last string
	err := s.DB.QueryRow("SELECT value FROM settings WHERE key='last_sync'").Scan(&last)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		result["lastSync"] = last
	} else {
		result["lastSync"] = nil
	}
	return result, nil
}

func (s *Store) Sync() (map[string]any, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	at := nowUTC()
	r, err := tx.Exec("UPDATE memory SET status='superseded',updated_at=? WHERE status='verified' AND expires_at IS NOT NULL AND expires_at<=?", at, at)
	if err != nil {
		return nil, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT INTO settings(key,value) VALUES('last_sync',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", at); err != nil {
		return nil, err
	}
	id, err := ID()
	if err != nil {
		return nil, err
	}
	payload := fmt.Sprintf(`{"expired":%d,"online":false}`, n)
	if _, err = tx.Exec("INSERT INTO events(id,kind,payload,created_at) VALUES(?,'sync',?,?)", id, payload, at); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"expiredMemories": n, "note": "Local maintenance complete; skills are not automatically installed or updated."}, nil
}

func (s *Store) Digest(days int) (map[string]any, error) {
	if days < 1 || days > 3650 {
		return nil, errors.New("days must be between 1 and 3650")
	}
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
	events, err := s.eventsSince(since)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Query("SELECT id,type,scope,project_id,content,source,status,updated_at FROM memory WHERE created_at>=? ORDER BY created_at DESC LIMIT 30", since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memories, err := readMemories(rows)
	if err != nil {
		return nil, err
	}
	return map[string]any{"since": since, "events": events, "memories": memories}, nil
}

func (s *Store) eventsSince(since string) ([]Event, error) {
	rows, err := s.DB.Query("SELECT id,kind,project_id,payload,created_at FROM events WHERE created_at>=? ORDER BY created_at DESC LIMIT 30", since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var event Event
		var payload string
		if err := rows.Scan(&event.ID, &event.Kind, &event.ProjectID, &payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		event.Payload = json.RawMessage(payload)
		events = append(events, event)
	}
	return events, rows.Err()
}
