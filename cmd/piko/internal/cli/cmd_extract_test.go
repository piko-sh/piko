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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunExtractNoSubcommandPrintsHelp(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO(nil, &stdout, &stderr)

	require.Equal(t, 0, code, "help output should exit 0")
	require.Empty(t, stderr.String(), "no error output when showing help")
	help := stdout.String()
	for _, name := range []string{"generate", "discover", "init", "check"} {
		require.Containsf(t, help, name, "help must list subcommand %s", name)
	}
}

func TestRunExtractHelpFlagPrintsHelp(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"-h", "--help"} {
		t.Run(strings.TrimLeft(flag, "-"), func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := RunExtractWithIO([]string{flag}, &stdout, &stderr)
			require.Equal(t, 0, code)
			require.Contains(t, stdout.String(), "Subcommands:")
		})
	}
}

func TestRunExtractUnknownSubcommandFails(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"bogus"}, &stdout, &stderr)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "Unknown subcommand: bogus")
	require.Contains(t, stderr.String(), "Subcommands:")
}

func TestRunExtractGenerateMissingManifestFails(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"generate", "--manifest", "/nonexistent/piko-symbols.yaml"}, &stdout, &stderr)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "Error")
}

func TestRunExtractGenerateHelpFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"generate", "--help"}, &stdout, &stderr)

	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "Usage: piko extract generate")
}

func TestRunExtractDiscoverHelpFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"discover", "--help"}, &stdout, &stderr)

	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "Usage: piko extract discover")
}

func TestRunExtractInitHelpFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"init", "--help"}, &stdout, &stderr)

	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "Usage: piko extract init")
}

func TestRunExtractCheckHelpFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"check", "--help"}, &stdout, &stderr)

	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "Usage: piko extract check")
}
