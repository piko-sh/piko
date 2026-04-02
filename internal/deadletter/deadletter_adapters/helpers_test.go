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

package deadletter_adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

type testEntry struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func newTestMemoryDLQ() *MemoryDeadLetterQueue[testEntry] {
	dlq := NewMemoryDeadLetterQueue[testEntry]()

	memDLQ, ok := dlq.(*MemoryDeadLetterQueue[testEntry])
	if !ok {
		panic("NewMemoryDeadLetterQueue did not return *MemoryDeadLetterQueue")
	}

	return memDLQ
}

func newTestDiskDLQ(t *testing.T) (*DiskDeadLetterQueue[testEntry], *safedisk.MockSandbox) {
	t.Helper()

	sandbox := safedisk.NewMockSandbox("/dlq", safedisk.ModeReadWrite)
	t.Cleanup(func() { _ = sandbox.Close() })
	raw := NewDiskDeadLetterQueue[testEntry](
		"/dlq/deadletters.jsonl",
		WithDeadLetterSandbox[testEntry](sandbox),
	)

	diskDLQ, ok := raw.(*DiskDeadLetterQueue[testEntry])
	if !ok {
		t.Fatal("NewDiskDeadLetterQueue did not return *DiskDeadLetterQueue")
	}

	return diskDLQ, sandbox
}

func testCtx() context.Context {
	return context.Background()
}

func injectMemoryEntry(m *MemoryDeadLetterQueue[testEntry], id string, ts time.Time, data testEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[id] = wrappedEntry[testEntry]{
		ID:        id,
		Timestamp: ts,
		Data:      data,
	}
}

func writeRawDiskEntries(t *testing.T, sandbox *safedisk.MockSandbox, fileName string, entries []wrappedEntry[testEntry]) {
	t.Helper()

	data := make([]byte, 0, len(entries)*64)
	for _, e := range entries {
		line, err := json.Marshal(e)
		require.NoError(t, err)
		data = append(data, line...)
		data = append(data, '\n')
	}

	require.NoError(t, sandbox.WriteFile(fileName, data, 0o600))
}

func writeRawLines(t *testing.T, sandbox *safedisk.MockSandbox, fileName string, lines []string) {
	t.Helper()

	data := make([]byte, 0, len(lines)*64)
	for _, line := range lines {
		data = append(data, []byte(line)...)
		data = append(data, '\n')
	}

	require.NoError(t, sandbox.WriteFile(fileName, data, 0o600))
}

func sampleEntry(n int) testEntry {
	return testEntry{
		Message: fmt.Sprintf("entry-%d", n),
		Code:    n,
	}
}
