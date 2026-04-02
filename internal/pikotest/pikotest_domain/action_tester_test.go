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

package pikotest_domain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/pikotest/pikotest_domain"
)

type testAction struct {
	daemon_dto.ActionMetadata
}

func TestActionTester_Invoke_Success(t *testing.T) {
	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return map[string]string{"status": "ok"}, nil
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	result := tester.Invoke(context.Background(), nil)

	result.AssertSuccess()
	result.AssertNoHelpers()

	data, ok := result.Data().(map[string]string)
	require.True(t, ok, "expected map[string]string data")
	assert.Equal(t, "ok", data["status"])
}

func TestActionTester_Invoke_Error(t *testing.T) {
	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return nil, errors.New("something went wrong")
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	result := tester.Invoke(context.Background(), nil)

	result.AssertError()
	result.AssertErrorContains("something went wrong")
}

func TestActionTester_Invoke_WithArgs(t *testing.T) {
	var receivedArgs map[string]any

	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, arguments map[string]any) (any, error) {
			receivedArgs = arguments
			return "ok", nil
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	tester.Invoke(context.Background(), map[string]any{"name": "Alice", "age": 30})

	assert.Equal(t, "Alice", receivedArgs["name"])
	assert.Equal(t, 30, receivedArgs["age"])
}

func TestActionTester_Invoke_NilArgs_DefaultsToEmpty(t *testing.T) {
	var receivedArgs map[string]any

	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, arguments map[string]any) (any, error) {
			receivedArgs = arguments
			return nil, nil
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	tester.Invoke(context.Background(), nil)

	require.NotNil(t, receivedArgs, "nil arguments should be converted to empty map")
	assert.Empty(t, receivedArgs)
}

func TestActionTester_Invoke_WithContext(t *testing.T) {
	type ctxKey string

	var capturedCtx context.Context

	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(ctx context.Context, _ any, _ map[string]any) (any, error) {
			capturedCtx = ctx
			return nil, nil
		},
	}

	ctx := context.WithValue(context.Background(), ctxKey("db"), "mock-db")
	tester := pikotest_domain.NewActionTester(t, entry)
	tester.Invoke(ctx, nil)

	require.NotNil(t, capturedCtx)
	assert.Equal(t, "mock-db", capturedCtx.Value(ctxKey("db")))
}

func TestActionTester_Invoke_WithHelpers(t *testing.T) {
	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, action any, _ map[string]any) (any, error) {
			a, ok := action.(*testAction)
			if !ok {
				return nil, fmt.Errorf("expected *testAction, got %T", action)
			}
			a.Response().AddHelper("showToast", "Created", "success")
			return map[string]string{"id": "123"}, nil
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	result := tester.Invoke(context.Background(), nil)

	result.AssertSuccess()
	result.AssertHelper("showToast")
}

func TestActionResultView_Data(t *testing.T) {
	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return "hello world", nil
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	result := tester.Invoke(context.Background(), nil)

	assert.Equal(t, "hello world", result.Data())
}

func TestActionResultView_Err(t *testing.T) {
	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return nil, errors.New("test error")
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	result := tester.Invoke(context.Background(), nil)

	require.NotNil(t, result.Err())
	assert.Equal(t, "test error", result.Err().Error())
}

func TestActionResultView_Result(t *testing.T) {
	entry := daemon_adapters.ActionHandlerEntry{
		Name: "test.action",
		Create: func() any {
			return &testAction{}
		},
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return "data", nil
		},
	}

	tester := pikotest_domain.NewActionTester(t, entry)
	result := tester.Invoke(context.Background(), nil)

	raw := result.Result()
	require.NotNil(t, raw)
	require.NotNil(t, raw.Response)
	assert.Equal(t, "data", raw.Response.Data)
	assert.Nil(t, raw.Err)
}
