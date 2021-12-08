package deployment

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Work with a deployment",
	Long: `Delopy a project, e.g.
	nitric deployment create
	nitric deployment delete
	nitric deployment list
`,
}

var deploymentCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "create a new application deployment",
	Long:  `Creates a new Nitric application deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

var deploymentListCmd = &cobra.Command{
	Use:   "list [name]",
	Short: "list deployments done for this stack",
	Long:  `Lists Nitric application deployments done for this stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

func init() {
	deploymentCmd.AddCommand(deploymentCreateCmd)
	deploymentCmd.AddCommand(deploymentListCmd)
}

func RootCommand() *cobra.Command {
	return deploymentCmd
}
