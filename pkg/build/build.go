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

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/runtime"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

func Create(s *stack.Stack, t *target.Target) error {
	cr, err := containerengine.Discover()
	if err != nil {
		return err
	}
	for _, f := range s.Functions {
		for _, script := range f.BuildScripts {
			cmd := exec.Command(script)
			cmd.Dir = path.Join(s.Dir, f.Context)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}

		fh, err := os.CreateTemp(s.Dir, "Dockerfile.*")
		if err != nil {
			return err
		}
		defer func() { os.Remove(fh.Name()) }()

		rt, err := runtime.NewRunTimeFromHandler(f.Handler)
		if err != nil {
			return err
		}
		err = rt.FunctionDockerfile(f.ContextDirectory, f.VersionString(s), t.Provider, fh)
		if err != nil {
			return err
		}
		fh.Close()

		buildArgs := map[string]string{"PROVIDER": t.Provider}
		err = cr.Build(path.Base(fh.Name()), f.ContextDirectory, f.ImageTagName(s, t.Provider), buildArgs, f.Excludes)
		if err != nil {
			return err
		}
	}

	for _, c := range s.Containers {
		buildArgs := map[string]string{"PROVIDER": t.Provider}
		err := cr.Build(path.Join(c.Context, c.Dockerfile), c.ContextDirectory, c.ImageTagName(s, t.Provider), buildArgs, []string{})
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateBaseDev builds images for code-as-config
func CreateBaseDev(s *stack.Stack) error {
	ce, err := containerengine.Discover()
	if err != nil {
		return err
	}
	imagesToBuild := map[string]string{}
	for _, f := range s.Functions {
		rt, err := runtime.NewRunTimeFromHandler(f.Handler)
		if err != nil {
			return err
		}
		lang := strings.Replace(path.Ext(f.Handler), ".", "", 1)
		_, ok := imagesToBuild[lang]
		if ok {
			continue
		}

		f, err := os.CreateTemp(s.Dir, fmt.Sprintf("%s.*.dockerfile", lang))
		if err != nil {
			return err
		}

		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		if err := rt.FunctionDockerfileForCodeAsConfig(f); err != nil {
			return err
		}

		if err := ce.Build(path.Base(f.Name()), s.Dir, rt.DevImageName(), map[string]string{}, []string{}); err != nil {
			return err
		}
		imagesToBuild[lang] = rt.DevImageName()
	}

	return nil
}

func List(s *stack.Stack) ([]containerengine.Image, error) {
	cr, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}
	images := []containerengine.Image{}
	for _, f := range s.Functions {
		imgs, err := cr.ListImages(s.Name, f.Name)
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			images = append(images, imgs...)
		}
	}
	for _, c := range s.Containers {
		imgs, err := cr.ListImages(s.Name, c.Name)
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			images = append(images, imgs...)
		}
	}
	return images, nil
}
