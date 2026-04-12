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

func TestSession_ShortVar_Simple(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "a := 1")
	require.Equal(t, int64(1), mustSubmit(t, sess, "a"))
}

func TestSession_ShortVar_InferredStringType(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, `s := "hi"`)
	require.Equal(t, int64(2), mustSubmit(t, sess, "len(s)"))
}

func TestSession_ShortVar_InferredFloatType(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, "f := 1.5")
	require.InDelta(t, 4.5, mustSubmit(t, sess, "f * 3"), 1e-9)
}

func TestSession_ShortVar_MultiValueReturnsClearError(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	_, err := sess.Submit(context.Background(), "x, y := 1, 2")
	require.Error(t, err)
	require.ErrorContains(t, err, "var")
}

func TestSession_ShortVar_BlankIdentReturnsClearError(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	_, err := sess.Submit(context.Background(), "_ := 1")
	require.Error(t, err)
}

func TestSession_ShortVar_InsideIfNotRewritten(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)

	result, err := sess.Submit(context.Background(), `if x := 5; x > 0 { _ = x }
0`)
	require.NoError(t, err)
	require.Equal(t, int64(0), result)

	_, err = sess.Submit(context.Background(), "x")
	require.Error(t, err)
}
