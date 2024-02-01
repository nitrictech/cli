package paths

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// This is an alternative implementation of useful functions from the filepath package, that uses Afero to access the file system.

// Glob returns a list of files that match the given pattern.
func Glob(fs afero.Fs, dir string, pattern string, recursive bool) ([]string, error) {
	if dir == "" {
		dir = "."
	}
	if recursive {
		return globRecursive(fs, dir, pattern)
	}
	return globNonRecursive(fs, dir, pattern)
}

func fileMatchesPattern(file fs.FileInfo, pattern string) (bool, error) {
	if file.IsDir() {
		return false, nil
	}

	matched, err := filepath.Match(pattern, file.Name())
	if err != nil {
		return false, err
	}

	return matched, nil
}

func globNonRecursive(fs afero.Fs, dir string, pattern string) ([]string, error) {
	var matches []string

	files, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		matched, err := fileMatchesPattern(file, pattern)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, file.Name())
		}
	}

	return matches, err
}

func globRecursive(fs afero.Fs, dir string, pattern string) ([]string, error) {
	var matches []string
	// walk is recursive, it will look in sub-directories
	err := afero.Walk(fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		matched, err := fileMatchesPattern(info, pattern)
		if err != nil {
			return err
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}

// Re-export functions that don't directly interact with the file	system.

var (
	Join = filepath.Join
	Rel  = filepath.Rel
)
