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

package testutil

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolchainEnvCapsProcs(t *testing.T) {
	t.Parallel()

	environment := ToolchainEnv()

	require.NotEmpty(t, environment)
	assert.Contains(t, environment, "GOMAXPROCS=2")
	assert.Len(t, environment, len(os.Environ())+1)
}

func TestToolchainEnvAppendsExtraAfterTheCap(t *testing.T) {
	t.Parallel()

	environment := ToolchainEnv("GOWORK=off", "CGO_ENABLED=0")

	capIndex := slices.Index(environment, "GOMAXPROCS=2")
	workIndex := slices.Index(environment, "GOWORK=off")
	cgoIndex := slices.Index(environment, "CGO_ENABLED=0")

	require.NotEqual(t, -1, capIndex)
	require.NotEqual(t, -1, workIndex)
	require.NotEqual(t, -1, cgoIndex)
	assert.Less(t, capIndex, workIndex)
	assert.Less(t, workIndex, cgoIndex)
}

func TestToolchainEnvLetsCallersOverrideTheCap(t *testing.T) {
	t.Parallel()

	environment := ToolchainEnv("GOMAXPROCS=8")

	last := slices.Index(environment, "GOMAXPROCS=8")
	require.NotEqual(t, -1, last)
	assert.Greater(t, last, slices.Index(environment, "GOMAXPROCS=2"))
}

func TestToolchainEnvDoesNotMutateTheProcessEnvironment(t *testing.T) {
	t.Parallel()

	before := os.Environ()
	_ = ToolchainEnv("GOWORK=off")

	assert.Equal(t, before, os.Environ())
}
