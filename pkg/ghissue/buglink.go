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

package ghissue

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/utils"
)

type Diagnostics struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	GoVersion  string `json:"goVersion"`
	CliVersion string `json:"cliVersion"`
}

type GHIssue struct {
	Diagnostics Diagnostics `json:"diagnostics"`
	Command     string      `json:"command"`
	Error       string      `json:"error"`
	StackTrace  string      `json:"stacktrace"`
}

var diag = Diagnostics{
	OS:         runtime.GOOS,
	Arch:       runtime.GOARCH,
	GoVersion:  runtime.Version(),
	CliVersion: utils.Version,
}

func BugLink(err interface{}) string {
	issue := GHIssue{
		Diagnostics: diag,
		Error:       fmt.Sprint(err),
		StackTrace:  string(debug.Stack()),
		Command:     strings.Join(os.Args, " "),
	}

	title := "Command '" + issue.Command + "' panicked: " + utils.StringTrunc(issue.Error, 50)
	b, _ := yaml.Marshal(issue)

	issueUrl, _ := url.Parse("https://github.com/nitrictech/cli/issues/new")

	q := issueUrl.Query()
	q.Add("title", title)
	q.Add("body", string(b))
	issueUrl.RawQuery = q.Encode()

	return issueUrl.String()
}
