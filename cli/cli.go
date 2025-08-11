// Package cli provides the command line interface for the bucket application.
package cli

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/bucket/bucket"
	"github.com/urfave/cli/v2"
)

var Workspace *bucket.Workspace

var GlobalError error
var Time time.Time

var Commands = []*cli.Command{
	ADD, CLEAN, DEBUG, // LIST, REMOVE, RUN, SEARCH, UPDATE,
}

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
		Workspace, err = bucket.GlobalConfig.MakeWorkspace()

		if err != nil {
			log.Print("failed to initialize workspace: ", err)
			return err
		}

		if loadDatabase {
			for _, c := range Workspace.Contexts {
				err = c.LoadPluginDatabase()
				if err != nil {
					return fmt.Errorf("failed to load database for %s: %v", c.Name, err)
				}
			}
		}

		if bucket.DEBUG {
			bucket.LogContexts(Workspace)
		}

		return nil
	}
}

func ShutdownContexts(c *cli.Context) error {
	if Workspace != nil {
		Workspace.CloseWorkspace()
		if len(Workspace.Contexts) <= 1 {
			fmt.Println()
		}
	} else {
		fmt.Println()
	}

	dur := time.Since(Time).Truncate(time.Millisecond)

	if GlobalError != nil {
		fmt.Print(GlobalError.Error())
		fmt.Printf("FAILURE (took %v)\n", dur)
	} else {
		fmt.Printf("SUCCESS (took %v)\n", dur)
	}

	return nil
}
