package cli

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/MRtecno98/afero"
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
			const REPO = "modrinth"

			if oc.Platform == nil {
				return cli.Exit("no platform set", 1)
			}

			res, _, err := oc.Repositories[REPO].Search(c.Args().Get(0), 5)
			if err != nil {
				return err
			}

			if len(res) > 5 {
				res = res[:5]
			} else if len(res) == 0 {
				return cli.Exit("no plugins found", 1)
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

			oc.Fs.MkdirAll(oc.Platform.PluginsFolder(), 0755)
			folder := afero.NewBasePathFs(oc.Fs, oc.Platform.PluginsFolder())

			for _, f := range files {
				if !f.Optional() {
					log.Printf("Downloading %s\n", f.Name())

					data, err := f.Download()
					if err != nil {
						return err
					}

					defer data.Close()

					fd, err := folder.OpenFile(f.Name(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
					if err != nil {
						return err
					}

					defer fd.Close()

					if _, err := io.Copy(fd, data); err != nil {
						return err
					}

					if err := fd.Close(); err != nil {
						return err
					}

					if err := f.Verify(); err != nil {
						return err
					} else {
						log.Printf("File %s verified\n", f.Name())
					}

					pl, err := oc.Platform.LoadPlugin(f.Name())
					if err != nil {
						return err
					}

					err = oc.SavePlugin(bucket.CachedMatch(pl, ver, oc.Repositories[REPO], 1.0))
					if err != nil {
						return err
					}

					log.Printf("Plugin %s saved\n", pl.GetName())
				} else {
					log.Printf("Skipping optional file %s\n", f.Name())
				}
			}

			return nil
		})
	},
}
