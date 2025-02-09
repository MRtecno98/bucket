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
			size, err := oc.DbSize()
			if err != nil {
				return err
			}

			log.Printf("deleting database file (%d KB)\n", size/1024)
			return oc.CleanCache()
		})
	},
}
