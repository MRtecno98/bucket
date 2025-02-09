package tasks

import (
	"log"
	"os"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/MRtecno98/bucket/cli"
)

var AddPlugin = &bucket.Task{
	Name:     "add",
	Priority: 0,

	Func: func(oc *bucket.OpenContext, log *log.Logger) error {
		res, _, err := oc.Repositories["spigotmc"].Search(os.Args[2], 5)
		if err != nil {
			return err
		}

		if len(res) > 5 {
			res = res[:5]
		}

		options := make([]string, len(res))
		for i, v := range res {
			options[i] = v.GetName()
		}

		log.Println("\n- Select a plugin to install")

		n, err := cli.TableSelect(options, os.Stderr)

		if err != nil {
			return err
		}

		log.Printf("Selected %s", options[n])

		return nil
	},
}
