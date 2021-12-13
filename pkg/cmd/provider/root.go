package provider

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Work with a provider",
	Long: `List availabe providers, e.g.
	nitric provider list
`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "list providers",
	Long:  `Lists Nitric providers.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

func RootCommand() *cobra.Command {
	providerCmd.AddCommand(providerListCmd)
	return providerCmd
}
