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

// +build linux

package containerengine

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	buildahDefine "github.com/containers/buildah/define"
	"github.com/containers/podman/v3/pkg/bindings"
	"github.com/containers/podman/v3/pkg/bindings/containers"
	"github.com/containers/podman/v3/pkg/bindings/images"
	"github.com/containers/podman/v3/pkg/bindings/network"
	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/specgen"
)

type podman struct {
	ctx context.Context
}

var _ ContainerEngine = &podman{}

func newPodman() (ContainerEngine, error) {
	cmd := exec.Command("podman", "--version")
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	cmd = exec.Command("systemctl", "is-active", "docker")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		fmt.Println("podman.socket not available, starting..")
		cmd = exec.Command("systemctl", "start", "--user", "podman.socket")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return nil, err
		}
	}

	socket := "unix:" + os.Getenv("XDG_RUNTIME_DIR") + "/podman/podman.sock"
	connText, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		fmt.Printf("Error connecting to %s : %v\n", socket, err)
	}

	return &podman{ctx: connText}, err
}

func (p *podman) Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error {
	buildArgs["PROVIDER"] = provider
	opts := buildahDefine.BuildOptions{
		ContextDirectory: path,
		Output:           imageTag,
		Args:             buildArgs,
		In:               os.Stdin,
		Out:              os.Stdout,
		Err:              os.Stderr,
	}
	ctx, cancel := context.WithTimeout(p.ctx, buildTimeout())
	defer cancel()

	report, err := images.Build(ctx, []string{dockerfile}, entities.BuildOptions{BuildOptions: opts})
	fmt.Println("imageID ", report.ID)
	return err
}

func (p *podman) ListImages(stackName, containerName string) ([]Image, error) {
	opts := &images.ListOptions{
		Filters: map[string][]string{
			"reference": {
				fmt.Sprintf("localhost/%s-%s-*", stackName, containerName),
			},
		},
	}
	imageSummaries, err := images.List(p.ctx, opts)
	if err != nil {
		return nil, err
	}
	imgs := []Image{}
	for _, i := range imageSummaries {
		nameParts := strings.Split(i.Names[0], ":")
		imgs = append(imgs, Image{
			ID:         i.ID,
			Repository: nameParts[0],
			Tag:        nameParts[1],
			CreatedAt:  time.Unix(i.Created, 0).Local().String(),
		})
	}
	return imgs, err
}

func (p *podman) Pull(rawImage string) error {
	_, err := images.Pull(p.ctx, rawImage, nil)
	return err
}

func (p *podman) NetworkCreate(name string) error {
	ok, err := network.Exists(p.ctx, name, nil)
	if err == nil && ok {
		return nil
	}
	_, err = network.Create(p.ctx, &network.CreateOptions{Name: &name})
	return err
}

func (p *podman) CreateWithSpec(s *specgen.SpecGenerator) (string, error) {
	resp, err := containers.CreateWithSpec(p.ctx, s, nil)
	if err != nil {
		return "", err
	}
	return resp.ID, err
}

func (p *podman) Start(nameOrID string) error {
	return containers.Start(p.ctx, nameOrID, nil)
}

func (p *podman) CopyFromArchive(nameOrID string, path string, reader io.Reader) error {
	copyFn, err := containers.CopyFromArchive(p.ctx, nameOrID, path, reader)
	if err != nil {
		return err
	}
	return copyFn()
}

func (p *podman) ContainersListByLabel(match map[string]string) ([]entities.ListContainer, error) {
	labelSelector := []string{}
	t := true
	for k, v := range match {
		labelSelector = append(labelSelector, k+"="+v)
	}
	return containers.List(p.ctx, &containers.ListOptions{
		All:     &t,
		Filters: map[string][]string{"label": labelSelector},
	})
}

func (p *podman) RemoveByLabel(name, value string) error {
	t := true
	cons, err := containers.List(p.ctx, &containers.ListOptions{
		All: &t,
		Filters: map[string][]string{
			"label": {name + "=" + value},
		},
	})
	if err != nil {
		return err
	}
	for _, c := range cons {
		fmt.Println("remove ", c.Names)
		err = containers.Remove(p.ctx, c.ID, &containers.RemoveOptions{Force: &t})
		if err != nil {
			return err
		}
	}
	return nil
}
