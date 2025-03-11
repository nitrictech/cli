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

package add_website

import (
	"fmt"
	"strings"
)

type Framework struct {
	Name         string
	Value        string
	Description  string
	BuildCommand string
	DevCommand   string
	DevURL       string
	OutputDir    string

	createCommand    string
	npmCreateCommand string

	// link to install the dependency if it doesn't exist
	InstallLink string
}

func (f Framework) GetItemValue() string {
	return f.Name
}

func (f Framework) GetItemDescription() string {
	return ""
}

// get dev command with package manager
func (f Framework) GetDevCommand(packageManager string, path string) string {
	return getCommand(f.DevCommand, packageManager, path)
}

// get build command with package manager
func (f Framework) GetBuildCommand(packageManager string, path string) string {
	return getCommand(f.BuildCommand, packageManager, path)
}

func getCommand(command string, packageManager string, path string) string {
	// if command has no string interpolation, return it as is
	if !strings.Contains(command, "%s") {
		return command
	}

	// baseUrl := ""
	// if path != "" {
	// 	baseUrl = fmt.Sprintf(" --base-url %s", path)
	// }

	if packageManager == "npm" {
		return fmt.Sprintf(command, "npm run")
	}

	return fmt.Sprintf(command, packageManager)
}

// get create command with package manager
func (f Framework) GetCreateCommand(packageManager string, path string) string {
	if packageManager == "npm" {
		return fmt.Sprintf(f.npmCreateCommand, path)
	}

	return fmt.Sprintf(f.createCommand, packageManager, path)
}

var frameworks = []Framework{
	{
		Name:             "Astro",
		Value:            "astro",
		BuildCommand:     "%s build",
		DevCommand:       "%s dev --port 3000",
		DevURL:           "http://localhost:3000",
		createCommand:    "%s creates astro %s --template minimal --no",
		npmCreateCommand: "npm create astro@latest %s -- --template minimal --no",
		OutputDir:        "dist",
	},
	{
		Name:             "React (Vite)",
		Value:            "react",
		BuildCommand:     "%s build",
		DevCommand:       "%s dev --port 3000",
		DevURL:           "http://localhost:3000",
		OutputDir:        "dist",
		createCommand:    "%s create vite %s --template react-ts",
		npmCreateCommand: "npm create vite@latest %s -- --template react-ts",
	},
	{
		Name:             "Vue (Vite)",
		Value:            "vue",
		BuildCommand:     "%s build",
		DevCommand:       "%s dev --port 3000",
		DevURL:           "http://localhost:3000",
		OutputDir:        "dist",
		createCommand:    "%s create vite %s --template vue-ts",
		npmCreateCommand: "npm create vite@latest %s -- --template vue-ts",
	},
	{
		Name:             "Svelte (Vite)",
		Value:            "svelte",
		BuildCommand:     "%s build",
		DevCommand:       "%s dev --port 3000",
		DevURL:           "http://localhost:3000",
		OutputDir:        "dist",
		createCommand:    "%s create vite %s --template svelte-ts",
		npmCreateCommand: "npm create vite@latest %s -- --template svelte-ts",
	},
	{
		Name:          "Hugo",
		Value:         "hugo",
		BuildCommand:  "hugo",
		DevCommand:    "hugo server --port 3000",
		DevURL:        "http://localhost:3000",
		OutputDir:     "public",
		createCommand: "%s new site %s",
		InstallLink:   "https://gohugo.io/installation/",
	},
}
