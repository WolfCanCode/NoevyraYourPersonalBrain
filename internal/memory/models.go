package memory

import (
	"encoding/json"
)

type Memory struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Scope     string  `json:"scope"`
	ProjectID *string `json:"projectId"`
	Content   string  `json:"content"`
	Source    string  `json:"source"`
	Status    string  `json:"status"`
	UpdatedAt string  `json:"updatedAt"`
}

type Input struct{ Content, Type, Scope, ProjectID, Source, ExpiresAt string }

type Event struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	ProjectID *string         `json:"projectId"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"createdAt"`
}

type Skill struct {
	Ref        string `json:"ref"`
	Keyword    string `json:"keyword"`
	ApprovedAt string `json:"approved_at"`
}

type StoredProfile struct {
	ID               string `json:"id"`
	ProjectID        string `json:"projectId"`
	Root             string `json:"root"`
	ProfileJSON      string `json:"-"`
	CapabilitiesJSON string `json:"-"`
	CreatedAt        string `json:"createdAt"`
}
