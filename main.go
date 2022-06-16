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
)

var w *bucket.Workspace

func main() {
	log.SetPrefix("bucket: ")

	(&cli.App{
		Name:  "bucket",
		Usage: "manages spigot servers and plugins",

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
			return
		},

		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "adds a plugin to the server",
				Action: func(c *cli.Context) error {
					fmt.Println("add plugin:", c.Args().First())
					return nil
				},
			},
		},
	}).Run(os.Args)
}
