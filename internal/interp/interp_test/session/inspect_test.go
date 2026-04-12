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

package session_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_domain"
)

func TestSession_Inspect_EmptyOnFresh(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	state := sess.Inspect()
	require.Empty(t, state.Imports)
	require.Empty(t, state.Declarations)
	require.Zero(t, state.SubmitCount)
}

func TestSession_Inspect_ReportsKinds(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "var x = 1")
	mustSubmit(t, sess, "func f() int { return 0 }")
	mustSubmit(t, sess, "type T int")
	mustSubmit(t, sess, "const K = 42")
	state := sess.Inspect()

	kinds := map[string]interp_domain.SessionDeclKind{}
	for _, d := range state.Declarations {
		kinds[d.Name] = d.Kind
	}
	require.Equal(t, interp_domain.SessionDeclVar, kinds["x"])
	require.Equal(t, interp_domain.SessionDeclFunc, kinds["f"])
	require.Equal(t, interp_domain.SessionDeclType, kinds["T"])
	require.Equal(t, interp_domain.SessionDeclConst, kinds["K"])
}

func TestSession_Inspect_ReportsImports(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, `import "strings"`)
	mustSubmit(t, sess, `import "fmt"`)
	state := sess.Inspect()
	require.ElementsMatch(t, []string{"strings", "fmt"}, state.Imports)
}

func TestSession_Inspect_AfterResetReportsEmpty(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "var x = 1")
	mustSubmit(t, sess, `import "strings"`)
	sess.Reset()
	state := sess.Inspect()
	require.Empty(t, state.Imports)
	require.Empty(t, state.Declarations)
	require.Zero(t, state.SubmitCount)
}

func TestSession_Inspect_SubmitCountIncrements(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "1+1")
	mustSubmit(t, sess, "2+2")
	mustSubmit(t, sess, "3+3")
	require.Equal(t, 3, sess.Inspect().SubmitCount)
}
