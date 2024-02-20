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
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/nitrictech/cli/pkg/paths"
)

type Provider struct {
	organization string
	name         string
	version      string
}

const semverRegex = `@(latest|(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*)))?`

// Provider format <org>/<provider>@<semver>
const providerIdRegex = `\w+\/\w+` + semverRegex

func providerIdSeparators(r rune) bool {
	const versionSeparator = '@'

	const orgSeparator = '/'

	return r == versionSeparator || r == orgSeparator
}

func providerFromId(providerId string) (*Provider, error) {
	match, err := regexp.MatchString(providerIdRegex, providerId)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred parsing provider ID %s (%w)", providerId, err)
	}

	if !match {
		return nil, fmt.Errorf("invalid provider format %s, valid example is nitric/aws@1.2.3", providerId)
	}

	providerParts := strings.FieldsFunc(providerId, providerIdSeparators)

	return &Provider{
		organization: providerParts[0],
		name:         providerParts[1],
		version:      providerParts[2],
	}, nil
}

const nitricOrg = "nitric"

func providerFilePath(prov *Provider) string {
	provDir := paths.NitricProviderDir()
	os := runtime.GOOS

	if os == "windows" {
		return filepath.Join(provDir, prov.organization, fmt.Sprintf("%s-%s%s", prov.name, prov.version, ".exe"))
	}

	return filepath.Join(provDir, prov.organization, fmt.Sprintf("%s-%s", prov.name, prov.version))
}

// NewProvider - Returns a new provider instance based on the given providerId string
// The providerId string is in the form of <org-name>/<provider-name>@<version>
func NewProvider(providerId string) (*Provider, error) {
	provider, err := providerFromId(providerId)
	if err != nil {
		return nil, err
	}

	if provider.organization == nitricOrg {
		// v0 providers are not supported, still permit the 'development' version 0.0.1
		if strings.HasPrefix(provider.version, "0.") && provider.version != "0.0.1" {
			return nil, fmt.Errorf("nitric providers prior to version 1.0.0 are not supported by this version of the CLI.")
		}
	}

	return provider, nil
}

// func NewDeploymentEngine(provider *Provider) (DeploymentEngine, error) {
// 	baseNitricDeployment := &nitricDeployment{binaryRemoteDeployment: baseBinaryDeployment}

// 	// Format provider file location
// 	providerFilePath := providerFilePath(provider)
// 	if provider.organization == nitricOrg {
// 		// attempt to install
// 		providerFile, err = ensureProviderExists(provider)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return NewProviderExecutable(providerFilePath)
// }
