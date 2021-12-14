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
	"path"

	"github.com/nitrictech/newcli/pkg/functiondockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

func BuildCreate(s *stack.Stack, t *target.Target) error {
	cr, err := DiscoverContainerRuntime()
	if err != nil {
		return err
	}
	for _, f := range s.Functions {
		fh, err := os.CreateTemp("", "Dockerfile.*")
		if err != nil {
			return err
		}
		err = functiondockerfile.Generate(&f, f.VersionString(s), t.Provider, fh)
		if err != nil {
			return err
		}
		err = cr.Build(fh.Name(), f.ContextDirectory(), f.ImageTagName(s, t.Provider), t.Provider, map[string]string{})
		if err != nil {
			return err
		}
	}

	for _, c := range s.Containers {
		err := cr.Build(path.Join(c.ContextDirectory(), c.Dockerfile), c.ContextDirectory(), c.ImageTagName(s, t.Provider), t.Provider, map[string]string{})
		if err != nil {
			return err
		}
	}
	return nil
}

type Image struct {
	ID         string `yaml:"id"`
	Repository string `yaml:"repository,omitempty"`
	Tag        string `yaml:"tag,omitempty`
	CreatedAt  string `yaml:CreatedAt`
}

type StackImages struct {
	Name       string             `yaml:"name"`
	Functions  map[string][]Image `yaml:"functionImages,omitempty"`
	Containers map[string][]Image `yaml:"containerImages,omitempty"`
}

func BuildList(s *stack.Stack) (*StackImages, error) {
	cr, err := DiscoverContainerRuntime()
	if err != nil {
		return nil, err
	}
	si := &StackImages{
		Name:       s.Name,
		Functions:  map[string][]Image{},
		Containers: map[string][]Image{},
	}
	for n, f := range s.Functions {
		images, err := cr.ListImages(s.Name, f.Name())
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			si.Functions[n] = images
		}
	}
	for n, c := range s.Containers {
		images, err := cr.ListImages(s.Name, c.Name())
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			si.Functions[n] = images
		}
	}
	return si, nil
}
