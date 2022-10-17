package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/bucket/bucket"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	_ "github.com/MRtecno98/bucket/bucket/platforms"
	"github.com/MRtecno98/bucket/bucket/repositories"
	_ "github.com/MRtecno98/bucket/bucket/repositories"
)

var w *bucket.Workspace

func main() {
	log.SetPrefix("bucket: ")
	log.SetFlags(0)

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

			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"f"},
				Usage:   "selects `FILE` as the configuration file",
				Value:   bucket.ConfigName,
			},
		},

		Before: func(c *cli.Context) (err error) {
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

			w, err = bucket.GlobalConfig.MakeWorkspace()

			if err != nil {
				log.Print("failed to initialize workspace: ", err)
				return err
			}

			fmt.Println("Available contexts: ")
			for _, v := range w.Contexts {
				fmt.Printf("\tName: %s\n", v.Name)
				fmt.Printf("\t\tURL: %s\n", v.URL)
				fmt.Printf("\t\tFilesystem: %s %v\n", v.Fs.Name(), v.Fs)
				fmt.Printf("\t\tPlatform: %v\n\n", v.PlatformName())
			}

			return
		},

		After: func(c *cli.Context) error {
			if w != nil {
				w.CloseWorkspace()
			}

			return nil
		},

		ExitErrHandler: func(c *cli.Context, err error) {
			cli.HandleExitCoder(cli.Exit(err, 1))
		},

		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "adds a plugin to the server",
				Action: func(c *cli.Context) error {
					return w.RunWithContext("test-add", func(oc *bucket.OpenContext, log *log.Logger) error {
						pls, errs, err := oc.Platform.Plugins()
						pl, ok := pls[0].(bucket.Depender)

						if ok {
							log.Println(errs, err, pls[0].GetName(), pl.GetDependencies())
						} else {
							log.Println(errs, err, pls[0])
						}

						mpl, err := repositories.NewModrinthRepository().Get(c.Args().First())
						if err != nil {
							return err
						}

						latest, err := mpl.GetLatestVersion()
						if err != nil {
							return err
						}

						log.Println()
						log.Println(mpl.GetName(), mpl.GetAuthors(), latest.GetName())

						log.Printf("add plugin: %s compat: %v", c.Args().First(), mpl.Compatible(oc.Platform.Type()))
						return nil
					})
				},
			},
		},
	}).Run(os.Args)
}
