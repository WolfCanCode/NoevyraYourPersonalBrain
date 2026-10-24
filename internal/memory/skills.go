package memory

import (
	"errors"
	"regexp"
	"strings"
)

var skillRef = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+$`)

func (s *Store) Trust(ref, keyword string) error {
	if !skillRef.MatchString(ref) || strings.TrimSpace(keyword) == "" {
		return errors.New("use owner/repo@skill and a nonempty keyword")
	}
	_, err := s.DB.Exec("INSERT INTO approved_skills(ref,keyword,approved_at) VALUES(?,?,?) ON CONFLICT(ref) DO UPDATE SET keyword=excluded.keyword,approved_at=excluded.approved_at", ref, keyword, nowUTC())
	return err
}

func (s *Store) Skills() ([]Skill, error) {
	rows, err := s.DB.Query("SELECT ref,keyword,approved_at FROM approved_skills ORDER BY keyword")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Skill{}
	for rows.Next() {
		var item Skill
		if err = rows.Scan(&item.Ref, &item.Keyword, &item.ApprovedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
