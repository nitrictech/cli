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

package main

import (
	"bufio"
	"log"
	"os"

	"github.com/nitrictech/cli/pkg/cmd"
)

type section struct {
	startLine string
	genFunc   func(*section) []string
	endLine   string
}

var sections = map[string]*section{
	"common": {
		startLine: "## Common Commands",
		genFunc:   generatedCommonCommands,
		endLine:   "## Help with Commands",
	},
	"complete": {
		startLine: "## Complete Reference",
		genFunc:   generatedCompleteCommands,
		endLine:   "## Get in touch",
	},
}

func generatedCommonCommands(s *section) []string {
	return cmd.CommonCommandsUsage()
}

func generatedCompleteCommands(s *section) []string {
	return cmd.AllCommandsUsage()
}

func readmeContent(readmePath string) ([]string, error) {
	file, err := os.Open(readmePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	output := []string{}
	var inSection *section
	for scanner.Scan() {
		if inSection == nil {
			for k, v := range sections {
				if v.startLine == scanner.Text() {
					inSection = sections[k]
					delete(sections, k)
					output = append(output, scanner.Text())
					output = append(output, inSection.genFunc(v)...)
					break
				}
			}
		}
		if inSection != nil {
			if scanner.Text() == inSection.endLine {
				inSection = nil
			}
		}
		if inSection == nil {
			output = append(output, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return output, nil
}

func main() {
	readmePath := os.Args[1]
	output, err := readmeContent(readmePath)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.OpenFile(readmePath, os.O_RDWR, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for _, line := range output {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			log.Print(err)
		}
	}
}
