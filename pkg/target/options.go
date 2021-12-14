package target

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	cmd.Flags().StringVarP(&target, "target", "t", "local", "use this to refer to a target in the configuration")
	cmd.Flags().StringVarP(&target, "provider", "p", "local", "the provider to deploy to")
	if !providerOnly {
		cmd.Flags().StringVarP(&target, "name", "n", "", "The name of the deployment")
		cmd.Flags().StringVarP(&target, "region", "r", "", "the region to deploy to")
	}
}
