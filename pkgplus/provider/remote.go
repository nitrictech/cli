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

package provider

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	deploy "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
)

type DeploymentClient struct {
	address     string
	interactive bool
}

func (p *DeploymentClient) dialConnection() (*grpc.ClientConn, error) {
	if p.address == "" {
		p.address = "127.0.0.1:50051"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, p.address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (p *DeploymentClient) Up(deploymentRequest *deploy.DeploymentUpRequest) (<-chan *deploy.DeploymentUpEvent, <-chan error) {
	eventChan := make(chan *deploy.DeploymentUpEvent)
	errorChan := make(chan error)
	go func() {
		defer close(eventChan)

		conn, err := p.dialConnection()
		if err != nil {
			errorChan <- fmt.Errorf("failed to connect to provider: %w", err)
			return
		}
		defer conn.Close()

		client := deploy.NewDeploymentClient(conn)

		op, err := client.Up(context.Background(), deploymentRequest)
		if err != nil {
			errorChan <- err
			return
		}

		for {
			evt, err := op.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errorChan <- err
				} else {
					fmt.Println("got EOF")
				}
				break
			}

			eventChan <- evt
		}
	}()

	return eventChan, errorChan
}

func (p *DeploymentClient) Down(deploymentRequest *deploy.DeploymentDownRequest) (<-chan *deploy.DeploymentDownEvent, <-chan error) {
	eventChan := make(chan *deploy.DeploymentDownEvent)
	errorChan := make(chan error)

	go func() {
		defer close(eventChan)

		conn, err := p.dialConnection()
		if err != nil {
			errorChan <- fmt.Errorf("failed to connect to provider: %w", err)
			return
		}
		defer conn.Close()

		client := deploy.NewDeploymentClient(conn)

		op, err := client.Down(context.Background(), deploymentRequest)
		if err != nil {
			errorChan <- err
			return
		}

		for {
			evt, err := op.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errorChan <- err
				}
				break
			}

			eventChan <- evt
		}
	}()

	return eventChan, errorChan
}

func NewDeploymentClient(address string, interactive bool) *DeploymentClient {
	return &DeploymentClient{
		address:     address,
		interactive: interactive,
	}
}
