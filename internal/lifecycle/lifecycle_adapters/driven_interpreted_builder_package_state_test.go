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

package lifecycle_adapters

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/safedisk"
)

func packageStateProject(t *testing.T) (string, map[string]map[string]string) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"cmd/main/main.go":                  "package main\n\nimport _ \"example.com/site/internal/publishing\"\n\nfunc main() {}\n",
		"cmd/main/main_test.go":             "package main\n\nimport _ \"example.com/site/internal/drafts\"\n",
		"internal/publishing/publishing.go": "package publishing\n\nimport _ \"example.com/site/internal/content\"\n\nvar Preview bool\n\nvar (\n\tLimit, offset = 10, 0\n)\n",
		"internal/content/content.go":       "package content\n\nvar files = 1\n\nconst Name = \"x\"\n",
		"internal/drafts/drafts.go":         "package drafts\n\nvar Shown bool\n",
	}
	for name, source := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	}
	userPackages := map[string]map[string]string{
		"internal/publishing": {"publishing.go": files["internal/publishing/publishing.go"]},
		"internal/content":    {"content.go": files["internal/content/content.go"]},
		"internal/drafts":     {"drafts.go": files["internal/drafts/drafts.go"]},
	}
	return root, userPackages
}

func TestWarnSeparatePackageStateNamesHostLinkedPackagesWithExportedVariables(t *testing.T) {
	t.Parallel()
	root, userPackages := packageStateProject(t)
	sandbox, err := safedisk.NewSandbox(root, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })
	output := new(bytes.Buffer)
	ctx := logger_domain.WithLogger(context.Background(),
		logger_domain.New(slog.New(slog.NewTextHandler(output, nil)), "package-state-test"))
	orchestrator := &InterpretedBuildOrchestrator{moduleName: "example.com/site"}

	orchestrator.warnSeparatePackageState(ctx, sandbox, userPackages, "example.com/site/cmd/main")
	orchestrator.warnSeparatePackageState(ctx, sandbox, userPackages, "example.com/site/cmd/main")

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	require.Len(t, lines, 1, "one warning, once, for the one linked package with exported variables:\n%s", output.String())
	assert.Contains(t, lines[0], "level=WARN")
	assert.Contains(t, lines[0], "pkg_path=internal/publishing")
	assert.Contains(t, lines[0], "variables=Limit,Preview")
	assert.Contains(t, lines[0], "piko extract --include-project")
}

func TestWarnSeparatePackageStateIgnoresAMainOutsideTheModule(t *testing.T) {
	t.Parallel()
	root, userPackages := packageStateProject(t)
	sandbox, err := safedisk.NewSandbox(root, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })
	output := new(bytes.Buffer)
	ctx := logger_domain.WithLogger(context.Background(),
		logger_domain.New(slog.New(slog.NewTextHandler(output, nil)), "package-state-test"))
	orchestrator := &InterpretedBuildOrchestrator{moduleName: "example.com/site"}

	orchestrator.warnSeparatePackageState(ctx, sandbox, userPackages, "piko.sh/piko/cmd/piko")

	assert.Empty(t, output.String())
}

func TestExportedPackageVariables(t *testing.T) {
	t.Parallel()
	got := exportedPackageVariables(map[string]string{
		"a.go":      "package p\n\nvar B, a = 1, 2\n\nvar (\n\tA int\n\tB2 = \"x\"\n)\n\nconst C = 3\n\nfunc D() {}\n",
		"b.go":      "package p\n\nvar A int\n",
		"broken.go": "package p\n\nvar (",
	})
	assert.Equal(t, []string{"A", "B", "B2"}, got)
	assert.Empty(t, exportedPackageVariables(map[string]string{"c.go": "package p\n\nvar hidden = 1\n"}))
}
