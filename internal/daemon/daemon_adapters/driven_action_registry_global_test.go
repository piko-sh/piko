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

package daemon_adapters

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearGlobalActionRegistry_EmptyAfterClear(t *testing.T) {
	ClearGlobalActionRegistry()
	result := GetGlobalActionRegistry()
	assert.Empty(t, result)
}

func TestRegisterAction_SingleEntry(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	entry := ActionHandlerEntry{Name: "test.action", Method: "POST"}
	RegisterAction(entry)

	registry := GetGlobalActionRegistry()
	require.Len(t, registry, 1)
	assert.Equal(t, "test.action", registry["test.action"].Name)
	assert.Equal(t, "POST", registry["test.action"].Method)
}

func TestRegisterAction_MultipleEntries(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	RegisterAction(ActionHandlerEntry{Name: "action1", Method: "GET"})
	RegisterAction(ActionHandlerEntry{Name: "action2", Method: "POST"})

	registry := GetGlobalActionRegistry()
	assert.Len(t, registry, 2)
}

func TestRegisterAction_OverwritesExisting(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	RegisterAction(ActionHandlerEntry{Name: "action", Method: "GET"})
	RegisterAction(ActionHandlerEntry{Name: "action", Method: "POST"})

	registry := GetGlobalActionRegistry()
	require.Len(t, registry, 1)
	assert.Equal(t, "POST", registry["action"].Method)
}

func TestRegisterAction_PreservesAllFields(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	createFunction := func() any { return nil }
	invokeFunction := func(_ context.Context, action any, arguments map[string]any) (any, error) { return nil, nil }
	mw := func(next http.Handler) http.Handler { return next }

	entry := ActionHandlerEntry{
		Name:        "full.action",
		Method:      "PUT",
		Create:      createFunction,
		Invoke:      invokeFunction,
		HasSSE:      true,
		Middlewares: []func(http.Handler) http.Handler{mw},
	}
	RegisterAction(entry)

	registry := GetGlobalActionRegistry()
	got := registry["full.action"]
	assert.Equal(t, "full.action", got.Name)
	assert.Equal(t, "PUT", got.Method)
	assert.True(t, got.HasSSE)
	assert.NotNil(t, got.Create)
	assert.NotNil(t, got.Invoke)
	require.Len(t, got.Middlewares, 1)
}

func TestRegisterActions_MultipleAtOnce(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	entries := map[string]ActionHandlerEntry{
		"batch.action1": {Method: "GET"},
		"batch.action2": {Method: "POST"},
	}
	RegisterActions(entries)

	registry := GetGlobalActionRegistry()
	require.Len(t, registry, 2)
	assert.Equal(t, "batch.action1", registry["batch.action1"].Name)
	assert.Equal(t, "batch.action2", registry["batch.action2"].Name)
}

func TestRegisterActions_SetsNameFromMapKey(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	entries := map[string]ActionHandlerEntry{
		"correct.name": {Name: "wrong.name", Method: "GET"},
	}
	RegisterActions(entries)

	registry := GetGlobalActionRegistry()
	assert.Equal(t, "correct.name", registry["correct.name"].Name)
}

func TestRegisterActions_EmptyMap(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	RegisterActions(map[string]ActionHandlerEntry{})

	registry := GetGlobalActionRegistry()
	assert.Empty(t, registry)
}

func TestGetGlobalActionRegistry_ReturnsCopy(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	RegisterAction(ActionHandlerEntry{Name: "action", Method: "GET"})

	copy1 := GetGlobalActionRegistry()
	copy1["action"] = ActionHandlerEntry{Name: "modified"}

	copy2 := GetGlobalActionRegistry()
	assert.Equal(t, "action", copy2["action"].Name)
}

func TestGetGlobalActionRegistry_EmptyWhenNothingRegistered(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	registry := GetGlobalActionRegistry()
	assert.NotNil(t, registry)
	assert.Empty(t, registry)
}

func TestClearGlobalActionRegistry_ClearsAllEntries(t *testing.T) {
	ClearGlobalActionRegistry()

	RegisterAction(ActionHandlerEntry{Name: "a", Method: "GET"})
	RegisterAction(ActionHandlerEntry{Name: "b", Method: "POST"})
	RegisterAction(ActionHandlerEntry{Name: "c", Method: "PUT"})

	require.Len(t, GetGlobalActionRegistry(), 3)

	ClearGlobalActionRegistry()

	assert.Empty(t, GetGlobalActionRegistry())
}

func TestClearGlobalActionRegistry_CanReRegisterAfterClear(t *testing.T) {
	ClearGlobalActionRegistry()
	defer ClearGlobalActionRegistry()

	RegisterAction(ActionHandlerEntry{Name: "first", Method: "GET"})
	ClearGlobalActionRegistry()

	RegisterAction(ActionHandlerEntry{Name: "second", Method: "POST"})

	registry := GetGlobalActionRegistry()
	require.Len(t, registry, 1)
	assert.Equal(t, "second", registry["second"].Name)
	assert.Equal(t, "POST", registry["second"].Method)
}
