package bucket

import (
	"log"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/resolver"
)

type Context struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type OpenContext struct {
	Context
	Fs          afero.Fs
	LocalConfig *Config
}

type Workspace struct {
	Contexts []*OpenContext
}

func (w *Workspace) RunWithContext(name string, action func(*OpenContext)) {
	for _, c := range w.Contexts {
		log.Printf("Running [ %s ] for < %s >\n", name, c.Name)
		action(c)
	}
}

func (w *Workspace) CloseWorkspace() {
	w.RunWithContext("close", func(c *OpenContext) {
		c.Fs.Close()
	})
}

func (c Context) OpenContext() (*OpenContext, error) {
	fs, err := resolver.OpenUrl(c.URL)
	if err != nil {
		return nil, err
	}

	var conf *Config = nil
	conf, err = LoadFilesystemConfig(fs, ConfigName)
	if err == nil && conf != nil {
		conf.Collapse(GlobalConfig) // Also add base options
	}

	return &OpenContext{Context: c, Fs: fs, LocalConfig: conf}, nil
}

func CreateWorkspace(contexts ...Context) (*Workspace, error) {
	opened := make([]*OpenContext, len(contexts))

	for i, v := range contexts {
		op, err := v.OpenContext()
		if err != nil {
			return nil, err
		} else {
			opened[i] = op
		}
	}

	return &Workspace{Contexts: opened}, nil
}
