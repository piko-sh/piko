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

package telemetry_grpcfb

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		maxBytes  int
		truncated bool
	}{
		{name: "under limit", input: "hello", maxBytes: 10, want: "hello", truncated: false},
		{name: "exact limit", input: "hello", maxBytes: 5, want: "hello", truncated: false},
		{name: "ascii truncated", input: "hello", maxBytes: 3, want: "hel", truncated: true},
		{name: "rune boundary respected", input: "héllo", maxBytes: 2, want: "h", truncated: true},
		{name: "does not split rune", input: "aé", maxBytes: 2, want: "a", truncated: true},
		{name: "empty", input: "", maxBytes: 5, want: "", truncated: false},
		{name: "negative max", input: "x", maxBytes: -1, want: "", truncated: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, truncated := TruncateUTF8(tc.input, tc.maxBytes)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.truncated, truncated)
			assert.True(t, utf8.ValidString(got), "result must remain valid UTF-8")
		})
	}
}

func TestMarshalRejectsOversizedFrame(t *testing.T) {
	b := oversizedBatch()
	_, err := b.Marshal()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrFrameTooLarge), "want ErrFrameTooLarge, got %v", err)
}

func TestMarshalCapsOversizedString(t *testing.T) {
	b := &Batch{SiteID: "s", Logs: []LogLine{{Message: strings.Repeat("é", maxStringLen)}}}
	data, err := b.Marshal()
	require.NoError(t, err)
	var got Batch
	require.NoError(t, got.Unmarshal(data), "capped frame must pass the verifier")
	require.Len(t, got.Logs, 1)
	assert.LessOrEqual(t, len(got.Logs[0].Message), maxStringLen)
	assert.True(t, utf8.ValidString(got.Logs[0].Message))
}
