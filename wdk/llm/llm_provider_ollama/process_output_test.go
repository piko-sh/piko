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

package llm_provider_ollama

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputRelay(t *testing.T) {
	t.Parallel()

	longLine := strings.Repeat("y", maxOutputLineBytes+5)
	multiByteLine := strings.Repeat("a", maxOutputLineBytes-1) + "étail"

	testCases := []struct {
		name          string
		writes        []string
		wantOutputs   []string
		wantTruncated []string
		flush         bool
	}{
		{
			name:          "splits complete lines",
			writes:        []string{"first\nsecond\n"},
			wantOutputs:   []string{"first", "second"},
			wantTruncated: []string{"", ""},
		},
		{
			name:          "joins a line split across writes",
			writes:        []string{"par", "tial\r\n"},
			wantOutputs:   []string{"partial"},
			wantTruncated: []string{""},
		},
		{
			name:          "holds a trailing partial line until flushed",
			writes:        []string{"done\nunfinished"},
			flush:         true,
			wantOutputs:   []string{"done", "unfinished"},
			wantTruncated: []string{"", ""},
		},
		{
			name:          "cuts an over-long line and reports the dropped bytes",
			writes:        []string{longLine + "\nnext\n"},
			wantOutputs:   []string{strings.Repeat("y", maxOutputLineBytes), "next"},
			wantTruncated: []string{"5", ""},
		},
		{
			name:          "cuts at a rune boundary",
			writes:        []string{multiByteLine + "\n"},
			wantOutputs:   []string{strings.Repeat("a", maxOutputLineBytes-1)},
			wantTruncated: []string{strconv.Itoa(len("étail"))},
		},
		{
			name:          "flushing an empty relay logs nothing",
			writes:        nil,
			flush:         true,
			wantOutputs:   nil,
			wantTruncated: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, records := newCaptureContext(t)
			relay := newOutputRelay(ctx)

			for _, chunk := range testCase.writes {
				written, err := relay.Write([]byte(chunk))
				require.NoError(t, err)
				assert.Equal(t, len(chunk), written)
			}
			if testCase.flush {
				relay.Flush()
			}

			var outputs, truncated []string
			for _, record := range records.snapshot() {
				require.Equal(t, "Ollama output", record.message)
				outputs = append(outputs, record.attributes["output"])
				truncated = append(truncated, record.attributes["truncated_bytes"])
			}
			assert.Equal(t, testCase.wantOutputs, outputs)
			assert.Equal(t, testCase.wantTruncated, truncated)
		})
	}
}
