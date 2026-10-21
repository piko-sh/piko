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

package interpreted_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko"
)

const (
	memoryBudgetChildEnv        = "PIKO_INTERPRETED_MEMORY_BUDGET_CHILD"
	memoryBudgetCase            = "30_concurrent_large_struct_map"
	memoryBudgetHeapBytes       = 79_700_000
	memoryBudgetHeadroomPercent = 10
)

var (
	memoryBudgetReport = regexp.MustCompile(`heap_inuse_bytes=(\d+) build_ms=(\d+)`)
)

func TestInterpretedBuildMemoryBudget(t *testing.T) {
	if os.Getenv(memoryBudgetChildEnv) != "" {
		measureInterpretedBuild(t)
		return
	}
	if testing.Short() {
		t.Skip("skipping the interpreted build memory budget in short mode")
	}
	command := exec.Command(os.Args[0], "-test.run=^TestInterpretedBuildMemoryBudget$", "-test.count=1", "-test.v")
	command.Env = append(os.Environ(), memoryBudgetChildEnv+"=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "the measured build failed:\n%s", output)
	report := memoryBudgetReport.FindSubmatch(output)
	require.NotNil(t, report, "the measured build printed no report:\n%s", output)
	heap, err := strconv.ParseUint(string(report[1]), 10, 64)
	require.NoError(t, err)
	t.Logf("dev-i build of %s: heap in use %d bytes, built in %s ms", memoryBudgetCase, heap, report[2])
	limit := uint64(memoryBudgetHeapBytes) * (100 + memoryBudgetHeadroomPercent) / 100
	require.LessOrEqual(t, heap, limit,
		"the dev-i build now keeps %d bytes in use, over the %d-byte budget plus %d%%", heap, memoryBudgetHeapBytes, memoryBudgetHeadroomPercent)
}

func measureInterpretedBuild(t *testing.T) {
	resetGlobalStateForTestIsolation()
	tc := testCase{Name: memoryBudgetCase, Path: "testdata/" + memoryBudgetCase}
	spec := loadTestSpec(t, tc)
	server, _, cleanup := setupServer(t, tc, spec)
	defer cleanup()
	defer server.Close()

	started := time.Now()
	require.NoError(t, server.Generate(context.Background(), piko.RunModeDevInterpreted))
	built := time.Since(started)
	response := serveWithin(t, server.GetHandler(), spec.RequestURL, requestTimeout)
	_ = response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)

	runtime.GC()
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Printf("heap_inuse_bytes=%d build_ms=%d\n", stats.HeapInuse, built.Milliseconds())
}
