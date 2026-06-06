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

package db_engine_clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/wdk/db"
)

const (
	// replicationQueueDegradedRows is the number of pending replication-queue entries above
	// which the cluster is flagged degraded. The value is a conservative default; operators
	// can override it per-deployment by wrapping the check function or filtering output.
	replicationQueueDegradedRows = 1000

	// replicationQueueUnhealthyRows is the threshold above which the cluster is flagged
	// outright unhealthy.
	replicationQueueUnhealthyRows = 10000

	// mutationsDegradedCount is the number of in-progress mutations above which the table is
	// flagged degraded.
	mutationsDegradedCount = 50

	// mutationsUnhealthyCount is the threshold above which the table is flagged outright
	// unhealthy.
	mutationsUnhealthyCount = 500

	// partsDegradedCount is the number of active parts above which the table is flagged
	// degraded (a high part count can indicate failing merges).
	partsDegradedCount = 10000

	// partsUnhealthyCount is the threshold above which the part count is unhealthy.
	partsUnhealthyCount = 100000

	// healthCheckTimeout caps each diagnostic query so a hung database never blocks the
	// health endpoint.
	healthCheckTimeout = 5 * time.Second

	// healthStateHealthy is the standard diagnostic state for healthy results.
	//
	// The bootstrap probe translator (diagnosticStateToProbeState) matches the uppercase
	// healthprobe_dto.State spellings exactly, so these constants MUST be uppercase or every
	// non-healthy diagnostic silently falls through to the HEALTHY default. The postgres
	// engine uses the same uppercase convention.
	healthStateHealthy = "HEALTHY"

	// healthStateDegraded is the standard diagnostic state for degraded results.
	healthStateDegraded = "DEGRADED"

	// healthStateUnhealthy is the standard diagnostic state for unhealthy results.
	healthStateUnhealthy = "UNHEALTHY"
)

var (
	_ db.DatabaseHealthChecker = (*ClickHouseEngine)(nil)

	// errHealthProbeTimeout is the cause attached to each diagnostic query's deadline so a
	// caller inspecting context.Cause can tell a probe that exceeded healthCheckTimeout
	// apart from an unrelated cancellation of the parent context.
	errHealthProbeTimeout = errors.New("clickhouse health probe timed out")
)

// CheckHealth runs the ClickHouse-specific diagnostics and returns one entry per check.
//
// The diagnostics cover the active part count per shard (surfacing tables that are not
// merging fast enough), the replication queue depth (surfacing replicas falling behind),
// the in-progress mutations (surfacing async ALTER UPDATE/DELETE operations that have
// backed up), and a basic ping to confirm the connection is alive. Each diagnostic
// carries a state (HEALTHY / DEGRADED / UNHEALTHY), a human-readable message, and an
// optional numeric value the /healthz endpoint can render.
//
// This is a method on *ClickHouseEngine so the bootstrap probe can discover it via the
// db.DatabaseHealthChecker interface assertion; the sibling engines expose CheckHealth
// the same way. A compile-time guard in engine.go pins the interface so the receiver can
// never silently regress to a free function.
//
// Takes database (*sql.DB) to execute the diagnostics through.
//
// Returns []db.DatabaseHealthDiagnostic which holds one entry per check.
func (*ClickHouseEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	results := []db.DatabaseHealthDiagnostic{
		checkPing(ctx, database),
		checkActiveParts(ctx, database),
		checkReplicationQueue(ctx, database),
		checkPendingMutations(ctx, database),
	}
	return results
}

// checkPing executes a trivial SELECT against the connection. The underlying driver's
// connection pool may queue this behind other requests; the explicit deadline protects
// against indefinite waits.
//
// Takes database (*sql.DB) which the ping query runs against.
//
// Returns db.DatabaseHealthDiagnostic which reports connectivity state.
func checkPing(ctx context.Context, database *sql.DB) db.DatabaseHealthDiagnostic {
	ctx, cancel := context.WithTimeoutCause(ctx, healthCheckTimeout, errHealthProbeTimeout)
	defer cancel()

	var one int
	if err := database.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return db.DatabaseHealthDiagnostic{
			Name:    "clickhouse.connectivity",
			State:   healthStateUnhealthy,
			Message: fmt.Sprintf("ping failed: %s", err.Error()),
		}
	}
	return db.DatabaseHealthDiagnostic{
		Name:  "clickhouse.connectivity",
		State: healthStateHealthy,

		Message: "ping ok",
		Value:   strconv.Itoa(one),
	}
}

