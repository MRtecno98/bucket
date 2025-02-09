package cli

import (
	"log"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/urfave/cli/v2"
)

var DEBUG = &cli.Command{
	Name:   "debug",
	Usage:  "runs debug routine",
	Before: InitializeContexts(true),
	After:  ShutdownContexts,
	Action: func(c *cli.Context) error {
		return Workspace.RunWithContext("debug", func(oc *bucket.OpenContext, log *log.Logger) error {
			return bucket.DebugRoutine(oc, log)
		})
	},
}
