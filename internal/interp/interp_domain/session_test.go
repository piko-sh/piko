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

package interp_domain

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSession_NewSessionWiresService(t *testing.T) {
	t.Parallel()
	service := NewService()
	session := service.NewSession()

	require.NotNil(t, session)
	require.Same(t, service, session.Service(),
		"Service() should return the originating Service pointer")
}

func TestSession_InspectInitialState(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()
	state := session.Inspect()

	require.Empty(t, state.Imports)
	require.Empty(t, state.Declarations)
	require.Equal(t, 0, state.SubmitCount)
}

func TestSession_SubmitConstExpression(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()
	result, err := session.Submit(context.Background(), "1 + 2")
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestSession_SubmitDeclThenUseAcrossSubmits(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `var x = 10`)
	require.NoError(t, err)

	result, err := session.Submit(context.Background(), `x * 2`)
	require.NoError(t, err)
	require.Equal(t, int64(20), result)
}

func TestSession_InspectAfterSubmitTracksDecl(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `var counter = 0`)
	require.NoError(t, err)

	state := session.Inspect()
	require.Len(t, state.Declarations, 1)
	require.Equal(t, SessionDecl{Name: "counter", Kind: SessionDeclVar}, state.Declarations[0])
	require.Equal(t, 1, state.SubmitCount)
}

func TestSession_ResetClearsAllState(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()
	_, err := session.Submit(context.Background(), `var v = 1`)
	require.NoError(t, err)
	require.Len(t, session.Inspect().Declarations, 1)

	session.Reset()

	state := session.Inspect()
	require.Empty(t, state.Imports)
	require.Empty(t, state.Declarations)
	require.Equal(t, 0, state.SubmitCount)
	require.Empty(t, session.CompiledFunctions(),
		"CompiledFunctions after Reset should be empty")
}

func TestSession_RedeclareSameNameIsError(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `var v = 1`)
	require.NoError(t, err)

	_, err = session.Submit(context.Background(), `var v = 2`)
	require.Error(t, err, "redeclaring v in a subsequent submit should fail")
}

func TestSession_RedeclareAfterResetSucceeds(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `var v = 1`)
	require.NoError(t, err)

	session.Reset()

	_, err = session.Submit(context.Background(), `var v = 2`)
	require.NoError(t, err, "after Reset, v should be declarable again")
}

func TestSession_ShortVarAtTopLevelIsLifted(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `x := 5`)
	require.NoError(t, err)

	state := session.Inspect()
	require.Len(t, state.Declarations, 1)
	require.Equal(t, "x", state.Declarations[0].Name)
	require.Equal(t, SessionDeclVar, state.Declarations[0].Kind)
}

func TestSession_SetStderrCaptured(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()
	var buffer bytes.Buffer
	session.SetStderr(&buffer)
}

func TestSession_FunctionDeclVisibleAcrossSubmits(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `func double(x int) int { return x * 2 }`)
	require.NoError(t, err)

	result, err := session.Submit(context.Background(), `double(21)`)
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestSession_TypeDeclVisibleAcrossSubmits(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `type Point struct { X, Y int }`)
	require.NoError(t, err)

	state := session.Inspect()
	require.Len(t, state.Declarations, 1)
	require.Equal(t, SessionDecl{Name: "Point", Kind: SessionDeclType}, state.Declarations[0])
}

func TestSession_MultipleSubmitsIncrementCount(t *testing.T) {
	t.Parallel()
	session := NewService().NewSession()

	_, err := session.Submit(context.Background(), `var a = 1`)
	require.NoError(t, err)
	_, err = session.Submit(context.Background(), `var b = 2`)
	require.NoError(t, err)
	_, err = session.Submit(context.Background(), `a + b`)
	require.NoError(t, err)

	require.Equal(t, 3, session.Inspect().SubmitCount)
}

func TestSession_DeclKindStringRoundTrip(t *testing.T) {
	t.Parallel()
	require.Equal(t, "var", SessionDeclVar.String())
	require.Equal(t, "const", SessionDeclConst.String())
	require.Equal(t, "func", SessionDeclFunc.String())
	require.Equal(t, "type", SessionDeclType.String())
}
