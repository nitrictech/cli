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

	"github.com/samber/lo"
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
	RuntimeCsharp     RuntimeExt = "cs"
	RuntimeJvm        RuntimeExt = "jar"

	RuntimeUnknown RuntimeExt = ""
)

var commonIgnore = []string{".nitric/", "!.nitric/*.yaml", ".git/", ".idea/", ".vscode/", ".github/", "*.dockerfile", "*.dockerignore"}

func getDockerIgnores(dockerIgnorePath string, fs afero.Fs) ([]string, error) {
	// Check if the file exists
	exists, err := afero.Exists(fs, dockerIgnorePath)
	if err != nil {
		return nil, err
	}

	if exists {
		// Read the file
		content, err := afero.ReadFile(fs, dockerIgnorePath)
		if err != nil {
			return nil, err
		}

		// Split the content into lines
		lines := lo.Filter[string](strings.Split(string(content), "\n"), func(line string, index int) bool {
			return strings.TrimSpace(line) != ""
		})

		return lines, nil
	}

	return []string{}, nil
}

func customBuildContext(entrypointFilePath string, dockerfilePath string, baseDirectory string, buildArgs map[string]string, additionalIgnores []string, fs afero.Fs) (*RuntimeBuildContext, error) {
	// Get the dockerfile contents
	// dockerfilePath
	dockerfileContents, err := afero.ReadFile(fs, dockerfilePath)
	if err != nil {
		return nil, err
	}

	// ensure build args exists
	if buildArgs == nil {
		buildArgs = map[string]string{}
	} else {
		// Copy the build args to avoid modifying the original
		copiedBuildArgs := map[string]string{}
		for k, v := range buildArgs {
			copiedBuildArgs[k] = v
		}

		buildArgs = copiedBuildArgs
	}

	// Append handler to build args
	buildArgs["HANDLER"] = filepath.ToSlash(entrypointFilePath)

	return &RuntimeBuildContext{
		DockerfileContents: string(dockerfileContents),
		BaseDirectory:      baseDirectory, // uses the nitric project directory by default
		BuildArguments:     buildArgs,
		IgnoreFileContents: strings.Join(append(additionalIgnores, commonIgnore...), "\n"),
	}, nil
}

//go:embed csharp.dockerfile
var csharpDockerfile string
var csharpIgnores = append([]string{"obj/", "bin/"}, commonIgnore...)

func csharpBuildContext(entrypointFilePath string, baseDir string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	// Convert the service name to the name of the binary produced. i.e. services/hello.csproj -> hello
	handler := strings.ReplaceAll(filepath.Base(entrypointFilePath), ".csproj", "")

	return &RuntimeBuildContext{
		DockerfileContents: csharpDockerfile,
		BaseDirectory:      baseDir, // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": handler,
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, csharpIgnores...), "\n"),
	}, nil
}

//go:embed jvm.dockerfile
var jvmDockerfile string
var jvmIgnores = append([]string{"obj/", "bin/"}, commonIgnore...)

func jvmBuildContext(entrypointFilePath string, baseDir string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: jvmDockerfile,
		BaseDirectory:      baseDir, // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, jvmIgnores...), "\n"),
	}, nil
}

//go:embed python.dockerfile
var pythonDockerfile string
var pythonIgnores = append([]string{"__pycache__/", "*.py[cod]", "*$py.class"}, commonIgnore...)

func pythonBuildContext(entrypointFilePath string, baseDir string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: pythonDockerfile,
		BaseDirectory:      baseDir, // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, pythonIgnores...), "\n"),
	}, nil
}

//go:embed javascript.dockerfile
var javascriptDockerfile string
var javascriptIgnores = append([]string{"node_modules/"}, commonIgnore...)

func javascriptBuildContext(entrypointFilePath string, baseDir string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: javascriptDockerfile,
		BaseDirectory:      baseDir, // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, javascriptIgnores...), "\n"),
	}, nil
}

//go:embed typescript.dockerfile
var typescriptDockerfile string

func typescriptBuildContext(entrypointFilePath string, baseDir string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: typescriptDockerfile,
		BaseDirectory:      baseDir, // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, javascriptIgnores...), "\n"),
	}, nil
}

//go:embed dart.dockerfile
var dartDockerfile string
var dartIgnores = append([]string{}, commonIgnore...)

func dartBuildContext(entrypointFilePath string, baseDir string, additionalIgnores []string) (*RuntimeBuildContext, error) {
	return &RuntimeBuildContext{
		DockerfileContents: dartDockerfile,
		BaseDirectory:      baseDir, // use the nitric project directory
		BuildArguments: map[string]string{
			"HANDLER": filepath.ToSlash(entrypointFilePath),
		},
		IgnoreFileContents: strings.Join(append(additionalIgnores, dartIgnores...), "\n"),
	}, nil
}

const customDockerfileDocLink = "https://nitric.io/docs/reference/custom-containers#create-a-dockerfile-template"

// NewBuildContext - Creates a new runtime build context.
// if a dockerfile path is provided a custom runtime is assumed, otherwise the entrypoint file is used for automatic detection of language runtime.
func NewBuildContext(entrypointFilePath string, dockerfilePath string, baseDirectory string, buildArgs map[string]string, additionalIgnores []string, fs afero.Fs) (*RuntimeBuildContext, error) {
	if baseDirectory == "" {
		baseDirectory = "."
	}

	if dockerfilePath != "" {
		dockerIgnorePath := fmt.Sprintf("%s.dockerignore", dockerfilePath)

		dockerIgnores, err := getDockerIgnores(dockerIgnorePath, fs)
		if err != nil {
			return nil, err
		}

		additionalIgnores = append(additionalIgnores, dockerIgnores...)

		return customBuildContext(entrypointFilePath, dockerfilePath, baseDirectory, buildArgs, additionalIgnores, fs)
	}

	if fi, err := fs.Stat(entrypointFilePath); err == nil && fi.IsDir() {
		return nil, fmt.Errorf("nitric does not support directories by default, use a custom runtime with a Dockerfile see: %s", customDockerfileDocLink)
	}

	ext := filepath.Ext(entrypointFilePath)

	dockerIgnores, err := getDockerIgnores(".dockerignore", fs)
	if err != nil {
		return nil, err
	}

	additionalIgnores = append(additionalIgnores, dockerIgnores...)

	switch ext {
	case ".csproj":
		return csharpBuildContext(entrypointFilePath, baseDirectory, additionalIgnores)
	case ".jar":
		return jvmBuildContext(entrypointFilePath, baseDirectory, additionalIgnores)
	case ".py":
		return pythonBuildContext(entrypointFilePath, baseDirectory, additionalIgnores)
	case ".js":
		return javascriptBuildContext(entrypointFilePath, baseDirectory, additionalIgnores)
	case ".ts":
		return typescriptBuildContext(entrypointFilePath, baseDirectory, additionalIgnores)
	case ".dart":
		return dartBuildContext(entrypointFilePath, baseDirectory, additionalIgnores)
	default:
		return nil, fmt.Errorf("nitric does not support files with extension %s by default", ext)
	}
}
