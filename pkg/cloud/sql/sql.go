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
	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/netx"
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
	DatabaseStatusError              DatabaseStatus = "error"
)

type DatabaseServer struct {
	DatabaseName     string
	Status           string
	ResourceRegister *resources.ResourceRegister[resourcespb.SqlDatabaseResource]
	ConnectionString string
}

type (
	DatabaseName = string
	State        = map[DatabaseName]*DatabaseServer
)

type LocalSqlServer struct {
	projectName string
	containerId string
	port        int
	State       State
	sqlpb.UnimplementedSqlServer

	migrationRunner MigrationRunner

	bus EventBus.Bus
}

type MigrationRunner = func(fs afero.Fs, servers map[string]*DatabaseServer, databasesToMigrate map[string]*resourcespb.SqlDatabaseResource, useBuilder bool) error

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
		return "", err
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
		return err
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

func (l *LocalSqlServer) BuildAndRunMigrations(fs afero.Fs, databasesToMigrate map[string]*resourcespb.SqlDatabaseResource, useBuilder bool) error {
	// Update the migration status
	for dbName := range databasesToMigrate {
		l.State[dbName].Status = string(DatabaseStatusApplyingMigrations)
	}

	l.Publish(l.State)

	servers := map[string]*DatabaseServer{}

	// only run migrations for databases that are keys in databasesToMigrate
	for dbName := range databasesToMigrate {
		servers[dbName] = l.State[dbName]
	}

	err := l.migrationRunner(fs, servers, databasesToMigrate, useBuilder)
	if err != nil {
		return err
	}

	// Update the status to running
	for dbName := range databasesToMigrate {
		l.State[dbName].Status = string(DatabaseStatusActive)
	}

	l.Publish(l.State)

	return err
}

func (l *LocalSqlServer) RegisterDatabases(lrs resources.LocalResourcesState) {
	// reset the state
	l.State = make(State)

	// Check for new databases to migrate
	for dbName, r := range lrs.SqlDatabases.GetAll() {
		_, ok := l.State[dbName]

		if !ok {
			l.State[dbName] = &DatabaseServer{
				Status:           string(DatabaseStatusStarting),
				ResourceRegister: r,
				ConnectionString: "",
			}

			connectionString, err := l.ensureDatabaseExists(dbName)
			if err != nil {
				// Mark database as errored
				l.State[dbName].Status = string(DatabaseStatusError)
			}

			// Update the connection string
			l.State[dbName].ConnectionString = connectionString
			l.State[dbName].Status = string(DatabaseStatusActive)
		}
	}

	l.Publish(l.State)
}

func NewLocalSqlServer(projectName string, localResources *resources.LocalResourcesService, migrationRunner MigrationRunner) (*LocalSqlServer, error) {
	localSql := &LocalSqlServer{
		projectName:     projectName,
		State:           make(State),
		bus:             EventBus.New(),
		migrationRunner: migrationRunner,
	}

	err := localSql.start()
	if err != nil {
		return nil, err
	}

	// subscribe to local resources for migrations
	localResources.SubscribeToState(localSql.RegisterDatabases)

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
