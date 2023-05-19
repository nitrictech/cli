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

package runtime

import (
	_ "embed"
	"io"
	"path/filepath"
	"strings"
)

//go:embed csharp.dockerfile
var csharpDockerfile string

type csharp struct {
	rte     RuntimeExt
	handler string
}

var _ Runtime = &csharp{}

func (t *csharp) ContainerName() string {
	return strings.ToLower(strings.Replace(filepath.Base(t.handler), filepath.Ext(t.handler), "", 1))
}

func (t *csharp) BuildIgnore(additional ...string) []string {
	baseIgnores := append(commonIgnore, additional...)
	return append(baseIgnores, "obj/", "bin/")
}

func (t *csharp) BaseDockerFile(w io.Writer) error {
	_, err := w.Write([]byte(csharpDockerfile))
	return err
}

func (t *csharp) BuildArgs() map[string]string {
	return map[string]string{
		"HANDLER": strings.ToLower(strings.Split(filepath.Base(t.handler), ".")[0]),
	}
}