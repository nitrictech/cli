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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nitrictech/cli/pkg/command"
	"github.com/nitrictech/cli/pkg/utils"
	"github.com/nitrictech/cli/pkgplus/collector"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/project/stack"
	"github.com/nitrictech/cli/pkgplus/provider"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/build"
	stack_new "github.com/nitrictech/cli/pkgplus/view/tui/commands/stack"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
	"github.com/nitrictech/pearls/pkg/tui/inlinelist"
)

var (
	confirmDown   bool
	forceStack    bool
	forceNewStack bool
	envFile       string
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage stacks (the deployed app containing multiple resources e.g. collection, bucket, topic)",
	Long: `Manage stacks (the deployed app containing multiple resources e.g. collection, bucket, topic).

A stack is a named update target, and a single project may have many of them.`,
	Example: `nitric stack up
nitric stack down
nitric stack list
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Root().PersistentPreRun != nil {
			cmd.Root().PersistentPreRun(cmd, args)
		}

		// Respect existing pulumi configuration if one already exists
		// currPass := os.Getenv("PULUMI_CONFIG_PASSPHRASE")
		// currPassFile := os.Getenv("PULUMI_CONFIG_PASSPHRASE_FILE")
		// if currPass == "" && currPassFile == "" {
		// 	p, err := preferences.GetLocalPassPhraseFile()
		// 	// In non-CI environments we can generate the file to save a step.
		// 	// in CI environments this file would typically be lost, so it shouldn't auto-generate
		// 	if err != nil && !output.CI {
		// 		p, err = preferences.GenerateLocalPassPhraseFile()
		// 	}
		// 	if err != nil {
		// 		err = fmt.Errorf("unable to determine configured passphrase. See https://nitric.io/docs/guides/github-actions#configuring-environment-variables")
		// 	}
		// 	utils.CheckErr(err)

		// 	// Set the default
		// 	os.Setenv("PULUMI_CONFIG_PASSPHRASE_FILE", p)
		// }
	},
}

var newStackCmd = &cobra.Command{
	Use:   "new [stackName] [providerName]",
	Short: "Create a new Nitric stack",
	Long:  `Creates a new Nitric stack.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.IsTerminal() {
			return fmt.Errorf("the stack new command does not support non-interactive environments")
		}

		stackName := ""
		if len(args) >= 1 {
			stackName = args[0]
		}

		providerName := ""
		if len(args) >= 2 {
			providerName = args[1]
		}
		_, err := tea.NewProgram(stack_new.New(afero.NewOsFs(), stack_new.Args{
			StackName:    stackName,
			ProviderName: providerName,
			Force:        forceNewStack,
		})).Run()

		return err
	},
	Args:        cobra.MaximumNArgs(2),
	Annotations: map[string]string{"commonCommand": "yes"},
}

type SimpleStringListItem string

var _ inlinelist.ListItem = SimpleStringListItem("")

func (s SimpleStringListItem) GetItemDescription() string {
	return string(s)
}

func (s SimpleStringListItem) GetItemValue() string {
	return string(s)
}

