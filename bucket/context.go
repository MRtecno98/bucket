package bucket

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/resolver"
	"github.com/MRtecno98/bucket/bucket/util"
	"github.com/hashicorp/go-multierror"
)

const SIMILARITY_THRESHOLD float64 = 0.51

var DEFAULT_REPOSITORIES = [...]string{"spigotmc", "modrinth"}

type Context struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type OpenContext struct {
	Context
	Fs           afero.Afero
	LocalConfig  *Config
	Platform     Platform
	Repositories map[string]NamedRepository

	Plugins *SymmetricBiMap[string, CachedPlugin]

	Database *sql.DB
}

type Workspace struct {
	Contexts []*OpenContext
}

func (w *Workspace) RunWithContext(name string, action func(*OpenContext, *log.Logger) error) error {
	var res error = &multierror.Error{Errors: []error{}}
	var newline bool = false
	for _, c := range w.Contexts {
		fmt.Printf(":%s [%s]\n", name, c.Name)

		out := util.NewLookbackCountingWriter(os.Stdout, 2)
		logger := log.New(out, "", log.Lmsgprefix)

		err := action(c, logger)
		if err != nil {
			res = multierror.Append(fmt.Errorf("%s: %v", c.Name, err), res)
		}

		if out.BytesWritten > 0 {
			for _, v := range out.LastBytes {
				if v != '\n' {
					logger.Println()
				}
			}

			newline = true
		}

		if err != nil {
			logger.Printf(":%s [%s] FAILED: %s\n\n", name, c.Name, err)
			newline = true
		}
	}

	if !newline && len(w.Contexts) > 1 {
		fmt.Println()
	}

	if res.(*multierror.Error).Len() > 0 {
		return res
	} else {
		return nil
	}
}

func (w *Workspace) CloseWorkspace() {
	w.RunWithContext("close", func(c *OpenContext, log *log.Logger) error {
		c.CloseContext()
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

	ctx := &OpenContext{Context: c, Fs: afero.Afero{Fs: fs},
		LocalConfig:  conf,
		Repositories: make(map[string]NamedRepository),
		Plugins:      NewPluginBiMap()}

	return ctx, Parallelize(
		ctx.LoadRepositories,
		ctx.LoadPlatform,
		ctx.InitialiazeDatabase)
}

func (c *OpenContext) CloseContext() {
	if c.Database != nil {
		// c.SavePluginDatabase() // Maybe not necessary
		c.Database.Close()
	}

	c.Fs.Close()
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

	if rem, ok := c.Plugins.GetAny(plugin.GetIdentifier()); ok {
		return &rem, nil
	}

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

					if score >= 1.0 {
						break
					}
				}
			}

			if DEBUG {
				for _, v := range keys {
					log.Printf("%s candidate: %s\t\t\tscore: %f [%s]\n",
						plugin.GetName(), scores[v].GetName(), v, scores[v].GetRepository().Provider())
				}
			}

			if len(keys) == 0 {
				gerr = multierror.Append(gerr, fmt.Errorf(
					"%d candidates found for \"%s\" but none are compatible with platform \"%s\"",
					len(candidates), plugin.GetName(), c.Platform.Type().Name))
				continue
			}

			slices.Sort(keys)
			match := keys[len(keys)-1]

			if match < SIMILARITY_THRESHOLD {
				gerr = multierror.Append(gerr, fmt.Errorf(
					"%d candidates found for \"%s\" but none satisfy similarity treshold, closest match was %f",
					len(candidates), plugin.GetName(), match))
				continue
			}

			if local, ok := plugin.(LocalPlugin); ok {
				res := CachedMatch(local, scores[match], r, match)
				if err := c.SavePlugin(res); err != nil {
					return nil, err
				}

				return &res, nil
			}

			return scores[match], nil
		}
	}

	return nil, gerr
}

func (c *OpenContext) RepositoryByNameOrProvider(name string) *NamedRepository {
	if v, ok := c.Repositories[name]; ok {
		return &v
	}

	if v := c.RepositoryByProvider(name); v != nil {
		return v
	}

	return nil
}

func (c *OpenContext) RepositoryByProvider(provider string) *NamedRepository {
	for _, v := range c.Repositories {
		if v.Repository.Provider() == provider {
			return &v
		}
	}

	return nil
}

func (c *OpenContext) LoadRepositories() error {
	repos := c.Config().Repositories
	if len(repos) == 0 {
		repos = make([]RepositoryConfig, len(DEFAULT_REPOSITORIES))
		for i, v := range DEFAULT_REPOSITORIES {
			repos[i] = RepositoryConfig{Provider: v}
		}
	}

	for _, v := range repos {
		if r, err := v.MakeRepository(c); err != nil {
			return err
		} else {
			c.Repositories[v.GetName()] = *r
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
