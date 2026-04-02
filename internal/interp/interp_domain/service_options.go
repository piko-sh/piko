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
	"go/ast"
	"time"
)

// WithBuildTags sets custom build tags for //go:build constraint
// evaluation in CompileFileSet and CompileProgram. The current GOOS,
// GOARCH, and Go version are always included as default tags.
//
// Takes tags (string variadic) which are additional build tags to
// activate.
//
// Returns Option which applies the build tags to the service config.
func WithBuildTags(tags ...string) Option {
	return func(c *serviceConfig) {
		c.buildTags = append(c.buildTags, tags...)
	}
}

// WithEnv sets environment variable overrides for interpreted code.
// When set, os.Getenv, os.LookupEnv, and os.Environ in interpreted
// code read from this map instead of the host process environment.
//
// Takes env (map[string]string) which maps variable names to values.
//
// Returns Option which applies the environment overrides to the
// service config.
func WithEnv(env map[string]string) Option {
	return func(c *serviceConfig) {
		c.env = env
	}
}

// WithMaxExecutionTime sets the maximum duration for any single
// evaluation. Each public method wraps the caller's context with this
// deadline and the shorter limit wins.
//
// Takes d (time.Duration) which is the maximum execution time.
//
// Returns Option which applies the execution time limit.
func WithMaxExecutionTime(d time.Duration) Option {
	return func(c *serviceConfig) {
		c.maxExecutionTime = d
	}
}

// WithMaxAllocSize sets the maximum number of elements permitted in a
// single allocation (make slice, make chan, unsafe.String,
// unsafe.Slice). Zero means unlimited.
//
// Takes n (int) which is the maximum element count.
//
// Returns Option which applies the allocation limit.
func WithMaxAllocSize(n int) Option {
	return func(c *serviceConfig) {
		c.maxAllocSize = n
	}
}

// WithMaxGoroutines sets the maximum number of concurrent goroutines
// that interpreted code may spawn via go statements. Zero means
// unlimited.
//
// Takes n (int32) which is the goroutine limit.
//
// Returns Option which applies the goroutine limit.
func WithMaxGoroutines(n int32) Option {
	return func(c *serviceConfig) {
		c.maxGoroutines = n
	}
}

// WithMaxCallDepth sets the maximum call stack depth before a stack
// overflow error is raised. Zero uses the default (10000).
//
// Takes n (int) which is the call depth limit.
//
// Returns Option which applies the call depth limit.
func WithMaxCallDepth(n int) Option {
	return func(c *serviceConfig) {
		c.maxCallDepth = n
	}
}

// WithMaxOutputSize sets the maximum number of bytes that print and
// println may write before returning an error. Zero means unlimited.
//
// Takes n (int) which is the output byte limit.
//
// Returns Option which applies the output size limit.
func WithMaxOutputSize(n int) Option {
	return func(c *serviceConfig) {
		c.maxOutputSize = n
	}
}

// WithForceGoDispatch forces the VM to use the pure Go dispatch loop
// even on architectures with ASM threaded dispatch (amd64, arm64).
// Useful for testing dispatch parity across both paths.
//
// Returns Option which enables pure Go dispatch.
func WithForceGoDispatch() Option {
	return func(c *serviceConfig) {
		c.forceGoDispatch = true
	}
}

// WithDebugInfo enables debug information generation during
// compilation. When enabled, the compiler records source positions
// for each bytecode instruction and variable liveness ranges.
//
// Returns Option which enables debug info generation.
func WithDebugInfo() Option {
	return func(c *serviceConfig) {
		c.debugInfo = true
	}
}

// WithDebugger attaches a Debugger to the interpreter service. This
// implies WithDebugInfo() and WithForceGoDispatch() - debug info is
// needed for breakpoints and source mapping, and Go dispatch is
// required because the ASM loop does not support the debug hook.
//
// Takes debugger (*Debugger) which is the debugger to attach.
//
// Returns Option which attaches the debugger.
func WithDebugger(debugger *Debugger) Option {
	return func(c *serviceConfig) {
		c.debugInfo = true
		c.forceGoDispatch = true
		c.debugger = debugger
	}
}

// WithArenaFactory sets a custom factory for register arena allocation.
//
// When nil, the global sync.Pool is used. Each call to the factory
// should return a fresh RegisterArena for test isolation.
//
// Takes factory (func() *RegisterArena) which is the arena constructor.
//
// Returns Option which applies the arena factory.
func WithArenaFactory(factory func() *RegisterArena) Option {
	return func(c *serviceConfig) {
		c.arenaFactory = factory
	}
}

