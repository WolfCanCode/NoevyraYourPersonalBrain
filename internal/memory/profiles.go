package memory

import (
	"database/sql"
	"errors"
)

// UpsertProfile persists a new StoredProfile for the given project. When the project
// already has 3 stored profiles, the oldest is deleted before inserting (Req 2.1, 2.3, 2.8).
func (s *Store) UpsertProfile(projectID, root, profileJSON, capabilitiesJSON string) (StoredProfile, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return StoredProfile{}, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM project_profiles WHERE project_id=?", projectID).Scan(&count); err != nil {
		return StoredProfile{}, err
	}
	if count >= 3 {
		if _, err = tx.Exec("DELETE FROM project_profiles WHERE id=(SELECT id FROM project_profiles WHERE project_id=? ORDER BY created_at ASC LIMIT 1)", projectID); err != nil {
			return StoredProfile{}, err
		}
	}
	id, err := ID()
	if err != nil {
		return StoredProfile{}, err
	}
	at := nowUTC()
	if _, err = tx.Exec("INSERT INTO project_profiles(id,project_id,root,profile_json,capabilities_json,created_at) VALUES(?,?,?,?,?,?)", id, projectID, root, profileJSON, capabilitiesJSON, at); err != nil {
		return StoredProfile{}, err
	}
	if err = tx.Commit(); err != nil {
		return StoredProfile{}, err
	}
	return StoredProfile{ID: id, ProjectID: projectID, Root: root, ProfileJSON: profileJSON, CapabilitiesJSON: capabilitiesJSON, CreatedAt: at}, nil
}

// LatestProfile returns the most recently created StoredProfile for projectID,
// or (nil, nil) if none exists (Req 2.4, 7.1, 7.2).
func (s *Store) LatestProfile(projectID string) (*StoredProfile, error) {
	var p StoredProfile
	err := s.DB.QueryRow("SELECT id,project_id,root,profile_json,capabilities_json,created_at FROM project_profiles WHERE project_id=? ORDER BY created_at DESC LIMIT 1", projectID).Scan(&p.ID, &p.ProjectID, &p.Root, &p.ProfileJSON, &p.CapabilitiesJSON, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
