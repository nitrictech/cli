// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/cli/pkg/cmd/provider"
	"github.com/nitrictech/cli/pkg/cmd/run"
	cmdstack "github.com/nitrictech/cli/pkg/cmd/stack"
	cmdTarget "github.com/nitrictech/cli/pkg/cmd/target"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/target"
	"github.com/nitrictech/cli/pkg/tasklet"
	"github.com/nitrictech/cli/pkg/utils"
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if output.VerboseLevel > 1 {
			pterm.EnableDebugMessages()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

var configHelpTopic = &cobra.Command{
	Use:   "configuration",
	Short: "Configuration help",
	Long: `nitric CLI can be configured (using yaml format) in the following locations:
${HOME}/.nitric-config.yaml
${HOME}/.config/nitric/.nitric-config.yaml

An example of the format is:
  aliases:
    new: stack create

  targets:
    aws:
      region: eastus
      provider: aws
	azure:
	  region: eastus
	  provider: azure
  `,
}

func init() {
	initConfig()

	rootCmd.PersistentFlags().IntVarP(&output.VerboseLevel, "verbose", "v", 1, "set the verbosity of output (larger is more verbose)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/%s.yaml)", configFileName))
	rootCmd.PersistentFlags().VarP(output.OutputTypeFlag, "output", "o", "output format")
	err := rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return output.OutputTypeFlag.Allowed, cobra.ShellCompDirectiveDefault
	})
	cobra.CheckErr(err)

	newProjectCmd.Flags().BoolVarP(&force, "force", "f", false, "force project creation, even in non-empty directories.")
	rootCmd.AddCommand(newProjectCmd)
	rootCmd.AddCommand(cmdstack.RootCommand())
	rootCmd.AddCommand(provider.RootCommand())
	rootCmd.AddCommand(cmdTarget.RootCommand())
	rootCmd.AddCommand(run.RootCommand())
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configHelpTopic)
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
		viper.AddConfigPath(utils.NitricConfigDir())
		viper.SetConfigType("yaml")
		viper.SetConfigName(".nitric-config")
	}

	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	ensureConfigDefaults()
}

func ensureConfigDefaults() {
	needsWrite := false

	to := viper.GetDuration("build_timeout")
	if to == 0 {
		needsWrite = true
		viper.Set("build_timeout", 5*time.Minute)
	}

	if target.EnsureDefaultConfig() {
		needsWrite = true
	}

	if stack.EnsureRuntimeDefaults() {
		needsWrite = true
	}

	if needsWrite {
		tasklet.MustRun(tasklet.Runner{
			StartMsg: "Updating configfile to include defaults",
			Runner: func(_ output.Progress) error {
				// ensure .config/nitric exists
				err := os.MkdirAll(utils.NitricConfigDir(), os.ModePerm)
				if err != nil {
					return err
				}

				return viper.WriteConfigAs(filepath.Join(utils.NitricConfigDir(), ".nitric-config.yaml"))
			},
			StopMsg: "Configfile updated"}, tasklet.Opts{})
	}
}
