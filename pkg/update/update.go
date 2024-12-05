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

package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/paths"
	"github.com/nitrictech/cli/pkg/version"
	"github.com/nitrictech/cli/pkg/view/tui"
)

func FetchLatestCLIVersion() string {
	return fetchLatestVersion("cli", cacheCLIPath())
}

func FetchLatestProviderVersion() string {
	return fetchLatestVersion("nitric", cacheProviderPath())
}

func PrintOutdatedCLIWarning() {
	currentVersion := strings.TrimPrefix(version.Version, "v")
	latestVersion := FetchLatestCLIVersion()

	// don't generate warning for non-production versions
	if strings.Contains(currentVersion, "-") {
		return
	}

	updateAvailable := isOutdated(currentVersion, latestVersion)

	if updateAvailable {
		msg := fmt.Sprintf(`A new version of Nitric is available. To upgrade from version '%s' to '%s'`, currentVersion, latestVersion)
		msg += ", visit https://nitric.io/docs/installation for instructions and release notes."

		tui.Warning.Println(msg)
	}
}

func PrintOutdatedProviderWarning(providerName string) {
	latestVersion := FetchLatestProviderVersion()

	var currentVersion string

	providerParts := strings.SplitN(providerName, "@", 2)
	if len(providerParts) == 2 {
		providerName = providerParts[0]
		currentVersion = providerParts[1]
	} else {
		// the format of the provider name is checked elsewhere, so no need to print an error here
		return
	}

	// don't generate warning for non nitric provider versions
	if !strings.HasPrefix(providerName, "nitric/") {
		return
	}

	// don't generate warning for 0.0.1 versions (local builds)
	if currentVersion == "0.0.1" {
		return
	}

	updateAvailable := isOutdated(currentVersion, latestVersion)

	if updateAvailable {
		tui.Info.Println(fmt.Sprintf(`Update available for %s: '%s' → '%s'.`, providerName, currentVersion, latestVersion))
		tui.Info.Println(fmt.Sprintf("Visit https://github.com/nitrictech/nitric/compare/v%s...v%s for full changelog.", currentVersion, latestVersion))

		fmt.Println("")
	}
}

func fetchLatestVersion(repo string, cachePath string) string {
	latestVersionContents, err := os.ReadFile(cachePath)
	latestVersion := string(latestVersionContents)
	// if file does not exist or cache is expired, fetch and save latest version
	if err != nil || cacheExpired(cachePath) {
		owner := "nitrictech"
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

		response, err := http.Get(apiURL)
		if err != nil {
			// if there is an error due to being offline, timeout or rate limit. Skip check.
			return ""
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			cobra.CheckErr(err)
		}

		var releaseInfo struct {
			TagName string `json:"tag_name"`
		}

		err = json.Unmarshal(body, &releaseInfo)
		if err != nil {
			cobra.CheckErr(err)
		}

		latestVersion := strings.TrimPrefix(releaseInfo.TagName, "v")

		err = updateFile(latestVersion, cachePath)
		if err != nil {
			cobra.CheckErr(err)
		}
	}

	return latestVersion
}

func cacheExpired(cachePath string) bool {
	catchFileInfo, err := os.Stat(cachePath)
	if err != nil {
		return true
	}

	return time.Now().After(catchFileInfo.ModTime().Add(24 * time.Hour))
}

func cacheCLIPath() string {
	return filepath.Join(paths.NitricHomeDir(), ".cached-last-version-check")
}

func cacheProviderPath() string {
	return filepath.Join(paths.NitricHomeDir(), ".cached-last-provider-version-check")
}

func updateFile(latestVersion string, cachePath string) error {
	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create/update %s file", cachePath)
	}
	defer file.Close()

	_, err = file.WriteString(latestVersion)
	if err != nil {
		return err
	}

	return nil
}

func isOutdated(currentVersion string, latestVersion string) bool {
	// if current version is latest, no need to update
	if currentVersion == "latest" {
		return false
	}

	latestVer, err := semver.NewVersion(latestVersion)
	if err != nil {
		return false
	}

	currentVer, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false
	}

	return currentVer.LessThan(latestVer)
}
