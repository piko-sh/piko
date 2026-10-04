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

package interp_provider_pipit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitSettingResolve(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		setting    limitSetting[int]
		restricted bool
		want       int
	}{
		{name: "unconfigured and unrestricted is unlimited", setting: limitSetting[int]{}, restricted: false, want: 0},
		{name: "unconfigured and restricted uses the default", setting: limitSetting[int]{}, restricted: true, want: 7},
		{name: "configured value wins when unrestricted", setting: limitSetting[int]{value: 3, configured: true}, restricted: false, want: 3},
		{name: "configured value wins when restricted", setting: limitSetting[int]{value: 3, configured: true}, restricted: true, want: 3},
		{name: "configured zero lifts the restricted default", setting: limitSetting[int]{value: 0, configured: true}, restricted: true, want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, testCase.setting.resolve(testCase.restricted, 7))
		})
	}
}

func TestResourceLimitOptions(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		options     []ProviderOption
		wantOptions int
	}{
		{name: "development use is unlimited by default", options: nil, wantOptions: 0},
		{name: "restricted mode applies six default caps", options: []ProviderOption{WithRestrictedSymbolSurface()}, wantOptions: 6},
		{
			name: "every limit configured",
			options: []ProviderOption{
				WithMaxExecutionTime(time.Second), WithCostBudget(10), WithMaxCallDepth(10), WithMaxAllocSize(10),
				WithMaxOutputSize(10), WithMaxSourceSize(10), WithMaxStringSize(10), WithMaxGoroutines(10),
			},
			wantOptions: 8,
		},
		{
			name:        "explicit zero lifts a restricted cap",
			options:     []ProviderOption{WithRestrictedSymbolSurface(), WithMaxExecutionTime(0)},
			wantOptions: 5,
		},
		{
			name:        "negative values are treated as unlimited",
			options:     []ProviderOption{WithMaxGoroutines(-1), WithCostBudget(-1), WithMaxCallDepth(-5)},
			wantOptions: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider := NewProvider(testCase.options...)
			assert.Len(t, provider.limits.pipitOptions(provider.restrictedSymbolSurface), testCase.wantOptions)
		})
	}
}

func TestResourceLimitsReachTheInterpreter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		source  string
		options []ProviderOption
	}{
		{
			name:    "execution time limit stops a runaway init",
			options: []ProviderOption{WithMaxExecutionTime(50 * time.Millisecond)},
			source:  "package main\n\nfunc init() {\n\tfor {\n\t}\n}\n",
		},
		{
			name:    "cost budget stops a runaway init",
			options: []ProviderOption{WithCostBudget(10_000)},
			source:  "package main\n\nfunc init() {\n\tfor {\n\t}\n}\n",
		},
		{
			name:    "source size limit rejects a large program",
			options: []ProviderOption{WithMaxSourceSize(16)},
			source:  "package main\n\nvar Value = 1\n",
		},
		{
			name:    "allocation limit rejects a large make",
			options: []ProviderOption{WithMaxAllocSize(8)},
			source:  "package main\n\nvar n = 1024\n\nvar Values = make([]int, n)\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			interpreter, err := NewProvider(testCase.options...).NewInterpreterPool().Get()
			require.NoError(t, err)
			err = interpreter.CompileAndExecute(context.Background(), "main", map[string]map[string]string{
				"main": {"main.go": testCase.source},
			})
			require.Error(t, err)
		})
	}
}