var stackUpdateCmd = &cobra.Command{
	Use:     "update [-s stack]",
	Short:   "Create or update a deployed stack",
	Long:    `Create or update a deployed stack`,
	Example: `nitric stack update -s aws`,
	Run: func(cmd *cobra.Command, args []string) {
		// s, err := stack.ConfigFromOptions()

		fs := afero.NewOsFs()

		stackFiles, err := stack.GetAllStackFiles(fs)
		tui.CheckErr(err)

		if len(stackFiles) == 0 {
			// no stack files found
			// print error with suggestion for user to run stack new
			tui.CheckErr(fmt.Errorf("no stacks found in project, to create a new one run `nitric stack new`"))
		}

		// Step 0. Get the stack file, or proomptyboi if more than 1.
		stackSelection := stack.GetStackNameFromFileName(stackFiles[0])
		// if len(stackFiles) > 1 {
		// 	stackNames := make([]inlinelist.ListItem, len(stackFiles))
		// 	for i, stackFile := range stackFiles {
		// 		stackNames[i] = SimpleStringListItem(stack.GetStackNameFromFileName(stackFile))
		// 	}
		// 	// ask which one
		// 	// FIXME: currently a keyboard trap and not the UI we want
		// 	promptModel := listprompt.New(listprompt.Args{
		// 		Items:  stackNames,
		// 		Tag:    "stack",
		// 		Prompt: "Which stack would you like to update?",
		// 	})

		// 	selection, err := tea.NewProgram(promptModel).Run()
		// 	cobra.CheckErr(err)
		// 	stackSelection = selection.(listprompt.Model).Choice()
		// }

		stackConfig, err := stack.ConfigFromName[map[string]any](fs, stackSelection)
		tui.CheckErr(err)

		proj, err := project.FromFile(fs, "")
		tui.CheckErr(err)

		// make provider from the provider name
		// providerName := stackConfig.Provider

		// Step 0a. Locate/Download provider where applicable.
		prov, err := provider.NewProvider(stackConfig.Provider)
		tui.CheckErr(err)

		providerFilePath, err := provider.EnsureProviderExists(fs, prov)
		tui.CheckErr(err)

		// Build the Project's Services (Containers)
		buildUpdates, err := proj.BuildServices(fs)
		tui.CheckErr(err)

		prog := tea.NewProgram(build.NewModel(buildUpdates))
		// blocks but quits once the above updates channel is closed by the build process
		_, err = prog.Run()
		tui.CheckErr(err)

		// Step 2. Start the collectors and containers (respectively in pairs)
		// Step 3. Merge requirements from collectors into a specification
		serviceRequirements, err := proj.CollectServicesRequirements()
		tui.CheckErr(err)

		spec, err := collector.ServiceRequirementsToSpec(proj.Name, map[string]string{}, serviceRequirements)
		tui.CheckErr(err)

		// Step 4. Start the deployment provider server
		providerProcess, err := provider.StartProviderExecutable(fs, providerFilePath)
		tui.CheckErr(err)
		defer providerProcess.Stop()

		// Step 5a. Send specification to provider for deployment
		deploymentClient := provider.NewDeploymentClient(providerProcess.Address, true)

		attributes := map[string]interface{}{}

		attributes["stack"] = stackConfig.Name
		attributes["project"] = proj.Name

		for k, v := range stackConfig.Config {
			attributes[k] = v
		}

		attributesStruct, err := structpb.NewStruct(attributes)
		tui.CheckErr(err)

		eventChannel, errorChan := deploymentClient.Up(&deploymentspb.DeployUpRequest{
			Spec:        spec,
			Attributes:  attributesStruct,
			Interactive: true,
		})

		// Step 5b. Communicate with server to share progress of ...
	ServerCommunication:
		for {
			select {
			case event, ok := <-eventChannel:
				if !ok {
					break ServerCommunication
				}

				switch event.Content.(type) {
				case *deploymentspb.DeployUpEvent_Message:
					fmt.Print(event.GetMessage().GetMessage())
				case *deploymentspb.DeployUpEvent_Result:
					fmt.Print(event.GetResult().Result.GetStringResult())
				}

			case err, ok := <-errorChan:
				if !ok {
					break ServerCommunication
				}
				tui.CheckErr(err)
			}
		}
	},
	Args:    cobra.MinimumNArgs(0),
	Aliases: []string{"up"},
}

