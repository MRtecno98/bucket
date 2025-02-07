package tasks

import (
	"log"

	"github.com/MRtecno98/bucket/bucket"
)

var AddPlugin = &bucket.Task{
	Name:     "add",
	Priority: 0,

	DependsOn: []*bucket.Task{
		{
			Name: "test-task",
			Func: func(oc *bucket.OpenContext, log *log.Logger) error {
				log.Println("test depend")
				return nil
			},
		},
	},

	Func: func(oc *bucket.OpenContext, log *log.Logger) error {
		log.Println("test new task system")
		return nil
	},
}
