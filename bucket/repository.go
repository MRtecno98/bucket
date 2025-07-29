package bucket

import (
	"context"
	"io"

	"github.com/go-resty/resty/v2"
)

type Repository interface {
	Provider() string

	Search(query string, max int) ([]RemotePlugin, int, error)
	SearchAll(query string, max int) ([]RemotePlugin, int, error)

	Get(identifier string) (RemotePlugin, error)
	Resolve(plugin Plugin) (RemotePlugin, []RemotePlugin, error)

	// SupportsDependencies() bool // Can just check if version.(Depender)
}

type HashRepository interface {
	GetByHash(hash string) (Plugin, error)
}

type RemotePlugin interface {
	Plugin
	PluginMetadata
	PlatformCompatible

	GetRepository() Repository

	GetLatestCompatible(PlatformType) (RemoteVersion, error)
	GetLatestVersion() (RemoteVersion, error)
	GetVersions(limit int) ([]RemoteVersion, error)
	GetVersionByID(identifier string) (RemoteVersion, error)
	GetVersionIdentifiers() ([]string, error)
}

type RemoteVersion interface {
	RemotePlugin
	PlatformCompatible
	NamedVersionable

	GetFiles() ([]RemoteFile, error)
}

type RemoteFile interface {
	Name() string
	Optional() bool
	Download() (io.ReadCloser, error)
	Verify() error
}

type LockRepository struct {
	Repository

	Lock context.Context
}

type HTTPRepository struct {
	Repository

	Endpoint   string
	HTTPClient *resty.Client
}

type NamedRepository struct {
	Repository
	RepositoryConfig
}

func NewHTTPRepository(endpoint string) *HTTPRepository {
	return &HTTPRepository{
		Endpoint:   endpoint,
		HTTPClient: resty.New().SetHeader("User-Agent", UserAgent).SetBaseURL(endpoint),
	}
}

var Repositories = make(map[string]RepositoryConstructor)

type RepositoryConstructor func(context.Context, *OpenContext, map[string]string) Repository

func RegisterRepository(name string, constr RepositoryConstructor) {
	Repositories[name] = constr
}

func GetVersionNames(p RemotePlugin) ([]string, error) {
	vers, err := p.GetVersions(0)
	if err != nil {
		return nil, err
	}

	var identifiers []string
	for _, v := range vers {
		identifiers = append(identifiers, v.GetVersion())
	}

	return identifiers, nil
}
