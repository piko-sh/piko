// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/sqldb/sqldb_domain"
	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

type fakeMigrationService struct {
	sqldb_domain.MigrationService
	upErr   error
	upCalls int
}

func (m *fakeMigrationService) Up(context.Context) (int, error) {
	m.upCalls++
	return 0, m.upErr
}

type fakeSeedService struct{ sqldb_domain.SeedService }

type fakeSQLToolchain struct {
	sqldb_domain.ToolchainPort
	generateErr  error
	migrationErr error
	result       *sqldb_dto.GenerationResult
	migrator     *fakeMigrationService
}

func (f *fakeSQLToolchain) Generate(context.Context, sqldb_dto.GenerateRequest) (*sqldb_dto.GenerationResult, error) {
	return f.result, f.generateErr
}

func (f *fakeSQLToolchain) NewMigrationService(*sql.DB, sqldb_dto.DialectConfig, fs.FS, string, sqldb_dto.MigrationServiceConfig) (sqldb_domain.MigrationService, error) {
	if f.migrationErr != nil {
		return nil, f.migrationErr
	}
	if f.migrator == nil {
		f.migrator = &fakeMigrationService{}
	}
	return f.migrator, nil
}

func (*fakeSQLToolchain) NewSeedService(*sql.DB, sqldb_dto.DialectConfig, fs.FS, string) (sqldb_domain.SeedService, error) {
	return &fakeSeedService{}, nil
}

type fakeSQLEngine struct {
	toolchain   *fakeSQLToolchain
	diagnostics []DatabaseHealthDiagnostic
}

func (*fakeSQLEngine) Dialect() string { return "sqlite" }

func (e *fakeSQLEngine) Toolchain() sqldb_domain.ToolchainPort { return e.toolchain }

func (e *fakeSQLEngine) CheckHealth(context.Context, *sql.DB) []DatabaseHealthDiagnostic {
	return e.diagnostics
}

func newFakeEngine() *fakeSQLEngine {
	return &fakeSQLEngine{toolchain: &fakeSQLToolchain{}}
}

func newQuerierContainer(t *testing.T) *Container {
	t.Helper()

	fakeDriverDB, err := fakeDB()
	require.NoError(t, err)
	require.NoError(t, fakeDriverDB.Close())
	runGlobalShutdownOnCleanup(t)

	c := NewContainer()
	c.serverConfig.Paths.BaseDir = new(t.TempDir())
	return c
}

func TestGetDatabaseService_OpensRegisteredDatabasesWithReplicasAndServices(t *testing.T) {
	c := newQuerierContainer(t)
	engine := newFakeEngine()
	engine.diagnostics = []DatabaseHealthDiagnostic{
		{Name: "wal", Value: "on", State: string(healthprobe_dto.StateHealthy)},
		{Name: "cache", Value: "92%", State: string(healthprobe_dto.StateDegraded), Message: "hit ratio low"},
	}
	migrations := fstest.MapFS{"001_init.up.sql": &fstest.MapFile{Data: []byte("SELECT 1;")}}
	c.AddDatabase("main", &DatabaseRegistration{
		DriverName:      fakeDriverName,
		DSN:             "primary",
		MigrationFS:     migrations,
		SeedFS:          migrations,
		MaxOpenConns:    8,
		MaxIdleConns:    4,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Second,
		EnableOTel:      true,
		EngineConfig:    EngineConfig{Engine: engine},
		Replicas: []Replica{
			{DSN: "replica-a", Weight: 2},
			{DSN: "replica-b", MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: time.Hour, ConnMaxIdleTime: time.Minute},
		},
	})

	service, err := c.GetDatabaseService()
	require.NoError(t, err)

	connection, err := c.GetDatabaseConnection("main")
	require.NoError(t, err)
	assert.Equal(t, 8, connection.Stats().MaxOpenConnections)

	driver, err := c.GetDatabaseDriver("main")
	require.NoError(t, err)
	assert.Equal(t, fakeDriverName, driver)

	reader, err := c.GetDatabaseReader("main")
	require.NoError(t, err)
	_, err = reader.ExecContext(context.Background(), "SELECT 1")
	require.NoError(t, err)

	writer, err := c.GetDatabaseWriter("main")
	require.NoError(t, err)
	assert.NotNil(t, writer)

	migrator, err := c.GetQuerierMigrationService("main")
	require.NoError(t, err)
	assert.Same(t, engine.toolchain.migrator, migrator)

	seeder, err := c.GetQuerierSeedService("main")
	require.NoError(t, err)
	assert.NotNil(t, seeder)

	require.NoError(t, c.runMigrationsIfConfigured("main"))
	assert.Equal(t, 1, engine.toolchain.migrator.upCalls)

	readiness := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateDegraded, readiness.State)
	require.Len(t, readiness.Dependencies, 1)
	assert.Len(t, readiness.Dependencies[0].Dependencies, 3)

	liveness := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	assert.Equal(t, healthprobe_dto.StateHealthy, liveness.State)
	assert.Equal(t, databaseServiceName, service.Name())
}

