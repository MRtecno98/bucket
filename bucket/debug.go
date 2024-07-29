package bucket

import "fmt"

const DEBUG = false

func LogContexts(w *Workspace) {
	fmt.Println("Available contexts: ")
	for _, v := range w.Contexts {
		fmt.Printf("\tName: %s\n", v.Name)
		fmt.Printf("\t\tURL: %s\n", v.URL)
		fmt.Printf("\t\tFilesystem: %s %v\n", v.Fs.Name(), v.Fs)
		fmt.Printf("\t\tPlatform: %v\n", v.PlatformName())
		fmt.Printf("\t\tRepositories: %d\n", len(v.Repositories))
		for i, r := range v.Repositories {
			fmt.Printf("\t\t  - %s: %v\n", v.Config().Repositories[i].Name, r)
		}

		fmt.Println()
	}
}
