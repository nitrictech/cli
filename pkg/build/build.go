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

		fh, err := os.CreateTemp("", "Dockerfile.*")
		if err != nil {
			return err
		}

		defer func() {
			fh.Close()
			os.Remove(fh.Name())
		}()

		err = functiondockerfile.Generate(&f, f.VersionString(s), t.Provider, fh)
		if err != nil {
			return err
		}
		buildArgs := map[string]string{"PROVIDER": t.Provider}
		if buildArgs["PROVIDER"] == "local" {
			buildArgs["PROVIDER"] = "dev"
		}
		err = cr.Build(fh.Name(), f.ContextDirectory(), f.ImageTagName(s, t.Provider), buildArgs)
		if err != nil {
			return err
		}
	}

	for _, c := range s.Containers {
		buildArgs := map[string]string{"PROVIDER": t.Provider}
		if buildArgs["PROVIDER"] == "local" {
			buildArgs["PROVIDER"] = "dev"
		}
		err := cr.Build(path.Join(c.ContextDirectory(), c.Dockerfile), c.ContextDirectory(), c.ImageTagName(s, t.Provider), buildArgs)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateBaseDev builds images for code-as-config
func CreateBaseDev(handlers []string) error {
	ce, err := containerengine.Discover()
	if err != nil {
		return err
	}

	imagesToBuild := map[string]string{}
	for _, h := range handlers {
		lang := strings.Replace(path.Ext(h), ".", "", 1)
		imagesToBuild[lang] = "nitric-" + lang + "-dev"
	}

	for lang, imageTag := range imagesToBuild {
		f, err := os.CreateTemp("", fmt.Sprintf("%s.*.dockerfile", lang))
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

		if err := ce.Build(f.Name(), ".", imageTag, map[string]string{}); err != nil {
			return err
		}
	}

	return nil
}

type StackImages struct {
	Name       string                             `yaml:"name"`
	Functions  map[string][]containerengine.Image `yaml:"functionImages,omitempty"`
	Containers map[string][]containerengine.Image `yaml:"containerImages,omitempty"`
}

func List(s *stack.Stack) (*StackImages, error) {
	cr, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}
	images := []containerengine.Image{}
	for _, f := range s.Functions {
		imgs, err := cr.ListImages(s.Name, f.Name())
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			images = append(images, imgs...)
		}
	}
	for _, c := range s.Containers {
		imgs, err := cr.ListImages(s.Name, c.Name())
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			images = append(images, imgs...)
		}
	}
	return images, nil
}
