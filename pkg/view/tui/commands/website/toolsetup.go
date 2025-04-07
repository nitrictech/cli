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
	"strings"
)

type Tool struct {
	Name                     string
	Value                    string
	Description              string
	buildCommand             CommandTemplate
	buildCommandSubSite      CommandTemplate
	devCommand               CommandTemplate
	devCommandSubSite        CommandTemplate
	devURL                   CommandTemplate
	OutputDir                string
	createCommand            CommandTemplate
	npmCreateCommand         CommandTemplate
	InstallLink              string
	SkipPackageManagerPrompt bool
}

func (f Tool) GetItemValue() string {
	return f.Name
}

func (f Tool) GetItemDescription() string {
	return f.Description
}

type CommandVars struct {
	PackageManager string
	Path           string
	Port           string
	BaseURL        string
}

func (f Tool) GetDevCommand(packageManager string, path string, port string) string {
	// if packageManager is npm, we need to add run to the command
	if packageManager == "npm" {
		packageManager = "npm run"
	}

	vars := CommandVars{
		PackageManager: packageManager,
		Path:           path,
		Port:           port,
		BaseURL:        path,
	}

	if path != "" && f.devCommandSubSite != "" {
		return f.devCommandSubSite.Format(vars)
	}

	return f.devCommand.Format(vars)
}

func (f Tool) GetBuildCommand(packageManager string, path string) string {
	// if packageManager is npm, we need to add run to the command
	if packageManager == "npm" {
		packageManager = "npm run"
	}

	vars := CommandVars{
		PackageManager: packageManager,
		Path:           path,
		BaseURL:        path,
	}

	if path != "" && f.buildCommandSubSite != "" {
		return f.buildCommandSubSite.Format(vars)
	}

	return f.buildCommand.Format(vars)
}

func (f Tool) GetCreateCommand(packageManager string, path string) string {
	vars := CommandVars{
		PackageManager: packageManager,
		Path:           path,
	}

	if packageManager == "npm" {
		return f.npmCreateCommand.Format(vars)
	}

	return f.createCommand.Format(vars)
}

func (f Tool) GetDevURL(port string, path string) string {
	vars := CommandVars{
		Port: port,
	}

	url := f.devURL.Format(vars)

	// append the subsite path if it exists
	if path != "" && path != "/" {
		url = url + path
	}

	return url
}

type CommandTemplate string

func (t CommandTemplate) Format(vars CommandVars) string {
	cmd := string(t)
	cmd = strings.ReplaceAll(cmd, "{packageManager}", vars.PackageManager)
	cmd = strings.ReplaceAll(cmd, "{path}", vars.Path)
	cmd = strings.ReplaceAll(cmd, "{port}", vars.Port)
	cmd = strings.ReplaceAll(cmd, "{baseURL}", vars.BaseURL)

	// if the package manager is npm and using a run command,
	// we need to add a " -- " before the flags if it does not already exist
	if vars.PackageManager == "npm run" {
		// Find the first flag (starts with --)
		parts := strings.Split(cmd, " ")
		for i, part := range parts {
			if part == "--" {
				break // already has --
			}

			if strings.HasPrefix(part, "--") {
				// Insert " -- " before the first flag
				parts = append(parts[:i], append([]string{"--"}, parts[i:]...)...)
				cmd = strings.Join(parts, " ")

				break
			}
		}
	}

	return cmd
}

var tools = []Tool{
	{
		Name:                "Astro",
		Description:         "Static Site Generator (JS) — React, Vue, Markdown, and more",
		Value:               "astro",
		buildCommand:        "{packageManager} build",
		buildCommandSubSite: "{packageManager} build --base {baseURL}",
		devCommand:          "{packageManager} dev --port {port}",
		devCommandSubSite:   "{packageManager} dev --base {baseURL} --port {port}",
		devURL:              "http://localhost:{port}",
		createCommand:       "{packageManager} create astro {path} --no-git",
		npmCreateCommand:    "npm create astro@latest {path} -- --no-git",
		OutputDir:           "dist",
	},
	{
		Name:                "Vite",
		Description:         "Build Tool (JS) — React, Vue, Svelte, and more",
		Value:               "vite",
		buildCommand:        "{packageManager} build",
		buildCommandSubSite: "{packageManager} build --base {baseURL}",
		devCommand:          "{packageManager} dev --port {port}",
		devCommandSubSite:   "{packageManager} dev --base {baseURL} --port {port}",
		devURL:              "http://localhost:{port}",
		OutputDir:           "dist",
		createCommand:       "{packageManager} create vite {path}",
		npmCreateCommand:    "npm create vite@latest {path}",
	},
	{
		Name:                     "Hugo",
		Description:              "Static Site Generator (Go) — Markdown, HTML, and more",
		Value:                    "hugo",
		buildCommand:             "hugo",
		buildCommandSubSite:      "hugo --baseURL {baseURL}",
		devCommand:               "hugo server --port {port}",
		devCommandSubSite:        "hugo server --baseURL {baseURL} --port {port}",
		devURL:                   "http://localhost:{port}",
		OutputDir:                "public",
		createCommand:            "{packageManager} new site {path}",
		InstallLink:              "https://gohugo.io/installation",
		SkipPackageManagerPrompt: true,
	},
}
