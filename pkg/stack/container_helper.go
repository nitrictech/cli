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

var _ Compute = &Container{}

func (c *Container) Unit() *ComputeUnit {
	return &c.ComputeUnit
}

func (c *Container) SetContextDirectory(stackDir string) {
	if c.Context != "" {
		c.ContextDirectory = path.Join(stackDir, c.Context)
	} else {
		c.ContextDirectory = stackDir
	}
}

// ImageTagName returns the default image tag for a source image built from this function
// provider the provider name (e.g. aws), used to uniquely identify builds for specific providers
func (c *Container) ImageTagName(s *Stack, provider string) string {
	if c.Tag != "" {
		return c.Tag
	}
	providerString := ""
	if provider != "" {
		providerString = "-" + provider
	}
	return fmt.Sprintf("%s-%s%s", s.Name, c.Name, providerString)
}
