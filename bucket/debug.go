package bucket

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/MRtecno98/afero"
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
