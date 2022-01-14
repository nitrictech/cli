package run

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/pkg/errors"
)

type MinioServer struct {
	dir     string
	name    string
	cid     string
	ce      containerengine.ContainerEngine
	apiPort int
}

const (
	minioImage       = "minio/minio:latest"
	devVolume        = "/nitric/"
	runDir           = "./.nitric/run"
	runPerm          = os.ModePerm // NOTE: octal notation is important here!!!
	LabelRunID       = "io.nitric-run-id"
	LabelStackName   = "io.nitric-stack"
	LabelType        = "io.nitric-type"
	minioPort        = 9000
	minioConsolePort = 9001 // TODO: Determine if we would like to expose the console

)

// StartMinio -
func (m *MinioServer) Start() error {
	runDir, err := filepath.Abs(m.dir)

	if err != nil {
		return err
	}

	os.MkdirAll(runDir, runPerm)

	// TODO: Create new buckets on the fly
	//for bName := range l.s.Buckets {
	//	os.MkdirAll(path.Join(nitricRunDir, "buckets", bName), runPerm)
	//}
	ports, err := freeport.Take(2)
	if err != nil {
		return errors.WithMessage(err, "freeport.Take")
	}

	port := uint16(ports[0])
	consolePort := uint16(ports[1])

	err = m.ce.Pull(minioImage)
	if err != nil {
		return err
	}

	cID, err := m.ce.ContainerCreate(&container.Config{
		Image: minioImage,
		Cmd:   []string{"minio", "server", "/nitric/buckets", "--console-address", fmt.Sprintf(":%d", consolePort)},
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", minioPort)):        struct{}{},
			nat.Port(fmt.Sprintf("%d/tcp", minioConsolePort)): struct{}{},
		},
	}, &container.HostConfig{
		AutoRemove: true,
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", minioPort)): []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", port),
				},
			},
			nat.Port(fmt.Sprintf("%d/tcp", minioConsolePort)): []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", consolePort),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Source: runDir,
				Type:   mount.TypeBind,
				Target: devVolume,
			},
		},
		NetworkMode: container.NetworkMode("bridge"),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}, "minio-"+m.name)
	if err != nil {
		return err
	}
	m.cid = cID
	m.apiPort = minioPort

	return m.ce.Start(cID)
}

func (m *MinioServer) GetApiPort() int {
	return m.apiPort
}

func (m *MinioServer) Stop() error {
	defaultTimeout := time.Duration(5) * time.Second
	return m.ce.Stop(m.cid, &defaultTimeout)
}

func NewMinio(dir string, name string) (*MinioServer, error) {
	ce, err := containerengine.Discover()

	if err != nil {
		return nil, err
	}

	return &MinioServer{
		ce:   ce,
		dir:  dir,
		name: name,
	}, nil
}
