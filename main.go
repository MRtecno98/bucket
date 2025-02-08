package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/bucket/bucket"
	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"

	_ "github.com/MRtecno98/bucket/bucket/platforms"
	_ "github.com/MRtecno98/bucket/bucket/repositories"
	"github.com/MRtecno98/bucket/bucket/tasks"

	_ "github.com/mattn/go-sqlite3"
)

var w *bucket.Workspace

var globalError error

var stamp time.Time
var profile bool

func InitializeContexts(loadDatabase bool) func(*cli.Context) error {
	return func(c *cli.Context) error {
		bucket.LoadSystemConfig(afero.NewOsFs(), c.String("config"))

		clic := c.String("context")
		if clic != "." || len(bucket.GlobalConfig.ActiveContexts) == 0 {
			split := strings.Split(clic, ",")

			cfg := bucket.GlobalConfig
			cfg.ActiveContexts = make([]string, 0, len(split))

			cliCount := 0
			for _, v := range split {
				if slices.Contains(cfg.ContextNames(), v) {
					cfg.ActiveContexts = append(cfg.ActiveContexts, v)
				} else {
					name := "<cli" + strconv.Itoa(cliCount) + ">"
					cliCount++
					cfg.Contexts = append(cfg.Contexts,
						bucket.Context{Name: name, URL: v},
					)

					cfg.ActiveContexts = append(cfg.ActiveContexts, name)
				}
			}
		}

		var err error
		w, err = bucket.GlobalConfig.MakeWorkspace()

		if err != nil {
			log.Print("failed to initialize workspace: ", err)
			return err
		}

		if loadDatabase {
			for _, c := range w.Contexts {
				err = c.LoadPluginDatabase()
				if err != nil {
					return fmt.Errorf("failed to load database for %s: %v", c.Name, err)
				}
			}
		}

		if bucket.DEBUG {
			bucket.LogContexts(w)
			fmt.Println()
		}

		return nil
	}
}

func ShutdownContexts(c *cli.Context) error {
	if w != nil {
		w.CloseWorkspace()
		if len(w.Contexts) <= 1 {
			fmt.Println()
		}
	}

	dur := time.Since(stamp).Truncate(time.Millisecond)

	if globalError != nil {
		fmt.Print(globalError.Error())
		fmt.Printf("FAILURE (took %v)\n", dur)
	} else {
		fmt.Printf("SUCCESS (took %v)\n", dur)
	}

	return nil
}

func main() {
	log.SetPrefix("bucket: ")
	log.SetFlags(0)

	defer pprof.StopCPUProfile()

	(&cli.App{
		Name:  "bucket",
		Usage: "manages spigot servers and plugins",

		UseShortOptionHandling: true,
		Suggest:                true,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "context",
				Aliases:     []string{"c"},
				Usage:       "selects `URL` as the working directory",
				Value:       ".",
				DefaultText: "current directory",
			},

			// TODO: Add repositories argument

			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"f"},
				Usage:   "selects `FILE` as the configuration file",
				Value:   bucket.ConfigName,
			},

			&cli.BoolFlag{
				Name:        "parallel",
				Aliases:     []string{"j"},
				Usage:       "disables multithreaded processes",
				Value:       true,
				Destination: &bucket.GlobalConfig.Multithread,
			},

			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"v"},
				Usage:       "enables debug mode",
				Value:       bucket.DEBUG,
				Destination: &bucket.DEBUG,
			},

			&cli.BoolFlag{
				Name:        "cpuprofile",
				Usage:       "write a cpu profile",
				Destination: &profile,
			},

			&cli.BoolFlag{
				Name:    "plain",
				Usage:   "disables ANSI formatted output",
				EnvVars: []string{"bucket.plain"},
			},
		},

		Before: func(c *cli.Context) error {
			if profile {
				log.Println("starting CPU profile")

				f, err := os.Create(bucket.NewProfileFilename())
				if err != nil {
					return err
				}

				err = pprof.StartCPUProfile(f)
				if err != nil {
					return err
				}
			}

			if c.Bool("plain") {
				os.Setenv("bucket.plain", "true")
			}

			stamp = time.Now()
			return nil
		},

		ExitErrHandler: func(c *cli.Context, err error) {
			if err != nil {
				globalError = multierror.Append(globalError, err)
			}
		},

		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "adds a plugin to the server",
				Before:  InitializeContexts(true),
				After:   ShutdownContexts,
				Action: func(c *cli.Context) error {
					return w.RunTaskWithContext(tasks.AddPlugin)
				},
			},

			{
				Name:   "debug",
				Usage:  "runs debug routine",
				Before: InitializeContexts(true),
				After:  ShutdownContexts,
				Action: func(c *cli.Context) error {
					return w.RunWithContext("debug", func(oc *bucket.OpenContext, log *log.Logger) error {
						return bucket.DebugRoutine(oc, log)
					})
				},
			},

			{
				Name:    "clean",
				Aliases: []string{"c"},
				Usage:   "discards the plugin cache",
				Before:  InitializeContexts(false),
				After:   ShutdownContexts,
				Action: func(c *cli.Context) error {
					return w.RunWithContext("clean", func(oc *bucket.OpenContext, log *log.Logger) error {
						log.Println("deleting database file")
						return oc.CleanCache()
					})
				},
			},
		},
	}).Run(os.Args)
}