// checkActiveParts inspects system.parts and reports the total active-part count. A
// consistently high count indicates merges are falling behind ingest.
//
// Takes database (*sql.DB) which the count query runs against.
//
// Returns db.DatabaseHealthDiagnostic which reports the active-part state.
func checkActiveParts(ctx context.Context, database *sql.DB) db.DatabaseHealthDiagnostic {
	ctx, cancel := context.WithTimeoutCause(ctx, healthCheckTimeout, errHealthProbeTimeout)
	defer cancel()

	var count int64
	err := database.QueryRowContext(ctx,
		`SELECT count() FROM system.parts WHERE active`,
	).Scan(&count)
	if err != nil {
		return db.DatabaseHealthDiagnostic{
			Name:    "clickhouse.active_parts",
			State:   healthStateUnhealthy,
			Message: fmt.Sprintf("query failed: %s", err.Error()),
		}
	}
	state, message := classifyActiveParts(count)
	return db.DatabaseHealthDiagnostic{
		Name:    "clickhouse.active_parts",
		State:   state,
		Message: message,
		Value:   strconv.FormatInt(count, 10),
	}
}

// classifyActiveParts maps a part count to a (state, message) pair using the
// partsDegradedCount / partsUnhealthyCount thresholds. Extracted so the classification
// logic can be unit-tested without requiring a live database connection.
//
// Takes count (int64), the number of active parts to classify.
//
// Returns state (string), the diagnostic state name.
// Returns message (string), the human-readable explanation.
func classifyActiveParts(count int64) (state string, message string) {
	if count >= partsUnhealthyCount {
		return healthStateUnhealthy, fmt.Sprintf("active part count %d above unhealthy threshold %d", count, partsUnhealthyCount)
	}
	if count >= partsDegradedCount {
		return healthStateDegraded, fmt.Sprintf("active part count %d above degraded threshold %d", count, partsDegradedCount)
	}
	return healthStateHealthy, fmt.Sprintf("%d active parts", count)
}

// checkReplicationQueue inspects system.replication_queue for any replica that has fallen
// behind the leader. Reports the number of pending queue entries across all replicated
// tables.
//
// system.replication_queue is absent on non-replicated deployments; that specific
// "unknown table" case is benign and reported healthy. Any other failure (timeout,
// permission, connectivity) is a real problem and must surface as unhealthy rather than
// being silently swallowed as healthy.
//
// Takes database (*sql.DB) which the count query runs against.
//
// Returns db.DatabaseHealthDiagnostic which reports the replication-queue state.
func checkReplicationQueue(ctx context.Context, database *sql.DB) db.DatabaseHealthDiagnostic {
	return countSystemTableDiagnostic(ctx, database, countDiagnosticSpec{
		name:          "clickhouse.replication_queue",
		query:         `SELECT count() FROM system.replication_queue`,
		absentMessage: "replication queue not present (non-replicated deployment)",
		classify:      classifyReplicationQueue,
	})
}

// countDiagnosticSpec parameterises countSystemTableDiagnostic for one system-table count
// probe.
type countDiagnosticSpec struct {
	// classify maps a non-error count to its (state, message) pair.
	classify func(int64) (state string, message string)

	// name is the diagnostic name (e.g. "clickhouse.replication_queue").
	name string

	// query is the count(*) probe SQL.
	query string

	// absentMessage is the message reported (as healthy) when the system table is absent.
	absentMessage string
}

// countSystemTableDiagnostic runs a count(*) probe and maps the result through
// spec.classify.
//
// A missing system table (UNKNOWN_TABLE) is the benign non-replicated / non-mutating
// deployment case and is reported healthy with spec.absentMessage; any other error
// (timeout, permission, connectivity) is a real failure and surfaces as unhealthy rather
// than being swallowed. Shared by the replication-queue and pending-mutations probes so
// their error handling stays in lockstep.
//
// Takes database (*sql.DB) which the probe query runs against.
// Takes spec (countDiagnosticSpec) which supplies the query, name, absent message, and
// classifier for one probe.
//
// Returns db.DatabaseHealthDiagnostic which reports the probe state.
func countSystemTableDiagnostic(ctx context.Context, database *sql.DB, spec countDiagnosticSpec) db.DatabaseHealthDiagnostic {
	ctx, cancel := context.WithTimeoutCause(ctx, healthCheckTimeout, errHealthProbeTimeout)
	defer cancel()

	var count int64
	if err := database.QueryRowContext(ctx, spec.query).Scan(&count); err != nil {
		if isUnknownTableError(err) {
			return db.DatabaseHealthDiagnostic{
				Name:    spec.name,
				State:   healthStateHealthy,
				Message: spec.absentMessage,
			}
		}
		return db.DatabaseHealthDiagnostic{
			Name:    spec.name,
			State:   healthStateUnhealthy,
			Message: fmt.Sprintf("query failed: %s", err.Error()),
		}
	}
	state, message := spec.classify(count)
	return db.DatabaseHealthDiagnostic{
		Name:    spec.name,
		State:   state,
		Message: message,
		Value:   strconv.FormatInt(count, 10),
	}
}

