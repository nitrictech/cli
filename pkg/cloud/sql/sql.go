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
	"encoding/hex"
	"fmt"
	"maps"
	"net"
	"net/netip"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/afero"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/nitrictech/cli/pkg/cloud/resources"
	"github.com/nitrictech/cli/pkg/collector"
	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/exit"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/cli/pkg/project/migrations"
	"github.com/nitrictech/nitric/core/pkg/env"
	"github.com/nitrictech/nitric/core/pkg/logger"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	sqlpb "github.com/nitrictech/nitric/core/pkg/proto/sql/v1"
)

type DatabaseStatus string

const (
	DatabaseStatusStarting           DatabaseStatus = "starting"
	DatabaseStatusBuildingMigrations DatabaseStatus = "building migrations"
	DatabaseStatusApplyingMigrations DatabaseStatus = "applying migrations"
	DatabaseStatusActive             DatabaseStatus = "active"
)

type DatabaseState struct {
	*migrations.LocalMigration

	Status           string
	ResourceRegister *resources.ResourceRegister[resourcespb.SqlDatabaseResource]
	ConnectionString string
}

type (
	DatabaseName = string
	State        = map[DatabaseName]*DatabaseState
)

type LocalSqlServer struct {
	projectName string
	containerId string
	port        int
	State       State
	sqlpb.UnimplementedSqlServer

	bus EventBus.Bus
}

var _ sqlpb.SqlServer = (*LocalSqlServer)(nil)

const localDatabaseTopic = "local_database"

func (l *LocalSqlServer) SubscribeToState(subscriberFunction func(State)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localDatabaseTopic, subscriberFunction)
}

func (l *LocalSqlServer) Publish(state State) {
	l.bus.Publish(localDatabaseTopic, maps.Clone(state))
}

func (l *LocalSqlServer) GetState() State {
	return maps.Clone(l.State)
}

func (l *LocalSqlServer) ensureDatabaseExists(databaseName string) (string, error) {
	// Ensure the database exists
	// Connect to the PostgreSQL instance
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("user=postgres password=localsecret host=localhost port=%d dbname=postgres sslmode=disable", l.port))
	if err != nil {
		exit.GetExitService().Exit(err)
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

	// Return the connection string of the new database
	return fmt.Sprintf("postgresql://postgres:localsecret@localhost:%d/%s?sslmode=disable", l.port, databaseName), nil
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
	volume, err := dockerClient.VolumeCreate(context.Background(), volume.CreateOptions{
		Driver: "local",
		Name:   fmt.Sprintf("%s-local-sql", l.projectName),
	})
	if err != nil {
		// FIXME: Use error container type to validate here
		if !strings.Contains(err.Error(), "name already in use") {
			exit.GetExitService().Exit(fmt.Errorf("failed to create volume: %w", err))
		}
	}

	newLis, err := netx.GetNextListener(netx.MinPort(5432))
	if err != nil {
		return err
	}

	freeport := newLis.Addr().(*net.TCPAddr).Port

	l.port = freeport

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

	return dockerClient.ContainerStart(context.Background(), l.containerId, container.StartOptions{})
}

func (l *LocalSqlServer) Stop() error {
	dockerClient, err := docker.New()
	if err != nil {
		return err
	}

	err = dockerClient.ContainerStop(context.Background(), l.containerId, container.StopOptions{})
	if err != nil {
		return err
	}

	l.containerId = ""

	return nil
}

