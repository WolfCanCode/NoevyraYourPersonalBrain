-- Schema snapshot from the retired Node SQLite runtime. Keep independent of Go migrations.
PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;
    CREATE TABLE IF NOT EXISTS memory (
      id TEXT PRIMARY KEY,
      type TEXT NOT NULL DEFAULT 'semantic',
      scope TEXT NOT NULL CHECK(scope IN ('personal','project')),
      project_id TEXT,
      content TEXT NOT NULL,
      source TEXT NOT NULL,
      status TEXT NOT NULL CHECK(status IN ('candidate','verified','superseded','rejected')),
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      expires_at TEXT
    );
    CREATE INDEX IF NOT EXISTS ix_memory_scope ON memory(scope,project_id,status);
    CREATE TABLE IF NOT EXISTS events (
      id TEXT PRIMARY KEY,
      kind TEXT NOT NULL,
      project_id TEXT,
      payload TEXT NOT NULL,
      created_at TEXT NOT NULL
    );
    CREATE INDEX IF NOT EXISTS ix_events_created_at ON events(created_at);
    CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
    CREATE TABLE IF NOT EXISTS approved_skills (
      ref TEXT PRIMARY KEY, keyword TEXT NOT NULL, approved_at TEXT NOT NULL
    );
  CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(content, content='memory', content_rowid='rowid');
      CREATE TRIGGER IF NOT EXISTS memory_fts_ai AFTER INSERT ON memory BEGIN
        INSERT INTO memory_fts(rowid,content) VALUES(new.rowid,new.content);
      END;
      CREATE TRIGGER IF NOT EXISTS memory_fts_ad AFTER DELETE ON memory BEGIN
        INSERT INTO memory_fts(memory_fts,rowid,content) VALUES('delete',old.rowid,old.content);
      END;
      CREATE TRIGGER IF NOT EXISTS memory_fts_au AFTER UPDATE OF content ON memory BEGIN
        INSERT INTO memory_fts(memory_fts,rowid,content) VALUES('delete',old.rowid,old.content);
        INSERT INTO memory_fts(rowid,content) VALUES(new.rowid,new.content);
      END;
