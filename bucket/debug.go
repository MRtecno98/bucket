package bucket

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"unsafe"

	"github.com/MRtecno98/afero"
	"github.com/hashicorp/go-multierror"
)

var DEBUG = false

func LogContexts(w *Workspace) {
	fmt.Println("Available contexts: ")
	for _, v := range w.Contexts {
		var fs afero.Fs = v.Fs.Fs
		if o, ok := v.Fs.Fs.(*afero.BasePathFs); ok {
			// Super hacky way to get the source filesystem
			val := reflect.ValueOf(o).Elem()
			fd := val.FieldByName("source")
			fd = reflect.NewAt(fd.Type(), unsafe.Pointer(fd.UnsafeAddr())).Elem()

			if fs, ok = fd.Interface().(afero.Fs); !ok {
				fs = o
			}
		}

		fmt.Printf("\tName: %s\n", v.Name)
		fmt.Printf("\t\tURL: %s\n", v.URL)
		fmt.Printf("\t\tFilesystem: %s %v\n", fs.Name(), fs)
		fmt.Printf("\t\tPlatform: %v\n", v.PlatformName())
		fmt.Printf("\t\tRepositories: %d\n", len(v.Repositories))
		for n, r := range v.Repositories {
			fmt.Printf("\t\t  - %s: %v\n", n, r)
		}

		fmt.Println()
	}
}

func DebugRoutine(oc *OpenContext, logger *log.Logger) error {
	if oc.Platform == nil {
		// TODO: make so that we don't have to repeat this for every action
		return fmt.Errorf("no platform detected")
	}

	pls, perrs, err := oc.Platform.Plugins()
	if perrs != nil {
		return multierror.Append(err, perrs...)
	}

	if err != nil {
		return err
	}

	var wait sync.WaitGroup
	wait.Add(len(pls))

	var errs error = nil

	for _, pli := range pls {
		f := func(pl Plugin) {
			defer wait.Done()
			res, err := oc.ResolvePlugin(pl)
			if err != nil {
				multierror.Append(errs,
					fmt.Errorf("error resolving plugin %s: %v", pl.GetName(), err))
				return
			}

			ver, err := res.GetLatestVersion()
			if err != nil {
				multierror.Append(errs,
					fmt.Errorf("error getting latest version for %s: %v", res.GetIdentifier(), err))
				return
			}

			var ind float64
			if c, ok := res.(*CachedPlugin); ok {
				ind = c.Confidence
			} else {
				ind = ComparisonIndex(pl, res)
			}

			logger.Printf("found plugin: %s [%s] %s %s%s %f\n", pl.GetName(), res.GetRepository().Provider(), res.GetName(),
				ver.GetName(), res.GetAuthors(), ind)
		}

		if GlobalConfig.Multithread {
			go f(pli)
		} else {
			f(pli)
		}
	}

	wait.Wait()

	return errs
}
