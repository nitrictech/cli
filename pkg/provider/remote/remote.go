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

package remote

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider/types"
	deploy "github.com/nitrictech/nitric/core/pkg/api/nitric/deploy/v1"
)

type remoteDeployment struct {
	cfc       types.ConfigFromCode
	stackName string
	provider  string
	conn      *grpc.ClientConn
	ce        containerengine.ContainerEngine
	// Container id populated after a call to Start
	cid string
}

var _ types.Provider = &remoteDeployment{}

func New(cfc types.ConfigFromCode, name, provider string, envMap map[string]string, opts *types.ProviderOpts) (types.Provider, error) {
	ce, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	return &remoteDeployment{
		ce:        ce,
		cfc:       cfc,
		stackName: name,
		provider:  provider,
	}, nil
}

func (p *remoteDeployment) start() error {
	pterm.Info.Println("Starting remote deployment server")

	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	hc := &container.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(homedir, ".aws"),
				Target: "/root/.aws",
			},
		},
		PortBindings: map[nat.Port][]nat.PortBinding{
			"50051/tcp": {},
		},
	}

	cc := &container.Config{
		Image: p.provider,
		Env: []string{
			"PULUMI_CONFIG_PASSPHRASE=set-to-this",
			"PULUMI_BACKEND_URL=file://~",
		},
	}

	pterm.Debug.Print(containerengine.Cli(cc, hc))

	cID, err := p.ce.ContainerCreate(cc, hc, nil, "deployment-server")
	if err != nil {
		return err
	}

	p.cid = cID

	err = p.ce.Start(cID)
	if err != nil {
		return err
	}

	pterm.Info.Println("Connecting to deployment server")
	p.conn, err = grpc.Dial("172.17.0.2:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

	return err
}

func (p *remoteDeployment) stop() error {
	timeout := time.Second * 5
	return p.ce.Stop(p.cid, &timeout)
}

func (p *remoteDeployment) AskAndSave() error {
	return errors.New("not supported on remote deployment servers")
}

func (p *remoteDeployment) SupportedRegions() []string {
	return []string{}
}

func (p *remoteDeployment) List() (interface{}, error) {
	return nil, errors.New("not supported for remote deployments")
}

func (a *remoteDeployment) Preview(log output.Progress) (string, error) {
	return "", errors.New("not supported for remote deployments")
}

func (p *remoteDeployment) Up(log output.Progress) (*types.Deployment, error) {
	err := p.start()
	if err != nil {
		return nil, err
	}

	defer p.stop()

	sc, err := stackConfig(p.cfc.ProjectDir(), p.stackName, p.provider)
	if err != nil {
		return nil, err
	}

	req, err := p.cfc.ToUpRequest()
	if err != nil {
		return nil, err
	}

	for k, v := range sc {
		switch k {
		case "name":
			req.Attributes["x-nitric-stack"] = v.(string)
		default:
			req.Attributes[k] = fmt.Sprintf("%v", v)
		}
	}

	client := deploy.NewDeployServiceClient(p.conn)

	op, err := client.Up(context.Background(), req)
	if err != nil {
		return nil, err
	}

	res := &types.Deployment{
		Summary:      &types.Summary{},
		ApiEndpoints: map[string]string{},
	}

	for {
		evt, err := op.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return res, nil
			}

			return res, err
		}

		switch t := evt.Content.(type) {
		case *deploy.DeployUpEvent_Message:
			log.Debugf(t.Message.Message)
		case *deploy.DeployUpEvent_Result:
			if !t.Result.Success {
				return res, errors.New("failed to deploy")
			}
			/*	for k, v := range res.Outputs {
					if strings.HasPrefix(k, "api:") {
						d.ApiEndpoints[strings.TrimPrefix(k, "api:")] = fmt.Sprint(v.Value)
					}
				}
			*/

			return res, nil
		}
	}
}

func (p *remoteDeployment) Down(log output.Progress) (*types.Summary, error) {
	err := p.start()
	if err != nil {
		return nil, err
	}

	defer p.stop()

	sc, err := stackConfig(p.cfc.ProjectDir(), p.stackName, p.provider)
	if err != nil {
		return nil, err
	}

	req := &deploy.DeployDownRequest{
		Attributes: map[string]string{
			"x-nitric-project": p.cfc.ProjectName(),
		},
	}

	for k, v := range sc {
		switch k {
		case "name":
			req.Attributes["x-nitric-stack"] = v.(string)
		default:
			req.Attributes[k] = fmt.Sprintf("%v", v)
		}
	}

	client := deploy.NewDeployServiceClient(p.conn)

	op, err := client.Down(context.Background(), req)
	if err != nil {
		return nil, err
	}

	res := &types.Summary{}

	for {
		evt, err := op.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return res, nil
			}

			return res, err
		}

		switch t := evt.Content.(type) {
		case *deploy.DeployDownEvent_Message:
			log.Debugf(t.Message.Message)
		case *deploy.DeployDownEvent_Result: // TODO - handle errors
			return res, nil
		}
	}
}
