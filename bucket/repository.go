package bucket

import (
	"context"
	"io"

	"github.com/go-resty/resty/v2"
)

type Repository interface {
	Search(query string, max int) ([]Plugin, int, error)
	SearchAll(query string, max int) ([]Plugin, int, error)

	Get(identifier string) (RemotePlugin, error)
	Resolve(plugin Plugin) (RemotePlugin, error)

	// SupportsDependencies() bool // Can just check if version.(Depender)
}

type HashRepository interface {
	GetByHash(hash string) (Plugin, error)
}

type RemotePlugin interface {
	Plugin
	PluginMetadata
	PlatformCompatible

	GetLatestVersion() (RemoteVersion, error)
	GetVersions() ([]RemoteVersion, error)
	GetVersion(identifier string) (RemoteVersion, error)
	GetVersionIdentifiers() ([]string, error)
}

type RemoteVersion interface {
	RemotePlugin
	PlatformCompatible

	GetName() string
	GetIdentifier() string
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

type HttpRepository struct {
	Repository

	Endpoint   string
	HttpClient *resty.Client
}

func NewHttpRepository(endpoint string) *HttpRepository {
	return &HttpRepository{
		Endpoint:   endpoint,
		HttpClient: resty.New().SetHeader("User-Agent", USER_AGENT).SetBaseURL(endpoint),
	}
}
