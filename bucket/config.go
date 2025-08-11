package bucket

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/MRtecno98/afero"
	"gopkg.in/yaml.v2"
)

const ConfigName string = "bucketrc.yml"

var GlobalConfig *Config = &Config{}

type Config struct {
	ActiveContexts []string           `yaml:"active-contexts"`
	Contexts       []Context          `yaml:"contexts"`
	Platform       string             `yaml:"platform"`
	Multithread    bool               `yaml:"multithread"`
	SumDB          string             `yaml:"sumdb,omitempty"`
	Repositories   []RepositoryConfig `yaml:"repositories"`
}

type RepositoryConfig struct {
	Name     string            `yaml:"name"`
	Provider string            `yaml:"provider"`
	Options  map[string]string `yaml:"options"`
}

func (rc *RepositoryConfig) GetName() string {
	if rc.Name == "" {
		return rc.Provider
	} else {
		return rc.Name
	}
}

func (rc *RepositoryConfig) MakeRepository(oc *OpenContext) (*NamedRepository, error) {
	if constr, ok := Repositories[rc.Provider]; ok {
		return &NamedRepository{RepositoryConfig: *rc,
			Repository: constr(context.TODO(), oc, rc.Options)}, nil
	} else {
		return nil, fmt.Errorf("unknown repository: %s", rc.Name)
	}
}

func (c *Config) MakeWorkspace() (*Workspace, error) {
	active := make([]Context, len(c.ActiveContexts))

	for i, v := range c.ActiveContexts {
		for _, cx := range c.Contexts {
			if cx.Name == v {
				active[i] = cx
			}
		}
	}

	return CreateWorkspace(active...)
}

func (c *Config) ContextNames() []string {
	res := make([]string, len(c.Contexts))

	for i, v := range c.Contexts {
		res[i] = v.Name
	}

	return res
}

func (c *Config) Collapse(o *Config) {
	cv := reflect.ValueOf(c).Elem()
	ov := reflect.ValueOf(o).Elem()

	for i := 0; i < cv.NumField(); i++ {
		cf := cv.Field(i)
		of := ov.FieldByName(cv.Type().Field(i).Name)

		switch of.Type().Kind() {
		case reflect.Array:
		case reflect.Slice:
			cf.Set(reflect.AppendSlice(cf, of))

		default:
			if cf.IsZero() {
				cf.Set(of)
			}
		}
	}
}

func LoadSystemConfig(fs afero.Fs, base string) *Config {
	paths := []string{base}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".bucket", ConfigName))
	}

	conf := Config{}
	for _, v := range paths {
		parse, _ := LoadFilesystemConfig(fs, v)
		if parse != nil {
			conf.Collapse(parse)
		}
	}

	GlobalConfig.Collapse(&conf)

	return GlobalConfig
}

func LoadFilesystemConfig(fs afero.Fs, path string) (conf *Config, err error) {
	if file, err := fs.Open(path); err == nil {
		if conf, err = LoadConfigFrom(file); err != nil {
			log.Println("Found config file while opening context but failed to parse it", err)
			return nil, err
		}
	}

	return
}

func LoadConfigFrom(f io.Reader) (*Config, error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var c Config
	if yaml.Unmarshal(data, &c) != nil {
		return nil, err
	}

	return &c, nil
}
