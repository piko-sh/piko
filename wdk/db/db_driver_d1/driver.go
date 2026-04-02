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

package db_driver_d1

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

const driverName = "d1"

func init() {
	sql.Register(driverName, &d1Driver{})
}

// Config holds the credentials needed to connect to a Cloudflare D1 database.
type Config struct {
	// APIToken is a Cloudflare API token with D1:Edit permission.
	APIToken string

	// AccountID is the Cloudflare account identifier.
	AccountID string

	// DatabaseID is the D1 database UUID.
	DatabaseID string
}

// d1Driver implements driver.Driver.
type d1Driver struct{}

// Open parses the DSN and returns a new connection.
//
// The DSN format is "accountID/databaseID?token=apiToken".
//
// Takes dsn (string) which encodes the account ID, database ID, and API token.
//
// Returns driver.Conn which is the opened D1 connection.
// Returns error when the DSN is malformed or the API client cannot be created.
func (*d1Driver) Open(dsn string) (driver.Conn, error) {
	config, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: %w", err)
	}

	api, err := cloudflare.NewWithAPIToken(config.APIToken)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: creating API client: %w", err)
	}

	return &d1Conn{
		api:        api,
		rc:         cloudflare.AccountIdentifier(config.AccountID),
		databaseID: config.DatabaseID,
	}, nil
}

// Open opens a D1 database using the provided configuration and returns a
// standard *sql.DB handle.
//
// Takes config (Config) which provides the API token, account ID, and
// database ID for the D1 database.
//
// Returns *sql.DB which is the configured database connection pool.
// Returns error when the configuration is invalid or the connection fails.
func Open(config Config) (*sql.DB, error) {
	if config.APIToken == "" {
		return nil, errors.New("db_driver_d1: APIToken must not be empty")
	}
	if config.AccountID == "" {
		return nil, errors.New("db_driver_d1: AccountID must not be empty")
	}
	if config.DatabaseID == "" {
		return nil, errors.New("db_driver_d1: DatabaseID must not be empty")
	}

	dsn := config.AccountID + "/" + config.DatabaseID + "?token=" + url.QueryEscape(config.APIToken)
	return sql.Open(driverName, dsn)
}

// DriverName returns the database/sql driver name used by this package.
//
// Returns string which is "d1".
func DriverName() string {
	return driverName
}

// parseDSN extracts Config fields from a DSN string.
//
// The expected format is "accountID/databaseID?token=apiToken".
//
// Takes dsn (string) which is the data source name to parse.
//
// Returns Config which contains the extracted credentials.
// Returns error when the DSN is missing required components.
func parseDSN(dsn string) (Config, error) {
	pathPart, queryPart, _ := strings.Cut(dsn, "?")

	accountID, databaseID, found := strings.Cut(pathPart, "/")
	if !found {
		return Config{}, errors.New("invalid DSN: expected format accountID/databaseID?token=apiToken")
	}

	if accountID == "" {
		return Config{}, errors.New("invalid DSN: accountID is empty")
	}
	if databaseID == "" {
		return Config{}, errors.New("invalid DSN: databaseID is empty")
	}

	values, err := url.ParseQuery(queryPart)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DSN query: %w", err)
	}

	apiToken := values.Get("token")
	if apiToken == "" {
		return Config{}, errors.New("invalid DSN: token parameter is required")
	}

	return Config{
		APIToken:   apiToken,
		AccountID:  accountID,
		DatabaseID: databaseID,
	}, nil
}
