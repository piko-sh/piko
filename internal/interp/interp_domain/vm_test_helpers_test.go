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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestVM(t *testing.T) *VM {
	t.Helper()
	return newVM(context.Background(), newGlobalStore(), NewSymbolRegistry(nil))
}

func newTestVMWithContext(ctx context.Context, t *testing.T) *VM {
	t.Helper()
	return newVM(ctx, newGlobalStore(), NewSymbolRegistry(nil))
}

func newTestVMWithSymbols(t *testing.T, exports SymbolExports) *VM {
	t.Helper()
	return newVM(context.Background(), newGlobalStore(), NewSymbolRegistry(exports))
}

func executeTestBytecode(t *testing.T, compiledFunction *CompiledFunction) (any, error) {
	t.Helper()
	vm := newTestVM(t)
	return vm.execute(compiledFunction)
}

func executeTestBytecodeExpect(t *testing.T, compiledFunction *CompiledFunction, expect any) {
	t.Helper()
	result, err := executeTestBytecode(t, compiledFunction)
	require.NoError(t, err)
	require.Equal(t, expect, result)
}

func executeTestBytecodeExpectError(t *testing.T, compiledFunction *CompiledFunction, target error) {
	t.Helper()
	_, err := executeTestBytecode(t, compiledFunction)
	require.Error(t, err)
	require.True(t, errors.Is(err, target), "expected error %v, got %v", target, err)
}
