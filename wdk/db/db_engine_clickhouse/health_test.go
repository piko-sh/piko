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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/db"
)

func TestHealth_StateConstantsMatchProbeContract(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "HEALTHY", healthStateHealthy)
	assert.Equal(t, "DEGRADED", healthStateDegraded)
	assert.Equal(t, "UNHEALTHY", healthStateUnhealthy)
}

func TestHealth_EngineSatisfiesHealthChecker(t *testing.T) {
	t.Parallel()

	var engine any = NewClickHouseEngine()
	_, ok := engine.(db.DatabaseHealthChecker)
	assert.True(t, ok, "*ClickHouseEngine must satisfy db.DatabaseHealthChecker")
}

func TestHealth_IsUnknownTableError(t *testing.T) {
	t.Parallel()

	assert.True(t, isUnknownTableError(assertError("Code: 60. DB::Exception: Table system.replication_queue doesn't exist")))
	assert.True(t, isUnknownTableError(assertError("Code: 60 DB::Exception: ...")))
	assert.True(t, isUnknownTableError(assertError("UNKNOWN_TABLE")))
	assert.False(t, isUnknownTableError(assertError("Code: 159. DB::Exception: Timeout exceeded")))

	assert.False(t, isUnknownTableError(assertError("Code: 600. DB::Exception: TOO_MANY_PARTS")))
	assert.False(t, isUnknownTableError(assertError("Code: 47. DB::Exception: column foo does not exist")))
	assert.False(t, isUnknownTableError(nil))
}

func assertError(message string) error {
	if message == "" {
		return nil
	}
	return &stringError{message: message}
}

type stringError struct{ message string }

func (e *stringError) Error() string { return e.message }

func TestHealth_ClassifyActivePartsHealthy(t *testing.T) {
	t.Parallel()

	state, message := classifyActiveParts(0)
	assert.Equal(t, healthStateHealthy, state)
	assert.Contains(t, message, "0 active parts")
}

func TestHealth_ClassifyActivePartsDegraded(t *testing.T) {
	t.Parallel()

	state, message := classifyActiveParts(partsDegradedCount)
	assert.Equal(t, healthStateDegraded, state)
	assert.Contains(t, message, "degraded threshold")
}

func TestHealth_ClassifyActivePartsUnhealthy(t *testing.T) {
	t.Parallel()

	state, message := classifyActiveParts(partsUnhealthyCount + 1)
	assert.Equal(t, healthStateUnhealthy, state)
	assert.Contains(t, message, "unhealthy threshold")
}

func TestHealth_ClassifyReplicationQueueHealthy(t *testing.T) {
	t.Parallel()

	state, _ := classifyReplicationQueue(0)
	assert.Equal(t, healthStateHealthy, state)
}

func TestHealth_ClassifyReplicationQueueDegraded(t *testing.T) {
	t.Parallel()

	state, _ := classifyReplicationQueue(replicationQueueDegradedRows)
	assert.Equal(t, healthStateDegraded, state)
}

func TestHealth_ClassifyReplicationQueueUnhealthy(t *testing.T) {
	t.Parallel()

	state, _ := classifyReplicationQueue(replicationQueueUnhealthyRows + 1)
	assert.Equal(t, healthStateUnhealthy, state)
}

func TestHealth_ClassifyPendingMutationsHealthy(t *testing.T) {
	t.Parallel()

	state, _ := classifyPendingMutations(0)
	assert.Equal(t, healthStateHealthy, state)
}

func TestHealth_ClassifyPendingMutationsDegraded(t *testing.T) {
	t.Parallel()

	state, _ := classifyPendingMutations(mutationsDegradedCount)
	assert.Equal(t, healthStateDegraded, state)
}

func TestHealth_ClassifyPendingMutationsUnhealthy(t *testing.T) {
	t.Parallel()

	state, _ := classifyPendingMutations(mutationsUnhealthyCount + 1)
	assert.Equal(t, healthStateUnhealthy, state)
}

func TestHealth_ThresholdsOrdered(t *testing.T) {
	t.Parallel()

	assert.Less(t, partsDegradedCount, partsUnhealthyCount, "degraded threshold must be below unhealthy threshold")
	assert.Less(t, replicationQueueDegradedRows, replicationQueueUnhealthyRows)
	assert.Less(t, mutationsDegradedCount, mutationsUnhealthyCount)
}
