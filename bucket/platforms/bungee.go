package platform

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
)

var BungeeTypePlatform = bucket.PlatformType{
	Name:    "bungeecoord",
	Install: InstallBungeecoord,
	Detect:  DetectBungeecoord,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return &BungeePlatform{bucket.ContextPlatform{Context: context}}
	},
}

func init() {
	bucket.RegisterPlatform(BungeeTypePlatform, 0)
}

type BungeePlatform struct {
	bucket.ContextPlatform
}

func (p *BungeePlatform) Type() bucket.PlatformType {
	return BungeeTypePlatform
}

func (p *BungeePlatform) PluginsFolder() string {
	return "plugins"
}

func (p *BungeePlatform) Plugins() ([]bucket.Plugin, error) {
	return nil, nil // TODO: Analyze plugins in context
}

func DetectBungeecoord(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "net\\md_5\\bungee")
	})

	if err != nil {
		log.Println("Error during platform check:", err)
	}

	if res {
		return &BungeePlatform{bucket.ContextPlatform{Context: context}}, nil
	} else {
		return nil, nil
	}
}

func InstallBungeecoord(context *bucket.OpenContext) error {
	return nil
}
