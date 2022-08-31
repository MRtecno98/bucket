package bucket

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/zipfs"
	"golang.org/x/exp/maps"
)

type PlatformType struct {
	Name    string
	Install func(context *OpenContext) error
	Detect  func(context *OpenContext) (Platform, error)
	Build   func(context *OpenContext) Platform
}

type Platform interface {
	Type() PlatformType

	PluginsFolder() string
	Plugins() ([]Plugin, error)
}

type ContextPlatform struct {
	Context *OpenContext
}

type PrioritizedPlatform struct {
	Platform PlatformType
	Priority int
}

var platforms = map[string]PrioritizedPlatform{}

func DetectPlatform(context *OpenContext) (Platform, error) {
	plts := maps.Values(platforms)
	sort.Slice(plts, func(a, b int) bool {
		return plts[a].Priority > plts[b].Priority
	})

	for _, plt := range plts {
		if plt.Platform.Detect == nil {
			continue
		}

		platform, err := plt.Platform.Detect(context)
		if err != nil {
			return nil, err
		}

		if platform != nil {
			return platform, nil
		}
	}

	return nil, nil
}

func RegisterPlatform(platform PlatformType, priority int) {
	platforms[platform.Name] = PrioritizedPlatform{platform, priority}
}

func DetectFromJars(context *OpenContext, filter func(jar *afero.Afero) bool) (bool, error) {
	files, err := context.Fs.ReadDir("")
	if err != nil {
		return false, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".jar" {
			opened, err := context.Fs.Open(file.Name())
			if err != nil {
				return false, err
			}

			defer opened.Close()

			reader, err := zip.NewReader(opened, file.Size())
			if err != nil {
				return false, err
			}

			jar := &afero.Afero{Fs: zipfs.New(reader)}
			if filter(jar) {
				return true, nil
			}
		}
	}

	return false, nil
}

func DetectJarPath(context *OpenContext, filter func(path string) bool) (bool, error) {
	return DetectFromJars(context, func(fs *afero.Afero) bool {
		err := fs.Walk("", func(path string, info os.FileInfo, err error) error {
			if filter(path) {
				return io.EOF
			} else {
				return nil
			}
		})

		if err != nil && err != io.EOF {
			log.Println("Error during jar platform check:", err)
		}

		return err == io.EOF
	})
}
