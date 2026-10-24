package intelligence

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
)

var htmlTag = regexp.MustCompile(`<[^>]+>`)

func cleanRemoteText(value string) string {
	if strings.Contains(value, "<html") || strings.Contains(value, "<body") {
		value = htmlTag.ReplaceAllString(value, " ")
		value = html.UnescapeString(value)
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

type sourceChunk struct{ title, content string }

// chunksFromSkillsIndex parses the AAS skills_index.json format (a JSON array
// of skill objects) into one sourceChunk per skill so that each skill is
// individually FTS5-searchable by title, description, category, and tags.
func chunksFromSkillsIndex(data []byte) []sourceChunk {
	var skills []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Category    string   `json:"category"`
		Tags        []string `json:"tags"`
		Risk        string   `json:"risk"`
	}
	if err := json.Unmarshal(data, &skills); err != nil {
		return nil
	}
	result := make([]sourceChunk, 0, len(skills))
	for _, s := range skills {
		name := s.Name
		if name == "" {
			name = s.ID
		}
		if name == "" || s.Description == "" {
			continue
		}
		content := s.Description
		if s.Category != "" {
			content += "\ncategory: " + s.Category
		}
		if len(s.Tags) > 0 {
			content += "\ntags: " + strings.Join(s.Tags, ", ")
		}
		if s.Risk != "" && s.Risk != "none" {
			content += "\nrisk: " + s.Risk
		}
		result = append(result, sourceChunk{name, content})
	}
	return result
}

func chunks(value string) []sourceChunk {
	value = cleanRemoteText(value)
	if value == "" {
		return nil
	}
	// Detect AAS skills_index.json: a JSON array of skill objects.
	// Route to the dedicated per-skill chunker instead of the markdown chunker.
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "[") &&
		strings.Contains(trimmed, `"description"`) &&
		strings.Contains(trimmed, `"category"`) {
		if result := chunksFromSkillsIndex([]byte(trimmed)); len(result) > 0 {
			return result
		}
	}
	result := []sourceChunk{}
	title := "Overview"
	lines := []string{}
	flush := func() {
		content := strings.TrimSpace(strings.Join(lines, "\n"))
		for len(content) > 0 {
			cut := len(content)
			if cut > 6000 {
				cut = 6000
				if boundary := strings.LastIndex(content[:cut], "\n\n"); boundary > 1000 {
					cut = boundary
				}
			}
			part := strings.TrimSpace(content[:cut])
			if part != "" {
				result = append(result, sourceChunk{title, part})
			}
			content = strings.TrimSpace(content[cut:])
		}
		lines = nil
	}
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if heading == "" {
			lines = append(lines, line)
			continue
		}
		flush()
		title = truncateTokens(heading, 40)
	}
	flush()
	return result
}
