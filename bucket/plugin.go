package bucket

type Plugin interface {
	GetName() string
	GetVersion() string
}
