package bucket

import (
	"io"

	"github.com/go-resty/resty/v2"
)

type Repository interface {
	Search(query string, max int) ([]Plugin, error)
	SearchAll(query string) ([]Plugin, error)

	Get(identifier string) (RemotePlugin, error)
	Resolve(plugin Plugin) (RemotePlugin, error)

	// SupportsDependencies() bool // Can just check if version.(Depender)
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
