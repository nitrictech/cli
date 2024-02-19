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
	"fmt"
	"runtime"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/spf13/afero"
)

// Gets a default provider string and translates it into a file name that can be retrieved from
// our github releases
func providerFileName(prov *Provider) string {
	// Get the OS name
	os := runtime.GOOS
	platform := runtime.GOARCH

	// tarballs are the default archive type
	archive := "tar.gz"
	if os == "windows" {
		// We use zips for windows
		archive = "zip"
	}

	if platform == "amd64" {
		platform = "x86_64"
	}

	// Return the archive uri in the form of
	// {PROVIDER}_{OS}_{PLATFORM}.{ARCHIVE}
	// e.g. gcp_linux_x86_64.tar.gz
	return strings.ToLower(fmt.Sprintf("%s_%s_%s.%s", prov.name, os, platform, archive))
}

func defaultDownloadUri(prov *Provider) string {
	fileName := providerFileName(prov)

	if prov.version == "latest" {
		return fmt.Sprintf("https://github.com/nitrictech/nitric/releases/latest/download/%s", fileName)
	}

	return fmt.Sprintf("https://github.com/nitrictech/nitric/releases/download/v%s/%s", prov.version, fileName)
}

// EnsureProviderExists - Ensures that the provider exists on the local file system
// If it doesn't exist and is a nitric provider, it will attempt to download it from the nitric releases
func EnsureProviderExists(fs afero.Fs, prov *Provider) (string, error) {
	// Check to see if the provider already exists
	provFile := providerFilePath(prov)

	// Check if the provider we're after actually exists already
	_, err := fs.Stat(provFile)

	if err != nil && prov.organization == nitricOrg {
		// If the provider is apart of the nitric org attempt to download it from the core nitric releases
		if prov.organization == nitricOrg {
			if err := getter.GetFile(provFile, defaultDownloadUri(prov)); err != nil {
				return "", fmt.Errorf("error downloading file %s (%w)", defaultDownloadUri(prov), err)
			}
		} else {
			// Not a nitric release so should be installed manually
			// TODO: Make CLI assistant method for getting third-part provider releases
			// nitric provider install custom provider --url "https://github.com/my-org/my-project/releases"
			return "", fmt.Errorf("could not locate provider at %s, please check that it exists and is executable", provFile)
		}
	}

	return provFile, nil
}

// func checkPulumiLoginState() (bool, error) {
// 	cmd := exec.Command("pulumi", "whoami")
// 	_, err := cmd.Output()
// 	// if no pulumi login state detected
// 	if err != nil {
// 		var confirm string

// 		pterm.Warning.Print("No pulumi config detected")
// 		fmt.Println("")
// 		pterm.Info.Printf("For more information on best practices for production deployments, see docs %s", "https://nitric.io/docs/deployment#self-hosting")
// 		fmt.Println("")

// 		err := survey.AskOne(&survey.Select{
// 			Message: "To deploy we require you to be logged in to pulumi. We can automatically configure this to be a local login?",
// 			Default: "No",
// 			Options: []string{"Yes", "No"},
// 		}, &confirm)
// 		if err != nil {
// 			return false, err
// 		}

// 		if confirm != "Yes" {
// 			pterm.Info.Println("Cancelling deployment. You can log into pulumi using `pulumi login --local`.")
// 			os.Exit(0)
// 		}

// 		fmt.Println("Configuring ephemeral pulumi local login. To remove this message in the future and persist stack information use `pulumi login --local`")

// 		return true, nil
// 	}

// 	return false, nil
// }
