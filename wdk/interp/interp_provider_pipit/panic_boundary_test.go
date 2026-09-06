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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/modules"
)

func TestInterpreterPanicError(t *testing.T) {
	t.Parallel()
	err := interpreterPanicError(context.Background(), "running a test operation", "interpreter exploded")
	require.ErrorIs(t, err, errInterpreterPanic)
	assert.Contains(t, err.Error(), "interpreter exploded")
	assert.Contains(t, err.Error(), "running a test operation")
	assert.NotContains(t, err.Error(), "goroutine")
}

func TestInterpreterBoundariesRecoverPanics(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		call func() error
		name string
	}{
		{
			name: "compile and execute",
			call: func() error {
				adapter := &interpreterAdapter{service: nil, bytecodeEmissionDirectory: ""}
				return adapter.CompileAndExecute(context.Background(), "main", map[string]map[string]string{"main": {"main.go": "package main\n"}})
			},
		},
		{
			name: "module load",
			call: func() error {
				pool := newPoolAdapter(nil, []pendingModuleLoad{{bundle: nil, ref: modules.ModuleRef{Path: "example.com/panicking", Version: "", Pin: ""}}}, "")
				return pool.LoadModules(context.Background())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var err error
			require.NotPanics(t, func() { err = testCase.call() })
			require.ErrorIs(t, err, errInterpreterPanic)
		})
	}
}
