package filepath

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// This is an alternative implementation of useful functions from the filepath package, that uses Afero to access the file system.

// FIXME: work with sub-dirs (recursive)
// Glob returns a list of files that match the given pattern.
func Glob(fs afero.Fs, pattern string) ([]string, error) {
	var matches []string
	err := afero.Walk(fs, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if matched {
				matches = append(matches, path)
			}
		}
		return nil
	})
	return matches, err
}

// Re-export functions that don't directly interact with the file	system.

var Join = filepath.Join
var Rel = filepath.Rel
