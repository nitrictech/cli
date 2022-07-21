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
	"fmt"
	"os"
	"os/exec"
	"strings"

	multierror "github.com/missionMeteora/toolkit/errors"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/aws"
	"github.com/nitrictech/cli/pkg/provider/pulumi/azure"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/provider/pulumi/gcp"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/utils"
)

type pulumiDeployment struct {
	proj *project.Project
	sc   *stack.Config
	prov common.PulumiProvider
	opts *types.ProviderOpts
}

type stackSummary struct {
	Name             string `json:"name"`
	Deployed         bool   `json:"deployed"`
	LastUpdate       string `json:"lastUpdate,omitempty"`
	UpdateInProgress bool   `json:"updateInProgress"`
	ResourceCount    *int   `json:"resourceCount,omitempty"`
	URL              string `json:"url,omitempty"`
}

var (
	_ types.Provider = &pulumiDeployment{}
)

func New(p *project.Project, sc *stack.Config, envMap map[string]string, opts *types.ProviderOpts) (types.Provider, error) {
	pv := exec.Command("pulumi", "version")

	err := pv.Run()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, errors.WithMessage(err, "Please install pulumi from https://www.pulumi.com/docs/get-started/install/")
		}

		return nil, err
	}

	var prov common.PulumiProvider

	switch sc.Provider {
	case stack.Aws:
		prov = aws.New(p, sc, envMap)
	case stack.Azure:
		prov = azure.New(p, sc, envMap)
	case stack.Gcp:
		prov = gcp.New(p, sc, envMap)
	default:
		return nil, utils.NewNotSupportedErr("pulumi provider " + sc.Provider + " not suppored")
	}

	return &pulumiDeployment{
		proj: p,
		sc:   sc,
		prov: prov,
		opts: opts,
	}, nil
}

func (p *pulumiDeployment) Ask() (*stack.Config, error) {
	return p.prov.Ask()
}

func (p *pulumiDeployment) TryPullImages() error {
	if !strings.Contains(os.Getenv("PULUMI_BACKEND_URL"), "://") {
		return nil
	}

	// s3://<bucket-path>, azblob://<container-path>, gs://<bucket-path>, or file://<fs-path>
	backend := strings.Split(os.Getenv("PULUMI_BACKEND_URL"), ":")[0]
	backendProv := p.prov

	switch backend {
	case "s3":
		if p.sc.Provider != stack.Aws {
			backendProv = aws.New(p.proj, p.sc, nil)
		}
	case "azblob":
		if p.sc.Provider != stack.Azure {
			backendProv = azure.New(p.proj, p.sc, nil)
		}
	case "gs":
		if p.sc.Provider != stack.Gcp {
			backendProv = gcp.New(p.proj, p.sc, nil)
		}
	default:
		return nil
	}

	ps, err := backendProv.ReadPulumiStack(context.TODO(), os.Getenv("PULUMI_BACKEND_URL"))
	if err != nil || ps == nil {
		return err
	}

	merr := multierror.ErrorList{}

	for _, r := range ps.Checkpoint.Latest.Resources {
		if r.Type != "nitric:Image" {
			continue
		}

		err = p.prov.TryPullImage(context.TODO(), r.Outputs["imageUri"].(string))
		if err != nil {
			merr.Push(err)
		}
	}

	return merr.Err()
}

func (p *pulumiDeployment) SupportedRegions() []string {
	return p.prov.SupportedRegions()
}

func (p *pulumiDeployment) load(log output.Progress) (*auto.Stack, error) {
	if err := p.prov.Validate(); err != nil {
		return nil, err
	}

	stackName := p.proj.Name + "-" + p.sc.Name
	ctx := context.Background()

	s, err := auto.UpsertStackInlineSource(ctx, stackName, p.proj.Name, p.prov.Deploy,
		auto.SecretsProvider("passphrase"),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(p.proj.Name),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			Main:    p.proj.Dir,
		}))
	if err != nil {
		return nil, errors.WithMessage(err, "UpsertStackInlineSource")
	}

	// Cancel all previously running stacks
	if p.opts.Force {
		// It will only return an error if the stack isn't in an updating state, so we can just ignore it
		_ = s.Cancel(ctx)
	}

	for _, plug := range p.prov.Plugins() {
		log.Busyf("Installing Pulumi plugin %s:%s", plug.Name, plug.Version)

		err = s.Workspace().InstallPlugin(ctx, plug.Name, plug.Version)
		if err != nil {
			return nil, errors.WithMessage(err, "InstallPlugin "+plug.String())
		}
	}

	err = p.prov.Configure(ctx, &s)
	if err != nil {
		return nil, errors.WithMessage(err, "Configure")
	}

	log.Busyf("Refreshing the Pulumi stack")

	_, err = s.Refresh(ctx)
	if err != nil && strings.Contains(err.Error(), "[409] Conflict") {
		return &s, errors.WithMessage(fmt.Errorf("Stack conflict occurred. If you are sure an update is not in progress, use --force to override the stack state."), "Refresh")
	}

	return &s, errors.WithMessage(err, "Refresh")
}

func (p *pulumiDeployment) Up(log output.Progress) (*types.Deployment, error) {
	s, err := p.load(log)
	if err != nil {
		return nil, errors.WithMessage(err, "loading pulumi stack")
	}

	res, err := s.Up(context.Background(), updateLoggingOpts(log)...)
	if err != nil {
		return nil, errors.WithMessage(err, "Updating pulumi stack "+res.Summary.Message)
	}

	defer p.prov.CleanUp()

	d := &types.Deployment{
		ApiEndpoints: map[string]string{},
	}

	for k, v := range res.Outputs {
		if strings.HasPrefix(k, "api:") {
			d.ApiEndpoints[strings.TrimPrefix(k, "api:")] = fmt.Sprint(v.Value)
		}
	}

	return d, nil
}

func (p *pulumiDeployment) List() (interface{}, error) {
	projectName := p.proj.Name

	ws, err := auto.NewLocalWorkspace(context.Background(),
		auto.SecretsProvider("passphrase"),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			Main:    p.proj.Dir,
		}))
	if err != nil {
		return nil, errors.WithMessage(err, "UpsertStackInlineSource")
	}

	sl, err := ws.ListStacks(context.Background())
	if err != nil {
		return nil, errors.WithMessage(err, "ListStacks")
	}

	stackName := p.proj.Name + "-" + p.sc.Name
	result := []stackSummary{}

	for _, st := range sl {
		if strings.HasPrefix(st.Name, stackName) {
			var stackListOutput = stackSummary{
				Name:             st.Name,
				Deployed:         *st.ResourceCount > 0,
				LastUpdate:       st.LastUpdate,
				UpdateInProgress: st.UpdateInProgress,
				ResourceCount:    st.ResourceCount,
				URL:              st.URL,
			}

			result = append(result, stackListOutput)
		}
	}

	return result, nil
}

func (a *pulumiDeployment) Down(log output.Progress) error {
	s, err := a.load(log)
	if err != nil {
		return err
	}

	res, err := s.Destroy(context.Background(), destroyLoggingOpts(log)...)
	if err != nil {
		return errors.WithMessage(err, res.Summary.Message)
	}

	return nil
}
