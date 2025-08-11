package bucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

const SumfileName string = "bucket.sum"
const SumfileHeader string = "# bucket local plugin database"

const SumDBFile string = "file"

type SumfileDatabase struct {
	Name string

	lock    sync.Mutex
	ctx     *OpenContext
	plugins *SymmetricBiMap[string, CachedPlugin]
}

func NewNamedSumfileDatabase(name string) *SumfileDatabase {
	return &SumfileDatabase{
		Name:    name,
		plugins: NewPluginBiMap(),
	}
}

func NewSumfileDatabase() *SumfileDatabase {
	return NewNamedSumfileDatabase(SumfileName)
}

func (db *SumfileDatabase) Plugins() *SymmetricBiMap[string, CachedPlugin] {
	return db.plugins
}

func (db *SumfileDatabase) _parseError(err error) error {
	if err == nil {
		return nil
	}

	if db.Name != "" {
		return fmt.Errorf("sumfile: %w (%s)", err, db.Name)
	} else {
		return fmt.Errorf("sumfile: %w", err)
	}
}

func (db *SumfileDatabase) InitializeDatabase(ctx *OpenContext) error {
	ok, err := ctx.Fs.Exists(db.Name)
	if err != nil {
		return db._parseError(err)
	}

	if !ok {
		f, err := ctx.Fs.OpenFile(db.Name, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return db._parseError(err)
		}

		_, err = f.Write([]byte(SumfileHeader + "\n\n[]\n"))
		if err != nil {
			defer f.Close()
			return db._parseError(err)
		}
	}

	db.ctx = ctx

	return nil
}

func (db *SumfileDatabase) LoadPluginDatabase() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	f, err := db.ctx.Fs.OpenFile(db.Name, os.O_RDONLY, 0644)
	if err != nil {
		return db._parseError(err)
	}

	defer f.Close()

	var data []byte
	if data, err = io.ReadAll(f); err != nil {
		return db._parseError(err)
	}

	data = bytes.TrimPrefix(data, []byte(SumfileHeader))

	var plugins []CachedRecord
	if err = json.Unmarshal(data, &plugins); err != nil {
		return db._parseError(err)
	}

	for _, plugin := range plugins {
		plugin, err := plugin.CachedPlugin(db.ctx)
		if err != nil {
			return db._parseError(err)
		}

		db.plugins.Put(*plugin)
	}

	return nil
}

func (db *SumfileDatabase) SavePlugin(plugin CachedPlugin) error {
	db.plugins.Put(plugin)
	return db.SavePluginDatabase()
}

func (db *SumfileDatabase) SavePluginDatabase() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	f, err := db.ctx.Fs.OpenFile(db.Name, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return db._parseError(err)
	}

	defer f.Close()

	data, err := json.MarshalIndent(db.plugins.Values(), "", "  ")
	if err != nil {
		return db._parseError(err)
	}

	if _, err = f.Write([]byte(SumfileHeader + "\n\n" + string(data) + "\n")); err != nil {
		return db._parseError(err)
	}

	return nil
}

func (db *SumfileDatabase) DBSize() (int64, error) {
	info, err := db.ctx.Fs.Stat(db.Name)
	if err != nil {
		return 0, db._parseError(err)
	}

	return info.Size(), nil
}

func (db *SumfileDatabase) CleanCache() error {
	if err := db.ctx.Fs.Remove(db.Name); err != nil {
		return db._parseError(err)
	}

	db.plugins = NewPluginBiMap()

	return nil
}

func (db *SumfileDatabase) CloseDatabase() error {
	db.plugins = nil
	return nil
}
