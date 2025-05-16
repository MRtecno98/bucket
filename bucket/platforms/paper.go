package platforms

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
)

var PaperTypePlatform = bucket.PlatformType{
	Name:       "paper",
	Compatible: []string{"spigot"},
	Install:    InstallPaper,
	Detect:     DetectPaper,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return NewPaperPlatform(context) // Go boilerplate
	},
}

func init() {
	bucket.RegisterPlatform(PaperTypePlatform, 10)
}

type PaperPlatform struct {
	SpigotPlatform
}

func (p *PaperPlatform) Type() bucket.PlatformType {
	return PaperTypePlatform
}

func NewPaperPlatform(context *bucket.OpenContext) *PaperPlatform {
	spigot := NewSpigotPlatform(context)
	p := spigot.PluginProvider.(bucket.JarPluginPlatform[SpigotPluginDescriptor])

	p.PluginFiles = append(p.PluginFiles, "paper-plugin.yml")
	return &PaperPlatform{
		SpigotPlatform: SpigotPlatform{
			PluginCachePlatform: bucket.PluginCachePlatform{
				PluginProvider: p,
			},
		},
	}
}

func DetectPaper(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "paperclip")
	})

	if err != nil {
		log.Println("Error during platform check:", err)
	}

	if res {
		return NewPaperPlatform(context), nil
	} else {
		return nil, nil
	}
}

func InstallPaper(context *bucket.OpenContext) error {
	return nil
}
