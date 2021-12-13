package target

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/newcli/pkg/pflagext"
)

var (
	target   string
	name     string
	provider string
	region   string
)

func FromOptions() *Target {
	t := Target{}
	if target == "" {
		t.Name = "local"
		t.Provider = "local"
	} else {
		targets := map[string]Target{}
		cobra.CheckErr(mapstructure.Decode(viper.GetStringMap("targets"), &targets))
		t = targets[target]
	}
	if name != "" {
		t.Name = name
	}
	if provider != "" {
		t.Provider = provider
	}
	if region != "" {
		t.Region = region
	}
	return &t
}

func AddOptions(cmd *cobra.Command, providerOnly bool) {
	targetsMap := viper.GetStringMap("targets")
	targets := []string{}
	for k := range targetsMap {
		targets = append(targets, k)
	}

	cmd.Flags().VarP(pflagext.NewStringEnumVar(&target, targets, "local"), "target", "t", "use this to refer to a target in the configuration")
	cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return targets, cobra.ShellCompDirectiveDefault
	})

	providers := []string{"local", "aws", "azure", "gcp", "digitalocean"}
	cmd.Flags().VarP(pflagext.NewStringEnumVar(&provider, providers, "local"), "provider", "p", "the provider to deploy to")
	cmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return providers, cobra.ShellCompDirectiveDefault
	})

	if !providerOnly {
		cmd.Flags().StringVarP(&target, "name", "n", "", "The name of the deployment")
		cmd.Flags().StringVarP(&target, "region", "r", "", "the region to deploy to")
	}
}
