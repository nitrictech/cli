package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/operations/start"
)

var noBrowser bool

var startCmd = &cobra.Command{
	Use:         "start",
	Short:       "Run nitric services locally for development and testing",
	Long:        `Run nitric services locally for development and testing`,
	Example:     `nitric start`,
	Annotations: map[string]string{"commonCommand": "yes"},
	Run: func(cmd *cobra.Command, args []string) {
		start.Run(context.TODO(), noBrowser)
	},
	Args: cobra.ExactArgs(0),
}

func init() {
	startCmd.PersistentFlags().BoolVar(
		&noBrowser,
		"no-browser",
		false,
		"disable browser opening for local dashboard, note: in CI mode the browser opening feature is disabled",
	)

	rootCmd.AddCommand(startCmd)
}
