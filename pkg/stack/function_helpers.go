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

package stack

import (
	"fmt"
	"path"
)

const DefaulMembraneVersion = "v0.12.1-rc.5"

var _ Compute = &Function{}

func (f *Function) Unit() *ComputeUnit {
	return &f.ComputeUnit
}

func (f *Function) VersionString(s *Stack) string {
	if f.Version != "" {
		return f.Version
	}
	return DefaulMembraneVersion
}

func (f *Function) WithPrivateInfo(name, stackDir string) *Function {
	f.name = name
	if f.Context != "" {
		f.contextDirectory = path.Join(stackDir, f.Context)
	} else {
		f.contextDirectory = stackDir
	}
	return f
}

func (f *Function) ContextDirectory() string {
	return f.contextDirectory
}

// ImageTagName returns the default image tag for a source image built from this function
// provider the provider name (e.g. aws), used to uniquely identify builds for specific providers
func (f *Function) ImageTagName(s *Stack, provider string) string {
	if f.Tag != "" {
		return f.Tag
	}
	providerString := ""
	if provider != "" {
		providerString = "-" + provider
	}
	return fmt.Sprintf("%s-%s%s", s.Name, f.Name, providerString)
}
