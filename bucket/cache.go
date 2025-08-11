package bucket

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type PluginDatabase interface {
	InitializeDatabase(*OpenContext) error

	LoadPluginDatabase() error
	SavePlugin(plugin CachedPlugin) error
	SavePluginDatabase() error
	DBSize() (int64, error)
	CleanCache() error
	CloseDatabase() error

	Plugins() *SymmetricBiMap[string, CachedPlugin]
}

type CachedRecord struct {
	metadata json.RawMessage

	Repository string `json:"repository"`
	File       string `json:"path"`

	Name             string `json:"name"`
	LocalIdentifier  string `json:"local_identifier"`
	RemoteIdentifier string `json:"remote_identifier"`

	Authors     []string `json:"authors,omitempty"`
	Description string   `json:"description,omitempty"`
	Website     string   `json:"website,omitempty"`

	Confidence float64 `json:"confidence"`
}

type CachedPlugin struct {
	RemotePlugin `json:"metadata"`
	CachedRecord

	Repository NamedRepository `json:"-"`
}

func (r *CachedRecord) UnmarshalJSON(data []byte) error {
	type alias CachedRecord

	if err := json.Unmarshal(data, (*alias)(r)); err != nil {
		return err
	}

	var tmp struct {
		Metadata json.RawMessage `json:"metadata"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	r.metadata = tmp.Metadata
	return nil
}

func (r *CachedRecord) CachedPlugin(ctx *OpenContext) (*CachedPlugin, error) {
	var plugin CachedPlugin

	plugin.CachedRecord = *r

	repo := ctx.RepositoryByNameOrProvider(r.Repository)
	if repo == nil {
		return nil, fmt.Errorf("repository %s not found for plugin record %s", r.Repository, plugin.LocalIdentifier)
	}

	plugin.Repository = *repo

	remote := reflect.New(plugin.Repository.PluginType()).Interface().(RemotePlugin)
	if err := json.Unmarshal(r.metadata, &remote); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin metadata: %w", err)
	}

	plugin.RemotePlugin = remote

	return &plugin, nil
}

func CachedMatch(local *LocalPlugin, remote RemotePlugin, repo NamedRepository, conf float64) CachedPlugin {
	return CachedPlugin{
		RemotePlugin: remote,
		Repository:   repo,
		CachedRecord: CachedRecord{
			Name:             local.GetName(),
			Repository:       repo.GetName(),
			LocalIdentifier:  local.GetIdentifier(),
			RemoteIdentifier: remote.GetIdentifier(),
			Authors:          remote.GetAuthors(),
			Description:      remote.GetDescription(),
			Website:          remote.GetWebsite(),
			File:             local.File.Name(),
			Confidence:       conf,
		},
	}
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

	return cp.ForceRequest()
}

func (cp *CachedPlugin) ForceRequest() error {
	remote, err := cp.Repository.Get(cp.RemoteIdentifier)
	if err != nil {
		return err
	}

	cp.RemotePlugin = remote

	return nil
}

func (cp *CachedPlugin) GetName() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetName()
	}

	return cp.Name
}

func (cp *CachedPlugin) GetIdentifier() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetIdentifier()
	}

	return cp.RemoteIdentifier
}

func (cp *CachedPlugin) GetAuthors() []string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetAuthors()
	}

	return cp.Authors
}

func (cp *CachedPlugin) GetDescription() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetDescription()
	}

	return cp.Description
}

func (cp *CachedPlugin) GetWebsite() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetWebsite()
	}

	return cp.Website
}

func (cp *CachedPlugin) requestIfMissing() error {
	if cp.RemotePlugin == nil {
		return cp.Request()
	} else {
		return nil
	}
}

func (cp *CachedPlugin) GetLatestVersion() (RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetLatestVersion()
}

func (cp *CachedPlugin) GetVersions(limit int) ([]RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetVersions(limit)
}

func (cp *CachedPlugin) GetVersionByID(version string) (RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetVersionByID(version)
}

func (cp *CachedPlugin) GetVersionIdentifiers() ([]string, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetVersionIdentifiers()
}

func (cp *CachedPlugin) GetLatestCompatible(plt PlatformType) (RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetLatestCompatible(plt)
}

func (cp *CachedPlugin) GetRepository() Repository {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetRepository()
	}

	return cp.Repository.Repository
}
