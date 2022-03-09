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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/cmd/run"
	cmdstack "github.com/nitrictech/cli/pkg/cmd/stack"
	"github.com/nitrictech/cli/pkg/ghissue"
	"github.com/nitrictech/cli/pkg/output"
)

const usageTemplate = `Nitric - The fastest way to build serverless apps

To start with nitric, run the 'nitric new' command:

    $ nitric new

This will guide you through project creation, including selecting from available templates.
%s
For further details visit our docs https://nitric.io/docs`

func usageString() string {
	return fmt.Sprintf(usageTemplate, strings.Join(CommonCommandsUsage(), "\n"))
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nitric",
	Short: "CLI for Nitric applications",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if output.VerboseLevel > 1 {
			pterm.EnableDebugMessages()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer func() {
		if err := recover(); err != nil {
			pterm.Error.Println("An unexpected error occurred, please create a github issue by clicking on the link below")
			fmt.Println(ghissue.BugLink(err))
		}
	}()

	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&output.VerboseLevel, "verbose", "v", 1, "set the verbosity of output (larger is more verbose)")
	rootCmd.PersistentFlags().VarP(output.OutputTypeFlag, "output", "o", "output format")
	err := rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return output.OutputTypeFlag.Allowed, cobra.ShellCompDirectiveDefault
	})
	cobra.CheckErr(err)

	newProjectCmd.Flags().BoolVarP(&force, "force", "f", false, "force project creation, even in non-empty directories.")
	rootCmd.AddCommand(newProjectCmd)
	rootCmd.AddCommand(cmdstack.RootCommand())
	rootCmd.AddCommand(run.RootCommand())
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(feedbackCmd)
	addAlias("stack update", "up", true)
	addAlias("stack down", "down", true)
	addAlias("stack list", "list", false)
	rootCmd.Long = usageString()
}

func addAlias(from, to string, commonCommand bool) {
	cmd, _, err := rootCmd.Find(strings.Split(from, " "))
	cobra.CheckErr(err)

	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["alias:to"] = to
	alias := &cobra.Command{
		Annotations: map[string]string{"alias:from": from},
		Use:         to,
		Short:       cmd.Short,
		Long:        cmd.Long,
		Example:     cmd.Example,
		Run: func(cmd *cobra.Command, args []string) {
			newArgs := []string{os.Args[0]}
			newArgs = append(newArgs, strings.Split(from, " ")...)
			newArgs = append(newArgs, args...)
			os.Args = newArgs
			cobra.CheckErr(rootCmd.Execute())
		},
		DisableFlagParsing: true, // the real command will parse the flags
	}
	if commonCommand {
		alias.Annotations["commonCommand"] = "yes"
	}
	rootCmd.AddCommand(alias)
}

func CommonCommandsUsage() []string {
	cmdH := []string{
		"",
		"Common commands in the CLI that you’ll be using:",
		""}
	cmdH = append(cmdH, cmdUsage([]string{}, rootCmd, true)...)
	return append(cmdH, "")
}

func AllCommandsUsage() []string {
	cmdH := []string{
		"",
		"Documentation for all available commands:",
		""}
	cmdH = append(cmdH, cmdUsage([]string{}, rootCmd, false)...)
	return append(cmdH, "")
}

// cmdUsage returns the command usage for commonOnly commands or all.
// if all commands, then the aliases are group with the full command.
func cmdUsage(prefix []string, c *cobra.Command, commonOnly bool) []string {
	cmdH := []string{}
	args := append(prefix, c.Use)
	use := strings.Join(args, " ")

	add := true
	if _, ok := c.Annotations["commonCommand"]; commonOnly && !ok {
		add = false
	}
	if _, ok := c.Annotations["alias:from"]; !commonOnly && ok {
		add = false
	}
	if !c.HasParent() {
		add = false
	}

	if add {
		cmdH = append(cmdH, fmt.Sprintf("- %-22s : %s", use, c.Short))
		if _, ok := c.Annotations["alias:to"]; ok {
			use = "nitric " + c.Annotations["alias:to"]
			cmdH = append(cmdH, fmt.Sprintf("  (alias: %s)", use))
		}
	}

	for _, sc := range c.Commands() {
		cmdH = append(cmdH, cmdUsage(append(prefix, c.Use), sc, commonOnly)...)
	}
	return cmdH
}
