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

package sql

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v4"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/nitric/core/pkg/logger"
	sqlpb "github.com/nitrictech/nitric/core/pkg/proto/sql/v1"
)

type LocalSqlServer struct {
	projectName string
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
	_, err = conn.Exec(context.Background(), fmt.Sprintf(`CREATE DATABASE "%s"`, databaseName))
	if err != nil {
		// If the database already exists, don't treat it as an error
		if strings.Contains(err.Error(), "already exists") {
			logger.Debugf("Database %s already exists", databaseName)
		} else {
			return "", err
		}
	}

	// TODO: Run migrations/seeds if necessary

	// Return the connection string of the new database
	return fmt.Sprintf("postgresql://postgres:localsecret@localhost:%d/%s?sslmode=disable", port, databaseName), nil
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

	// create a persistent volume for the database
	volume, err := dockerClient.VolumeCreate(context.Background(), volume.VolumeCreateBody{
		Driver: "local",
		Name:   fmt.Sprintf("%s-local-sql", l.projectName),
	})
	if err != nil {
		// FIXME: Use error container type to validate here
		if !strings.Contains(err.Error(), "name already in use") {
			log.Fatalf("Failed to create volume: %v", err)
		}
	}

	newLis, err := netx.GetNextListener(netx.MinPort(5432))
	if err != nil {
		return err
	}

	freeport := newLis.Addr().(*net.TCPAddr).Port

	_ = newLis.Close()

	l.containerId, err = dockerClient.ContainerCreate(&container.Config{
		Image: "postgres",
		Env: []string{
			"POSTGRES_PASSWORD=localsecret",
			"PGDATA=/var/lib/postgresql/data/pgdata",
		},
	}, &container.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volume.Name,
				Target: "/var/lib/postgresql/data",
			},
		},
		PortBindings: map[nat.Port][]nat.PortBinding{
			"5432/tcp": {
				{
					HostPort: fmt.Sprint(freeport),
				},
			},
		},
	}, nil, fmt.Sprintf("nitric-%s-local-sql", l.projectName))
	if err != nil {
		return err
	}

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

// create a function that will execute a query on the local database
func (l *LocalSqlServer) Query(ctx context.Context, connectionString string, query string) ([]*orderedmap.OrderedMap[string, any], error) {
	// Connect to the PostgreSQL instance using the provided connection string
	conn, err := pgx.Connect(ctx, connectionString)
	if err != nil {
		return nil, err
	}

	defer conn.Close(ctx)

	// Begin transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Split commands from string
	commands := strings.Split(query, ";")

	results := []*orderedmap.OrderedMap[string, any]{}

	// Execute each command
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}

		rows, err := tx.Query(ctx, command)
		if err != nil {
			_ = tx.Rollback(ctx)

			return nil, err
		}

		if rows.Next() {
			// Process the query results
			results, err = processRows(rows)
			rows.Close()

			if err != nil {
				_ = tx.Rollback(ctx)

				return nil, err
			}
		} else {
			rows.Close()
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return results, nil
}

func NewLocalSqlServer(projectName string) (*LocalSqlServer, error) {
	localSql := &LocalSqlServer{
		projectName: projectName,
	}

	err := localSql.start()
	if err != nil {
		return nil, err
	}

	return localSql, nil
}

func processRows(rows pgx.Rows) ([]*orderedmap.OrderedMap[string, any], error) {
	fieldDescriptions := rows.FieldDescriptions()
	numColumns := len(fieldDescriptions)

	results := []*orderedmap.OrderedMap[string, any]{}

	for {
		values := make([]interface{}, numColumns)
		valuePointers := make([]interface{}, numColumns)

		for i := range values {
			valuePointers[i] = &values[i]
		}

		err := rows.Scan(valuePointers...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := orderedmap.New[string, any]()

		for i, val := range values {
			row.Set(string(fieldDescriptions[i].Name), val)
		}

		results = append(results, row)

		if !rows.Next() {
			break
		}
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row iteration failed: %w", rows.Err())
	}

	return results, nil
}
