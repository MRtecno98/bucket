package platforms

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
	"gopkg.in/yaml.v2"
)

var BungeeTypePlatform = bucket.PlatformType{
	Name:    "bungeecoord",
	Install: InstallBungeecoord,
	Detect:  DetectBungeecoord,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return NewBungeePlatform(context) // Go boilerplate
	},
}

func init() {
	bucket.RegisterPlatform(BungeeTypePlatform, 0)
}

type BungeePlatform struct {
	bucket.PluginCachePlatform
}

func (p *BungeePlatform) Type() bucket.PlatformType {
	return BungeeTypePlatform
}

func NewBungeePlatform(context *bucket.OpenContext) *BungeePlatform {
	return &BungeePlatform{
		PluginCachePlatform: bucket.PluginCachePlatform{
			PluginProvider: bucket.JarPluginPlatform[SpigotPluginDescriptor]{
				ContextPlatform: bucket.ContextPlatform{Context: context},
				Decode:          bucket.BufferedDecode(yaml.Unmarshal),
				PluginFiles:     []string{"bungee.yml"},
				PluginFolder:    "plugins",
			},
		},
	}
}

func DetectBungeecoord(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "net\\md_5\\bungee")
	})

	if err != nil {
		log.Println("Error during platform check:", err)
	}

	if res {
		return NewBungeePlatform(context), nil
	} else {
		return nil, nil
	}
}

func InstallBungeecoord(context *bucket.OpenContext) error {
	return nil
}