func TestGetDatabaseService_ReportsMissingServicesForARegisteredDatabase(t *testing.T) {
	c := newQuerierContainer(t)
	database, err := fakeDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	c.AddDatabase("external", &DatabaseRegistration{DB: database})

	driver, err := c.GetDatabaseDriver("external")
	require.NoError(t, err)
	assert.Equal(t, "pre-opened", driver)

	_, err = c.GetQuerierMigrationService("external")
	require.ErrorIs(t, err, ErrNoMigrationFilesystem)
	_, err = c.GetQuerierSeedService("external")
	require.ErrorIs(t, err, ErrNoSeedFilesystem)
	require.NoError(t, c.runMigrationsIfConfigured("external"))

	_, err = c.GetDatabaseDriver("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
	err = c.runMigrationsIfConfigured("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
}

func TestRunMigrationsIfConfigured_WrapsMigrationFailures(t *testing.T) {
	c := newQuerierContainer(t)
	engine := newFakeEngine()
	engine.toolchain.migrator = &fakeMigrationService{upErr: errors.New("syntax error")}
	c.AddDatabase("main", &DatabaseRegistration{
		DriverName:   fakeDriverName,
		MigrationFS:  fstest.MapFS{},
		EngineConfig: EngineConfig{Engine: engine},
	})

	err := c.runMigrationsIfConfigured("main")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax error")
}

func TestGetDatabaseService_FailsForInvalidRegistrations(t *testing.T) {
	unreachable := newPingDB(errors.New("refused"))
	t.Cleanup(func() { _ = unreachable.Close() })
	failingToolchain := newFakeEngine()
	failingToolchain.toolchain.migrationErr = errors.New("bad dialect")

	testCases := []struct {
		registration *DatabaseRegistration
		name         string
		wantErr      string
	}{
		{name: "no driver", registration: &DatabaseRegistration{}, wantErr: ErrNoDriverConfigured.Error()},
		{name: "unknown driver", registration: &DatabaseRegistration{DriverName: "piko-missing-driver"}, wantErr: "open"},
		{name: "replica through the engine driver", registration: &DatabaseRegistration{
			EngineConfig: EngineConfig{DriverName: fakeDriverName},
			Replicas:     []Replica{{DSN: "replica"}},
		}},
		{name: "migrations without toolchain", registration: &DatabaseRegistration{DriverName: fakeDriverName, MigrationFS: fstest.MapFS{}}, wantErr: sqldb_domain.ErrNoSQLToolchain.Error()},
		{name: "seeds without toolchain", registration: &DatabaseRegistration{DriverName: fakeDriverName, SeedFS: fstest.MapFS{}}, wantErr: sqldb_domain.ErrNoSQLToolchain.Error()},
		{name: "migration service failure", registration: &DatabaseRegistration{
			DriverName:   fakeDriverName,
			MigrationFS:  fstest.MapFS{},
			EngineConfig: EngineConfig{Engine: failingToolchain},
		}, wantErr: "bad dialect"},
		{name: "pre-opened ping failure", registration: &DatabaseRegistration{DB: unreachable}, wantErr: "refused"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newQuerierContainer(t)
			c.AddDatabase("main", tc.registration)

			_, err := c.GetDatabaseService()

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)

			_, err = c.GetDatabaseConnection("main")
			require.Error(t, err)
			_, err = c.GetDatabaseReader("main")
			require.Error(t, err)
			_, err = c.GetDatabaseWriter("main")
			require.Error(t, err)
			_, err = c.GetDatabaseDriver("main")
			require.Error(t, err)
			_, err = c.GetQuerierMigrationService("main")
			require.Error(t, err)
			_, err = c.GetQuerierSeedService("main")
			require.Error(t, err)
		})
	}
}

