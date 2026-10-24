package memory

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS memory (
 id TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'semantic',
 scope TEXT NOT NULL CHECK(scope IN ('personal','project')), project_id TEXT,
 content TEXT NOT NULL, source TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('candidate','verified','superseded','rejected')),
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, expires_at TEXT
);
CREATE INDEX IF NOT EXISTS ix_memory_scope ON memory(scope,project_id,status);
CREATE TABLE IF NOT EXISTS events (
 id TEXT PRIMARY KEY, kind TEXT NOT NULL, project_id TEXT, payload TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_events_created_at ON events(created_at);
CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS approved_skills (ref TEXT PRIMARY KEY, keyword TEXT NOT NULL, approved_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS knowledge_sources (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, url TEXT NOT NULL UNIQUE, tags TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL CHECK(status IN ('active','paused')), etag TEXT, last_modified TEXT,
 content_hash TEXT, last_synced_at TEXT, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS knowledge_items (
 id TEXT PRIMARY KEY, source_id TEXT NOT NULL REFERENCES knowledge_sources(id) ON DELETE CASCADE,
 title TEXT NOT NULL, content TEXT NOT NULL, tags TEXT NOT NULL DEFAULT '', content_hash TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('active','superseded')), quality REAL NOT NULL DEFAULT 0,
 updated_at TEXT NOT NULL, UNIQUE(source_id,content_hash)
);
CREATE INDEX IF NOT EXISTS ix_knowledge_source_status ON knowledge_items(source_id,status);
CREATE TABLE IF NOT EXISTS context_packs (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, task TEXT NOT NULL, budget_tokens INTEGER NOT NULL,
 estimated_tokens INTEGER NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS context_pack_items (
 pack_id TEXT NOT NULL REFERENCES context_packs(id) ON DELETE CASCADE,
 item_kind TEXT NOT NULL CHECK(item_kind IN ('memory','knowledge')), item_id TEXT NOT NULL,
 score REAL NOT NULL, estimated_tokens INTEGER NOT NULL,
 PRIMARY KEY(pack_id,item_kind,item_id)
);
CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(content, content='memory', content_rowid='rowid');
CREATE TRIGGER IF NOT EXISTS memory_fts_ai AFTER INSERT ON memory BEGIN
 INSERT INTO memory_fts(rowid,content) VALUES(new.rowid,new.content); END;
CREATE TRIGGER IF NOT EXISTS memory_fts_ad AFTER DELETE ON memory BEGIN
 INSERT INTO memory_fts(memory_fts,rowid,content) VALUES('delete',old.rowid,old.content); END;
CREATE TRIGGER IF NOT EXISTS memory_fts_au AFTER UPDATE OF content ON memory BEGIN
 INSERT INTO memory_fts(memory_fts,rowid,content) VALUES('delete',old.rowid,old.content);
 INSERT INTO memory_fts(rowid,content) VALUES(new.rowid,new.content); END;
CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(title,content,tags,content='knowledge_items',content_rowid='rowid');
CREATE TRIGGER IF NOT EXISTS knowledge_fts_ai AFTER INSERT ON knowledge_items BEGIN
 INSERT INTO knowledge_fts(rowid,title,content,tags) VALUES(new.rowid,new.title,new.content,new.tags); END;
CREATE TRIGGER IF NOT EXISTS knowledge_fts_ad AFTER DELETE ON knowledge_items BEGIN
 INSERT INTO knowledge_fts(knowledge_fts,rowid,title,content,tags) VALUES('delete',old.rowid,old.title,old.content,old.tags); END;
CREATE TRIGGER IF NOT EXISTS knowledge_fts_au AFTER UPDATE OF title,content,tags ON knowledge_items BEGIN
 INSERT INTO knowledge_fts(knowledge_fts,rowid,title,content,tags) VALUES('delete',old.rowid,old.title,old.content,old.tags);
 INSERT INTO knowledge_fts(rowid,title,content,tags) VALUES(new.rowid,new.title,new.content,new.tags); END;
CREATE TABLE IF NOT EXISTS project_profiles (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, root TEXT NOT NULL,
 profile_json TEXT NOT NULL, capabilities_json TEXT NOT NULL DEFAULT '[]',
 created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_project_profiles_pid_ts ON project_profiles(project_id,created_at);
`

// initializeDatabase applies schema changes and legacy FTS rebuilds atomically.
func initializeDatabase(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 3 {
		return fmt.Errorf("database schema %d is newer than supported version 3", version)
	}
	if _, err := tx.Exec(schema); err != nil {
		return err
	}
	// FTS count(*) reads the external content table, not the actual index.
	// Legacy stores need a transactional rebuild rather than a row-count check.
	for _, migration := range []struct {
		version int
		query   string
	}{
		{1, `INSERT INTO memory_fts(memory_fts) VALUES('rebuild');`},
		{2, `INSERT INTO knowledge_fts(knowledge_fts) VALUES('rebuild');`},
	} {
		if version >= migration.version {
			continue
		}
		if _, err := tx.Exec(migration.query); err != nil {
			return err
		}
	}
	// Version 3 adds project_profiles through the schema's idempotent DDL.
	if _, err := tx.Exec(`PRAGMA user_version=3;`); err != nil {
		return err
	}
	return tx.Commit()
}
