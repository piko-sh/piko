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
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_helpers"
)

type equalityProbeStatus int

func TestEqualityHelperUnwrap_ComparesUnderlyingValues(t *testing.T) {
	t.Parallel()

	leftBox := equalityProbeStatus(1)
	rightBox := equalityProbeStatus(1)
	leftAdapter := &pikoStringerAdapter{underlying: reflect.ValueOf(&leftBox).Elem()}
	rightAdapter := &pikoStringerAdapter{underlying: reflect.ValueOf(&rightBox).Elem()}
	arguments := []reflect.Value{reflect.ValueOf(leftAdapter), reflect.ValueOf(rightAdapter)}

	require.False(t,
		generator_helpers.EvaluateStrictEquality(arguments[0].Interface(), arguments[1].Interface()),
		"comparing the adapter envelopes must not match before unwrapping")

	unwrapPikoAdapterArguments(reflect.ValueOf(generator_helpers.EvaluateStrictEquality), arguments)

	require.Equal(t, reflect.Int, arguments[0].Kind())
	require.Equal(t, reflect.Int, arguments[1].Kind())
	assert.True(t,
		generator_helpers.EvaluateStrictEquality(arguments[0].Interface(), arguments[1].Interface()),
		"after unwrapping the source-level values must compare equal")
}

func TestEqualityHelperUnwrap_KeepsAdaptersForRenderingFunctions(t *testing.T) {
	t.Parallel()

	adapter := &pikoStringerAdapter{underlying: reflect.ValueOf(equalityProbeStatus(1))}
	arguments := []reflect.Value{reflect.ValueOf(adapter)}

	unwrapPikoAdapterArguments(reflect.ValueOf(strings.ToUpper), arguments)

	assert.Equal(t, reflect.Pointer, arguments[0].Kind(),
		"a function that is not an identity or equality helper must still receive the Stringer adapter")
}
