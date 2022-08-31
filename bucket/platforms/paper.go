package platform

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
)

var PaperTypePlatform = bucket.PlatformType{
	Name:    "paper",
	Install: InstallPaper,
	Detect:  DetectPaper,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return &PaperPlatform{SpigotPlatform{bucket.ContextPlatform{Context: context}}}
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

func DetectPaper(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "paperclip")
	})

	if err != nil {
		log.Println("Error during platform check:", err)
	}

	if res {
		return &PaperPlatform{SpigotPlatform{bucket.ContextPlatform{Context: context}}}, nil
	} else {
		return nil, nil
	}
}

func InstallPaper(context *bucket.OpenContext) error {
	return nil
}
