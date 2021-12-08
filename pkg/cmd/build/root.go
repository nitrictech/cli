package build

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Work with a build",
	Long: `Build a project, e.g.
	nitric build create
`,
}

var buildCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "create a new application build",
	Long:  `Creates a new Nitric application build.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

var buildListCmd = &cobra.Command{
	Use:   "list [name] [template]",
	Short: "list builds done for this stack",
	Long:  `Lists Nitric application builds done for this stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

func init() {
	buildCmd.AddCommand(buildCreateCmd)
	buildCmd.AddCommand(buildListCmd)
}

func RootCommand() *cobra.Command {
	return buildCmd
}
