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

package generate

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestFormShimSourceReplay(t *testing.T) {
	shapes := map[int]bool{}
	for index := range uint64(12) {
		c := New("form-shim-contracts", 7, index)
		want := c.Expected.FormShim
		require.NotNil(t, want, "missing form model")
		require.Equal(t, "/intake", want.Page)
		shapes[len(want.Steps)] = true
		page := c.Files["pages/intake.pk"]
		require.Equal(t, len(want.Steps), strings.Count(page, "<form "))
		for _, step := range want.Steps {
			require.Contains(t, page, `data-case="`+step.Name+`"`)
			require.Contains(t, page, step.Label)
			if step.Ticket != "" {
				require.Contains(t, page, step.Ticket)
			}
		}
		for _, name := range []string{"internal/formfixture/provider.go", "actions/intake/forms.go"} {
			_, err := parser.ParseFile(token.NewFileSet(), name, c.Files[name], parser.AllErrors)
			require.NoError(t, err, name)
		}
		if index == 0 {
			directory := filepath.Join(t.TempDir(), "saved")
			require.NoError(t, casefile.Write(directory, &c))
			round, err := casefile.Read(directory)
			require.NoError(t, err)
			assert.Equal(t, c.Hash(), round.Hash())
			assert.Equal(t, want, round.Expected.FormShim)
			assert.Equal(t, page, round.Files["pages/intake.pk"])
		}
	}
	assert.True(t, shapes[6], "missing sequence variant")
	assert.True(t, shapes[7], "missing quota variant")
}
