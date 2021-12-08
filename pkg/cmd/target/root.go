package target

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "work with target objects",
	Long: `Choose an action to perform on a target, e.g.
	nitric target list
`,
}

// Flags
var force bool

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured targets",
	Long:  `Lists configured taregts.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v", force)
	},
	Args: cobra.MaximumNArgs(2),
}

func init() {
	targetCmd.AddCommand(targetListCmd)
}

func RootCommand() *cobra.Command {
	return targetCmd
}
