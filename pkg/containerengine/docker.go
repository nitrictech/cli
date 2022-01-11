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

package containerengine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/jhoonb/archivex"
	"github.com/pkg/errors"
)

type docker struct {
	cli *client.Client
}

var _ ContainerEngine = &docker{}

func newDocker() (ContainerEngine, error) {
	cmd := exec.Command("docker", "--version")
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	cmd = exec.Command("docker", "ps")
	err = cmd.Run()
	if err != nil {
		fmt.Println("docker daemon not running, please start it..")
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &docker{cli: cli}, err
}

func (d *docker) Build(dockerfile, srcPath, imageTag string, buildArgs map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout())
	defer cancel()

	tar := new(archivex.TarFile)
	dockerBuildContext := bytes.Buffer{}
	err := tar.CreateWriter("src.tar", &dockerBuildContext)
	if err != nil {
		return err
	}
	err = tar.AddAll(srcPath, false)
	if err != nil {
		return err
	}
	if strings.Contains(dockerfile, "/tmp") {
		// copy the generated dockerfile into the tar.
		df, err := os.Open(dockerfile)
		if err != nil {
			return err
		}
		s, err := os.Stat(dockerfile)
		if err != nil {
			return err
		}
		err = tar.Add(s.Name(), df, s)
		if err != nil {
			return err
		}
		dockerfile = s.Name()
	}
	tar.Close()
	opts := types.ImageBuildOptions{
		SuppressOutput: false,
		Dockerfile:     dockerfile,
		Tags:           []string{imageTag},
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
	}
	res, err := d.cli.ImageBuild(ctx, &dockerBuildContext, opts)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return print(res.Body)
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

type Line struct {
	Stream string `json:"stream"`
	Status string `json:"status"`
}

func print(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		line := &Line{}
		json.Unmarshal([]byte(lastLine), line)
		if len(line.Stream) > 0 {
			fmt.Print(line.Stream)
		}
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	return scanner.Err()
}

func (d *docker) ListImages(stackName, containerName string) ([]Image, error) {
	opts := types.ImageListOptions{}
	opts.Filters.Add("reference", fmt.Sprintf("localhost/%s-%s-*", stackName, containerName))
	imageSummaries, err := d.cli.ImageList(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	imgs := []Image{}
	for _, i := range imageSummaries {
		nameParts := strings.Split(i.ID, ":")
		imgs = append(imgs, Image{
			ID:         i.ID,
			Repository: nameParts[0],
			Tag:        nameParts[1],
			CreatedAt:  time.Unix(i.Created, 0).Local().String(),
		})
	}
	return imgs, err
}

func (d *docker) Pull(rawImage string) error {
	resp, err := d.cli.ImagePull(context.Background(), rawImage, types.ImagePullOptions{})
	if err != nil {
		return errors.WithMessage(err, "Pull")
	}
	defer resp.Close()
	print(resp)
	return nil
}

func (d *docker) NetworkCreate(name string) error {
	_, err := d.cli.NetworkInspect(context.Background(), name, types.NetworkInspectOptions{})
	if err == nil {
		// it already exists, no need to create.
		return nil
	}
	_, err = d.cli.NetworkCreate(context.Background(), name, types.NetworkCreate{})
	return err
}

func (d *docker) ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (string, error) {
	resp, err := d.cli.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, nil, name)
	if err != nil {
		return "", errors.WithMessage(err, "ContainerCreate")
	}
	return resp.ID, nil
}

func (d *docker) Start(nameOrID string) error {
	return d.cli.ContainerStart(context.Background(), nameOrID, types.ContainerStartOptions{})
}

func (d *docker) CopyFromArchive(nameOrID string, path string, reader io.Reader) error {
	return d.cli.CopyToContainer(context.Background(), nameOrID, path, reader, types.CopyToContainerOptions{})
}

func (d *docker) ContainersListByLabel(match map[string]string) ([]types.Container, error) {
	opts := types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(),
	}
	for k, v := range match {
		opts.Filters.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	return d.cli.ContainerList(context.Background(), opts)
}

func (d *docker) RemoveByLabel(name, value string) error {
	opts := types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(),
	}
	opts.Filters.Add("label", fmt.Sprintf("%s=%s", name, value))

	res, err := d.cli.ContainerList(context.Background(), opts)
	if err != nil {
		return err
	}
	for _, con := range res {
		err = d.cli.ContainerRemove(context.Background(), con.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *docker) ContainerWait(containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return d.cli.ContainerWait(context.Background(), containerID, condition)
}

func (d *docker) ContainerExec(containerName string, cmd []string, workingDir string) error {
	ctx := context.Background()
	rst, err := d.cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		WorkingDir: workingDir,
		Cmd:        cmd,
	})
	if err != nil {
		return err
	}
	err = d.cli.ContainerExecStart(ctx, rst.ID, types.ExecStartCheck{})
	if err != nil {
		return err
	}

	for {
		res, err := d.cli.ContainerExecInspect(ctx, rst.ID)
		if err != nil {
			return err
		}
		if res.Running {
			continue
		}
		if res.ExitCode == 0 {
			return nil
		}
		return fmt.Errorf("%s %v exited with %d", containerName, cmd, res.ExitCode)
	}
}
