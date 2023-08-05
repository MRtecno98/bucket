package bucket

import "github.com/MRtecno98/afero"

type Plugin interface {
	GetName() string
	GetIdentifier() string
}

type Versionable interface {
	GetVersion() string
}

type PluginDescriptor interface {
	Plugin
	Versionable
}

type LocalPlugin struct {
	PluginDescriptor
	File afero.File
}

type Dependency struct {
	Name     string
	Required bool
	// MinVersion string // Not for now
}

type Depender interface {
	GetDependencies() []Dependency
}

type PluginMetadata interface {
	GetAuthors() []string
	GetDescription() string
	GetWebsite() string
}
