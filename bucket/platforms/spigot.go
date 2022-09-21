package platform

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
)

type SpigotPluginDescriptor struct {
	bucket.Plugin

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
	Name:    "spigot",
	Install: InstallSpigot,
	Detect:  DetectSpigot,
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

func (pl SpigotPluginDescriptor) GetName() string {
	return pl.Name
}

func (pl SpigotPluginDescriptor) GetVersion() string {
	return pl.Version
}

func NewSpigotPlatform(context *bucket.OpenContext) *SpigotPlatform {
	return &SpigotPlatform{
		PluginCachePlatform: bucket.PluginCachePlatform{
			PluginProvider: bucket.JarPluginPlatform[SpigotPluginDescriptor]{
				ContextPlatform: bucket.ContextPlatform{Context: context},
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
