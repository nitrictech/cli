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
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/commands/build"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a Nitric project",
	Long:  `Build all services in a nitric project as docker container images`,
	Run: func(cmd *cobra.Command, args []string) {
		// info.Run(cmd.Context())
		fs := afero.NewOsFs()

		proj, err := project.FromFile(fs, "")
		tui.CheckErr(err)

		updates, err := proj.BuildServices(fs)
		tui.CheckErr(err)

		prog := teax.NewProgram(build.NewModel(updates, "Building Services"))
		// blocks but quits once the above updates channel is closed by the build process
		_, err = prog.Run()
		tui.CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(tui.AddDependencyCheck(buildCmd, tui.Docker, tui.DockerBuildx))
}
