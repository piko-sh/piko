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

package templates

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProject_GeneratesParseableGo(t *testing.T) {
	t.Parallel()

	dest := filepath.Join(t.TempDir(), "smoke-project")
	data := ScaffoldData{
		ProjectName:       "smoke-project",
		ModuleName:        "example.com/smoke-project",
		DestinationPath:   dest,
		PikoVersion:       "v0.0.0",
		EnableInterpreted: true,
		EnableAgents:      true,
		EnableValidator:   true,
		EnableSonicJSON:   true,
	}
	err := CreateProject(data)
	require.NoErrorf(t, err, "CreateProject() = %v", err)

	goMod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	require.NoErrorf(t, err, "reading generated go.mod: %v", err)
	assert.Containsf(t, string(goMod), "module example.com/smoke-project", "generated go.mod is missing the module declaration:\n%s", goMod)

	assert.Truef(t, regexp.MustCompile(`(?m)^\s*piko\.sh/piko\s+v`).Match(goMod), "generated go.mod is missing the framework root require:\n%s", goMod)

	fset := token.NewFileSet()
	goFiles := 0
	walkErr := filepath.WalkDir(dest, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		goFiles++
		if _, perr := parser.ParseFile(fset, path, nil, parser.AllErrors|parser.SkipObjectResolution); perr != nil {
			rel, _ := filepath.Rel(dest, path)
			assert.Failf(t, "generated Go file does not parse", "generated %s does not parse: %v", rel, perr)
		}
		return nil
	})
	require.NoErrorf(t, walkErr, "walking the scaffold: %v", walkErr)
	require.NotZero(t, goFiles, "scaffold produced no .go files")
}
