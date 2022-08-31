package platform

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
)

var SpigotTypePlatform = bucket.PlatformType{
	Name:    "spigot",
	Install: InstallSpigot,
	Detect:  DetectSpigot,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return &SpigotPlatform{bucket.ContextPlatform{Context: context}}
	},
}

func init() {
	bucket.RegisterPlatform(SpigotTypePlatform, 5)
}

type SpigotPlatform struct {
	bucket.ContextPlatform
}

func (p *SpigotPlatform) Type() bucket.PlatformType {
	return SpigotTypePlatform
}

func (p *SpigotPlatform) PluginsFolder() string {
	return "plugins"
}

func (p *SpigotPlatform) Plugins() ([]bucket.Plugin, error) {
	return nil, nil // TODO: Analyze plugins in context
}

func DetectSpigot(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "org\\spigotmc")
	})

	if err != nil {
		log.Println("Error during platform check:", err)
	}

	if res {
		return &SpigotPlatform{bucket.ContextPlatform{Context: context}}, nil
	} else {
		return nil, nil
	}
}

func InstallSpigot(context *bucket.OpenContext) error {
	return nil
}