var stackDeleteCmd = &cobra.Command{
	Use:   "down [-s stack]",
	Short: "Undeploy a previously deployed stack, deleting resources",
	Long:  `Undeploy a previously deployed stack, deleting resources`,
	Example: `nitric stack down -s aws

# To not be prompted, use -y
nitric stack down -s aws -y`,
	Run: func(cmd *cobra.Command, args []string) {
		fs := afero.NewOsFs()

		stackFiles, err := stack.GetAllStackFiles(fs)
		cobra.CheckErr(err)

		if len(stackFiles) == 0 {
			// no stack files found
			// print error with suggestion for user to run stack new
			cobra.CheckErr(fmt.Errorf("no stacks found in project, to create a new one run `nitric stack new`"))
		}

		// Step 0. Get the stack file, or proomptyboi if more than 1.
		stackSelection := stack.GetStackNameFromFileName(stackFiles[0])
		// if len(stackFiles) > 1 {
		// 	stackNames := make([]inlinelist.ListItem, len(stackFiles))
		// 	for i, stackFile := range stackFiles {
		// 		stackNames[i] = SimpleStringListItem(stack.GetStackNameFromFileName(stackFile))
		// 	}
		// 	// ask which one
		// 	// FIXME: currently a keyboard trap and not the UI we want
		// 	promptModel := listprompt.New(listprompt.Args{
		// 		Items:  stackNames,
		// 		Tag:    "stack",
		// 		Prompt: "Which stack would you like to update?",
		// 	})

		// 	selection, err := tea.NewProgram(promptModel).Run()
		// 	cobra.CheckErr(err)
		// 	stackSelection = selection.(listprompt.Model).Choice()
		// }

		stackConfig, err := stack.ConfigFromName[map[string]any](fs, stackSelection)
		cobra.CheckErr(err)

		proj, err := project.FromFile(fs, "")
		cobra.CheckErr(err)

		// make provider from the provider name
		// providerName := stackConfig.Provider

		// Step 0a. Locate/Download provider where applicable.
		prov, err := provider.NewProvider(stackConfig.Provider)
		cobra.CheckErr(err)

		providerFilePath, err := provider.EnsureProviderExists(fs, prov)
		cobra.CheckErr(err)

		// Step 4. Start the deployment provider server
		providerProcess, err := provider.StartProviderExecutable(fs, providerFilePath)
		cobra.CheckErr(err)
		defer providerProcess.Stop()

		// Step 5a. Send specification to provider for deployment
		deploymentClient := provider.NewDeploymentClient(providerProcess.Address, true)

		attributes := map[string]interface{}{}

		attributes["stack"] = stackConfig.Name
		attributes["project"] = proj.Name

		for k, v := range stackConfig.Config {
			attributes[k] = v
		}

		attributesStruct, err := structpb.NewStruct(attributes)
		cobra.CheckErr(err)

		eventChannel, errorChan := deploymentClient.Down(&deploymentspb.DeployDownRequest{
			Attributes:  attributesStruct,
			Interactive: true,
		})

		// Step 5b. Communicate with server to share progress of ...
	ServerCommunication:
		for {
			select {
			case event, ok := <-eventChannel:
				if !ok {
					break ServerCommunication
				}

				switch event.Content.(type) {
				case *deploymentspb.DeployDownEvent_Message:
					fmt.Print(event.GetMessage().GetMessage())
				case *deploymentspb.DeployDownEvent_Result:
					// fmt.Print(event.GetResult())
				}

			case err, ok := <-errorChan:
				if !ok {
					break ServerCommunication
				}
				cobra.CheckErr(err)
			}
		}
	},
	Args: cobra.ExactArgs(0),
}

func init() {
	stackCmd.AddCommand(newStackCmd)
	newStackCmd.Flags().BoolVarP(&forceNewStack, "force", "f", false, "force stack creation.")

	stackCmd.AddCommand(command.AddDependencyCheck(stackUpdateCmd, command.Pulumi, command.Docker))
	stackUpdateCmd.Flags().StringVarP(&envFile, "env-file", "e", "", "--env-file config/.my-env")
	stackUpdateCmd.Flags().BoolVarP(&forceStack, "force", "f", false, "force override previous deployment")
	// utils.CheckErr(stack.AddOptions(stackUpdateCmd, false))

	stackCmd.AddCommand(command.AddDependencyCheck(stackDeleteCmd, command.Pulumi))
	stackDeleteCmd.Flags().BoolVarP(&confirmDown, "yes", "y", false, "confirm the destruction of the stack")
	// utils.CheckErr(stack.AddOptions(stackDeleteCmd, false))

	rootCmd.AddCommand(stackCmd)

	addAlias("stack update", "up", true)
	addAlias("stack down", "down", true)
}
