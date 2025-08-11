package bucket

import (
	"archive/zip"
	"cmp"
	"slices"
	"sync"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/zipfs"
)

const UserAgent = "bucket/0.1 (MRtecno98/bucket)"

func OpenJar(file afero.File) (*afero.Afero, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	reader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return nil, err
	}

	return &afero.Afero{Fs: zipfs.New(reader)}, nil
}

func Decamel(camel string, sep string) string {
	var result string
	for i, r := range camel {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result += sep
		}
		result += string(r)
	}
	return result
}

func Distinct[T cmp.Ordered](slice []T) []T {
	slices.Sort(slice)
	return slices.Compact(slice)
}

func Parallelize(multi bool, tasks ...func() error) error {
	var wait sync.WaitGroup
	wait.Add(len(tasks))

	errs := make(chan error, len(tasks))

	for _, task := range tasks {
		f := func(task func() error) {
			defer wait.Done()

			if err := task(); err != nil {
				errs <- err
			}
		}

		if multi {
			go f(task)
		} else {
			f(task)
		}
	}

	wait.Wait()

	select {
	case err := <-errs:
		return err
	default:
	}

	return nil
}
