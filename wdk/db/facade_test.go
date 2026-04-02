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

package db_test

import (
	"testing"

	"piko.sh/piko/wdk/db"
)

func TestConstants(t *testing.T) {
	if db.DirectionUp == db.DirectionDown {
		t.Error("DirectionUp and DirectionDown must be distinct")
	}
}

func TestErrLockNotAcquired(t *testing.T) {
	if db.ErrLockNotAcquired == nil {
		t.Error("ErrLockNotAcquired must not be nil")
	}
}

func TestDialectFactories(t *testing.T) {
	dialects := []struct {
		name    string
		factory func() db.DialectConfig
	}{
		{"PostgresDialect", db.PostgresDialect},
		{"PostgresPgBouncerDialect", db.PostgresPgBouncerDialect},
		{"MySQLDialect", db.MySQLDialect},
		{"SQLiteDialect", db.SQLiteDialect},
	}

	for _, tt := range dialects {
		t.Run(tt.name, func(t *testing.T) {
			dialect := tt.factory()
			if dialect.CreateTableSQL == "" {
				t.Error("CreateTableSQL must not be empty")
			}
			if dialect.PlaceholderFunc == nil {
				t.Error("PlaceholderFunc must not be nil")
			}
			if dialect.LockStrategy == nil {
				t.Error("LockStrategy must not be nil")
			}
		})
	}
}

func TestNewFSFileReader(t *testing.T) {
	reader := db.NewFSFileReader(nil)
	if reader == nil {
		t.Error("NewFSFileReader must return a non-nil reader")
	}
}

func TestWithNonBlockingLock(t *testing.T) {
	opt := db.WithNonBlockingLock()
	if opt == nil {
		t.Error("WithNonBlockingLock must return a non-nil option")
	}
}

func TestGetDatabaseConnectionWithoutBootstrap(t *testing.T) {
	_, err := db.GetDatabaseConnection("test")
	if err == nil {
		t.Error("GetDatabaseConnection must return error without bootstrap")
	}
}

func TestGetMigrationServiceWithoutBootstrap(t *testing.T) {
	_, err := db.GetMigrationService("test")
	if err == nil {
		t.Error("GetMigrationService must return error without bootstrap")
	}
}

func TestGetDatabaseReaderWithoutBootstrap(t *testing.T) {
	_, err := db.GetDatabaseReader("test")
	if err == nil {
		t.Error("GetDatabaseReader must return error without bootstrap")
	}
}
