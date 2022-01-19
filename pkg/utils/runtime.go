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

package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type Runtime string

const (
	RuntimeTypescript Runtime = "ts"
	RuntimeJavascript Runtime = "js"
	RuntimePython     Runtime = "python"
	RuntimeGolang     Runtime = "go"
	RuntimeJava       Runtime = "java"

	RuntimeUnknown Runtime = ""
)

func NewRunTimeFromFilename(file string) (Runtime, error) {
	rt := Runtime(strings.Replace(filepath.Ext(file), ".", "", -1))
	switch rt {
	case RuntimeGolang:
		return RuntimeGolang, nil
	case RuntimeJavascript:
		return RuntimeJavascript, nil
	case RuntimePython:
		return RuntimePython, nil
	case RuntimeTypescript:
		return RuntimeTypescript, nil
	default:
		return RuntimeUnknown, errors.New("runtime '" + string(rt) + "' not supported")
	}
}

func (r Runtime) String() string {
	return string(r)
}

func (r Runtime) DevImageName() string {
	return fmt.Sprintf("nitric-%s-dev", r)
}

func ImagesToBuild(handlers []string) (map[string]string, error) {
	imagesToBuild := map[string]string{}
	for _, h := range handlers {
		rt, err := NewRunTimeFromFilename(h)
		if err != nil {
			return nil, err
		}
		imagesToBuild[rt.String()] = rt.DevImageName()
	}
	return imagesToBuild, nil
}
