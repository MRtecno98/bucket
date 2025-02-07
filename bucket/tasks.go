package bucket

import (
	"log"
	"slices"
)

var TaskComparator = func(a, b *Task) int {
	return a.Compare(b)
}

type TaskFunc func(*OpenContext, *log.Logger) error

type Task struct {
	Name     string
	Func     TaskFunc
	Priority int

	DependsOn []*Task
	AfterTaks []*Task
}

func (t *Task) Compare(other *Task) int {
	if t.Priority == other.Priority {
		return 0
	} else if t.Priority > other.Priority {
		return 1
	} else {
		return -1
	}
}

func (t *Task) Depends() []*Task {
	t.sort()
	return t.DependsOn
}

func (t *Task) After() []*Task {
	t.sort()
	return t.AfterTaks
}

func (t *Task) InnerFunc() TaskFunc {
	return t.Func
}

func (t *Task) sort() {
	slices.SortFunc(t.DependsOn, TaskComparator)
	slices.SortFunc(t.AfterTaks, TaskComparator)
}

func (t *Task) Run(oc *OpenContext, log *log.Logger) error {
	t.sort()

	if len(t.DependsOn) != 0 {
		for _, d := range t.DependsOn {
			if err := d.Run(oc, log); err != nil {
				return err
			}
		}
	}

	if err := t.Func(oc, log); err != nil {
		return err
	}

	if len(t.AfterTaks) != 0 {
		for _, a := range t.AfterTaks {
			if err := a.Run(oc, log); err != nil {
				return err
			}
		}
	}

	return nil
}
