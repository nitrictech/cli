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

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/paths"
	"github.com/nitrictech/cli/pkg/version"
)

func FetchLatestVersion() string {
	latestVersionContents, err := os.ReadFile(cachePath())
	latestVersion := string(latestVersionContents)
	// if file does not exist or cache is expired, fetch and save latest version
	if err != nil || cacheExpired() {
		owner := "nitrictech"
		repo := "cli"
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

		err = updateFile(latestVersion)
		if err != nil {
			cobra.CheckErr(err)
		}
	}

	return latestVersion
}

func PrintOutdatedWarning() {
	currentVersion := strings.TrimPrefix(version.Version, "v")
	latestVersion := FetchLatestVersion()

	// don't generate warning for non-production versions
	if strings.Contains(currentVersion, "-") {
		return
	}

	if currentVersion < latestVersion {
		msg := fmt.Sprintf(`A new version of Nitric is available. To upgrade from version '%s' to '%s'`, currentVersion, latestVersion)
		msg += ", visit https://nitric.io/docs/installation for instructions and release notes.\n"

		pterm.Warning.Printf(msg)
	}
}

func cacheExpired() bool {
	catchFileInfo, err := os.Stat(cachePath())
	if err != nil {
		return true
	}

	return time.Now().After(catchFileInfo.ModTime().Add(24 * time.Hour))
}

func cachePath() string {
	return filepath.Join(paths.NitricHomeDir(), ".cached-last-version-check")
}

func updateFile(latestVersion string) error {
	file, err := os.Create(cachePath())
	if err != nil {
		return fmt.Errorf("failed to create/update .cached-last-version-check file")
	}
	defer file.Close()

	_, err = file.WriteString(latestVersion)
	if err != nil {
		return err
	}

	return nil
}
