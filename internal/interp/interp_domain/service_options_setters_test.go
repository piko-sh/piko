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
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	optionSetterTestBudget = 12345
)

func TestWithMaxArenaSizeBytes_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxArenaSizeBytes(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, uint64(optionSetterTestBudget), service.config.maxArenaBytes,
		"WithMaxArenaSizeBytes should set the canonical maxArenaBytes field")
	require.Equal(t, uint64(optionSetterTestBudget), service.config.maxArenaSizeBytes,
		"WithMaxArenaSizeBytes should also set the mirror maxArenaSizeBytes alias")
}

func TestWithMaxConstantPoolSize_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxConstantPoolSize(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxConstantPoolSize)
}

func TestWithMaxSpecialisations_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxSpecialisations(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxSpecialisations)
}

func TestWithMaxMethods_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxMethods(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxMethods)
}

func TestWithMaxExpressionDepth_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxExpressionDepth(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxExpressionDepth)
}

func TestWithVerifierIterationLimit_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithVerifierIterationLimit(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.verifierIterationLimit)
}

func TestWithMaxSourceSize_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxSourceSize(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxSourceSize)
}

func TestWithMaxStringSize_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxStringSize(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxStringSize)
}

func TestWithMaxLiteralElements_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxLiteralElements(optionSetterTestBudget))

	require.NotNil(t, service.config)
	require.Equal(t, optionSetterTestBudget, service.config.maxLiteralElements)
}

func TestWithYieldInterval_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithYieldInterval(uint32(optionSetterTestBudget)))

	require.NotNil(t, service.config)
	require.Equal(t, uint32(optionSetterTestBudget), service.config.yieldInterval)
}

func TestWithCostBudget_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithCostBudget(int64(optionSetterTestBudget)))

	require.NotNil(t, service.config)
	require.Equal(t, int64(optionSetterTestBudget), service.config.costBudget)
}

func TestWithCostTable_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	table := &CostTable{}
	service := NewService(WithCostTable(table))

	require.NotNil(t, service.config)
	require.Same(t, table, service.config.costTable,
		"WithCostTable must record the exact pointer for identity-checks")
}

func TestWithFeatures_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	const restrictedFeatures InterpFeature = 1
	service := NewService(WithFeatures(restrictedFeatures))

	require.NotNil(t, service.config)
	require.Equal(t, restrictedFeatures, service.config.features)
}

func TestWithBytecodeVerification_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithBytecodeVerification(false))

	require.NotNil(t, service.config)
	require.True(t, service.config.bytecodeVerificationDisabled,
		"WithBytecodeVerification(false) should record disable flag")

	service = NewService(WithBytecodeVerification(true))
	require.False(t, service.config.bytecodeVerificationDisabled,
		"WithBytecodeVerification(true) should clear disable flag")
}

func TestWithDeniedImports_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithDeniedImports("unsafe", "runtime"))

	require.NotNil(t, service.config)
	require.Contains(t, service.config.deniedImports, "unsafe")
	require.Contains(t, service.config.deniedImports, "runtime")
}

func TestWithImportAllowlist_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	service := NewService(WithImportAllowlist("host/widgets"))

	require.NotNil(t, service.config)
	require.Contains(t, service.config.allowedImports, "host/widgets")
	require.Contains(t, service.config.allowedImports, importAllowlistSentinel,
		"a configured allowlist must stay non-empty so it is distinguishable from no allowlist")

	empty := NewService(WithImportAllowlist())
	require.NotEmpty(t, empty.config.allowedImports,
		"an empty allowlist must still install the sentinel so it denies all external imports")
}

func TestWithCompilationSnapshot_PlumbedToConfig(t *testing.T) {
	t.Parallel()
	var captured *CompiledFileSet
	callback := func(cfs *CompiledFileSet) { captured = cfs }

	service := NewService(WithCompilationSnapshot(callback))

	require.NotNil(t, service.config)
	require.NotNil(t, service.config.compilationSnapshotCallback,
		"WithCompilationSnapshot must record the callback")
	_ = captured
}
