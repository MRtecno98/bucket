package platforms

import (
	"log"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
)

var PurpurTypePlatform = bucket.PlatformType{
	Name:    "purpur",
	Install: InstallPurpur,
	Detect:  DetectPurpur,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return NewPurpurPlatform(context) // Go boilerplate
	},
}

func init() {
	bucket.RegisterPlatform(PurpurTypePlatform, 20)
}

type PurpurPlatform struct {
	PaperPlatform
}

func (p *PurpurPlatform) Type() bucket.PlatformType {
	return PurpurTypePlatform
}

func NewPurpurPlatform(context *bucket.OpenContext) *PurpurPlatform {
	return &PurpurPlatform{*NewPaperPlatform(context)}
}

func DetectPurpur(context *bucket.OpenContext) (bucket.Platform, error) {
	res, err := bucket.DetectJarPath(context, func(path string) bool {
		return strings.Contains(path, "purpurmc")
	})

	if err != nil {
		log.Println("error during platform check:", err)
	}

	if res {
		return NewPurpurPlatform(context), nil
	} else {
		return nil, nil
	}
}

func InstallPurpur(context *bucket.OpenContext) error {
	return nil
}
