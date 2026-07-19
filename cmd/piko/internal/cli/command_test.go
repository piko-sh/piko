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

package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommands_AllRegistered(t *testing.T) {
	t.Parallel()

	expectedCommands := []string{"get", "describe", "info", "watch", "diagnostics", "tui"}

	for _, name := range expectedCommands {
		_, ok := commands[name]
		assert.True(t, ok, "expected command %q to be registered", name)
	}
}

func TestCommands_NeedsConnection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		needsConnection bool
	}{
		{name: "get", needsConnection: true},
		{name: "describe", needsConnection: true},
		{name: "info", needsConnection: true},
		{name: "watch", needsConnection: true},
		{name: "diagnostics", needsConnection: false},
		{name: "tui", needsConnection: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			command, ok := commands[tc.name]
			require.True(t, ok, "command %q not found", tc.name)
			assert.Equal(t, tc.needsConnection, command.needsConnection, "command %q needsConnection = %v, want %v", tc.name, command.needsConnection, tc.needsConnection)
		})
	}
}

func TestRunCommandWithIO_UnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	exitCode := RunCommandWithIO("nonexistent", nil, &stdout, &stderr)

	assert.Equal(t, 1, exitCode, "exit code = %d, want 1", exitCode)
	got := stderr.String()
	assert.NotEmpty(t, got, "expected error output on stderr")
}
