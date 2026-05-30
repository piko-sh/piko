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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSession_FuncRedecl_ErrorsButPreservesState(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "func f() int { return 1 }")
	_, err := sess.Submit(context.Background(), "func f() int { return 2 }")
	require.Error(t, err)
	require.ErrorContains(t, err, "redeclared")
	require.Equal(t, int64(1), mustSubmit(t, sess, "f()"))
}

func TestSession_VarRedecl_ErrorsButPreservesState(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "var x = 1")
	_, err := sess.Submit(context.Background(), "var x = 2")
	require.Error(t, err)
	require.ErrorContains(t, err, "redeclared")
	require.Equal(t, int64(1), mustSubmit(t, sess, "x"))
}

func TestSession_TypeRedecl_ErrorsButPreservesState(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "type T struct{ N int }")
	_, err := sess.Submit(context.Background(), "type T struct{ M int }")
	require.Error(t, err)
	require.ErrorContains(t, err, "redeclared")
	require.Equal(t, int64(7), mustSubmit(t, sess, "T{N: 7}.N"))
}

func TestSession_ConstRedecl_ErrorsButPreservesState(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "const K = 5")
	_, err := sess.Submit(context.Background(), "const K = 9")
	require.Error(t, err)
	require.ErrorContains(t, err, "redeclared")
	require.Equal(t, int64(15), mustSubmit(t, sess, "K * 3"))
}

func TestSession_DifferentKindRedecl_Errors(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "var f = 1")
	_, err := sess.Submit(context.Background(), "func f() {}")
	require.Error(t, err)
	require.ErrorContains(t, err, "redeclared")
	require.Equal(t, int64(1), mustSubmit(t, sess, "f"))
}

func TestSession_ShortVarRedecl_ErrorsButPreservesState(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "x := 1")
	_, err := sess.Submit(context.Background(), "x := 2")
	require.Error(t, err)
	require.ErrorContains(t, err, "redeclared")
	require.Equal(t, int64(1), mustSubmit(t, sess, "x"))
}
