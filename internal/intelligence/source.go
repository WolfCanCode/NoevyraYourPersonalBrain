package intelligence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

const maxSourceBytes = 4 * 1024 * 1024

type Source struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	URL          string  `json:"url"`
	Tags         string  `json:"tags"`
	Status       string  `json:"status"`
	ETag         *string `json:"etag,omitempty"`
	LastModified *string `json:"lastModified,omitempty"`
	ContentHash  *string `json:"contentHash,omitempty"`
	LastSyncedAt *string `json:"lastSyncedAt,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

type SourceSync struct {
	SourceID string `json:"sourceId"`
	Name     string `json:"name"`
	Changed  bool   `json:"changed"`
	Items    int    `json:"items"`
	Error    string `json:"error,omitempty"`
}

var starterSources = []struct {
	name, url, tags string
}{
	{"Go code review comments", "https://raw.githubusercontent.com/golang/wiki/master/CodeReviewComments.md", "go,review,style"},
	{"OWASP secure coding with AI", "https://raw.githubusercontent.com/OWASP/CheatSheetSeries/master/cheatsheets/Secure_Coding_with_AI_Cheat_Sheet.md", "ai,agent,security,review"},
	{"MCP security best practices", "https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/main/docs/docs/2026-07-28/tutorials/security/security_best_practices.mdx", "mcp,security,oauth,agent"},
	// Agentic Awesome Skills catalog — 2 700+ community skills indexed locally for fast, offline search.
	{"AAS skills catalog", "https://raw.githubusercontent.com/sickn33/agentic-awesome-skills/main/skills_index.json", "skills,catalog,aas"},
}

func normalizeTags(value string) string {
	set := map[string]bool{}
	for _, item := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		item = strings.TrimSpace(item)
		if item != "" {
			set[item] = true
		}
	}
	values := make([]string, 0, len(set))
	for item := range set {
		values = append(values, item)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func validateSourceURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return "", errors.New("source must be an absolute HTTPS URL without credentials or fragment")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", errors.New("source must use HTTPS; HTTP is accepted only for localhost")
	}
	return u.String(), nil
}

// AddSource is an explicit trust action. Fetched text remains untrusted evidence and is never executed.
func AddSource(store *memory.Store, name, rawURL, tags string) (Source, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return Source{}, errors.New("source name must contain 1-120 characters")
	}
	cleanURL, err := validateSourceURL(rawURL)
	if err != nil {
		return Source{}, err
	}
	id, err := memory.ID()
	if err != nil {
		return Source{}, err
	}
	at := nowUTC()
	tags = normalizeTags(tags)
	_, err = store.DB.Exec(`INSERT INTO knowledge_sources(id,name,url,tags,status,created_at) VALUES(?,?,?,?, 'active', ?)
		ON CONFLICT(url) DO UPDATE SET name=excluded.name,tags=excluded.tags,status='active'`, id, name, cleanURL, tags, at)
	if err != nil {
		return Source{}, err
	}
	var source Source
	err = store.DB.QueryRow(`SELECT id,name,url,tags,status,etag,last_modified,content_hash,last_synced_at,created_at FROM knowledge_sources WHERE url=?`, cleanURL).
		Scan(&source.ID, &source.Name, &source.URL, &source.Tags, &source.Status, &source.ETag, &source.LastModified, &source.ContentHash, &source.LastSyncedAt, &source.CreatedAt)
	return source, err
}

func ListSources(store *memory.Store) ([]Source, error) {
	rows, err := store.DB.Query(`SELECT id,name,url,tags,status,etag,last_modified,content_hash,last_synced_at,created_at FROM knowledge_sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Source{}
	for rows.Next() {
		var source Source
		if err = rows.Scan(&source.ID, &source.Name, &source.URL, &source.Tags, &source.Status, &source.ETag, &source.LastModified, &source.ContentHash, &source.LastSyncedAt, &source.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, source)
	}
	return items, rows.Err()
}

// SeedSources registers a small, reviewable catalog of primary-source documents.
// The command is explicit; it does not fetch until source sync runs.
func SeedSources(store *memory.Store) ([]Source, error) {
	items := make([]Source, 0, len(starterSources))
	for _, starter := range starterSources {
		source, err := AddSource(store, starter.name, starter.url, starter.tags)
		if err != nil {
			return nil, err
		}
		items = append(items, source)
	}
	return items, nil
}

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
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if heading != "" {
				flush()
				title = truncateTokens(heading, 40)
				continue
			}
		}
		lines = append(lines, line)
	}
	flush()
	return result
}

func sourceClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 4 {
				return errors.New("too many source redirects")
			}
			if _, err := validateSourceURL(req.URL.String()); err != nil {
				return fmt.Errorf("unsafe source redirect: %w", err)
			}
			if len(via) > 0 && !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname()) {
				return errors.New("cross-host source redirect rejected")
			}
			return nil
		},
	}
}

func fetchSource(ctx context.Context, client *http.Client, source Source) ([]byte, string, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, "", "", false, err
	}
	req.Header.Set("Accept", "text/markdown,text/plain,application/json,text/html;q=0.5")
	req.Header.Set("User-Agent", "personal-ai-brain/0.2")
	if source.ETag != nil {
		req.Header.Set("If-None-Match", *source.ETag)
	}
	if source.LastModified != nil {
		req.Header.Set("If-Modified-Since", *source.LastModified)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, "", "", false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		return nil, response.Header.Get("ETag"), response.Header.Get("Last-Modified"), false, nil
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, "", "", false, fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	reader := io.LimitReader(response.Body, maxSourceBytes+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", "", false, err
	}
	if len(body) > maxSourceBytes {
		return nil, "", "", false, errors.New("source exceeds 2 MiB limit")
	}
	return body, response.Header.Get("ETag"), response.Header.Get("Last-Modified"), true, nil
}

func saveSource(store *memory.Store, source Source, body []byte, etag, modified string) (int, bool, error) {
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	if source.ContentHash != nil && *source.ContentHash == hash {
		_, err := store.DB.Exec(`UPDATE knowledge_sources SET etag=?,last_modified=?,last_synced_at=? WHERE id=?`, nullString(etag), nullString(modified), nowUTC(), source.ID)
		return 0, false, err
	}
	parts := chunks(string(body))
	if len(parts) == 0 {
		return 0, false, errors.New("source contains no usable text")
	}
	tx, err := store.DB.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE knowledge_items SET status='superseded',updated_at=? WHERE source_id=? AND status='active'`, nowUTC(), source.ID); err != nil {
		return 0, false, err
	}
	for _, part := range parts {
		partSum := sha256.Sum256([]byte(part.title + "\x00" + part.content))
		partHash := hex.EncodeToString(partSum[:])
		idSum := sha256.Sum256([]byte(source.ID + "\x00" + partHash))
		id := hex.EncodeToString(idSum[:16])
		_, err = tx.Exec(`INSERT INTO knowledge_items(id,source_id,title,content,tags,content_hash,status,updated_at)
			VALUES(?,?,?,?,?,?,'active',?)
			ON CONFLICT(source_id,content_hash) DO UPDATE SET title=excluded.title,tags=excluded.tags,status='active',updated_at=excluded.updated_at`, id, source.ID, part.title, part.content, source.Tags, partHash, nowUTC())
		if err != nil {
			return 0, false, err
		}
	}
	if _, err = tx.Exec(`UPDATE knowledge_sources SET etag=?,last_modified=?,content_hash=?,last_synced_at=? WHERE id=?`, nullString(etag), nullString(modified), hash, nowUTC(), source.ID); err != nil {
		return 0, false, err
	}
	if err = tx.Commit(); err != nil {
		return 0, false, err
	}
	return len(parts), true, nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// SyncSources pulls only sources explicitly trusted with AddSource.
func SyncSources(ctx context.Context, store *memory.Store) ([]SourceSync, error) {
	sources, err := ListSources(store)
	if err != nil {
		return nil, err
	}
	client := sourceClient()
	reports := make([]SourceSync, 0, len(sources))
	for _, source := range sources {
		if source.Status != "active" {
			continue
		}
		report := SourceSync{SourceID: source.ID, Name: source.Name}
		body, etag, modified, fetched, fetchErr := fetchSource(ctx, client, source)
		if fetchErr != nil {
			report.Error = fetchErr.Error()
			reports = append(reports, report)
			continue
		}
		if !fetched {
			_, fetchErr = store.DB.Exec("UPDATE knowledge_sources SET last_synced_at=? WHERE id=?", nowUTC(), source.ID)
		} else {
			report.Items, report.Changed, fetchErr = saveSource(store, source, body, etag, modified)
		}
		if fetchErr != nil {
			report.Error = fetchErr.Error()
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func PauseSource(store *memory.Store, id string) (bool, error) {
	result, err := store.DB.Exec("UPDATE knowledge_sources SET status='paused' WHERE id=? AND status='active'", id)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func KnowledgeCount(store *memory.Store) (int, error) {
	var count int
	err := store.DB.QueryRow("SELECT count(*) FROM knowledge_items WHERE status='active'").Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return count, err
}
