package bucket

import (
	"fmt"
	"log"
	"os"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/resolver"
	"github.com/MRtecno98/bucket/bucket/util"
	"github.com/hashicorp/go-multierror"
)

type Context struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type OpenContext struct {
	Context
	Fs          afero.Afero
	LocalConfig *Config
	Platform    Platform
}

type Workspace struct {
	Contexts []*OpenContext
}

func (w *Workspace) RunWithContext(name string, action func(*OpenContext, *log.Logger) error) error {
	var res error
	for _, c := range w.Contexts {
		fmt.Printf(":%s [%s]\n", name, c.Name)

		out := util.NewCountingWriter(os.Stdout)
		logger := log.New(out, "", log.Lmsgprefix)

		err := action(c, logger)
		res = multierror.Append(err, res)

		if out.BytesWritten > 0 {
			logger.Print("\n")
		}

		if err != nil {
			logger.Printf(":%s [%s] FAILED: %s\n\n", name, c.Name, err)
		}
	}

	if res.(*multierror.Error).Len() > 0 {
		return res
	} else {
		return nil
	}
}

func (w *Workspace) CloseWorkspace() {
	w.RunWithContext("close", func(c *OpenContext, log *log.Logger) error {
		c.Fs.Close()
		return nil
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

	ctx := &OpenContext{Context: c, Fs: afero.Afero{Fs: fs}, LocalConfig: conf}

	return ctx, ctx.LoadPlatform()
}

func (c *OpenContext) PlatformName() string {
	if c.Platform != nil {
		return c.Platform.Type().Name
	} else {
		return "none"
	}
}

func (c *OpenContext) LoadPlatform() error {
	if c.LocalConfig != nil && c.LocalConfig.Platform != "" {
		if pltype, ok := platforms[c.LocalConfig.Platform]; ok {
			c.Platform = pltype.Platform.Build(c)
		} else {
			return fmt.Errorf("unknown platform: %s", c.LocalConfig.Platform)
		}
	} else {
		plat, err := DetectPlatform(c)
		if err != nil {
			return err
		}

		c.Platform = plat
	}

	return nil
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
