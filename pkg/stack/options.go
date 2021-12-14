package stack

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	stackPath string
)

func wrapStatError(err error) error {
	if os.IsNotExist(err) {
		return errors.WithMessage(err, "Please provide the correct path to the stack (eg. -s ./nitric.yaml)")
	}
	if os.IsPermission(err) {
		return errors.WithMessagef(err, "Please make sure that %s has the correct permissions", stackPath)
	}
	return err
}

func FromOptions() (*Stack, error) {
	ss, err := os.Stat(stackPath)
	if err != nil {
		return nil, wrapStatError(err)
	}
	if ss.IsDir() {
		stackPath = path.Join(stackPath, "nitric.yaml")
	}
	_, err = os.Stat(stackPath)
	if err != nil {
		return nil, wrapStatError(err)
	}

	return FromFile(stackPath)
}

func AddOptions(cmd *cobra.Command) {
	wd, err := os.Getwd()
	cobra.CheckErr(err)
	cmd.Flags().StringVarP(&stackPath, "stack", "s", wd, "path to the nitric.yaml stack")
}
