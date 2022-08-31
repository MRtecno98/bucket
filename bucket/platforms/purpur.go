package platform

import "github.com/MRtecno98/bucket/bucket"

var PurpurTypePlatform = bucket.PlatformType{
	Name:    "purpur",
	Install: InstallPurpur,
	Detect:  nil,
	Build: func(context *bucket.OpenContext) bucket.Platform {
		return &PurpurPlatform{
			PaperPlatform{SpigotPlatform{
				bucket.ContextPlatform{Context: context}}}}
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

// No detector, don't know how to distinguish paperclip target between paper and purpur
// (But then plugins are the same, so it doesn't really matter)
// Must be selected manually in bucketrc.yml file

/* func DetectPurpur(context *bucket.OpenContext) (Platform, error) {
	// TODO: Find a way to detect purpur
} */

func InstallPurpur(context *bucket.OpenContext) error {
	return nil
}
