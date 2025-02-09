package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/urfave/cli/v2"
)

var ADD = &cli.Command{
	Name:    "add",
	Aliases: []string{"a"},
	Usage:   "adds a plugin to the server",
	Before:  InitializeContexts(true),
	After:   ShutdownContexts,

	Args:      true,
	ArgsUsage: " name",

	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			return cli.Exit("missing plugin name", 1)
		}

		return Workspace.RunWithContext("add", func(oc *bucket.OpenContext, log *log.Logger) error {
			res, _, err := oc.Repositories["modrinth"].Search(c.Args().Get(0), 5)
			if err != nil {
				return err
			}

			if len(res) > 5 {
				res = res[:5]
			}

			options := make([]string, len(res))
			for i, v := range res {
				options[i] = fmt.Sprintf("[%s] %s", v.GetRepository().Provider(), v.GetName())
			}

			log.Println("Select a plugin to install")

			n, err := TableSelect(options, os.Stderr)

			if err != nil {
				return err
			}

			log.Printf("Selected %s\n\n", options[n])

			pl := res[n]
			ver, err := pl.GetLatestVersion()
			if err != nil {
				return err
			}

			log.Printf("Installing %s [%s]\n", pl.GetName(), ver.GetIdentifier())

			files, err := ver.GetFiles()
			if err != nil {
				return err
			}

			for _, f := range files {
				if !f.Optional() {
					log.Printf("Downloading %s\n", f.Name())
					// TODO: Download file
				} else {
					log.Printf("Skipping optional file %s\n", f.Name())
				}
			}

			return nil
		})
	},
}
