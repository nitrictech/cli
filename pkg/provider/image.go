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
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/nitrictech/cli/pkg/docker"
)

type ProviderImage struct {
	// unique name/reference for the image - registry-host[:port]/][username/]repository[:tag]
	imageName string

	containerId string
}

func (pi *ProviderImage) Install() error {
	d, err := docker.New()
	if err != nil {
		return err
	}

	_, _, err = d.ImageInspectWithRaw(context.Background(), pi.imageName)
	if err == nil {
		return nil
	}

	if !client.IsErrNotFound(err) {
		return fmt.Errorf("error inspecting image: %w", err)
	}

	fmt.Printf("provider image %s not found locally, pulling\n", pi.imageName)

	err = d.ImagePull(pi.imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("error pulling image: %w", err)
	}

	return nil
}

func (pi *ProviderImage) Start(options *StartOptions) (string, error) {
	// Start a new container
	fmt.Printf("Starting container: %s\n", pi.imageName)

	client, err := docker.New()
	if err != nil {
		return "", err
	}

	fmt.Printf("Creating container: %s\n", pi.imageName)

	env := []string{}
	for k, v := range options.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	workspacePath, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("error starting provider: %w", err)
	}

	const providerPort = "50051"

	hostConfig := &container.HostConfig{
		AutoRemove: false,
		Binds: []string{
			fmt.Sprintf("%s:/workspace", workspacePath),
			// Bind the docker sock for docker in docker
			"/var/run/docker.sock:/var/run/docker.sock",
		},
		PortBindings: nat.PortMap{
			// TODO: Make the port dynamic
			nat.Port(providerPort): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: providerPort,
				},
			},
		},
	}

	containerConfig := &container.Config{
		Image: pi.imageName,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(providerPort): struct{}{},
		},
	}

	fmt.Printf("Creating container: %s\n", pi.imageName)

	if pi.containerId == "" {
		pi.containerId, err = client.ContainerCreate(containerConfig, hostConfig, nil, "")
		if err != nil {
			return "", err
		}
	}

	fmt.Printf("Starting container: %s\n", pi.containerId)

	err = client.ContainerStart(context.Background(), pi.containerId, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	// TODO: Split stdout and stderr
	stdOutAtt, err := client.ContainerAttach(context.Background(), pi.containerId, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return "", err
	}

	go func() {
		defer stdOutAtt.Close()

		_, err := io.Copy(writerFunc(func(p []byte) (n int, err error) {
			options.StdOut <- string(p)
			return len(p), nil
		}), stdOutAtt.Reader)
		if err != nil {
			options.StdErr <- fmt.Sprintf("error reading container stdout: %s", err)
		}
	}()

	return fmt.Sprintf("127.0.0.1:%s", providerPort), nil
}

type writerFunc func(p []byte) (n int, err error)

func (wf writerFunc) Write(p []byte) (n int, err error) {
	return wf(p)
}

func (pi *ProviderImage) Stop() error {
	client, err := docker.New()
	if err != nil {
		return fmt.Errorf("error creating Docker client: %w", err)
	}

	// Stop the container
	return client.ContainerStop(context.Background(), pi.containerId, nil)
}

func (pi *ProviderImage) Uninstall() error {
	client, err := docker.New()
	if err != nil {
		return err
	}

	// Remove the container
	err = client.ContainerRemove(context.Background(), pi.containerId, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// Remove the image
	_, err = client.ImageRemove(context.Background(), pi.imageName, types.ImageRemoveOptions{})

	return err
}

// NewImageProvider - Returns a new image provider instance based on the given image name [registry-host[:port]/][username/]repository[:tag]
func NewImageProvider(imageName string) *ProviderImage {
	return &ProviderImage{
		imageName: imageName,
	}
}
