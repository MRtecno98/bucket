package bucket

import (
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type PlatformType struct {
	Name    string
	Install func(context *OpenContext) error
	Detect  func(context *OpenContext) (Platform, error)
	Build   func(context *OpenContext) Platform
}

type PluginProvider interface {
	PluginsFolder() string
	Plugins() ([]Plugin, []error, error)
	LoadPlugin(filename string) (Plugin, error)
}

type Platform interface {
	Type() PlatformType

	PluginProvider
}

type ContextPlatform struct {
	Context *OpenContext
}

type PluginCachePlatform struct {
	PluginProvider
	PluginsCache []Plugin
}

type JarPluginPlatform[PluginType Plugin] struct {
	ContextPlatform
	PluginFile   string
	PluginFolder string
}

func (p *PluginCachePlatform) Plugins() ([]Plugin, []error, error) {
	if p.PluginsCache != nil {
		return p.PluginsCache, nil, nil
	} else {
		plugins, errs, err := p.PluginProvider.Plugins()
		if err != nil {
			p.PluginsCache = plugins
		}

		return plugins, errs, err
	}
}

func (p JarPluginPlatform[PluginType]) PluginsFolder() string {
	return p.PluginFolder
}

func (p JarPluginPlatform[PluginType]) Plugins() ([]Plugin, []error, error) {
	ok, err := p.Context.Fs.DirExists(p.PluginsFolder())
	if err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, errors.New("invalid server layout: plugins folder does not exist")
	}

	files, err := p.Context.Fs.ReadDir(p.PluginsFolder())
	if err != nil {
		return nil, nil, err
	}

	plugins := make([]Plugin, 0)
	errs := make([]error, 0)

	c := make(chan struct {
		Plugin Plugin
		Error  error
	}, len(files))

	count := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".jar") {
			count++
			go func() {
				plugin, err := p.LoadPlugin(file.Name())

				c <- struct {
					Plugin Plugin
					Error  error
				}{plugin, err}
			}()
		}
	}

	for i := 0; i < count; i++ {
		r := <-c

		if r.Error != nil {
			errs = append(errs, err)
			continue
		}

		plugins = append(plugins, r.Plugin)
	}

	if len(errs) > 0 {
		err = errors.New("some plugins couldn't be loaded")
	}

	return plugins, errs, err
}

func (p JarPluginPlatform[PluginType]) LoadPlugin(filename string) (Plugin, error) {
	file, err := p.Context.Fs.Open(p.PluginsFolder() + "/" + filename)
	if err != nil {
		return nil, err
	}

	jar, err := OpenJar(file)
	if err != nil {
		return nil, err
	}

	descriptor, err := jar.Open(p.PluginFile)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(descriptor)
	if err != nil {
		return nil, err
	}

	var pl PluginType
	yaml.Unmarshal(data, &pl)

	return pl, nil
}
