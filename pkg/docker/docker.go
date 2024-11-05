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

package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/view/tui"
)

type Docker struct {
	*client.Client
	// logger ContainerLogger
}

func VerifyDockerIsAvailable() error {
	// Create a new Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("error creating Docker client: %w", err)
	}

	// Ensure the client is closed when the function exits
	defer func() {
		if closeErr := cli.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	// Perform a Docker operation to verify availability
	if _, pingErr := cli.Ping(context.Background()); pingErr != nil {
		return fmt.Errorf("Docker daemon is not running, please start the docker daemon and try again")
	}

	return nil
}

func New() (*Docker, error) {
	if err := VerifyDockerIsAvailable(); err != nil {
		return nil, err
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Docker{Client: dockerClient}, err
}

var builderLock = sync.Mutex{}

type BuildxBuilder struct {
	Name string
}

// Create a known nitric container builder to allow custom cache configuration
func (d *Docker) createBuildxBuilder() (*BuildxBuilder, error) {
	builderLock.Lock()
	defer builderLock.Unlock() // Create a known fixed nitric builder to allow caching

	builderName := "nitric"

	cmd := exec.Command("docker", "buildx", "create", "--name", builderName, "--bootstrap", "--driver=docker-container", "--node", "nitric0")

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &BuildxBuilder{Name: builderName}, nil
}

func (d *Docker) Build(dockerfile, srcPath, imageTag string, buildArgs map[string]string, excludes []string, buildLogger io.Writer) error {
	// If docker is available, create a buildx builder
	var builder *BuildxBuilder

	if err := tui.DockerAvailable(); err == nil {
		var err error

		builder, err = d.createBuildxBuilder()
		if err != nil {
			return err
		}
	}

	// write a temporary dockerignore file
	ignoreFile, err := os.Create(fmt.Sprintf("%s.dockerignore", dockerfile))
	if err != nil {
		return err
	}

	_, err = ignoreFile.Write([]byte(strings.Join(excludes, "\n")))
	if err != nil {
		return err
	}

	err = ignoreFile.Close()
	if err != nil {
		return err
	}

	defer func() {
		os.Remove(ignoreFile.Name())
	}()

	buildArgsCmd := make([]string, 0)
	for k, v := range buildArgs {
		buildArgsCmd = append(buildArgsCmd, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	args := []string{
		"buildx", "build", srcPath, "-f", dockerfile, "-t", imageTag, "--load", "--platform", "linux/amd64",
	}
	// Podman doesn't support builder containers
	if builder != nil {
		args = append(args, fmt.Sprintf("--builder=%s", builder.Name))
	}

	args = append(args, buildArgsCmd...)

	cacheTo := ""
	cacheFrom := ""

	dockerBuildCache := os.Getenv("DOCKER_BUILD_CACHE")
	if dockerBuildCache != "" {
		imageCache := filepath.Join(dockerBuildCache, imageTag)

		cacheTo = fmt.Sprintf("--cache-to=type=local,dest=%s", imageCache)
		cacheFrom = fmt.Sprintf("--cache-from=type=local,src=%s", imageCache)
	}

	dockerBuildCacheDest := os.Getenv("DOCKER_BUILD_CACHE_DEST")
	if dockerBuildCacheDest != "" {
		imageCache := filepath.Join(dockerBuildCacheDest, imageTag)

		cacheTo = fmt.Sprintf("--cache-to=type=local,dest=%s", imageCache)
	}

	dockerBuildCacheSrc := os.Getenv("DOCKER_BUILD_CACHE_SRC")
	if dockerBuildCacheSrc != "" {
		imageCache := filepath.Join(dockerBuildCacheSrc, imageTag)

		cacheFrom = fmt.Sprintf("--cache-from=type=local,src=%s", imageCache)
	}

	if cacheTo != "" {
		args = append(args, cacheTo)
	}

	if cacheFrom != "" {
		args = append(args, cacheFrom)
	}

	// The args should be compatible with either docker or podman
	baseCommand := "docker"

	if err := tui.DockerAvailable(); err != nil {
		if err := tui.PodmanAvailable(); err == nil {
			baseCommand = "podman"
		} else {
			return errors.New("Docker or Podman is required, see https://docs.docker.com/engine/install/ for docker installation instructions")
		}
	}

	cmd := exec.Command(baseCommand, args...)

	if buildLogger == nil {
		buildLogger = io.Discard
	}

	cmd.Stdout = buildLogger
	cmd.Stderr = buildLogger

	return cmd.Run()
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

		err := json.Unmarshal([]byte(lastLine), line)
		if err != nil {
			return err
		}

		text := strings.TrimRightFunc(line.Stream, unicode.IsSpace)

		if len(text) > 0 {
			log.Default().Println(text)
		}
	}

	errLine := &ErrorLine{}

	err := json.Unmarshal([]byte(lastLine), errLine)
	if err != nil {
		return err
	}

	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	return scanner.Err()
}

// func (d *Docker) ListImages(stackName, containerName string) ([]Image, error) {
// 	opts := types.ImageListOptions{Filters: filters.NewArgs()}
// 	opts.Filters.Add("reference", fmt.Sprintf("%s-%s-*", stackName, containerName))

// 	imageSummaries, err := d.Client.ImageList(context.Background(), opts)
// 	if err != nil {
// 		return nil, err
// 	}

// 	imgs := []Image{}

// 	for _, i := range imageSummaries {
// 		nameParts := strings.Split(i.RepoTags[0], ":")
// 		id := strings.Split(i.ID, ":")[1][0:12]

// 		imgs = append(imgs, Image{
// 			ID:         id,
// 			Repository: nameParts[0],
// 			Tag:        nameParts[1],
// 			CreatedAt:  time.Unix(i.Created, 0).Local().String(),
// 		})
// 	}

// 	return imgs, err
// }

func (d *Docker) ImagePull(rawImage string, opts types.ImagePullOptions) error {
	resp, err := d.Client.ImagePull(context.Background(), rawImage, opts)
	if err != nil {
		return errors.WithMessage(err, "Pull")
	}

	defer resp.Close()

	return print(resp)
}

func (d *Docker) ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (string, error) {
	resp, err := d.Client.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, nil, name)
	if err != nil {
		return "", errors.WithMessage(err, "ContainerCreate")
	}

	return resp.ID, nil
}

func (d *Docker) RemoveByLabel(labels map[string]string) error {
	opts := container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(),
	}

	for name, value := range labels {
		opts.Filters.Add("label", fmt.Sprintf("%s=%s", name, value))
	}

	res, err := d.Client.ContainerList(context.Background(), opts)
	if err != nil {
		return err
	}

	for _, con := range res {
		err = d.Client.ContainerRemove(context.Background(), con.ID, container.RemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}

	return nil
}

// func (d *Docker) Logger(stackPath string) ContainerLogger {
// 	if d.logger != nil {
// 		return d.logger
// 	}

// 	logPath, _ := utils.NewNitricLogFile(stackPath)
// 	d.logger = newSyslog(logPath)

// 	return d.logger
// }

func (d *Docker) Version() string {
	sv, _ := d.Client.ServerVersion(context.Background())
	b, _ := yaml.Marshal(sv)

	return string(b)
}