func (l *LocalSqlServer) ConnectionString(ctx context.Context, req *sqlpb.SqlConnectionStringRequest) (*sqlpb.SqlConnectionStringResponse, error) {
	connectionString, err := l.ensureDatabaseExists(req.DatabaseName)
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
	commands := SQLSplit(query)

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

func (l *LocalSqlServer) BuildAndRunMigrations(databasesToMigrate map[string]*resourcespb.SqlDatabaseResource) error {
	fs := afero.NewOsFs()

	serviceRequirements := collector.MakeDatabaseServiceRequirements(databasesToMigrate)

	migrationImageContexts, err := collector.GetMigrationImageBuildContexts(serviceRequirements, fs)
	if err != nil {
		exit.GetExitService().Exit(fmt.Errorf("failed to get migration image build contexts: %w", err))
	}

	if len(migrationImageContexts) > 0 {
		updates, err := migrations.BuildMigrationImages(fs, migrationImageContexts)
		if err != nil {
			exit.GetExitService().Exit(fmt.Errorf("failed to build migration images: %w", err))
		}

		// wait for updates to complete
		for update := range updates {
			if update.Err != nil {
				exit.GetExitService().Exit(fmt.Errorf("failed to build migration image: %w", update.Err))
			}
		}

		// run the migrations
		localMigrations := []migrations.LocalMigration{}

		// Update the migration status
		for dbName := range databasesToMigrate {
			l.State[dbName].Status = string(DatabaseStatusApplyingMigrations)
		}

		l.Publish(l.State)

		for dbName := range databasesToMigrate {
			migration, ok := l.State[dbName]

			if ok {
				localMigrations = append(localMigrations, *migration.LocalMigration)
			}
		}

		err = migrations.RunMigrations(localMigrations)
		if err != nil {
			exit.GetExitService().Exit(fmt.Errorf("failed to run migrations: %w", err))
		}

		// Update the status to running
		for dbName := range databasesToMigrate {
			l.State[dbName].Status = string(DatabaseStatusActive)
		}

		l.Publish(l.State)
	}

	return nil
}

func (l *LocalSqlServer) HandleUpdates(lrs resources.LocalResourcesState) {
	databasesToMigrate := make(map[string]*resourcespb.SqlDatabaseResource)

	// Check for new databases to migrate
	for dbName, r := range lrs.SqlDatabases.GetAll() {
		_, ok := l.State[dbName]

		connectionString, err := l.ensureDatabaseExists(dbName)
		if err != nil {
			exit.GetExitService().Exit(fmt.Errorf("failed to ensure database exists: %w", err))
		}

		if !ok {
			l.State[dbName] = &DatabaseState{
				Status:           string(DatabaseStatusStarting),
				ResourceRegister: r,
				ConnectionString: connectionString,
			}

			l.Publish(l.State)
		}

		migrationPath := r.Resource.Migrations.GetMigrationsPath()

		if migrationPath != "" && l.State[dbName].LocalMigration == nil {
			dockerHost := "host.docker.internal"

			if goruntime.GOOS == "linux" {
				host := env.GetEnv("NITRIC_DOCKER_HOST", "172.17.0.1")

				dockerHost = host.String()
			}

			l.State[dbName].Status = string(DatabaseStatusBuildingMigrations)
			l.State[dbName].LocalMigration = &migrations.LocalMigration{
				DatabaseName: dbName,
				// Replace localhost with host.docker.internal to allow the container to connect to the host
				ConnectionString: strings.Replace(connectionString, "localhost", dockerHost, 1),
			}

			databasesToMigrate[dbName] = r.Resource

			l.Publish(l.State)
		} else {
			l.State[dbName].Status = string(DatabaseStatusActive)
			l.Publish(l.State)
		}
	}

	if len(databasesToMigrate) > 0 {
		err := l.BuildAndRunMigrations(databasesToMigrate)
		if err != nil {
			exit.GetExitService().Exit(fmt.Errorf("failed to build and run migrations: %w", err))
		}
	}
}

func NewLocalSqlServer(projectName string, localResources *resources.LocalResourcesService) (*LocalSqlServer, error) {
	localSql := &LocalSqlServer{
		projectName: projectName,
		State:       make(State),
		bus:         EventBus.New(),
	}

	err := localSql.start()
	if err != nil {
		return nil, err
	}

	// subscribe to local resources for migrations
	localResources.SubscribeToState(localSql.HandleUpdates)

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
			// format values if necessary
			switch v := val.(type) {
			case time.Time:
				if v.UTC().Hour() == 0 && v.UTC().Minute() == 0 && v.UTC().Second() == 0 {
					val = v.Format("2006-01-02")
				} else {
					val = v.Format("2006-01-02 15:04:05")
				}
			case netip.Prefix:
				val = v.Addr().String()
			case net.HardwareAddr:
				val = v.String()
			case pgtype.Interval:
				val = formatInterval(v)
			case pgtype.Bits:
				var result string
				for _, b := range v.Bytes {
					result += fmt.Sprintf("%08b", b)
				}

				val = result
			case []uint8:
				val = fmt.Sprintf("\\x%s", hex.EncodeToString(v))
			case [16]uint8:
				u, err := uuid.FromBytes(v[:])
				if err != nil {
					return nil, fmt.Errorf("failed to parse UUID: %w", err)
				}

				val = u.String()
			}

			row.Set(fieldDescriptions[i].Name, val)
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

func formatInterval(interval pgtype.Interval) string {
	years := interval.Months / 12
	months := interval.Months % 12
	days := interval.Days

	// Calculate hours, minutes, and seconds from microseconds
	totalSeconds := interval.Microseconds / 1e6
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	parts := []string{}
	if years != 0 {
		parts = append(parts, fmt.Sprintf("%d year%s", years, pluralSuffix(years)))
	}

	if months != 0 {
		parts = append(parts, fmt.Sprintf("%d mon%s", months, pluralSuffix(months)))
	}

	if days != 0 {
		parts = append(parts, fmt.Sprintf("%d day%s", days, pluralSuffix(days)))
	}

	if hours != 0 || minutes != 0 || seconds != 0 {
		parts = append(parts, fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds))
	}

	return strings.Join(parts, " ")
}

func pluralSuffix(value int32) string {
	if value == 1 {
		return ""
	}

	return "s"
}
