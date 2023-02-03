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

package provider

import (
	"regexp"
	"strings"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider/remote"
	"github.com/spf13/cobra"
)

var (
	version     string
	downloadURL string
)

var providerCmd = &cobra.Command{
	Use:     "provider",
	Short:   "Manange providers (install, list, rm)",
	Example: `nitric provider install`,
}

var providerInstallCmd = &cobra.Command{
	Use:   "install [NAME] [flags]",
	Short: "Install a provider",
	Example: `
nitric provider install aws # Install the latest version of the AWS provider
nitric provider install aws --version 0.1.0 # Install a specific version of the AWS provider
nitric provider install allmine --url https://example.com/foo.tar.gz # Install a custom provider from a URL
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := &remote.ProviderInstall{
			Name: args[0],
		}

		if version != "" {
			p.Version = version
		}

		if downloadURL != "" {
			p.URL = downloadURL

			if p.Version == "" && strings.Contains(p.URL, "releases/download/v") {
				verMatch := regexp.MustCompile(`\/releases\/download\/([0-9v.]*)\/*`)
				p.Version = verMatch.FindStringSubmatch(p.URL)[1]
			}
		}

		if p.Version == "" {
			p.Version = "latest"
		}

		return remote.Install(p)
	},
	Args: cobra.ExactArgs(1),
}

var providerRmCmd = &cobra.Command{
	Use:     "rm [NAME]",
	Short:   "remove a previously installed provider",
	Example: `nitric provider rm aws`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return remote.Remove(args[0])
	},
	Args: cobra.ExactArgs(1),
}

var providerListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all providers",
	Example: `nitric provider list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, err := remote.List()
		if err != nil {
			return err
		}

		output.Print(providers)

		return nil
	},
	Args:    cobra.ExactArgs(0),
	Aliases: []string{"ls"},
}

func RootCommand() *cobra.Command {
	providerInstallCmd.Flags().StringVar(&version, "version", "", "--version 1.2.3")
	providerInstallCmd.Flags().StringVar(&downloadURL, "url", "", "--url https://example.com/foo.tar.gz")

	providerCmd.AddCommand(providerInstallCmd)

	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerRmCmd)

	return providerCmd
}
