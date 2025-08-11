package bucket

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/MRtecno98/afero/sqlitevfs"
)

const DatabaseName = "bucket.db"

const SumDBSqlite string = "sqlite"

type SqliteDatabase struct {
	Name string

	conn *sql.DB
	ctx  *OpenContext

	plugins *SymmetricBiMap[string, CachedPlugin]
}

func NewNamedSqliteDatabase(name string) *SqliteDatabase {
	return &SqliteDatabase{
		Name:    name,
		plugins: NewPluginBiMap(),
	}
}

func NewSqliteDatabase() *SqliteDatabase {
	return NewNamedSqliteDatabase(DatabaseName)
}

func (db *SqliteDatabase) Plugins() *SymmetricBiMap[string, CachedPlugin] {
	return db.plugins
}

func (db *SqliteDatabase) InitializeDatabase(ctx *OpenContext) error {
	/* if _, ok := c.Fs.Fs.(*afero.MemMapFs); ok {
		// TODO: Fix database for in-memory filesystem
		log.Printf("%s: in-memory filesystem not supported for database\n", c.Name)
		return nil
	} */

	sqlitevfs.RegisterVFS(ctx.Name, ctx.Fs)

	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?vfs=%s", db.Name, ctx.Name))
	if err != nil {
		return err
	}

	db.conn = conn
	db.ctx = ctx

	if err = db.InitializeTables(); err != nil {
		db.CloseDatabase()
		return fmt.Errorf("sql: %w", err)
	}

	return nil
}

func (db *SqliteDatabase) LoadPluginDatabase() error {
	rows, err := db.conn.Query(`SELECT * FROM plugins`)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var plugin CachedPlugin
		var authors string
		var repo string

		if err := rows.Scan(&plugin.LocalIdentifier,
			&plugin.RemoteIdentifier, &plugin.File,
			&plugin.Name, &repo, &plugin.Confidence, &authors,
			&plugin.Description, &plugin.Website); err != nil {
			return err
		}

		plugin.Authors = strings.Split(authors, ",")
		repository := db.ctx.RepositoryByNameOrProvider(repo)
		if repository == nil {
			log.Printf("warn: repository %s not found for plugin record %s\n", repo, plugin.LocalIdentifier)
			continue
		}

		plugin.Repository = *repository

		db.plugins.Put(plugin)
	}

	return nil
}

func (db *SqliteDatabase) SavePlugin(plugin CachedPlugin) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := db._savePlugin(plugin); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("plugin save: %w", err)
	}

	db.plugins.Put(plugin)
	return nil
}

func (db *SqliteDatabase) _savePlugin(plugins ...CachedPlugin) error {
	var q strings.Builder
	var args []any

	if len(plugins) == 0 {
		return nil
	}

	args = make([]any, 0, len(plugins)*9)

	q.WriteString(`REPLACE INTO plugins 
		(identifier, remote_identifier, 
		 filename, 
		 name, repository, confidence,
		 authors, description, website) 
		 VALUES `)

	for i, plugin := range plugins {
		q.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if i != len(plugins)-1 {
			q.WriteString(", ")
		}

		args = append(args,
			plugin.LocalIdentifier, plugin.RemoteIdentifier, plugin.File,
			plugin.GetName(), plugin.Repository.GetName(), plugin.Confidence,
			strings.Join(plugin.GetAuthors(), ","),
			plugin.GetDescription(), plugin.GetWebsite())
	}

	if _, err := db.conn.Exec(q.String(), args...); err != nil {
		return fmt.Errorf("db save (no data was modified): %w", err)
	}

	return nil
}

func (db *SqliteDatabase) SavePluginDatabase() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	if db._savePlugin(db.plugins.Values()...) != nil {
		return tx.Rollback()
	}

	return tx.Commit()
}

func (db *SqliteDatabase) InitializeTables() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS plugins (
		identifier VARCHAR(255) PRIMARY KEY,
		remote_identifier VARCHAR(255),
		filename TEXT,
		name VARCHAR(255),
		repository VARCHAR(255),
		confidence REAL,
		authors TEXT,
		description TEXT,
		website TEXT
	);`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
	CREATE INDEX IF NOT EXISTS plugins_remote_id ON plugins (remote_identifier);
	`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
	CREATE INDEX IF NOT EXISTS plugins_name ON plugins (filename);
	`); err != nil {
		return err
	}

	// We need a transaction to force the database to write the changes
	// if we're using an in-memory or remote filesystem
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *SqliteDatabase) DBSize() (int64, error) {
	inf, err := db.ctx.Fs.Stat(DatabaseName)
	if err != nil {
		return -1, nil
	}

	return inf.Size(), nil
}

func (db *SqliteDatabase) CleanCache() error {
	if db.conn != nil {
		if err := db.conn.Close(); err != nil {
			return err
		}
	}

	if err := db.ctx.Fs.Remove(DatabaseName); err != nil {
		return err
	}

	db.conn = nil
	return nil
}

func (db *SqliteDatabase) CloseDatabase() error {
	if db.conn != nil {
		if err := db.conn.Close(); err != nil {
			return fmt.Errorf("close database: %w", err)
		}
	}

	db.conn = nil
	db.ctx = nil
	db.plugins = nil

	return nil
}
