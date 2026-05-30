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

package driven_system_symbols

import (
	"reflect"
	"sync"
)

// wrappedSyncOnceValue bridges sync.OnceValue[T any] for interpreted code. The
// generator's existing registration only covers sync.OnceValue[int] which is useless for
// non-int users; this wrapper accepts any func() T expressed as `func() any` (the
// interpreter's untyped callback shape) and returns a memoising `func() any`.
//
// The wrapper preserves OnceValue's semantics - single execution, concurrent-safe, value
// cached for all subsequent calls - by delegating to sync.OnceValue[any].
//
// Takes producer (func() any) which yields the value to memoise.
//
// Returns func() any which returns the same value on every call.
func wrappedSyncOnceValue(producer func() any) func() any {
	return sync.OnceValue(producer)
}

// wrappedSyncOnceValues bridges sync.OnceValues[T1, T2 any]. Same strategy as
// wrappedSyncOnceValue but for two return values.
//
// Takes producer (func() (any, any)) which yields the value pair.
//
// Returns func() (any, any) which returns the same pair on every call.
func wrappedSyncOnceValues(producer func() (any, any)) func() (any, any) {
	return sync.OnceValues(producer)
}

func init() {
	if _, ok := Symbols["sync"]; ok {
		Symbols["sync"]["OnceValue"] = reflect.ValueOf(wrappedSyncOnceValue)
		Symbols["sync"]["OnceValues"] = reflect.ValueOf(wrappedSyncOnceValues)
	}
}
