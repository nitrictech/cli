/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/newcli/pkg/cmd/build"
	"github.com/nitrictech/newcli/pkg/cmd/deployment"
	"github.com/nitrictech/newcli/pkg/cmd/provider"
	"github.com/nitrictech/newcli/pkg/cmd/stack"
	"github.com/nitrictech/newcli/pkg/cmd/target"
	"github.com/nitrictech/newcli/pkg/output"
)

const configFileName = ".nitric-config"

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nitric",
	Short: "helper CLI for nitric applications",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

var configHelpTopic = &cobra.Command{
	Use:   "configuration",
	Short: "Configuraton help",
	Long: `nitric CLI can be configured (using yaml format) in the following locations:
${HOME}/.nitric-config.yaml
${HOME}/.config/nitric/.nitric-config.yaml

An example of the format is:
  aliases:
    new: stack create

  targets:
    local:
      provider: local
    test-app:
      region: eastus
      provider: aws
      name: myApp
  `,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/%s.yaml)", configFileName))
	rootCmd.PersistentFlags().VarP(output.OutputTypeFlag, "output", "o", "output format")

	rootCmd.AddCommand(build.RootCommand())
	rootCmd.AddCommand(deployment.RootCommand())
	rootCmd.AddCommand(provider.RootCommand())
	rootCmd.AddCommand(stack.RootCommand())
	rootCmd.AddCommand(target.RootCommand())
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configHelpTopic)

	initConfig()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".nitric" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(path.Join(home, ".config", "nitric"))
		viper.SetConfigType("yaml")
		viper.SetConfigName(".nitric-config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	aliases := map[string]string{}
	cobra.CheckErr(mapstructure.Decode(viper.GetStringMap("aliases"), &aliases))
	for n, aliasString := range aliases {
		alias := &cobra.Command{
			Use:   n,
			Short: "alias for: " + aliasString,
			Long:  "Custom alias command for " + aliasString,
			Run: func(cmd *cobra.Command, args []string) {
				newArgs := []string{os.Args[0]}
				newArgs = append(newArgs, strings.Split(aliasString, " ")...)
				newArgs = append(newArgs, args...)
				os.Args = newArgs
				rootCmd.Execute()
			},
			DisableFlagParsing: true, // the real command will parse the flags
		}
		rootCmd.AddCommand(alias)
	}
}
