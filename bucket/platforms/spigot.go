package platforms

import (
	"log"
	"slices"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
	"gopkg.in/yaml.v2"
)

type SpigotPluginDescriptor struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	MainClass   string   `yaml:"main"`
	Description string   `yaml:"description"`
	ApiVersion  string   `yaml:"api-version"`
	LoadPhase   string   `yaml:"load"`
	Author      string   `yaml:"author"`
	Authors     []string `yaml:"authors"`
	Website     string   `yaml:"website"`
	Depends     []string `yaml:"depend"`
	SoftDepends []string `yaml:"softdepend"`
	LoadBefore  []string `yaml:"loadbefore"`
	Prefix      string   `yaml:"prefix"`
	Libraries   []string `yaml:"libraries"`

	Commands map[string]struct {
		Description       string   `yaml:"description"`
		Aliases           []string `yaml:"aliases"`
		Permission        string   `yaml:"permission"`
		PermissionMessage string   `yaml:"permission-message"`
		Usage             string   `yaml:"usage"`
	} `yaml:"commands"`

	Permissions map[string]struct {
		Description string          `yaml:"description"`
		Default     string          `yaml:"default"`
		Children    map[string]bool `yaml:"children"`
	} `yaml:"permissions"`
}

var SpigotTypePlatform = bucket.PlatformType{
	Name: "spigot",
	// There's actually not a bukkit platform, but there may be
	// other derivatives of it other than spigot
	Compatible: []string{"bukkit"},
	Install:    InstallSpigot,
	Detect:     DetectSpigot,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return NewSpigotPlatform(context) // Go boilerplate
	},
}

func init() {
	bucket.RegisterPlatform(SpigotTypePlatform, 5)
}

type SpigotPlatform struct {
	bucket.PluginCachePlatform
}

func (p SpigotPlatform) Type() bucket.PlatformType {
	return SpigotTypePlatform
}

func (pl SpigotPluginDescriptor) GetDependencies() []bucket.Dependency {
	var deps []bucket.Dependency = make([]bucket.Dependency, 0, len(pl.Depends)+len(pl.SoftDepends))

	for _, dep := range pl.Depends {
		deps = append(deps, bucket.Dependency{Name: dep, Required: true})
	}

	for _, dep := range pl.SoftDepends {
		deps = append(deps, bucket.Dependency{Name: dep, Required: false})
	}

	return deps
}

func NewSpigotPlatform(context *bucket.OpenContext) *SpigotPlatform {
	return &SpigotPlatform{
		PluginCachePlatform: bucket.PluginCachePlatform{
			PluginProvider: bucket.JarPluginPlatform[SpigotPluginDescriptor]{
				ContextPlatform: bucket.ContextPlatform{Context: context},
				Decode:          bucket.BufferedDecode(yaml.Unmarshal),
				PluginFile:      "plugin.yml",
				PluginFolder:    "plugins",
			},
		},
	}
}

func DetectSpigot(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "org\\spigotmc")
	})

	if err != nil {
		log.Println("error during platform check:", err)
	}

	if res {
		return NewSpigotPlatform(context), nil
	} else {
		return nil, nil
	}
}

func InstallSpigot(context *bucket.OpenContext) error {
	return nil
}

func (pl SpigotPluginDescriptor) GetName() string {
	return pl.Name
}

func (pl SpigotPluginDescriptor) GetIdentifier() string {
	return pl.GetName()
}

func (pl SpigotPluginDescriptor) GetVersion() string {
	return pl.Version
}

func (pl SpigotPluginDescriptor) GetAuthors() []string {
	return slices.DeleteFunc(append(pl.Authors, pl.Author), func(s string) bool {
		return s == ""
	})
}

func (pl SpigotPluginDescriptor) GetDescription() string {
	return pl.Description
}

func (pl SpigotPluginDescriptor) GetWebsite() string {
	return pl.Website
}
