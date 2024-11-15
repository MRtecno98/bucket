package bucket

import (
	"fmt"
	"os"
)

func NewProfileFilename() string {
	for i := 0; ; i++ {
		filename := fmt.Sprintf("profile.%d.pprof", i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return filename
		}
	}
}
