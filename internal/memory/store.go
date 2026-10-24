// Package memory owns persistence and trust policy, shared by CLI and MCP.
package memory

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB   *sql.DB
	Path string
}

// nowUTC returns the current time formatted as a UTC ISO-8601 millisecond string.
func nowUTC() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }
func ID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// Open creates or migrates a private SQLite store before exposing its connection.
func Open(home string) (*Store, error) {
	file, err := prepareStorePath(home)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, err
	}
	// Pragmas apply to the sole connection; SQLite serializes writes across processes.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := initializeDatabase(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{DB: db, Path: file}, nil
}

func prepareStorePath(home string) (string, error) {
	if err := os.MkdirAll(home, 0700); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(home, 0700); err != nil {
			return "", err
		}
	}
	file := filepath.Join(home, "brain.sqlite")
	// Create with private permissions before SQLite can write contents.
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(file, 0600); err != nil {
			return "", err
		}
	}
	return file, nil
}

func (s *Store) Close() error { return s.DB.Close() }
