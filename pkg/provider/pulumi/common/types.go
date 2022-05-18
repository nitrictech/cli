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

package common

import (
	"context"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/cli/pkg/stack"
)

type Plugin struct {
	Name    string
	Version string
}

func (p *Plugin) String() string {
	return p.Name + " " + p.Version
}

type PulumiProvider interface {
	Validate() error
	Plugins() []Plugin
	Configure(context.Context, *auto.Stack) error
	Deploy(*pulumi.Context) error
	CleanUp()
	Ask() (*stack.Config, error)
	TryPullImages() error
	SupportedRegions() []string
}

func Tags(ctx *pulumi.Context, name string) pulumi.StringMap {
	return pulumi.StringMap{
		"x-nitric-project": pulumi.String(ctx.Project()),
		"x-nitric-stack":   pulumi.String(ctx.Stack()),
		"x-nitric-name":    pulumi.String(name),
	}
}

func IntValueOrDefault(v, def int) int {
	if v != 0 {
		return v
	}
	return def
}
