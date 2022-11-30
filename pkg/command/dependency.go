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

package command

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

type Dependency struct {
	// The display name of the Prerequisite
	name string

	// The command to run for the prequisite
	command string

	// The function to run to help out
	assist func() error
}

var Pulumi = &Dependency{
	name:    "Pulumi",
	command: "pulumi",
	assist: func() error {
		var resp bool
		_ = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Pulumi is required by %s but is not installed, would you like to install it?", "command"),
			Default: false,
		}, &resp)

		if !resp {
			return fmt.Errorf("pulumi is required to run %s. For installation instructions see: %s", "command", "<insert website>")
		}

		var installErr error
		platform := runtime.GOOS
		switch platform {
		case "darwin":
			cmd := exec.Command("sh", "-c", "curl -fsSL https://get.pulumi.com | sh")
			cmd.Stdout = os.Stdout
			installErr = cmd.Run()
		case "linux":
			cmd := exec.Command("sh", "-c", "curl -fsSL https://get.pulumi.com | sh")
			cmd.Stdout = os.Stdout
			installErr = cmd.Run()
		case "windows":
			cmd := exec.Command("powershell", "-NoProfile", "-InputFormat", "None", "-ExecutionPolicy", "Bypass", "-Command", "[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; iex ((New-Object System.Net.WebClient).DownloadString('https://get.pulumi.com/install.ps1'))")
			cmd.Stdout = os.Stdout
			installErr = cmd.Run()
			if installErr == nil {
				cmd := exec.Command("SET", "PATH=%PATH%;%USERPROFILE%\\.pulumi\\bin")
				cmd.Stdout = os.Stdout
				installErr = cmd.Run()
			}
		default:
			installErr = fmt.Errorf("platform %s not supported", platform)
		}

		return installErr
	},
}

var Docker = &Dependency{
	name:    "Docker",
	command: "docker",
	assist: func() error {
		return fmt.Errorf("docker is required to run this command. For installation instructions see: https://docs.docker.com/engine/install/")
	},
}

// AddDependencyCheck - Wraps a cobra command with a pre-run that
// will check for dependencies
func AddDependencyCheck(cmd *cobra.Command, deps ...*Dependency) *cobra.Command {
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		err := checkDependencies(deps...)
		cobra.CheckErr(err)
	}

	return cmd
}

func checkDependencies(deps ...*Dependency) error {
	if len(deps) == 0 {
		return nil
	}

	missing := make([]*Dependency, 0)

	for _, p := range deps {
		// check if the command exists on path
		if _, err := exec.LookPath(p.command); err != nil {
			// We don't have the dependency add it to our missing dependency
			missing = append(missing, p)
		}
	}

	if len(missing) > 0 {
		// need to do some prompts for install
		for _, p := range missing {
			// TODO: We may want to do dependency install prompts in batches
			// rather than one at a time
			err := p.assist()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
