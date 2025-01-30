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

	"github.com/samber/lo"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nitrictech/cli/pkg/collector"
	"github.com/nitrictech/cli/pkg/env"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/commands/build"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

var (
	debugEnvFile string
	debugFile    string
)

var debugCmd = &cobra.Command{
	Use:     "debug",
	Short:   "Debug Operations (utilities for debugging nitric applications)",
	Long:    `Debug Operations (utilities for debugging nitric applications).`,
	Example: `nitric debug spec`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Root().PersistentPreRun != nil {
			cmd.Root().PersistentPreRun(cmd, args)
		}
	},
}

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Output the nitric application cloud spec.",
	Long:  `Output the nitric application cloud spec.`,
	Run: func(cmd *cobra.Command, args []string) {
		fs := afero.NewOsFs()

		proj, err := project.FromFile(fs, "")
		tui.CheckErr(err)

		// Build the Project's Services (Containers)
		buildUpdates, err := proj.BuildServices(fs, !noBuilder)
		tui.CheckErr(err)

		batchBuildUpdates, err := proj.BuildBatches(fs, !noBuilder)
		tui.CheckErr(err)

		allBuildUpdates := lo.FanIn(10, buildUpdates, batchBuildUpdates)

		if isNonInteractive() {
			fmt.Println("building project services")
			for _, service := range proj.GetServices() {
				fmt.Printf("service matched '%s', auto-naming this service '%s'\n", service.GetFilePath(), service.Name)
			}

			// non-interactive environment
			for update := range buildUpdates {
				for _, line := range strings.Split(strings.TrimSuffix(update.Message, "\n"), "\n") {
					fmt.Printf("%s [%s]: %s\n", update.ServiceName, update.Status, line)
				}
			}
		} else {
			prog := teax.NewProgram(build.NewModel(allBuildUpdates, "Building Services"))
			// blocks but quits once the above updates channel is closed by the build process
			buildModel, err := prog.Run()
			tui.CheckErr(err)
			if buildModel.(build.Model).Err != nil {
				tui.CheckErr(fmt.Errorf("error building services"))
			}
		}

		// Step 2. Start the collectors and containers (respectively in pairs)
		// Step 3. Merge requirements from collectors into a specification
		serviceRequirements, err := proj.CollectServicesRequirements()
		tui.CheckErr(err)

		batchRequirements, err := proj.CollectBatchRequirements()
		tui.CheckErr(err)

		websiteRequirements, err := proj.CollectWebsiteRequirements()
		tui.CheckErr(err)

		additionalEnvFiles := []string{}

		if debugEnvFile != "" {
			additionalEnvFiles = append(additionalEnvFiles, envFile)
		}

		envVariables, err := env.ReadLocalEnv(additionalEnvFiles...)
		if err != nil && os.IsNotExist(err) {
			if !os.IsNotExist(err) {
				tui.CheckErr(err)
			}
			// If it doesn't exist set blank
			envVariables = map[string]string{}
		}

		spec, err := collector.ServiceRequirementsToSpec(proj.Name, envVariables, serviceRequirements, batchRequirements, websiteRequirements)
		tui.CheckErr(err)

		migrationImageContexts, err := collector.GetMigrationImageBuildContexts(serviceRequirements, batchRequirements, fs)
		tui.CheckErr(err)
		// Build images from contexts and provide updates on the builds

		if len(migrationImageContexts) > 0 {
			migrationBuildUpdates, err := project.BuildMigrationImages(fs, migrationImageContexts, !noBuilder)
			tui.CheckErr(err)

			if isNonInteractive() {
				fmt.Println("building project migration images")
				// non-interactive environment
				for update := range migrationBuildUpdates {
					for _, line := range strings.Split(strings.TrimSuffix(update.Message, "\n"), "\n") {
						if update.Status == project.ServiceBuildStatus_Error {
							tui.CheckErr(fmt.Errorf("error building migration images %s", update.Message))
						}

						fmt.Printf("%s [%s]: %s\n", update.ServiceName, update.Status, line)
					}
				}
			} else {
				prog := teax.NewProgram(build.NewModel(migrationBuildUpdates, "Building Database Migrations"))
				// blocks but quits once the above updates channel is closed by the build process
				buildModel, err := prog.Run()
				tui.CheckErr(err)
				if buildModel.(build.Model).Err != nil {
					tui.CheckErr(fmt.Errorf("error building services"))
				}
			}
		}

		outputFile := debugFile
		if outputFile == "" {
			outputFile = "./nitric-spec.json"
		}

		marshaler := protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}

		specJson, err := marshaler.Marshal(spec)
		tui.CheckErr(err)

		// output the spec
		err = os.WriteFile(outputFile, specJson, 0o644)
		tui.CheckErr(err)

		fmt.Printf("Successfully outputted deployment spec to %s\n", outputFile)
	},
	Aliases: []string{"spec"},
}

func init() {
	specCmd.Flags().StringVarP(&debugEnvFile, "env-file", "e", "", "--env-file config/.my-env")
	specCmd.Flags().StringVarP(&debugFile, "output", "o", "", "--file my-example-spec.json")
	specCmd.Flags().BoolVar(&noBuilder, "no-builder", false, "don't create a buildx container")

	// Debug spec
	debugCmd.AddCommand(specCmd)

	// Add Stack Commands
	rootCmd.AddCommand(debugCmd)

	addAlias("debug spec", "spec", true)
}
