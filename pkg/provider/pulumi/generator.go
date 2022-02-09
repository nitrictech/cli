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

package pulumi

import (
	"context"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/debug"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"

	"github.com/nitrictech/cli/pkg/provider/pulumi/aws"
	pulumitypes "github.com/nitrictech/cli/pkg/provider/pulumi/types"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/target"
	"github.com/nitrictech/cli/pkg/utils"
)

type pulumiDeployment struct {
	s *stack.Stack
	t *target.Target
	p pulumitypes.PulumiProvider
}

var (
	_ types.Provider = &pulumiDeployment{}
)

func New(s *stack.Stack, t *target.Target) (types.Provider, error) {
	var prov pulumitypes.PulumiProvider
	switch t.Provider {
	case target.Aws:
		prov = aws.New(s, t)
	default:
		return nil, utils.NewNotSupportedErr("pulumi provider " + t.Provider + " not suppored")
	}

	return &pulumiDeployment{
		s: s,
		t: t,
		p: prov,
	}, nil
}

func (p *pulumiDeployment) load(name string) (*auto.Stack, error) {
	projectName := p.s.Name
	stackName := p.s.Name + "-" + name
	ctx := context.Background()

	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, p.p.Deploy,
		auto.SecretsProvider("passphrase"),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			Main:    p.s.Dir,
		}))
	if err != nil {
		return nil, errors.WithMessage(err, "UpsertStackInlineSource")
	}

	err = s.Workspace().InstallPlugin(ctx, p.p.PluginName(), p.p.PluginVersion())
	if err != nil {
		return nil, errors.WithMessage(err, "InstallPlugin")
	}

	err = p.p.Configure(ctx, &s)
	if err != nil {
		return nil, errors.WithMessage(err, "Configure")
	}

	_, err = s.Refresh(ctx)
	return &s, errors.WithMessage(err, "Refresh")
}

func (p *pulumiDeployment) Apply(name string) error {
	s, err := p.load(name)
	if err != nil {
		return err
	}
	var loglevel uint = 2
	_ = optup.DebugLogging(debug.LoggingOptions{
		LogLevel:    &loglevel,
		LogToStdErr: true})

	res, err := s.Up(context.Background())
	defer p.p.CleanUp()
	if err != nil {
		return errors.WithMessage(err, res.Summary.Message)
	}
	return nil
}

func (p *pulumiDeployment) List() (interface{}, error) {
	projectName := p.s.Name

	ws, err := auto.NewLocalWorkspace(context.Background(),
		auto.SecretsProvider("passphrase"),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			Main:    p.s.Dir,
		}))
	if err != nil {
		return nil, errors.WithMessage(err, "UpsertStackInlineSource")
	}

	return ws.ListStacks(context.Background())
}

func (a *pulumiDeployment) Delete(name string) error {
	s, err := a.load(name)
	if err != nil {
		return err
	}
	res, err := s.Destroy(context.Background())
	if err != nil {
		return errors.WithMessage(err, res.Summary.Message)
	}
	return nil
}
