package stack

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "work with stack objects",
	Long: `Choose an action to perform on a stack, e.g.
	nitric stack create
`,
}

// Flags
var force bool

var stackCreateCmd = &cobra.Command{
	Use:   "create [name] [template]",
	Short: "create a new application stack",
	Long:  `Creates a new Nitric application stack from a template.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v", force)
	},
	Args: cobra.MaximumNArgs(2),
}

func init() {
	stackCreateCmd.Flags().BoolVarP(&force, "force", "f", false, "force stack creation, even in non-empty directories.")
	stackCmd.AddCommand(stackCreateCmd)
}

func RootCommand() *cobra.Command {
	return stackCmd
}
