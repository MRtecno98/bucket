package main

import (
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"

	_ "github.com/MRtecno98/bucket/bucket/platforms"
	_ "github.com/MRtecno98/bucket/bucket/repositories"

	c "github.com/MRtecno98/bucket/cli"

	_ "github.com/mattn/go-sqlite3"
)

var profile bool

func main() {
	log.SetPrefix("bucket: ")
	log.SetFlags(0)

	defer pprof.StopCPUProfile()

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print tool version",
	}

	(&cli.App{
		Name:  "bucket",
		Usage: "manages spigot servers and plugins",

		UseShortOptionHandling: true,
		Suggest:                true,

		Version: "v1.0.0",

		Authors: []*cli.Author{
			{
				Name:  "MRtecno98",
				Email: "mr.tecno98@gmail.com",
			},
		},

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

		Before: func(ctx *cli.Context) error {
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

			if ctx.Bool("plain") {
				os.Setenv("bucket.plain", "true")
			}

			c.Time = time.Now()
			return nil
		},

		ExitErrHandler: func(_ *cli.Context, err error) {
			if err != nil {
				c.GlobalError = multierror.Append(c.GlobalError, err)
			}
		},

		Commands: c.Commands,
	}).Run(os.Args)
}
