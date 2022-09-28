package bucket

import "io"

type Repository interface {
	Search(query string, max int) ([]Plugin, error)
	SearchAll(query string) ([]Plugin, error)

	Get(identifier string) (RemotePlugin, error)

	SupportsDependencies() bool
}

type RemotePlugin interface {
	Plugin
	PluginMetadata

	Download() (io.ReadCloser, error)
}
