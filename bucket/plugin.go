package bucket

type Plugin interface {
	GetName() string
	GetVersion() string
	GetIdentifier() string
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
