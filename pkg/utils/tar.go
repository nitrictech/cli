package utils

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/jhoonb/archivex"
)

func TarReaderFromString(name, src string) (io.Reader, error) {
	tar := new(archivex.TarFile)
	tarReader := bytes.Buffer{}
	err := tar.CreateWriter(name+".tar", &tarReader)
	if err != nil {
		return nil, err
	}

	err = tar.Add(src, strings.NewReader(src), NewStringFileInfo(name, src))
	if err != nil {
		return nil, err
	}

	tar.Close()

	return &tarReader, nil
}

func TarReaderFromPath(src string) (io.Reader, error) {
	tar := new(archivex.TarFile)
	tarReader := bytes.Buffer{}
	err := tar.CreateWriter(src+".tar", &tarReader)
	if err != nil {
		return nil, err
	}

	ss, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if ss.IsDir() {
		err = tar.AddAll(src, false)
		if err != nil {
			return nil, err
		}
	} else {
		file, err := os.Open(src)
		if err != nil {
			return nil, err
		}

		err = tar.Add(src, file, ss)
		if err != nil {
			return nil, err
		}
	}

	tar.Close()

	return &tarReader, nil
}
