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
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type pingConn struct{ pingErr error }

func (c pingConn) Ping(context.Context) error        { return c.pingErr }
func (pingConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (pingConn) Close() error                        { return nil }
func (pingConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }

type pingConnector struct{ pingErr error }

func (c pingConnector) Connect(context.Context) (driver.Conn, error) { return pingConn(c), nil }
func (c pingConnector) Driver() driver.Driver                        { return pingDriver(c) }

type pingDriver struct{ pingErr error }

func (d pingDriver) Open(string) (driver.Conn, error) { return pingConn(d), nil }

func newPingDB(pingErr error) *sql.DB {
	return sql.OpenDB(pingConnector{pingErr: pingErr})
}

func TestDatabaseServiceCheckLivenessAggregatesAndOrders(t *testing.T) {
	t.Parallel()

	healthyDB := newPingDB(nil)
	t.Cleanup(func() { _ = healthyDB.Close() })
	replicaDB := newPingDB(nil)
	t.Cleanup(func() { _ = replicaDB.Close() })
	downDB := newPingDB(errors.New("connection refused"))
	t.Cleanup(func() { _ = downDB.Close() })

	service := &databaseService{instances: map[string]*databaseInstance{
		"zeta":  {db: healthyDB},
		"alpha": {db: healthyDB, replicas: []*sql.DB{replicaDB}},
		"beta":  {db: downDB},
	}}

	status := service.checkLiveness(t.Context(), time.Now())

	require.Equal(t, healthprobe_dto.StateUnhealthy, status.State,
		"a single unreachable database must make the service unhealthy")
	require.Len(t, status.Dependencies, 3)
	require.Equal(t,
		[]string{"Database:alpha", "Database:beta", "Database:zeta"},
		[]string{status.Dependencies[0].Name, status.Dependencies[1].Name, status.Dependencies[2].Name},
		"dependencies must be deterministically ordered by name regardless of goroutine completion order")
}

func TestDatabaseServiceCheckLivenessHealthyWhenAllReachable(t *testing.T) {
	t.Parallel()

	primary := newPingDB(nil)
	t.Cleanup(func() { _ = primary.Close() })
	replica := newPingDB(nil)
	t.Cleanup(func() { _ = replica.Close() })

	service := &databaseService{instances: map[string]*databaseInstance{
		"main": {db: primary, replicas: []*sql.DB{replica}},
	}}

	status := service.checkLiveness(t.Context(), time.Now())

	require.Equal(t, healthprobe_dto.StateHealthy, status.State)
	require.Len(t, status.Dependencies, 1)
	require.Equal(t, healthprobe_dto.StateHealthy, status.Dependencies[0].State)
}
