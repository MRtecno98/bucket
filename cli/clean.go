package cli

import (
	"log"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/urfave/cli/v2"
)

var CLEAN = &cli.Command{
	Name:    "clean",
	Aliases: []string{"c"},
	Usage:   "discards the plugin cache",
	Before:  InitializeContexts(false),
	After:   ShutdownContexts,
	Action: func(c *cli.Context) error {
		return Workspace.RunWithContext("clean", func(oc *bucket.OpenContext, log *log.Logger) error {
			size, err := oc.DBSize()
			if err != nil {
				return err
			}

			log.Printf("deleting database file (%d KB)\n", size/1024)
			if err := oc.CleanCache(); err != nil {
				return err
			}

			if c.Args().Len() > 0 {
				if c.Args().Get(0) == "all" {
					var size int64

					if size, err = oc.PluginsSize(); err != nil {
						return err
					}

					log.Printf("deleting plugin cache (%.2f MB)\n", float64(size)/1024/1024)
					if err = oc.Fs.RemoveAll(oc.Platform.PluginsFolder()); err != nil {
						return err
					}
				}
			}

			return nil
		})
	},
}
