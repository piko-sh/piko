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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	// driverName is the database/sql driver registration name for D1.
	driverName = "d1"
)

var (
	_ driver.DriverContext = (*d1Driver)(nil)

	_ driver.Connector = (*d1Connector)(nil)
)

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

// d1Driver implements driver.Driver and driver.DriverContext.
type d1Driver struct{}

// d1Connector opens connections that share one D1 client, so every connection of a
// database handle draws on the same HTTP connection pool and rate limiter.
type d1Connector struct {
	// driver is the driver that created the connector.
	driver *d1Driver

	// client performs the D1 API calls for every connection.
	client *d1Client
}

// Open parses the DSN and returns a new connection that owns its own D1 client.
//
// The DSN format is "accountID/databaseID?token=apiToken". database/sql uses
// OpenConnector instead, so connections opened through sql.Open share one client.
//
// Takes dsn (string) which encodes the account ID, database ID, and API token.
//
// Returns driver.Conn which is the opened D1 connection.
// Returns error when the DSN is malformed or the API client cannot be created.
func (*d1Driver) Open(dsn string) (driver.Conn, error) {
	client, err := newClientFromDSN(dsn)
	if err != nil {
		return nil, err
	}
	return newConn(client, client), nil
}

// OpenConnector parses the DSN and returns a connector whose connections share one D1
// client.
//
// Takes dsn (string) which encodes the account ID, database ID, and API token.
//
// Returns driver.Connector which opens connections to the database.
// Returns error when the DSN is malformed or the API client cannot be created.
func (d *d1Driver) OpenConnector(dsn string) (driver.Connector, error) {
	client, err := newClientFromDSN(dsn)
	if err != nil {
		return nil, err
	}
	return &d1Connector{driver: d, client: client}, nil
}

// Connect returns a connection that shares the connector's D1 client.
//
// Returns driver.Conn which is the new connection.
// Returns error when ctx is already done.
func (c *d1Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return newConn(c.client, nil), nil
}

// Driver returns the driver that created the connector.
//
// Returns driver.Driver which is the D1 driver.
func (c *d1Connector) Driver() driver.Driver {
	return c.driver
}

// Close releases the idle HTTP connections of the shared client. database/sql calls it
// when the database handle is closed.
//
// Returns error which is always nil.
func (c *d1Connector) Close() error {
	c.client.close()
	return nil
}

// Open opens a D1 database using the provided configuration and returns a standard
// *sql.DB handle whose connections share one rate limiter and HTTP connection pool.
//
// Takes config (Config) which provides the API token, account ID, and database ID for the
// D1 database.
// Takes options (...Option) which override the request timeout, response size limit, and
// request rate.
//
// Returns *sql.DB which is the configured database connection pool.
// Returns error when the configuration is invalid or the API client cannot be created.
func Open(config Config, options ...Option) (*sql.DB, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	client, err := newD1Client(config, newClientOptions(options))
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: %w", err)
	}

	return sql.OpenDB(&d1Connector{driver: &d1Driver{}, client: client}), nil
}

// DriverName returns the database/sql driver name used for D1 registration.
//
// Returns string which is "d1".
func DriverName() string {
	return driverName
}

// validateConfig reports whether every required configuration field is set.
//
// Takes config (Config) which is the configuration to check.
//
// Returns error naming the first empty field, or nil when the configuration is complete.
func validateConfig(config Config) error {
	if config.APIToken == "" {
		return errors.New("db_driver_d1: APIToken must not be empty")
	}
	if config.AccountID == "" {
		return errors.New("db_driver_d1: AccountID must not be empty")
	}
	if config.DatabaseID == "" {
		return errors.New("db_driver_d1: DatabaseID must not be empty")
	}
	return nil
}

// newClientFromDSN parses dsn and builds a D1 client with the default limits.
//
// Takes dsn (string) which encodes the account ID, database ID, and API token.
//
// Returns *d1Client which is ready to issue queries.
// Returns error when the DSN is malformed or the API client cannot be created.
func newClientFromDSN(dsn string) (*d1Client, error) {
	config, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: %w", err)
	}
	client, err := newD1Client(config, newClientOptions(nil))
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: %w", err)
	}
	return client, nil
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
