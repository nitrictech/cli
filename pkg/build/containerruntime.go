package build

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type ContainerRuntime interface {
	Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error
}

func DiscoverContainerRuntime() (ContainerRuntime, error) {
	cmd := exec.Command("podman", "--version")
	err := cmd.Run()
	if err == nil {
		return &podman{}, nil
	}
	cmd = exec.Command("docker", "--version")
	err = cmd.Run()
	if err == nil {
		return &docker{}, nil
	}
	return nil, errors.New("neither podman nor docker found")
}

type podman struct{}

var _ ContainerRuntime = &podman{}

func (p *podman) Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error {
	args := []string{"build", path, "-f", dockerfile, "-t", imageTag, "--progress", "plain", "--build-arg=PROVIDER=" + provider}

	for key, val := range buildArgs {
		args = append(args, "--build-arg="+key+"="+val)
	}
	fmt.Println("podman", args)
	cmd := exec.Command("podman", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type docker struct{}

var _ ContainerRuntime = &docker{}

func (d *docker) Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error {
	args := []string{"build", path, "-f", dockerfile, "-t", imageTag, "--progress", "plain", "--build-arg PROVIDER=" + provider}

	for key, val := range buildArgs {
		args = append(args, "--build-arg "+key+"="+val)
	}
	fmt.Println("docker", args)
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	return cmd.Run()
}
