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

package main

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerUsesPikoBrowser(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, path := range paths {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, spec := range file.Imports {
			name, err := strconv.Unquote(spec.Path.Value)
			require.NoError(t, err)
			direct := strings.HasPrefix(name, "github.com/chromedp/") || name == "github.com/gobwas/ws" || strings.HasPrefix(name, "github.com/gobwas/ws/")
			assert.False(t, direct, "%s imports %s directly, use piko.sh/piko/wdk/browser", path, name)
		}
	}
}

func TestScriptStringQuotesText(t *testing.T) {
	quoted, err := scriptString("a\"b</script>\U0001F600")
	require.NoError(t, err)
	assert.Equal(t, "\"a\\\"b\\u003c/script\\u003e\U0001F600\"", quoted)
}

func TestSessionFailureNamesStepAndBudget(t *testing.T) {
	failure := errors.New("node not found")
	assert.Equal(t, "click: node not found", sessionFailure(t.Context(), "click", failure))
	assert.Equal(t, "node not found", sessionFailure(t.Context(), "", failure))

	expired, cancel := context.WithCancelCause(t.Context())
	cancel(errSessionBudget)
	assert.Equal(t, errSessionBudget.Error()+": click: node not found", sessionFailure(expired, "click", failure))
}

func TestStepContextCarriesStepCause(t *testing.T) {
	stepCtx, cancel := stepContext(t.Context(), 0)
	defer cancel()

	<-stepCtx.Done()

	require.ErrorIs(t, context.Cause(stepCtx), errStepBudget)
}
