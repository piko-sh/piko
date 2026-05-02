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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestAsyncExecPass_HardErrorOnUnsupportedEngine(t *testing.T) {
	t.Parallel()

	pass := &asyncExecPass{}
	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "PurgeOld",
			Command: querier_dto.QueryCommandAsyncExec,
			Line:    7,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{},
		Engine: &mockEngine{
			dialectFn:                func() string { return "postgres" },
			supportsAsyncMutationsFn: func() bool { return false },
		},
	}

	diagnostics := pass.Analyse(context)

	require.Len(t, diagnostics, 1)
	assert.Equal(t, querier_dto.CodeAsyncExecNotSupported, diagnostics[0].Code)
	assert.Equal(t, querier_dto.SeverityError, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, `"PurgeOld"`)
	assert.Contains(t, diagnostics[0].Message, "postgres")
	assert.Contains(t, diagnostics[0].Message, "asynchronous mutation semantics")
}

func TestAsyncExecPass_AcceptsAsyncExecOnSupportingEngine(t *testing.T) {
	t.Parallel()

	pass := &asyncExecPass{}
	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "PurgeOld",
			Command: querier_dto.QueryCommandAsyncExec,
			Line:    7,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			EngineSpecific: map[string]string{
				asyncExecBodyKey: "UPDATE name = 'x'",
			},
		},
		Engine: &mockEngine{
			dialectFn:                func() string { return "clickhouse" },
			supportsAsyncMutationsFn: func() bool { return true },
		},
	}

	diagnostics := pass.Analyse(context)
	assert.Empty(t, diagnostics)
}

func TestAsyncExecPass_RecommendsAsyncExecForExecOnSupportingEngine(t *testing.T) {
	t.Parallel()

	pass := &asyncExecPass{}
	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "PurgeOld",
			Command: querier_dto.QueryCommandExec,
			Line:    9,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			EngineSpecific: map[string]string{
				asyncExecBodyKey: "UPDATE name = 'x' WHERE id = 1",
			},
		},
		Engine: &mockEngine{
			dialectFn:                func() string { return "clickhouse" },
			supportsAsyncMutationsFn: func() bool { return true },
		},
	}

	diagnostics := pass.Analyse(context)

	require.Len(t, diagnostics, 1)
	assert.Equal(t, querier_dto.CodeAsyncExecRecommended, diagnostics[0].Code)
	assert.Equal(t, querier_dto.SeverityHint, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, `"PurgeOld"`)
	assert.Contains(t, diagnostics[0].Message, "asyncexec")
}

func TestAsyncExecPass_QuietOnExecWithoutAsyncMarker(t *testing.T) {
	t.Parallel()

	pass := &asyncExecPass{}
	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "InsertRow",
			Command: querier_dto.QueryCommandExec,
			Line:    3,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{},
		Engine: &mockEngine{
			dialectFn:                func() string { return "clickhouse" },
			supportsAsyncMutationsFn: func() bool { return true },
		},
	}

	diagnostics := pass.Analyse(context)
	assert.Empty(t, diagnostics)
}

func TestAsyncExecPass_QuietOnExecWithAsyncMarkerButUnsupportedEngine(t *testing.T) {
	t.Parallel()

	pass := &asyncExecPass{}
	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "ExecQuery",
			Command: querier_dto.QueryCommandExec,
			Line:    3,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			EngineSpecific: map[string]string{
				asyncExecBodyKey: "UPDATE name = 'x'",
			},
		},
		Engine: &mockEngine{
			dialectFn:                func() string { return "postgres" },
			supportsAsyncMutationsFn: func() bool { return false },
		},
	}

	diagnostics := pass.Analyse(context)
	assert.Empty(t, diagnostics)
}

func TestAsyncExecPass_QuietWhenEngineIsNil(t *testing.T) {
	t.Parallel()

	pass := &asyncExecPass{}
	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "PurgeOld",
			Command: querier_dto.QueryCommandAsyncExec,
			Line:    7,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{},
		Engine:      nil,
	}

	diagnostics := pass.Analyse(context)
	assert.Empty(t, diagnostics)
}

func TestAsyncExecPass_QuietOnNonExecCommands(t *testing.T) {
	t.Parallel()

	commands := []struct {
		name    string
		command querier_dto.QueryCommand
	}{
		{name: "one", command: querier_dto.QueryCommandOne},
		{name: "many", command: querier_dto.QueryCommandMany},
		{name: "execresult", command: querier_dto.QueryCommandExecResult},
		{name: "execrows", command: querier_dto.QueryCommandExecRows},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pass := &asyncExecPass{}
			context := &diagnosticContext{
				Filename: "test.sql",
				Query: &querier_dto.AnalysedQuery{
					Name:    "Q",
					Command: tc.command,
					Line:    1,
				},
				RawAnalysis: &querier_dto.RawQueryAnalysis{
					EngineSpecific: map[string]string{
						asyncExecBodyKey: "UPDATE x = 1",
					},
				},
				Engine: &mockEngine{
					dialectFn:                func() string { return "clickhouse" },
					supportsAsyncMutationsFn: func() bool { return true },
				},
			}

			diagnostics := pass.Analyse(context)
			assert.Empty(t, diagnostics)
		})
	}
}

func TestHasAsyncBodyMarker(t *testing.T) {
	t.Parallel()

	t.Run("nil analysis returns false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasAsyncBodyMarker(nil))
	})

	t.Run("nil engine specific map returns false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, hasAsyncBodyMarker(&querier_dto.RawQueryAnalysis{}))
	})

	t.Run("present marker returns true", func(t *testing.T) {
		t.Parallel()

		analysis := &querier_dto.RawQueryAnalysis{
			EngineSpecific: map[string]string{asyncExecBodyKey: "body"},
		}
		assert.True(t, hasAsyncBodyMarker(analysis))
	})

	t.Run("unrelated marker returns false", func(t *testing.T) {
		t.Parallel()

		analysis := &querier_dto.RawQueryAnalysis{
			EngineSpecific: map[string]string{"OTHER_KEY": "x"},
		}
		assert.False(t, hasAsyncBodyMarker(analysis))
	})
}
