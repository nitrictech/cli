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
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/cli/pkg/utils"
)

const (
	jvmRuntimeBaseImage = "adoptopenjdk/openjdk11:x86_64-alpine-jre-11.0.10_9"
	mavenOpenJDKImage   = "maven:3-openjdk-11"
)

type java struct {
	rte     RuntimeExt
	handler string
}

var _ Runtime = &java{}

func (t *java) DevImageName() string {
	return fmt.Sprintf("nitric-%s-dev", t.rte)
}

func (t *java) ContainerName() string {
	return filepath.Base(filepath.Dir(t.handler))
}

func (t *java) FunctionDockerfileForCodeAsConfig(w io.Writer) error {
	return utils.NewNotSupportedErr("code-as-config not supported on " + string(t.rte))
}

func (t *java) LaunchOptsForFunctionCollect(runCtx string) (LaunchOpts, error) {
	return LaunchOpts{}, utils.NewNotSupportedErr("code-as-config not supported on " + string(t.rte))
}

func (t *java) LaunchOptsForFunction(runCtx string) (LaunchOpts, error) {
	return LaunchOpts{}, utils.NewNotSupportedErr("code-as-config not supported on " + string(t.rte))
}

func (t *java) BuildIgnore() []string {
	return commonIgnore
}

func (t *java) FunctionDockerfile(funcCtxDir, version, provider string, w io.Writer) error {
	css := dockerfile.NewStateStore()

	buildCon, err := css.NewContainer(dockerfile.NewContainerOpts{
		From:   mavenOpenJDKImage,
		As:     layerBuild,
		Ignore: t.BuildIgnore(),
	})
	if err != nil {
		return err
	}

	err = mavenBuild(buildCon, funcCtxDir)
	if err != nil {
		return err
	}

	con, err := css.NewContainer(dockerfile.NewContainerOpts{
		From:   jvmRuntimeBaseImage,
		As:     layerFinal,
		Ignore: t.BuildIgnore(),
	})
	if err != nil {
		return err
	}

	err = con.Copy(dockerfile.CopyOptions{Src: t.handler, Dest: "function.jar", From: buildCon.Name()})
	if err != nil {
		return err
	}

	con.Config(dockerfile.ConfigOptions{
		WorkingDir: "/",
		Ports:      []int32{9001},
		Cmd:        []string{"java", "-jar", "function.jar"},
	})
	withMembrane(con, version, provider)

	_, err = w.Write([]byte(strings.Join(append(buildCon.Lines(), con.Lines()...), "\n")))

	return err
}

func mavenBuild(con dockerfile.ContainerState, contextDir string) error {
	pomFiles, err := utils.FindFilesInDir(contextDir, "pom.xml")
	if err != nil {
		return err
	}

	moduleDirs := []string{}

	for _, p := range pomFiles {
		moduleDirs = append(moduleDirs, path.Dir(p))

		err = con.Copy(dockerfile.CopyOptions{Src: p, Dest: filepath.Join("./", p)})
		if err != nil {
			return err
		}
	}

	con.Run(dockerfile.RunOptions{Command: []string{"mvn", "de.qaware.maven:go-offline-maven-plugin:resolve-dependencies"}})

	for _, d := range moduleDirs {
		err = con.Copy(dockerfile.CopyOptions{Src: d, Dest: filepath.Join("./", d)})
		if err != nil {
			return err
		}
	}

	con.Run(dockerfile.RunOptions{Command: []string{"mvn", "clean", "package"}})

	return nil
}
