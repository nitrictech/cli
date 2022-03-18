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
	"runtime"

	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/utils"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of this CLI",
	Long:  `All software has versions. This is Nitric's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Git commit: %s\n", utils.Commit)
		fmt.Printf("Build time: %s\n", utils.BuildTime)
		fmt.Printf("Nitric CLI: %s\n", utils.Version)
	},
}
