package bucket

import (
	"testing"

	"github.com/MRtecno98/afero"
)

var oc *OpenContext = &OpenContext{
	Context: Context{
		Name: "test",
		URL:  "test",
	},

	Fs:           afero.Afero{},
	Database:     nil,
	Repositories: nil,
	Plugins:      NewPluginBiMap(),

	LocalConfig: &Config{
		ActiveContexts: []string{"test"},
		Contexts:       []Context{{Name: "test", URL: "test"}},
		Platform:       "test",
		Multithread:    true,
		Repositories: []RepositoryConfig{
			{Name: "spigotmc", Provider: "spigotmc", Options: nil},
			{Name: "modrinth", Provider: "modrinth", Options: nil},
		},
	},
}

func init() {
	oc.LoadRepositories()
}

func TestResolve(t *testing.T) {

}
