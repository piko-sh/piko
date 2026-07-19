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

//go:build integration

// Package release_test proves the release publish protocol
// (internal/registry/registry_domain/release_publish.go) end to end against the real SQL
// registry DALs, SQLite and Postgres, which are the only backends implementing
// registry_domain.ReleasePublisher. The otter backend appears in every case's subtest tree as
// an explicit skip, so its absence from the protocol stays loud. The suite contains:
//
//   - publish stamps unstamped seed artefacts under the release layer key, proven by the
//     storage key resolving after publish and vanishing after retire
//   - two releases coexist on one shared backend and both storage keys resolve
//   - two nodes publishing an identical payload concurrently reference each blob exactly once
//   - a second publish under one release id with a different digest is rejected with
//     registry_domain.ErrReleaseDigestConflict and changes no reference counts
//   - a backend without ReleasePublisher (otter) publishes and retires as a no-op
//   - heartbeats advance monotonically and an out-of-order heartbeat never rewinds
//   - a stale publishing lease whose owner died mid-flight is taken over and published
//   - retire removes the layers, the lease, and the blob references, and emits GC hints for
//     blobs that reached zero
//   - a redeploy after a retire re-claims the release and republishes
//   - the reaper retires only stale foreign releases, never its own release or a live one
//
// Postgres is resolved in two ways. When PIKO_TEST_POSTGRES_DSN is set the suite pings that
// server with a ten second timeout and uses it, which is how a CI service container is
// supplied; a DSN that does not answer is treated exactly like an embedded start failure.
// Otherwise the suite starts an embedded Postgres, which is the convenient local path.
//
// When PIKO_REQUIRE_POSTGRES is set, a Postgres that cannot be reached is a FATAL error
// rather than a silent reduction in coverage. It is intended for CI environments that provide
// Postgres, because without it a suite run on a machine with no Postgres skips every Postgres
// case while still reporting PASS.
package release_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	"piko.sh/piko/internal/testutil/leakcheck"
)

const (
	postgresDSNEnv              = "PIKO_TEST_POSTGRES_DSN"
	postgresRequiredEnv         = "PIKO_REQUIRE_POSTGRES"
	embeddedPostgresDSN         = "postgres://piko_test:piko_test@localhost:15434/piko_test?sslmode=disable"
	externalPostgresPingTimeout = 10 * time.Second
)

var (
	postgresConnectionString  string
	postgresRequired          bool
	postgresUnavailableReason string
	embeddedPostgresInstance  *embeddedpostgres.EmbeddedPostgres
)

func TestMain(m *testing.M) {
	postgresRequired = envIsTrue(postgresRequiredEnv)
	resolvePostgres()

	if postgresConnectionString == "" && postgresRequired {
		_, _ = fmt.Fprintf(os.Stderr,
			"FATAL: %s is set but Postgres is unavailable: %s\n",
			postgresRequiredEnv, postgresUnavailableReason)
		stopEmbeddedPostgres()
		os.Exit(1)
	}

	code := m.Run()
	stopEmbeddedPostgres()

	if code == 0 {
		if err := leakcheck.FindLeaks(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(code)
}

func resolvePostgres() {
	if dsn := os.Getenv(postgresDSNEnv); dsn != "" {
		if err := pingPostgres(dsn); err != nil {
			postgresUnavailableReason = fmt.Sprintf(
				"%s is set but the server did not answer a ping within %s: %v",
				postgresDSNEnv, externalPostgresPingTimeout, err)
			return
		}
		postgresConnectionString = dsn
		return
	}
	startEmbeddedPostgres()
}

func pingPostgres(dsn string) error {
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), externalPostgresPingTimeout)
	defer cancel()
	return database.PingContext(ctx)
}

func startEmbeddedPostgres() {
	instance := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("piko_test").
			Password("piko_test").
			Database("piko_test").
			Port(15434).
			RuntimePath(filepath.Join(os.TempDir(), "piko-release-pg-test")).
			StartTimeout(120 * time.Second),
	)
	if err := instance.Start(); err != nil {
		postgresUnavailableReason = err.Error()
		return
	}
	embeddedPostgresInstance = instance
	postgresConnectionString = embeddedPostgresDSN
}

func stopEmbeddedPostgres() {
	if embeddedPostgresInstance != nil {
		_ = embeddedPostgresInstance.Stop()
	}
}

func envIsTrue(name string) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return true
	}
	return parsed
}
