package bucket

import (
	"archive/zip"

	"github.com/MRtecno98/afero"
	"github.com/MRtecno98/afero/zipfs"
)

const USER_AGENT = "bucket/0.1 (MRtecno98/bucket)"

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