func TestOpenReplicas_FailsWhenAReplicaCannotBeReached(t *testing.T) {
	c := newQuerierContainer(t)

	_, _, err := c.openReplicas(context.Background(), "main", "piko-missing-driver", &DatabaseRegistration{
		Replicas: []Replica{{DSN: "replica"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "replica 0 open")
}

func TestOpenReplicas_StopsWhenTheContextIsCancelled(t *testing.T) {
	c := newQuerierContainer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := c.openReplicas(ctx, "main", fakeDriverName, &DatabaseRegistration{
		Replicas: []Replica{{DSN: "replica"}},
	})

	require.ErrorIs(t, err, context.Canceled)
}

func TestReadinessReportsUnreachableConnections(t *testing.T) {
	t.Parallel()

	down := newPingDB(errors.New("connection refused"))
	t.Cleanup(func() { _ = down.Close() })
	healthy := newPingDB(nil)
	t.Cleanup(func() { _ = healthy.Close() })

	service := &databaseService{instances: map[string]*databaseInstance{
		"main": {db: healthy, replicas: []*sql.DB{down}},
	}}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Equal(t, "one or more databases have issues", status.Message)
}

func TestPoolUtilisationState_ClassifiesUtilisation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		stats sql.DBStats
		want  healthprobe_dto.State
	}{
		{stats: sql.DBStats{}, want: healthprobe_dto.StateHealthy},
		{stats: sql.DBStats{MaxOpenConnections: 10, InUse: 1}, want: healthprobe_dto.StateHealthy},
		{stats: sql.DBStats{MaxOpenConnections: 10, InUse: 8}, want: healthprobe_dto.StateDegraded},
		{stats: sql.DBStats{MaxOpenConnections: 10, InUse: 10}, want: healthprobe_dto.StateUnhealthy},
	}

	for _, tc := range testCases {
		state, message := poolUtilisationState(tc.stats)

		assert.Equal(t, tc.want, state)
		assert.Equal(t, tc.want == healthprobe_dto.StateHealthy, message == "")
	}
}

func TestDiagnosticStateToProbeState_MapsEngineStates(t *testing.T) {
	t.Parallel()

	assert.Equal(t, healthprobe_dto.StateDegraded, diagnosticStateToProbeState(string(healthprobe_dto.StateDegraded)))
	assert.Equal(t, healthprobe_dto.StateUnhealthy, diagnosticStateToProbeState(string(healthprobe_dto.StateUnhealthy)))
	assert.Equal(t, healthprobe_dto.StateHealthy, diagnosticStateToProbeState("unknown"))
}

func TestResolveDBSystem_MapsDriversToSemanticConventions(t *testing.T) {
	t.Parallel()

	testCases := map[string]*DatabaseRegistration{
		"postgresql": {DriverName: "pgx"},
		"mysql":      {DriverName: "mysql"},
		"sqlite":     {EngineConfig: EngineConfig{Engine: newFakeEngine()}},
		"duckdb":     {EngineConfig: EngineConfig{DriverName: "duckdb"}},
		"custom":     {DriverName: "custom"},
	}

	for want, registration := range testCases {
		assert.Equal(t, want, resolveDBSystem(registration))
	}
}

func TestResolveDriverName_PrefersExplicitNames(t *testing.T) {
	t.Parallel()

	database := &sql.DB{}
	assert.Equal(t, "pgx", resolveDriverName(&DatabaseRegistration{DB: database, EngineConfig: EngineConfig{DriverName: "pgx"}}))
	assert.Equal(t, "pre-opened", resolveDriverName(&DatabaseRegistration{DB: database}))
	assert.Equal(t, "mysql", resolveDriverName(&DatabaseRegistration{DriverName: "mysql"}))
	assert.Equal(t, "duckdb", resolveDriverName(&DatabaseRegistration{EngineConfig: EngineConfig{DriverName: "duckdb"}}))
}

func TestResolveMigrationDialect_PrefersTheEngineDialect(t *testing.T) {
	t.Parallel()

	registration := &DatabaseRegistration{MigrationDialect: sqldb_dto.DialectConfig{}}
	assert.Equal(t, registration.MigrationDialect, resolveMigrationDialect(registration))
}

func TestBuildProject_SQLModeWritesGeneratedQueryCode(t *testing.T) {
	baseDir := newTestProject(t, nil)
	outputDir := filepath.Join(baseDir, "db", "generated")
	engine := newFakeEngine()
	engine.toolchain.result = &sqldb_dto.GenerationResult{
		Files:      []sqldb_dto.GeneratedFile{{Name: "queries.go", Content: []byte("package generated\n")}},
		QueryCount: 3,
	}
	c := newResolvedContainer(t, baseDir, WithDatabase("main", &DatabaseRegistration{
		QueryFS:                  fstest.MapFS{"queries/users.sql": &fstest.MapFile{Data: []byte("SELECT 1;")}},
		GeneratedOutputDirectory: outputDir,
		EngineConfig:             EngineConfig{Engine: engine},
	}), WithDatabase("skipped", &DatabaseRegistration{}))

	err := BuildProject(context.Background(), GenerateModeSQL, c)

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(outputDir, "queries.go"))
	require.NoError(t, err)
	assert.Equal(t, "package generated\n", string(content))
}

func TestBuildProject_SQLModeReportsGenerationFailures(t *testing.T) {
	baseDir := newTestProject(t, nil)
	engine := newFakeEngine()
	engine.toolchain.generateErr = errors.New("parse failure")
	c := newResolvedContainer(t, baseDir, WithDatabase("main", &DatabaseRegistration{
		QueryFS:      fstest.MapFS{},
		EngineConfig: EngineConfig{Engine: engine},
	}))

	err := BuildProject(context.Background(), GenerateModeSQL, c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse failure")
}

func TestBuildProject_SQLModeStopsOnErrorDiagnostics(t *testing.T) {
	baseDir := newTestProject(t, nil)
	failure := sqldb_dto.SourceError{}
	failure.Filename = "queries/users.sql"
	failure.Message = "unknown column"
	failure.Severity = sqldb_dto.SeverityError
	engine := newFakeEngine()
	engine.toolchain.result = &sqldb_dto.GenerationResult{Diagnostics: []sqldb_dto.SourceError{failure}}
	c := newResolvedContainer(t, baseDir, WithDatabase("main", &DatabaseRegistration{
		QueryFS:      fstest.MapFS{},
		EngineConfig: EngineConfig{Engine: engine},
	}))

	err := BuildProject(context.Background(), GenerateModeSQL, c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown column")
}