// isUnknownTableError reports whether err is a ClickHouse "table does not exist" failure.
//
// The ClickHouse server returns SQLSTATE-free errors whose text carries the symbolic name
// UNKNOWN_TABLE and the numeric code 60; the clickhouse-go database/sql driver surfaces
// both in the error string. The substring check keeps this adapter free of a hard
// dependency on the driver's typed error. Used to distinguish the benign non-replicated /
// non-mutating deployment case (system table absent) from a genuine probe failure.
//
// The numeric match is anchored to the exact code-60 token ("Code: 60." / "Code: 60 ") so
// it does not prefix-match unrelated codes such as Code: 600+ (TOO_MANY_*). The bare
// "does not exist" / "doesn't exist" substrings are deliberately NOT matched on their own
// because they also appear in UNKNOWN_DATABASE and missing-column errors, which would
// otherwise mask genuine probe failures as HEALTHY.
//
// Takes err (error) which is the scan/query error to classify.
//
// Returns bool which is true when err indicates a missing system table.
func isUnknownTableError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "UNKNOWN_TABLE") ||
		strings.Contains(message, "Code: 60.") ||
		strings.Contains(message, "Code: 60 ")
}

// classifyReplicationQueue maps a replication-queue depth to a (state, message) pair
// using the replicationQueueDegradedRows / replicationQueueUnhealthyRows thresholds.
//
// Takes queueDepth (int64), the total queue entries summed across all replicas.
//
// Returns state (string), the diagnostic state name.
// Returns message (string), the human-readable explanation.
func classifyReplicationQueue(queueDepth int64) (state string, message string) {
	if queueDepth >= replicationQueueUnhealthyRows {
		return healthStateUnhealthy, fmt.Sprintf("replication queue %d above unhealthy threshold %d", queueDepth, replicationQueueUnhealthyRows)
	}
	if queueDepth >= replicationQueueDegradedRows {
		return healthStateDegraded, fmt.Sprintf("replication queue %d above degraded threshold %d", queueDepth, replicationQueueDegradedRows)
	}
	return healthStateHealthy, fmt.Sprintf("%d pending replication-queue entries", queueDepth)
}

// checkPendingMutations inspects system.mutations for any unfinished ALTER UPDATE / ALTER
// DELETE operations. Reports the count of not-yet-completed mutations.
//
// As with checkReplicationQueue, only the "unknown table" case (the system table is
// absent on minimal deployments) is treated as benign; any other failure surfaces as
// unhealthy.
//
// Takes database (*sql.DB) which the count query runs against.
//
// Returns db.DatabaseHealthDiagnostic which reports the pending-mutations state.
func checkPendingMutations(ctx context.Context, database *sql.DB) db.DatabaseHealthDiagnostic {
	return countSystemTableDiagnostic(ctx, database, countDiagnosticSpec{
		name:          "clickhouse.pending_mutations",
		query:         `SELECT count() FROM system.mutations WHERE not is_done`,
		absentMessage: "mutations system table not present",
		classify:      classifyPendingMutations,
	})
}

// classifyPendingMutations maps the count of unfinished mutations to a (state, message)
// pair using the mutationsDegradedCount / mutationsUnhealthyCount thresholds.
//
// Takes pending (int64), the number of in-progress mutations.
//
// Returns state (string), the diagnostic state name.
// Returns message (string), the human-readable explanation.
func classifyPendingMutations(pending int64) (state string, message string) {
	if pending >= mutationsUnhealthyCount {
		return healthStateUnhealthy, fmt.Sprintf("pending mutations %d above unhealthy threshold %d", pending, mutationsUnhealthyCount)
	}
	if pending >= mutationsDegradedCount {
		return healthStateDegraded, fmt.Sprintf("pending mutations %d above degraded threshold %d", pending, mutationsDegradedCount)
	}
	return healthStateHealthy, fmt.Sprintf("%d pending mutations", pending)
}
