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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructFieldGeneralT0EmittedForMapField(t *testing.T) {
	t.Parallel()

	source := `package main

type Container struct {
	lookup map[int]int
}

func read(container *Container, key int) int {
	return container.lookup[key]
}

func main() {
	container := &Container{lookup: map[int]int{1: 11}}
	_ = read(container, 1)
}
`
	cfs := compileFileSource(t, source)
	readFn, err := cfs.FindFunction("read")
	require.NoError(t, err)

	requireContainsOpcode(t, readFn, opGetStructFieldGeneral)
	requireNoOpcode(t, readFn, opGetField)
}

func TestStructFieldGeneralT0EmittedForChanField(t *testing.T) {
	t.Parallel()

	source := `package main

type Container struct {
	signals chan int
}

func read(container *Container) chan int {
	return container.signals
}

func main() {
	_ = read(&Container{signals: make(chan int, 1)})
}
`
	cfs := compileFileSource(t, source)
	readFn, err := cfs.FindFunction("read")
	require.NoError(t, err)

	requireContainsOpcode(t, readFn, opGetStructFieldGeneral)
	requireNoOpcode(t, readFn, opGetField)
}

func TestStructFieldGeneralT0ResultDetachedFromBackingStorage(t *testing.T) {
	t.Parallel()

	source := `package main

type Container struct {
	lookup map[int]int
}

func entry() int {
	container := &Container{lookup: map[int]int{1: 11}}
	first := container.lookup
	container.lookup = map[int]int{42: 99}
	return first[1]
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entry")
	require.NoError(t, err, "entry must execute without crashing")

	value, ok := extractInt64FromAny(result)
	require.True(t, ok, "entry must return an int-like value; got %T", result)
	require.Equalf(t, int64(11), value,
		"the captured map must remain bound to the original (pre-mutation) map; got %d", value)
}

func extractInt64FromAny(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	}
	return 0, false
}
