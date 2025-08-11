package bucket

import (
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/resolver"
	"github.com/MRtecno98/bucket/bucket/util"
	"github.com/hashicorp/go-multierror"
)

const SimilarityTreshold float64 = 0.51

var DefaultRepositories = [...]string{"spigotmc", "modrinth"}

type Context struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type OpenContext struct {
	Context
	PluginDatabase

	Fs           afero.Afero
	LocalConfig  *Config
	Platform     Platform
	Repositories map[string]NamedRepository
}

type Workspace struct {
	Contexts []*OpenContext
}

func (c *OpenContext) RunTask(task *Task) (error, bool) {
	var newline, n bool
	var err error

	for _, t := range task.Depends() {
		err, n = c.RunTask(t)
		newline = newline || n

		if err != nil {
			return err, newline || n
		}
	}

	err, n = c.Run(task.Name, task.Func)
	newline = newline || n

	if err != nil {
		return err, newline
	}

	for _, t := range task.After() {
		err, n = c.RunTask(t)
		newline = newline || n

		if err != nil {
			return err, newline || n
		}
	}

	return nil, newline
}

func (c *OpenContext) Run(name string, action TaskFunc) (error, bool) {
	var newline bool = false

	fmt.Printf(":%s [%s]\n", name, c.Name)

	out := util.NewLookbackCountingWriter(os.Stdout, 2)
	logger := log.New(out, "", log.Lmsgprefix)

	err := action(c, logger)

	if out.BytesWritten > 0 {
		for _, v := range slices.Clone(out.LastBytes) {
			if v != '\n' {
				logger.Println()
			}
		}

		newline = true
	}

	if err != nil {
		logger.Printf(":%s [%s] FAILED: %s\n\n", name, c.Name, err)
		return fmt.Errorf("%s: %v", c.Name, err), true
	}

	return nil, newline
}

func (w *Workspace) RunWithContext(name string, action TaskFunc) error {
	return w.RunTaskWithContext(&Task{Name: name, Func: action})
}

func (w *Workspace) RunTaskWithContext(task *Task) error {
	var res error = &multierror.Error{Errors: []error{}}
	var newline bool = false

	for _, c := range w.Contexts {
		err, n := c.RunTask(task)
		newline = newline || n

		multierror.Append(res, err)
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

	var conf *Config = &Config{}
	conf, err = LoadFilesystemConfig(fs, ConfigName)
	if err == nil {
		if conf == nil {
			conf = &Config{}
		}

		conf.Collapse(GlobalConfig) // Also add base options
	}

	sumdb, err := LoadSumDB(conf.SumDB)
	if err != nil {
		return nil, err
	}

	ctx := &OpenContext{Context: c, Fs: afero.Afero{Fs: fs},
		LocalConfig:    conf,
		Repositories:   make(map[string]NamedRepository),
		PluginDatabase: sumdb}

	return ctx, Parallelize(ctx.LocalConfig.Multithread,
		ctx.LoadRepositories,
		ctx.LoadPlatform,
		ctx.InitializeDatabase)
}

func LoadSumDB(name string) (PluginDatabase, error) {
	switch name {
	case SumDBSqlite:
		return NewSqliteDatabase(), nil
	case SumDBFile:
		return NewSumfileDatabase(), nil
	default:
		return nil, fmt.Errorf("unknown sumdb type: %s", name)
	}
}

func (c *OpenContext) CloseContext() {
	c.CloseDatabase()
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

func (c *OpenContext) PluginsSize() (int64, error) {
	var size int64 = 0

	if ex, err := c.Fs.DirExists(c.Platform.PluginsFolder()); !ex || err != nil {
		return 0, err
	}

	if err := c.Fs.Walk(c.Platform.PluginsFolder(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return size, nil
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

func (c *OpenContext) InitializeDatabase() error {
	if DEBUG {
		log.Printf("initializing database for %s\n", c.Name)
	}

	return c.PluginDatabase.InitializeDatabase(c)
}

func (c *OpenContext) ResolvePlugin(plugin Plugin) (RemotePlugin, error) {
	var gerr error

	if rem, ok := c.Plugins().GetAny(plugin.GetIdentifier()); ok {
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

			if match < SimilarityTreshold {
				gerr = multierror.Append(gerr, fmt.Errorf(
					"%d candidates found for \"%s\" but none satisfy similarity treshold, closest match was %f",
					len(candidates), plugin.GetName(), match))
				continue
			}

			if local, ok := plugin.(*LocalPlugin); ok {
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
		repos = make([]RepositoryConfig, len(DefaultRepositories))
		for i, v := range DefaultRepositories {
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
		err := Parallelize(GlobalConfig.Multithread,
			func() error {
				op, err := v.OpenContext()
				if err != nil {
					return err
				} else {
					opened[i] = op
					return nil
				}
			})

		if err != nil {
			return nil, err
		}
	}

	return &Workspace{Contexts: opened}, nil
}
