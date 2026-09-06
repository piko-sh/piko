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

package dbschema_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

var (
	forbiddenDependencies = []string{
		"piko.sh/piko/internal/bootstrap",
		"github.com/evanw/esbuild",
	}
)

func TestPackageDoesNotLinkTheFramework(t *testing.T) {
	t.Parallel()

	loaded, err := packages.Load(
		&packages.Config{Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps},
		"piko.sh/piko/wdk/dbschema",
	)
	require.NoError(t, err, "the package graph must load")
	require.Len(t, loaded, 1, "exactly one package must match")

	reached := make(map[string]struct{})
	var walk func(pkg *packages.Package)
	walk = func(pkg *packages.Package) {
		for path, imported := range pkg.Imports {
			if _, seen := reached[path]; seen {
				continue
			}
			reached[path] = struct{}{}
			walk(imported)
		}
	}
	walk(loaded[0])

	require.NotEmpty(t, reached, "the dependency walk must find something")

	for path := range reached {
		for _, forbidden := range forbiddenDependencies {
			assert.False(t, strings.HasPrefix(path, forbidden),
				"wdk/dbschema must not depend on %s, reached via %s", forbidden, path)
		}
	}
}
