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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortDigest(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "strips sha256 prefix", input: "sha256:2af3b81862c6abcdef1234567890", expect: "2af3b81862c6"},
		{name: "without prefix", input: "2af3b81862c6abcdef1234567890", expect: "2af3b81862c6"},
		{name: "short digest unchanged", input: "abc123", expect: "abc123"},
		{name: "empty string", input: "", expect: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, shortDigest(tt.input))
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name   string
		expect string
		input  int64
	}{
		{name: "bytes", input: 500, expect: "500 B"},
		{name: "kilobytes", input: 1536, expect: "1.5 KB"},
		{name: "megabytes", input: 1048576, expect: "1.0 MB"},
		{name: "gigabytes", input: 1610612736, expect: "1.5 GB"},
		{name: "large megabytes", input: 637_700_000, expect: "608.2 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, formatBytes(tt.input))
		})
	}
}
