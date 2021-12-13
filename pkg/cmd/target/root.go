package target

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/newcli/pkg/output"
	"github.com/nitrictech/newcli/pkg/target"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "work with target objects",
	Long: `Choose an action to perform on a target, e.g.
	nitric target list
`,
}

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured targets",
	Long:  `Lists configured taregts.`,
	Run: func(cmd *cobra.Command, args []string) {
		targets := map[string]target.Target{}
		cobra.CheckErr(mapstructure.Decode(viper.GetStringMap("targets"), &targets))
		output.Print(targets)
	},
	Args: cobra.MaximumNArgs(2),
}

func RootCommand() *cobra.Command {
	targetCmd.AddCommand(targetListCmd)
	return targetCmd
}
