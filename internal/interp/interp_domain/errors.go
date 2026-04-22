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

import "errors"

var (
	// errDivisionByZero is returned when an integer division or remainder
	// operation has a zero divisor.
	errDivisionByZero = errors.New("division by zero")

	// errStackOverflow is returned when the call stack exceeds the maximum
	// depth, indicating infinite recursion or excessively deep call chains.
	errStackOverflow = errors.New("stack overflow")

	// errIndexOutOfRange is returned when a slice, array, or string index
	// is outside the valid range [0, len).
	errIndexOutOfRange = errors.New("index out of range")

	// errNilPointerIndex is returned when an index operation is applied
	// to a nil pointer (e.g. indexing a nil *[N]T).
	errNilPointerIndex = errors.New("index of nil pointer")

	// errSliceOutOfRange is returned when slice bounds are outside the
	// valid range or low exceeds high.
	errSliceOutOfRange = errors.New("slice bounds out of range")

	// errInvalidOpcode is returned when the VM encounters an unrecognised
	// opcode during execution.
	errInvalidOpcode = errors.New("invalid opcode")

	// errCompilation is returned when the source code fails to compile.
	// The underlying error provides specific parsing or type-checking
	// details.
	errCompilation = errors.New("compilation failed")

	// errTypeCheck is returned when go/types rejects the source code.
	errTypeCheck = errors.New("type check failed")

	// errParse is returned when go/parser rejects the source code.
	errParse = errors.New("parse failed")

	// errEntrypointNotFound is returned when the requested entrypoint
	// function does not exist in the compiled file set.
	errEntrypointNotFound = errors.New("entrypoint not found")

	// errCyclicImport is returned when the import graph contains a
	// cycle, which is illegal in Go.
	errCyclicImport = errors.New("cyclic import detected")

	// errExecutionCancelled is returned when the execution context is
	// cancelled or its deadline is exceeded during evaluation.
	errExecutionCancelled = errors.New("execution cancelled")

	// errAllocationLimit is returned when a single allocation (make
	// slice, make chan, unsafe.String, unsafe.Slice) exceeds the
	// configured maximum size.
	errAllocationLimit = errors.New("allocation size limit exceeded")

	// errGoroutineLimit is returned when the number of goroutines
	// spawned by interpreted code exceeds the configured limit.
	errGoroutineLimit = errors.New("goroutine limit exceeded")

	// errOutputLimit is returned when print/println output exceeds
	// the configured maximum size.
	errOutputLimit = errors.New("output size limit exceeded")

	// errNoBytecodeStore is returned when bytecode save/load is
	// attempted without a configured BytecodeStorePort.
	errNoBytecodeStore = errors.New("no bytecode store configured")

	// errFeatureNotAllowed is returned when compiled code uses a
	// language feature that has been disabled via WithFeatures.
	errFeatureNotAllowed = errors.New("language feature not allowed")

	// errCostBudgetExceeded is returned when the runtime cost of
	// executing code exceeds the budget set via WithCostBudget.
	errCostBudgetExceeded = errors.New("cost budget exceeded")

	// errSourceSizeLimit is returned when the total source code size
	// exceeds the configured maximum set via WithMaxSourceSize.
	errSourceSizeLimit = errors.New("source size limit exceeded")

	// errStringLimit is returned when a string concatenation would
	// produce a result exceeding the configured maximum set via
	// WithMaxStringSize.
	errStringLimit = errors.New("string size limit exceeded")

	// errLiteralElementLimit is returned when a composite literal
	// (slice, array, map) has more elements than the configured
	// maximum set via WithMaxLiteralElements.
	errLiteralElementLimit = errors.New("literal element count limit exceeded")
)
