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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestWithBeforeMigration(t *testing.T) {
	t.Parallel()

	hookCalled := false
	hook := BeforeMigrationHook(func(_ context.Context, _ MigrationHookContext) error {
		hookCalled = true
		return nil
	})

	service := &migrationService{}
	option := WithBeforeMigration(hook)
	option(service)

	require.Len(t, service.beforeMigrationHooks, 1, "hook should be appended to beforeMigrationHooks")

	err := service.beforeMigrationHooks[0](context.Background(), MigrationHookContext{})
	require.NoError(t, err)
	assert.True(t, hookCalled, "the registered hook should have been called")
}

func TestWithAfterMigration(t *testing.T) {
	t.Parallel()

	hookCalled := false
	hook := AfterMigrationHook(func(_ context.Context, _ MigrationHookContext) error {
		hookCalled = true
		return nil
	})

	service := &migrationService{}
	option := WithAfterMigration(hook)
	option(service)

	require.Len(t, service.afterMigrationHooks, 1, "hook should be appended to afterMigrationHooks")

	err := service.afterMigrationHooks[0](context.Background(), MigrationHookContext{})
	require.NoError(t, err)
	assert.True(t, hookCalled, "the registered hook should have been called")
}

func TestWithBeforeRun(t *testing.T) {
	t.Parallel()

	hookCalled := false
	hook := BeforeRunHook(func(_ context.Context, _ MigrationRunHookContext) error {
		hookCalled = true
		return nil
	})

	service := &migrationService{}
	option := WithBeforeRun(hook)
	option(service)

	require.Len(t, service.beforeRunHooks, 1, "hook should be appended to beforeRunHooks")

	err := service.beforeRunHooks[0](context.Background(), MigrationRunHookContext{})
	require.NoError(t, err)
	assert.True(t, hookCalled, "the registered hook should have been called")
}

func TestWithAfterRun(t *testing.T) {
	t.Parallel()

	hookCalled := false
	hook := AfterRunHook(func(_ context.Context, _ MigrationRunHookContext, _ int) error {
		hookCalled = true
		return nil
	})

	service := &migrationService{}
	option := WithAfterRun(hook)
	option(service)

	require.Len(t, service.afterRunHooks, 1, "hook should be appended to afterRunHooks")

	err := service.afterRunHooks[0](context.Background(), MigrationRunHookContext{}, 0)
	require.NoError(t, err)
	assert.True(t, hookCalled, "the registered hook should have been called")
}

func TestWithNonBlockingLock(t *testing.T) {
	t.Parallel()

	service := &migrationService{}
	assert.False(t, service.nonBlockingLock, "nonBlockingLock should default to false")

	option := WithNonBlockingLock()
	option(service)

	assert.True(t, service.nonBlockingLock, "nonBlockingLock should be true after applying the option")
}

func TestMultipleHooksAppend(t *testing.T) {
	t.Parallel()

	var callOrder []int

	hookA := BeforeMigrationHook(func(_ context.Context, _ MigrationHookContext) error {
		callOrder = append(callOrder, 1)
		return nil
	})
	hookB := BeforeMigrationHook(func(_ context.Context, _ MigrationHookContext) error {
		callOrder = append(callOrder, 2)
		return nil
	})

	service := &migrationService{}
	WithBeforeMigration(hookA)(service)
	WithBeforeMigration(hookB)(service)

	require.Len(t, service.beforeMigrationHooks, 2, "both hooks should be present")

	for _, hook := range service.beforeMigrationHooks {
		err := hook(context.Background(), MigrationHookContext{
			Name:      "test_migration",
			Version:   1,
			Direction: querier_dto.MigrationDirectionUp,
		})
		require.NoError(t, err)
	}

	assert.Equal(t, []int{1, 2}, callOrder, "hooks should execute in append order")
}
