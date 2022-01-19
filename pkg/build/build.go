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

	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/functiondockerfile"
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
			cmd.Dir = path.Join(s.Path(), f.Context)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}

		fh, err := os.CreateTemp(s.Path(), "Dockerfile.*")
		if err != nil {
			return err
		}
		defer func() { os.Remove(fh.Name()) }()

		err = functiondockerfile.Generate(&f, f.VersionString(s), t.Provider, fh)
		if err != nil {
			return err
		}
		fh.Close()

		buildArgs := map[string]string{"PROVIDER": t.Provider}
		err = cr.Build(path.Base(fh.Name()), f.ContextDirectory(), f.ImageTagName(s, t.Provider), buildArgs)
		if err != nil {
			return err
		}
	}

	for _, c := range s.Containers {
		buildArgs := map[string]string{"PROVIDER": t.Provider}
		err := cr.Build(path.Join(c.Context, c.Dockerfile), c.ContextDirectory(), c.ImageTagName(s, t.Provider), buildArgs)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateBaseDev builds images for code-as-config
func CreateBaseDev(stackPath string, imagesToBuild map[string]string) error {
	ce, err := containerengine.Discover()
	if err != nil {
		return err
	}

	for lang, imageTag := range imagesToBuild {
		f, err := os.CreateTemp(stackPath, fmt.Sprintf("%s.*.dockerfile", lang))
		if err != nil {
			return err
		}

		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		if err := functiondockerfile.GenerateForCodeAsConfig("handler."+lang, f); err != nil {
			return err
		}

		if err := ce.Build(path.Base(f.Name()), stackPath, imageTag, map[string]string{}); err != nil {
			return err
		}
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