// WithCompilationSnapshot registers a callback that receives a
// snapshot of the compiled output at the end of CompileProgram,
// regardless of whether compilation succeeded or failed partway
// through. The snapshot contains all functions from packages that
// compiled successfully before the failure.
//
// Takes callback (func(*CompiledFileSet)) which receives the
// partial or complete compilation snapshot.
//
// Returns Option which registers the snapshot callback.
func WithCompilationSnapshot(callback func(*CompiledFileSet)) Option {
	return func(c *serviceConfig) {
		c.compilationSnapshotCallback = callback
	}
}

// WithFeatures sets the allowed language features for compilation.
//
// Features not present in the bitmask will cause a compile-time error
// when used. The default is InterpFeaturesAll.
//
// Takes features (InterpFeature) which is the bitmask of allowed
// features.
//
// Returns Option which applies the feature restrictions.
func WithFeatures(features InterpFeature) Option {
	return func(c *serviceConfig) {
		c.features = features
	}
}

// WithCostBudget sets the maximum total computation cost for a single
// execution.
//
// Each opcode consumes a cost from the budget based on the cost table.
// When the budget is exhausted, execution halts with
// errCostBudgetExceeded. Zero (the default) disables cost metering.
// Cost metering forces Go dispatch (not ASM) since the dispatch loop
// must check the budget on every instruction.
//
// Takes budget (int64) which is the total cost budget.
//
// Returns Option which enables cost metering.
func WithCostBudget(budget int64) Option {
	return func(c *serviceConfig) {
		c.costBudget = budget
	}
}

// WithCostTable sets a custom per-opcode cost table for cost metering.
//
// If not set, the default cost table is used. Use DefaultCostTable to
// get a copy of the default table and modify individual entries.
//
// Takes table (CostTable) which is the custom cost table.
//
// Returns Option which applies the custom cost table.
func WithCostTable(table *CostTable) Option {
	return func(c *serviceConfig) {
		c.costTable = table
	}
}

// WithMaxSourceSize sets the maximum total source code size in bytes
// accepted for compilation. Zero (the default) means no limit.
//
// Takes n (int) which is the maximum source size in bytes.
//
// Returns Option which enables source size limiting.
func WithMaxSourceSize(n int) Option {
	return func(c *serviceConfig) {
		c.maxSourceSize = n
	}
}

// WithMaxStringSize sets the maximum string length in bytes that a
// concatenation may produce at runtime. Zero (the default) means no
// limit.
//
// Takes n (int) which is the maximum string size in bytes.
//
// Returns Option which enables string size limiting.
func WithMaxStringSize(n int) Option {
	return func(c *serviceConfig) {
		c.maxStringSize = n
	}
}

// WithMaxLiteralElements sets the maximum number of elements allowed
// in a single composite literal (slice, array, map) at compile time.
// Zero (the default) means no limit.
//
// Takes n (int) which is the maximum element count.
//
// Returns Option which enables literal element limiting.
func WithMaxLiteralElements(n int) Option {
	return func(c *serviceConfig) {
		c.maxLiteralElements = n
	}
}

// WithYieldInterval sets the number of instructions between
// runtime.Gosched() calls for cooperative scheduling. This prevents a
// single interpreter VM from monopolising a CPU core in a
// multi-tenant environment.
//
// The value must be a power of two (e.g. 1024, 2048, 4096). Zero
// (the default) disables yielding. Enables Go dispatch (not ASM)
// since the ASM loop does not contain the yield check.
//
// Takes n (uint32) which is the instruction interval between yields.
//
// Returns Option which enables cooperative yielding.
func WithYieldInterval(n uint32) Option {
	return func(c *serviceConfig) {
		c.yieldInterval = n
	}
}

// findEvalFuncDecl locates the _eval_ function declaration in a file.
//
// Takes file (*ast.File) which is the parsed AST file to search.
//
// Returns *ast.FuncDecl which is the _eval_ function declaration, or
// nil when not found.
func findEvalFuncDecl(file *ast.File) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		functionDeclaration, ok := declaration.(*ast.FuncDecl)
		if ok && functionDeclaration.Name.Name == evalFuncName {
			return functionDeclaration
		}
	}
	return nil
}
