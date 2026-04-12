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

func TestEvalTrailingBareNilReturnsNil(t *testing.T) {
	t.Parallel()
	service := NewService()

	result, err := service.Eval(context.Background(), `
a := 1
_ = a
nil
`)
	require.NoError(t, err, "Service.Eval returned an unexpected error")
	require.Nil(t, result, "trailing bare nil should yield a nil interface")
}

func TestEvalTrailingVoidCallReturnsNil(t *testing.T) {
	t.Parallel()
	service := NewService()

	result, err := service.Eval(context.Background(), `
a := 1
_ = a
println("hi")
`)
	require.NoError(t, err, "Service.Eval returned an unexpected error")
	require.Nil(t, result, "trailing void call should yield a nil interface")
}

func TestEvalPureNilExpressionReturnsNil(t *testing.T) {
	t.Parallel()
	service := NewService()

	result, err := service.Eval(context.Background(), `nil`)
	require.NoError(t, err, "Service.Eval returned an unexpected error")
	require.Nil(t, result, "pure nil expression should yield a nil interface")
}

func TestEvalTrailingValueCallReturnsResult(t *testing.T) {
	t.Parallel()
	service := NewService()

	result, err := service.Eval(context.Background(), `
greet := func(name string) string {
    return "hello, " + name
}
greet("world")
`)
	require.NoError(t, err, "Service.Eval returned an unexpected error")
	require.Equal(t, "hello, world", result, "trailing value-returning call should propagate its result")
}
