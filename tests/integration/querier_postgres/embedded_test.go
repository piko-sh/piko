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

package querier_postgres_test

import (
	"log"
	"os"
	"path/filepath"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

var (
	embeddedDatabase     *embeddedpostgres.EmbeddedPostgres
	testConnectionString string
)

func startEmbeddedPostgres() {
	embeddedDatabase = embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("piko_test").
			Password("piko_test").
			Database("piko_test").
			Port(15432).
			RuntimePath(filepath.Join(os.TempDir(), "piko-querier-pg-test")).
			StartTimeout(120 * time.Second),
	)

	if err := embeddedDatabase.Start(); err != nil {
		log.Fatalf("starting embedded postgres: %v", err)
	}

	testConnectionString = "postgres://piko_test:piko_test@localhost:15432/piko_test?sslmode=disable"
}

func stopEmbeddedPostgres() {
	if embeddedDatabase != nil {
		_ = embeddedDatabase.Stop()
	}
}
