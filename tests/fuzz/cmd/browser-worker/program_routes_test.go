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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestBrowserProgramPathsValidatesRouteList(t *testing.T) {
	paths, err := browserProgramPaths(`["/a","/b?x=1"]`)
	require.NoError(t, err)
	assert.Equal(t, []string{"/a", "/b?x=1"}, paths)
	for _, raw := range []string{"{", strings.Repeat(" ", maxProgramPathsBytes+1), "[" + strings.TrimSuffix(strings.Repeat(`"/a",`, casefile.MaxBrowserProgramRoutes+1), ",") + "]"} {
		_, err := browserProgramPaths(raw)
		require.ErrorIs(t, err, errInvalidProgramPaths)
	}
	_, err = browserProgramPaths(`["https://elsewhere.invalid/"]`)
	require.Error(t, err)
}

func TestProgramRoutesBudget(t *testing.T) {
	assert.Zero(t, programRoutesBudget(nil))
	assert.Equal(t, 3*routeBudget, programRoutesBudget([]string{"/a", "/b", "/c"}))
}
