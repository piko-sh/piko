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

package goroutine

import (
	"context"
	"runtime/pprof"
)

// labelKeyComponent is the pprof label key identifying which part of the framework a
// goroutine belongs to. It mirrors the component string already passed to RecoverPanic.
const labelKeyComponent = "component"

// Label attaches pprof labels to the calling goroutine and returns the labelled context.
//
// Takes component (string) which identifies the goroutine, in the same
// "package.functionName" form used by RecoverPanic.
// Takes keyValues (...string) which are additional label key and value pairs.
//
// Returns context.Context which carries the labels, for passing to child goroutines.
func Label(ctx context.Context, component string, keyValues ...string) context.Context {
	if len(keyValues)%2 != 0 {
		keyValues = keyValues[:len(keyValues)-1]
	}

	pairs := make([]string, 0, len(keyValues)+2)
	pairs = append(pairs, labelKeyComponent, component)
	pairs = append(pairs, keyValues...)

	labelled := pprof.WithLabels(ctx, pprof.Labels(pairs...))
	pprof.SetGoroutineLabels(labelled)

	return labelled
}
