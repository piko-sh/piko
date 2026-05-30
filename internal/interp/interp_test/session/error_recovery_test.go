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

func TestSession_TypeErrorDoesNotMutateDecls(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "func f() int { return 1 }")
	_, err := sess.Submit(context.Background(), "this is not valid go +++")
	require.Error(t, err)
	require.Equal(t, int64(1), mustSubmit(t, sess, "f()"))
}

func TestSession_ParseErrorDoesNotMutateDecls(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "var x = 5")
	_, err := sess.Submit(context.Background(), "func broken( {")
	require.Error(t, err)
	require.Equal(t, int64(5), mustSubmit(t, sess, "x"))
}

func TestSession_UndefinedReferenceDoesNotMutateDecls(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "var x = 5")
	_, err := sess.Submit(context.Background(), "thisIdentifierDoesNotExist")
	require.Error(t, err)
	require.Equal(t, int64(5), mustSubmit(t, sess, "x"))
}

func TestSession_FailedDeclDoesNotPersist(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)

	_, err := sess.Submit(context.Background(), "func g() int { return missingSymbol }")
	require.Error(t, err)

	mustSubmit(t, sess, "func g() int { return 11 }")
	require.Equal(t, int64(11), mustSubmit(t, sess, "g()"))
}
