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
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/colour"
)

func runExplain(t *testing.T, arguments ...string) (int, string, string) {
	t.Helper()
	previous := colour.Enabled()
	colour.SetEnabled(false)
	t.Cleanup(func() { colour.SetEnabled(previous) })

	var stdout, stderr bytes.Buffer
	exitCode := RunExplainWithIO(arguments, &stdout, &stderr)
	return exitCode, stdout.String(), stderr.String()
}

func TestExplainKnownCode(t *testing.T) {
	exitCode, stdout, stderr := runExplain(t, "t141")

	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)
	require.True(t, strings.HasPrefix(stdout, "T141  Invalid comparison  (error)\n"))
	require.Contains(t, stdout, "\nWhen it is reported\n  ")
	require.Contains(t, stdout, "\nHow to fix it\n  ")
	require.Contains(t, stdout, "[T141]: Invalid operation: cannot strictly compare type 'app.Level' to 'string'")
	require.Contains(t, stdout, "More: https://piko.sh/diagnostics/T141\n")
}

func TestExplainByName(t *testing.T) {
	exitCode, stdout, _ := runExplain(t, "MissingRequiredProp")

	require.Equal(t, 0, exitCode)
	require.True(t, strings.HasPrefix(stdout, "T127  Missing required prop  (error)\n"))
}

func TestExplainUnusedCode(t *testing.T) {
	exitCode, stdout, _ := runExplain(t, "T144")

	require.Equal(t, 0, exitCode)
	require.Contains(t, stdout, "Not currently reported.")
}

func TestExplainListsEveryCode(t *testing.T) {
	exitCode, stdout, _ := runExplain(t)

	require.Equal(t, 0, exitCode)
	require.Contains(t, stdout, "Template expressions\n  T001  error")
	require.Contains(t, stdout, "\nTemplate analysis\n  T100  error")
	require.Contains(t, stdout, "\nRendering\n  R001  error")
	require.Contains(t, stdout, "T101  error or warning")
	require.Contains(t, stdout, "(not currently reported)")
	require.Contains(t, stdout, "Run 'piko explain <code>' for details")
}

func TestExplainJSON(t *testing.T) {
	exitCode, stdout, _ := runExplain(t, "--json", "R003")
	require.Equal(t, 0, exitCode)

	var entry struct {
		Code       string   `json:"code"`
		Severities []string `json:"severities"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &entry))
	require.Equal(t, "R003", entry.Code)
	require.Equal(t, []string{"warning"}, entry.Severities)

	exitCode, stdout, _ = runExplain(t, "--json")
	require.Equal(t, 0, exitCode)
	var catalogue struct {
		Codes []json.RawMessage `json:"codes"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &catalogue))
	require.Len(t, catalogue.Codes, 95)
}

func TestExplainSQLCode(t *testing.T) {
	exitCode, stdout, _ := runExplain(t, "q028")

	require.Equal(t, 0, exitCode)
	require.Contains(t, stdout, "Q028 is a SQL diagnostic")
	require.Contains(t, stdout, "https://seagull.sh/diagnostics/Q028")
}

func TestExplainUnknownCode(t *testing.T) {
	exitCode, stdout, stderr := runExplain(t, "T999")

	require.Equal(t, 1, exitCode)
	require.Empty(t, stdout)
	require.Contains(t, stderr, `"T999" is not a Piko diagnostic code`)
	require.NotContains(t, stderr, "Did you mean")
}

func TestExplainSuggestsAName(t *testing.T) {
	exitCode, _, stderr := runExplain(t, "ComparisonEror")

	require.Equal(t, 1, exitCode)
	require.Contains(t, stderr, "Did you mean T141?")
}

func TestExplainSeveralCodes(t *testing.T) {
	exitCode, stdout, _ := runExplain(t, "T141", "T999", "R003")

	require.Equal(t, 1, exitCode)
	require.Contains(t, stdout, "T141  Invalid comparison")
	require.Contains(t, stdout, "R003  Field read through nil")
}

func TestExplainUsage(t *testing.T) {
	exitCode, _, stderr := runExplain(t, "-h")
	require.Equal(t, 0, exitCode)
	require.Contains(t, stderr, "Usage: piko explain [--json] [code...]")

	exitCode, _, _ = runExplain(t, "--unknown-flag")
	require.Equal(t, 2, exitCode)
}

func TestWrapText(t *testing.T) {
	wrapped := wrapText("one two three four five six", "  ", 14)

	require.Equal(t, "  one two\n  three four\n  five six", wrapped)
	for line := range strings.SplitSeq(wrapped, "\n") {
		require.LessOrEqual(t, len(line), 14)
	}
}

func TestExplainColour(t *testing.T) {
	for _, name := range append(noColourVariables, forceColourVariables...) {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	t.Setenv("TERM", "xterm-256color")

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, reader.Close())
		require.NoError(t, writer.Close())
	})

	require.False(t, explainColour(writer))

	t.Setenv("FORCE_COLOR", "1")
	require.True(t, explainColour(writer))

	t.Setenv("NO_COLOR", "")
	require.False(t, explainColour(writer))
}
