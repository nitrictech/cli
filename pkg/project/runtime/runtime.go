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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type RuntimeBuildContext struct {
	DockerfileContents string
	BaseDirectory      string
	BuildArguments     map[string]string
	IgnoreFileContents string
}

type RuntimeExt = string

const (
	RuntimeTypescript RuntimeExt = "ts"
	RuntimeJavascript RuntimeExt = "js"
	RuntimePython     RuntimeExt = "py"
	RuntimeGolang     RuntimeExt = "go"
	RuntimeCsharp     RuntimeExt = "cs"
	RuntimeJvm        RuntimeExt = "jar"

	RuntimeUnknown RuntimeExt = ""
)

var commonIgnore = []string{".nitric/", "!.nitric/*.yaml", ".git/", ".idea/", ".vscode/", ".github/", "*.dockerfile", "*.dockerignore"}

func customBuildContext(entrypointFilePath string, dockerfilePath string, buildArgs map[string]string, additionalIgnores []string, fs afero.Fs) (*RuntimeBuildContext, error) {
	// Get the dockerfile contents
	// dockerfilePath
	dockerfileContents, err := afero.ReadFile(fs, dockerfilePath)
	if err != nil {
		return nil, err
	}

	// Get the ignore file contents

	// Append handler to build args
	buildArgs["HANDLER"] = filepath.ToSlash(entrypointFilePath)

	return &RuntimeBuildContext{
		DockerfileContents: string(dockerfileContents),
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments:     buildArgs,
		IgnoreFileContents: strings.Join(append(additionalIgnores, commonIgnore...), "\n"),
	}, nil
}

//go:embed csharp.dockerfile
var csharpDockerfile string
var csharpIgnores = append([]string{"obj/", "bin/"}, commonIgnore...)

func csharpBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: csharpDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, csharpIgnores...), "\n"),
	}, nil
}

//go:embed golang.dockerfile
var golangDockerfile string
var golangIgnores = append([]string{}, commonIgnore...)

func golangBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: golangDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, golangIgnores...), "\n"),
	}, nil
}

//go:embed jvm.dockerfile
var jvmDockerfile string
var jvmIgnores = append([]string{"obj/", "bin/"}, commonIgnore...)

func jvmBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: jvmDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, jvmIgnores...), "\n"),
	}, nil
}

//go:embed python.dockerfile
var pythonDockerfile string
var pythonIgnores = append([]string{"__pycache__/", "*.py[cod]", "*$py.class"}, commonIgnore...)

func pythonBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: pythonDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, pythonIgnores...), "\n"),
	}, nil
}

//go:embed javascript.dockerfile
var javascriptDockerfile string
var javascriptIgnores = append([]string{"node_modules/"}, commonIgnore...)

func javascriptBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: javascriptDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, javascriptIgnores...), "\n"),
	}, nil
}

//go:embed typescript.dockerfile
var typescriptDockerfile string

func typescriptBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: typescriptDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, javascriptIgnores...), "\n"),
	}, nil
}

//go:embed dart.dockerfile
var dartDockerfile string
var dartIgnores = append([]string{}, commonIgnore...)

func dartBuildContext(entrypointFilePath string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: dartDockerfile,
		BaseDirectory:      ".", // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, dartIgnores...), "\n"),
	}, nil
}

// NewBuildContext - Creates a new runtime build context.
// if a dockerfile path is provided a custom runtime is assumed, otherwise the entrypoint file is used for automatic detection of language runtime.
func NewBuildContext(entrypointFilePath string, dockerfilePath string, buildArgs map[string]string, additionalIgnores []string, fs afero.Fs) (*RuntimeBuildContext, error) {
	if dockerfilePath != "" {
		return customBuildContext(entrypointFilePath, dockerfilePath, buildArgs, additionalIgnores, fs)
	}

	ext := filepath.Ext(entrypointFilePath)

	switch ext {
	case ".cs":
		return csharpBuildContext(entrypointFilePath, additionalIgnores)
	case ".go":
		return golangBuildContext(entrypointFilePath, additionalIgnores)
	case ".jar":
		return jvmBuildContext(entrypointFilePath, additionalIgnores)
	case ".py":
		return pythonBuildContext(entrypointFilePath, additionalIgnores)
	case ".js":
		return javascriptBuildContext(entrypointFilePath, additionalIgnores)
	case ".ts":
		return typescriptBuildContext(entrypointFilePath, additionalIgnores)
	case ".dart":
		return dartBuildContext(entrypointFilePath, additionalIgnores)
	default:
		return nil, fmt.Errorf("nitric does not support files with extension %s by default", ext)
	}
}
