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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/interp/interp_piko_symbols"
	"pipit.sh/pipit"
)

func TestRunExtractNoSubcommandPrintsHelp(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO(nil, &stdout, &stderr)

	require.Equal(t, 0, code, "help output should exit 0")
	require.Empty(t, stderr.String(), "no error output when showing help")
	help := stdout.String()
	require.Contains(t, help, "Usage: piko extract <subcommand>")
	for _, name := range []string{"generate", "discover", "init", "check"} {
		require.Containsf(t, help, name, "help must list subcommand %s", name)
	}
}

func TestRunExtractSubcommandHelpNamesPiko(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"generate", "discover", "init", "check"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := RunExtractWithIO([]string{name, "--help"}, &stdout, &stderr)
			require.Equal(t, 0, code)
			require.Contains(t, stdout.String(), "Usage: piko extract "+name)
		})
	}
}

func TestRunExtractUnknownSubcommandFails(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"bogus"}, &stdout, &stderr)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "Unknown subcommand: bogus")
}

func TestPKScannerReadsScriptImports(t *testing.T) {
	t.Parallel()

	source := `<template><div></div></template>
<script type="application/x-go">
package main

import (
	"fmt"
	"example.com/fixture/pkg/local"
	header "example.com/fixture/partials/shared/header.pk"
)

var (
	_ = fmt.Sprint
	_ = local.Something
	_ = header
)
</script>`

	scanner := pkScanner{}
	require.True(t, scanner.Match("home.pk"))
	require.False(t, scanner.Match("home.go"))

	imports, err := scanner.Imports(context.Background(), "pages/home.pk", []byte(source))
	require.NoError(t, err)
	require.Equal(t, []string{"fmt", "example.com/fixture/pkg/local"}, imports,
		"component references resolve to partials, not Go packages")
}

func TestPKScannerIgnoresFilesWithoutScript(t *testing.T) {
	t.Parallel()

	imports, err := pkScanner{}.Imports(context.Background(), "pages/static.pk", []byte("<template><p>hi</p></template>"))
	require.NoError(t, err)
	require.Empty(t, imports)
}

func TestPKScannerImportsParseFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		source      string
		wantErr     string
		wantImports []string
	}{
		{
			name:    "broken i18n block without a script is reported",
			source:  `<template><p>hi</p></template><i18n lang="json">{"en": </i18n>`,
			wantErr: "parsing pages/broken.pk",
		},
		{
			name: "broken i18n block with a script is reported",
			source: `<template><p>hi</p></template>
<script type="application/x-go">
package main

import "fmt"

var _ = fmt.Sprint
</script>
<i18n lang="json">{"en": </i18n>`,
			wantErr: "parsing pages/broken.pk",
		},
		{
			name: "template diagnostics do not stop import discovery",
			source: `<template><p p-if="">hi</p></template>
<script type="application/x-go">
package main

import "fmt"

var _ = fmt.Sprint
</script>`,
			wantImports: []string{"fmt"},
		},
		{
			name: "script syntax errors do not stop import discovery",
			source: `<template><p>hi</p></template>
<script type="application/x-go">
package main

import "fmt"

func broken( {
</script>`,
			wantImports: []string{"fmt"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			imports, err := pkScanner{}.Imports(context.Background(), "pages/broken.pk", []byte(testCase.source))

			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				require.Nil(t, imports)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.wantImports, imports)
		})
	}
}

func TestProvidedSymbolPathsIncludesStandardLibraryAndRuntime(t *testing.T) {
	t.Parallel()

	provided := providedSymbolPaths()

	require.Contains(t, provided, "fmt", "the standard library is already provided")
	require.Contains(t, provided, "piko.sh/piko", "piko's runtime API is already provided")
	require.Len(t, provided, len(pipit.StandardLibraryPaths())+len(interp_piko_symbols.Symbols))
}

func writeExtractProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	files := map[string]string{
		"go.mod":                       "module example.com/fixture\n\ngo 1.22\n\nrequire example.com/dep v0.0.0\n\nreplace example.com/dep => ./dep\n",
		"dep/go.mod":                   "module example.com/dep\n\ngo 1.22\n",
		"dep/thing/thing.go":           "package thing\n\nfunc Value() int { return 1 }\n",
		"dep/nativeonly/nativeonly.go": "package nativeonly\n\nfunc Value() int { return 2 }\n",
		"pkg/local/local.go":           "package local\n\nimport \"example.com/dep/thing\"\n\nfunc Something() int { return thing.Value() }\n",
		"pkg/local/local_native.go":    "//go:build !interp\n\npackage local\n\nimport \"example.com/dep/nativeonly\"\n\nvar _ = nativeonly.Value\n",
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import (
	"fmt"
	"example.com/fixture/pkg/local"
)

var (
	_ = fmt.Sprint
	_ = local.Something
)
</script>`,
		"pages/missing.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import "example.com/missing/pkg"

var _ = pkg.Value
</script>`,
		"scripts/tool.go": "package main\n\nimport \"expvar\"\n\nvar _ = expvar.Do\n",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return root
}

func TestRunExtractDiscoverReadsPKProjects(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		reason    string
		arguments []string
		want      []string
	}{
		{
			name:      "follows the project's own packages by default",
			arguments: nil,
			want:      []string{"example.com/dep/thing"},
			reason: "the project's packages are compiled from source, so the third-party package pkg/local " +
				"imports is required, the !interp file's import is not, and fmt is already provided",
		},
		{
			name:      "--include-project reports the project's packages instead",
			arguments: []string{"--include-project"},
			want:      []string{"example.com/fixture/pkg/local"},
			reason:    "the old behaviour lists the project package the .pk script imports and does not follow it",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := writeExtractProject(t)
			var stdout, stderr bytes.Buffer
			code := RunExtractWithIO(append([]string{"discover", "--root", root}, testCase.arguments...), &stdout, &stderr)

			require.Equal(t, 0, code, stderr.String())
			require.Equal(t, testCase.want, strings.Fields(stdout.String()), testCase.reason)
		})
	}
}

func TestRunExtractDiscoverReachesActionsThroughPages(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	files := map[string]string{
		"go.mod":                    "module example.com/fixture\n\ngo 1.22\n\nrequire example.com/dep v0.0.0\n\nreplace example.com/dep => ./dep\n",
		"dep/go.mod":                "module example.com/dep\n\ngo 1.22\n",
		"dep/used/used.go":          "package used\n\nfunc Value() int { return 1 }\n",
		"dep/unused/unused.go":      "package unused\n\nfunc Value() int { return 2 }\n",
		"actions/contact/submit.go": "package contact\n\nimport \"example.com/dep/used\"\n\nvar Value = used.Value\n",
		"actions/orphan/orphan.go":  "package orphan\n\nimport \"example.com/dep/unused\"\n\nvar Value = unused.Value\n",
		"pages/contact.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import "example.com/fixture/actions/contact"

var _ = contact.Value
</script>`,
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}

	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"discover", "--root", root}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())
	require.Equal(t, []string{"example.com/dep/used"}, strings.Fields(stdout.String()),
		"an action a page imports is followed; one no page imports is never compiled, so its imports are not needed")
}

func TestRunExtractDiscoverWarnsAboutUnresolvedImports(t *testing.T) {
	t.Parallel()

	root := writeExtractProject(t)
	var stdout, stderr bytes.Buffer
	code := RunExtractWithIO([]string{"discover", "--root", root}, &stdout, &stderr)

	require.Equal(t, 0, code, stderr.String())
	require.Contains(t, stderr.String(), "warning: example.com/missing/pkg cannot be extracted:",
		"an import no module provides is reported rather than silently dropped")
	require.NotContains(t, stdout.String(), "example.com/missing/pkg")
}

func TestRunExtractHelpDocumentsIncludeProject(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"discover", "init", "check"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := RunExtractWithIO([]string{name, "--help"}, &stdout, &stderr)

			require.Equal(t, 0, code)
			require.Contains(t, stdout.String(), "--include-project")
		})
	}
}
