package build

import (
	"github.com/spf13/cobra"

	"github.com/nitrictech/newcli/pkg/build"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Work with a build",
	Long: `Build a project, e.g.
	nitric build create
`,
}

var buildCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create a new application build",
	Long:  `Creates a new Nitric application build.`,
	Run: func(cmd *cobra.Command, args []string) {
		t := target.FromOptions()
		s, err := stack.FromOptions()
		cobra.CheckErr(err)
		cobra.CheckErr(build.BuildCreate(s, t))
	},
	Args: cobra.MaximumNArgs(2),
}

var buildListCmd = &cobra.Command{
	Use:   "list [name] [template]",
	Short: "list builds done for this stack",
	Long:  `Lists Nitric application builds done for this stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		//s, err := stack.FromOptions()
		//cobra.CheckErr(err)
		//s.BuildList()
	},
	Args: cobra.MaximumNArgs(2),
}

func RootCommand() *cobra.Command {
	buildCmd.AddCommand(buildCreateCmd)
	target.AddOptions(buildCreateCmd, true)
	stack.AddOptions(buildCreateCmd)
	buildCmd.AddCommand(buildListCmd)
	return buildCmd
}
