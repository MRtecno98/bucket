package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/bucket/bucket"
	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"

	_ "github.com/MRtecno98/bucket/bucket/platforms"
	_ "github.com/MRtecno98/bucket/bucket/repositories"

	_ "github.com/mattn/go-sqlite3"
)

var w *bucket.Workspace

var globalError error

var stamp time.Time

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

	stamp = time.Now()

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
					return w.RunWithContext("add", func(oc *bucket.OpenContext, log *log.Logger) error {
						if oc.Platform == nil {
							// TODO: make so that we don't have to repeat this for every action
							return fmt.Errorf("no platform detected")
						}

						pls, _, err := oc.Platform.Plugins()
						if err != nil {
							return err
						}

						var wait sync.WaitGroup
						wait.Add(len(pls))
						for _, pli := range pls {
							go func(pl bucket.Plugin) {
								defer wait.Done()
								res, err := oc.ResolvePlugin(pl)
								if err != nil {
									log.Printf("error resolving plugin %s: %v\n", pl.GetName(), err)
									return
								}

								ver, err := res.GetLatestVersion()
								if err != nil {
									log.Printf("error getting latest version for %s: %v\n", res.GetIdentifier(), err)
									return
								}

								var ind float64
								if c, ok := res.(*bucket.CachedPlugin); ok {
									ind = c.Confidence
								} else {
									ind = bucket.ComparisonIndex(pl, res)
								}

								log.Printf("found plugin: %s\t%s %s%s\t%f\n", pl.GetName(), res.GetName(),
									ver.GetName(), res.GetAuthors(), ind)
							}(pli)
						}

						wait.Wait()

						return nil
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
