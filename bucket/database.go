package bucket

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/MRtecno98/afero/sqlitevfs"
)

const DATABASE_NAME = "bucket.db"

type CachedPlugin struct {
	RemotePlugin

	Repository NamedRepository
	File       string

	name             string
	LocalIdentifier  string
	RemoteIdentifier string

	authors     []string
	description string
	website     string

	Confidence float64
}

func CachedMatch(local LocalPlugin, remote RemotePlugin, repo NamedRepository, conf float64) CachedPlugin {
	return CachedPlugin{
		RemotePlugin:     remote,
		LocalIdentifier:  local.GetIdentifier(),
		RemoteIdentifier: remote.GetIdentifier(),
		Repository:       repo,
		File:             local.File.Name(),
		Confidence:       conf}
}

func NewPluginBiMap() *SymmetricBiMap[string, CachedPlugin] {
	return NewSymmetricBiMap(func(el CachedPlugin) (string, string) {
		return el.LocalIdentifier, el.RemoteIdentifier
	})
}

func (cp *CachedPlugin) Request() error {
	if cp.RemotePlugin != nil {
		return nil
	}

	remote, err := cp.Repository.Get(cp.RemoteIdentifier)
	if err != nil {
		return err
	}

	cp.RemotePlugin = remote

	return nil
}

func (c *OpenContext) InitialiazeDatabase() error {
	if DEBUG {
		log.Printf("initializing database for %s\n", c.Name)
	}

	/*
		if _, ok := c.Fs.Fs.(*afero.MemMapFs); ok {
			// TODO: Fix database for in-memory filesystem
			log.Printf("%s: in-memory filesystem not supported for database\n", c.Name)
			return nil
		} */

	sqlitevfs.RegisterVFS(c.Name, c.Fs)

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?vfs=%s", DATABASE_NAME, c.Name))
	if err != nil {
		return err
	}

	c.Database = db

	if err = c.CreateTables(); err != nil {
		db.Close()
		return fmt.Errorf("sql: %w", err)
	}

	return nil
}

func (c *OpenContext) LoadPluginDatabase() error {
	rows, err := c.Database.Query(`SELECT * FROM plugins`)
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
			&plugin.name, &repo, &plugin.Confidence, &authors,
			&plugin.description, &plugin.website); err != nil {
			return err
		}

		plugin.authors = strings.Split(authors, ",")
		repository := c.RepositoryByNameOrProvider(repo)
		if repository == nil {
			log.Printf("warn: repository %s not found for plugin record %s\n", repo, plugin.LocalIdentifier)
			continue
		}

		plugin.Repository = *repository

		c.Plugins.Put(plugin)
	}

	return nil
}

func (c *OpenContext) SavePlugin(plugin CachedPlugin) error {
	tx, err := c.Database.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := c._savePlugin(plugin); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("plugin save: %w", err)
	}

	c.Plugins.Put(plugin)
	return nil
}

func (c *OpenContext) _savePlugin(plugin CachedPlugin) error {
	if _, err := c.Database.Exec(
		`INSERT INTO plugins 
		(identifier, remote_identifier, 
		 filename, 
		 name, repository, confidence,
		 authors, description, website) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		plugin.LocalIdentifier, plugin.RemoteIdentifier, plugin.File,
		plugin.GetName(), plugin.Repository.GetName(), plugin.Confidence,
		strings.Join(plugin.GetAuthors(), ","),
		plugin.GetDescription(), plugin.GetWebsite()); err != nil {
		return fmt.Errorf("db save (no data was modified): %w", err)
	}

	return nil
}

func (c *OpenContext) SavePluginDatabase() error {
	tx, err := c.Database.Begin()
	if err != nil {
		return err
	}

	for _, plugin := range c.Plugins.Values() {
		if c._savePlugin(plugin) != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (c *OpenContext) CreateTables() error {
	tx, err := c.Database.Begin()
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
