// Package database opens SQLite and applies embedded, versioned migrations.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	goosedb "github.com/pressly/goose/v3/database"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: abs}
	q := u.Query()
	for _, pragma := range []string{"foreign_keys(1)", "journal_mode(WAL)", "busy_timeout(5000)", "synchronous(FULL)"} {
		q.Add("_pragma", pragma)
	}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	store, err := goosedb.NewStore(goose.DialectSQLite3, goose.DefaultTablename)
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectCustom, db, files, goose.WithStore(utcVersionStore{store}))
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

// Goose's default SQLite metadata uses a space-separated timestamp without Z.
// Keep its normal store behavior, but enforce the same UTC contract here too.
type utcVersionStore struct{ goosedb.Store }

func (s utcVersionStore) CreateVersionTable(ctx context.Context, db goosedb.DBTxConn) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE goose_db_version (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 version_id INTEGER NOT NULL,
 is_applied INTEGER NOT NULL,
 tstamp TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
 CHECK(length(tstamp)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',tstamp,'+0 seconds') IS tstamp)
 )`)
	return err
}
