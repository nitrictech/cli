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
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/avast/retry-go"
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
	"github.com/nitrictech/cli/pkg/utils"
)

type pulumiDeployment struct {
	proj      *project.Project
	stackName string
	provider  string
	prov      common.PulumiProvider
	opts      *types.ProviderOpts
}

type stackSummary struct {
	Name             string `json:"name"`
	Deployed         bool   `json:"deployed"`
	LastUpdate       string `json:"lastUpdate,omitempty"`
	UpdateInProgress bool   `json:"updateInProgress"`
	ResourceCount    *int   `json:"resourceCount,omitempty"`
	URL              string `json:"url,omitempty"`
}

type pulumiBackend struct{}

type pulumiAbout struct {
	Backend *pulumiBackend `json:"backend"`
}

var _ types.Provider = &pulumiDeployment{}

func New(p *project.Project, name, provider string, envMap map[string]string, opts *types.ProviderOpts) (types.Provider, error) {
	err := shimPulumi()
	if err != nil {
		return nil, err
	}

	pv := exec.Command("pulumi", "version")

	err = pv.Run()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, errors.WithMessage(err, "please install pulumi from https://www.pulumi.com/docs/get-started/install/")
		}

		return nil, err
	}

	var prov common.PulumiProvider

	switch provider {
	case types.Aws:
		prov, err = aws.New(p, name, envMap)
	case types.Azure:
		prov, err = azure.New(p, name, envMap)
	case types.Gcp:
		prov, err = gcp.New(p, name, envMap)
	default:
		return nil, utils.NewNotSupportedErr("pulumi provider " + provider + " not suppored")
	}

	if err != nil {
		return nil, err
	}

	return &pulumiDeployment{
		proj:      p,
		stackName: name,
		provider:  provider,
		prov:      prov,
		opts:      opts,
	}, nil
}

func (p *pulumiDeployment) AskAndSave() error {
	return p.prov.AskAndSave()
}

func (p *pulumiDeployment) SupportedRegions() []string {
	return p.prov.SupportedRegions()
}

//go:embed pulumi-shim.sh
var shimScript string

// Check if pulumi exists on path and if not shim it with the docker image
// this also requires some default setup
// NOTE: This can be dangerous in some cases and users serious about deploying
// to production should be educated on how to manage their pulumi stack states
func shimPulumi() error {
	// Only shim if pulumi cannot be found on path already
	_, err := exec.LookPath("pulumi")
	if err != nil {
		log.Default().Printf("pulumi not found on path, using pulumi docker image\n")
		// TODO: Check if pulumi exists on the local install
		shimDir := fmt.Sprintf("%s/.nitric/shims", os.Getenv("HOME"))

		// TODO: Warn user on what will happen and where there stack states will be stored
		// augment path to point to our pulumi shimming script
		// This will substitute any calls to pulumi with docker runs instead and add appropriate volume mounts
		// for pulumi state files
		err = os.WriteFile(filepath.Join(shimDir, "pulumi"), []byte(shimScript), fs.ModePerm)
		if err != nil {
			return err
		}
		// TODO: Test on windows
		os.Setenv("PATH", fmt.Sprintf("%s:%s", shimDir, os.Getenv("PATH")))

		// login to pulumi locally
		cmd := exec.Command("pulumi", "whoami")

		err = cmd.Run()
		if err != nil {
			// login to pulumi locally by default
			// TODO: Provide warning here...
			cmd = exec.Command("pulumi", "login", "--local")

			err = cmd.Run()
			if err != nil {
				return err
			}
		}
	} else {
		log.Default().Printf("pulumi found on path, using local pulumi binary\n")
	}

	return nil
}

func (p *pulumiDeployment) load(log output.Progress) (*auto.Stack, error) {
	if err := p.prov.Validate(); err != nil {
		return nil, err
	}

	stackName := p.proj.Name + "-" + p.stackName
	ctx := context.Background()

	aboutData, err := exec.Command("pulumi", "about", "-j").Output()
	if err != nil && strings.Contains(err.Error(), "executable file not found") {
		return nil, errors.WithMessage(err, "please install pulumi from https://www.pulumi.com/docs/get-started/install/")
	}

	// Default to local backend if not already logged in
	about := &pulumiAbout{}

	err = json.Unmarshal([]byte(strings.TrimSpace(string(aboutData))), about)
	if err != nil {
		return nil, errors.WithMessage(err, "please check your installation - https://nitric.io/docs/installation")
	}

	upsertOpts := []auto.LocalWorkspaceOption{
		auto.SecretsProvider("passphrase"),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(p.proj.Name),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			Main:    p.proj.Dir,
		}),
	}

	if about.Backend == nil {
		upsertOpts = append(upsertOpts, auto.EnvVars(map[string]string{
			"PULUMI_BACKEND_URL": "file://~",
		}))
	}

	s, err := auto.UpsertStackInlineSource(ctx, stackName, p.proj.Name, p.prov.Deploy, upsertOpts...)
	if err != nil {
		return nil, errors.WithMessage(err, "UpsertStackInlineSource")
	}

	// Cancel all previously running stacks
	if p.opts.Force {
		// It will only return an error if the stack isn't in an updating state, so we can just ignore it
		_ = s.Cancel(ctx)
	}

	// https://github.com/pulumi/pulumi/issues/9782
	buildkitInstall := exec.Command("pulumi", "plugin", "install", "resource", "docker-buildkit", "0.1.17", "--server", "https://github.com/MaterializeInc/pulumi-docker-buildkit/releases/download/v0.1.17")

	out, err := buildkitInstall.CombinedOutput()
	if err != nil {
		pl := &common.Plugin{Name: "docker-buildkit", Version: "0.1.17"}
		return nil, errors.WithMessagef(err, "InstallPlugin %s \n%s", pl.String(), out)
	}

	for _, plug := range p.prov.Plugins() {
		log.Busyf("Installing Pulumi plugin %s:%s", plug.Name, plug.Version)

		err = retry.Do(func() error {
			return s.Workspace().InstallPlugin(ctx, plug.Name, plug.Version)
		}, retry.Attempts(3), retry.Delay(time.Second))
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

	pLog := &pulumiLogger{
		Progress: log,
	}

	res, err := s.Up(context.Background(), updateLoggingOpts(pLog)...)
	summary := &types.Summary{Resources: pLog.resources}

	d := &types.Deployment{
		Summary:      summary,
		ApiEndpoints: map[string]string{},
	}

	if err != nil {
		return d, errors.WithMessage(err, "Updating pulumi stack "+res.Summary.Message)
	}

	defer p.prov.CleanUp()

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

	stackName := p.proj.Name + "-" + p.stackName
	result := []stackSummary{}

	for _, st := range sl {
		if strings.HasPrefix(st.Name, stackName) {
			stackListOutput := stackSummary{
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

func (a *pulumiDeployment) Down(log output.Progress) (*types.Summary, error) {
	pLog := &pulumiLogger{
		Progress: log,
	}

	s, err := a.load(log)
	if err != nil {
		return nil, err
	}

	res, err := s.Destroy(context.Background(), destroyLoggingOpts(pLog)...)
	summary := &types.Summary{
		Resources: pLog.resources,
	}

	if err != nil {
		return summary, errors.WithMessage(err, res.Summary.Message)
	}

	return summary, nil
}
