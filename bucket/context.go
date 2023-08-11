package bucket

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/resolver"
	"github.com/MRtecno98/bucket/bucket/util"
	"github.com/hashicorp/go-multierror"
)

const SIMILARITY_THRESHOLD float64 = 0.5

type Context struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type OpenContext struct {
	Context
	Fs           afero.Afero
	LocalConfig  *Config
	Platform     Platform
	Repositories []Repository
}

type Workspace struct {
	Contexts []*OpenContext
}

func (w *Workspace) RunWithContext(name string, action func(*OpenContext, *log.Logger) error) error {
	var res error
	for _, c := range w.Contexts {
		fmt.Printf(":%s [%s]\n", name, c.Name)

		out := util.NewLookbackCountingWriter(os.Stdout, 2)
		logger := log.New(out, "", log.Lmsgprefix)

		err := action(c, logger)
		res = multierror.Append(err, res)

		if out.BytesWritten > 0 {
			for _, v := range out.LastBytes {
				if v != '\n' {
					logger.Print("\n")
				}
			}
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

	if err := ctx.LoadRepositories(); err != nil {
		return nil, err
	}

	return ctx, ctx.LoadPlatform()
}

func (c *OpenContext) PlatformName() string {
	if c.Platform != nil {
		return c.Platform.Type().Name
	} else {
		return "none"
	}
}

func (c *OpenContext) Config() *Config {
	if c.LocalConfig != nil {
		return c.LocalConfig
	} else {
		return GlobalConfig
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

func (c *OpenContext) ResolvePlugin(plugin Plugin) (RemotePlugin, error) {
	var gerr error

	for _, r := range c.Repositories {
		if _, candidates, err := r.Resolve(plugin); err != nil {
			gerr = multierror.Append(gerr, err)
			continue
		} else if len(candidates) > 0 {
			keys := make([]float64, 0, len(candidates))
			scores := make(map[float64]RemotePlugin)
			for _, pl := range candidates {
				if pl.Compatible(c.Platform.Type()) {
					score := ComparisonIndex(plugin, pl)
					keys = append(keys, score)
					scores[score] = pl
				}
			}

			if len(keys) == 0 {
				gerr = multierror.Append(gerr, fmt.Errorf(
					"%d candidates found for \"%s\" but none are compatible with platform \"%s\"",
					len(candidates), plugin.GetName(), c.Platform.Type().Name))
				continue
			}

			sort.Float64s(keys)
			match := keys[len(keys)-1]

			if match < SIMILARITY_THRESHOLD {
				gerr = multierror.Append(gerr, fmt.Errorf(
					"%d candidates found for \"%s\" but none satisfy similarity treshold, closest match was %f",
					len(candidates), plugin.GetName(), match))
				continue
			}

			return scores[match], nil
		}
	}

	return nil, gerr
}

func (c *OpenContext) LoadRepositories() error {
	for _, v := range c.Config().Repositories {
		if constr, ok := Repositories[v.Name]; ok {
			c.Repositories = append(c.Repositories, constr(context.Background(), c, v.Options)) // TODO: Use another context
		} else {
			return fmt.Errorf("unknown repository: %s", v.Name)
		}
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
