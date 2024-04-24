package sql

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v4"

	"github.com/nitrictech/cli/pkg/docker"
	sqlpb "github.com/nitrictech/nitric/core/pkg/proto/sql/v1"
)

type LocalSqlServer struct {
	containerId string
	sqlpb.UnimplementedSqlServer
}

var _ sqlpb.SqlServer = (*LocalSqlServer)(nil)

func ensureDatabaseExists(databaseName string) (string, error) {
	port := 5432
	// Ensure the database exists
	// Connect to the PostgreSQL instance
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("user=postgres password=localsecret host=localhost port=%d dbname=postgres sslmode=disable", port))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	// Create the new database
	_, err = conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", databaseName))
	if err != nil {
		// If the database already exists, don't treat it as an error
		if strings.Contains(err.Error(), "already exists") {
			log.Fatal(err)
		} else {
			return "", err
		}
	}

	// TODO: Run migrations/seeds if necessary

	// Return the connection string of the new database
	return fmt.Sprintf("user=postgres password=localsecret host=localhost port=%d dbname=%s sslmode=disable", port, databaseName), nil
}

func (l *LocalSqlServer) start() error {
	if l.containerId != "" {
		// Already started, no-op
		return nil
	}

	// Start the local database container
	dockerClient, err := docker.New()
	if err != nil {
		return err
	}

	err = dockerClient.ImagePull("postgres:latest", types.ImagePullOptions{
		All: false,
	})
	if err != nil {
		return err
	}

	l.containerId, err = dockerClient.ContainerCreate(&container.Config{
		Image: "postgres",
		Env: []string{
			"POSTGRES_PASSWORD=localsecret",
			"PGDATA=/var/lib/postgresql/data/pgdata",
		},
		Volumes: map[string]struct{}{
			"./.nitric/local-sql:/var/lib/postgresql/data": {},
		},
	}, &container.HostConfig{
		AutoRemove: true,
		PortBindings: map[nat.Port][]nat.PortBinding{
			// TODO: Randomize port number to allow multiple starts
			"5432/tcp": {
				{
					HostPort: "5432",
				},
			},
		},
		// TODO: Randomize instance name to allow multiple starts
	}, nil, "nitric-local-sql")

	if err != nil {
		return err
	}

	// --name some-postgres \
	// -e POSTGRES_PASSWORD=mysecretpassword \
	// -e PGDATA=/var/lib/postgresql/data/pgdata \
	// -v /custom/mount:/var/lib/postgresql/data \

	return dockerClient.ContainerStart(context.Background(), l.containerId, types.ContainerStartOptions{})
}

func (l *LocalSqlServer) Stop() error {
	dockerClient, err := docker.New()
	if err != nil {
		return err
	}

	err = dockerClient.ContainerStop(context.Background(), l.containerId, nil)
	if err != nil {
		return err
	}

	l.containerId = ""
	return nil
}

func (l *LocalSqlServer) ConnectionString(ctx context.Context, req *sqlpb.SqlConnectionStringRequest) (*sqlpb.SqlConnectionStringResponse, error) {
	connectionString, err := ensureDatabaseExists(req.DatabaseName)
	if err != nil {
		return nil, err
	}

	// We can lazily create a new database instance here and return the connection information directly
	return &sqlpb.SqlConnectionStringResponse{
		ConnectionString: connectionString,
	}, nil
}

func NewLocalSqlServer() (*LocalSqlServer, error) {
	localSql := &LocalSqlServer{}

	err := localSql.start()
	if err != nil {
		return nil, err
	}

	return localSql, nil
}
