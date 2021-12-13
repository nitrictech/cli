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

package output

import (
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/newcli/pkg/pflagext"
)

var (
	allowedFormats = []string{"json", "yaml", "table"}
	defaultFormat  = "table"
	outputFormat   string
	OutputTypeFlag = pflagext.NewStringEnumVar(&outputFormat, allowedFormats, defaultFormat)
)

func Print(object interface{}) {
	switch outputFormat {
	case "json":
		printJson(object)
	case "yaml":
		printYaml(object)
	default:
		printTable(object)
	}
}

func printJson(object interface{}) {
	b, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func printYaml(object interface{}) {
	b, err := yaml.Marshal(object)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func printTable(object interface{}) {
	// TODO research a good printer
	spew.Dump(object)
}
