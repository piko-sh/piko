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

package interp_provider_piko

import (
	"context"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	"piko.sh/piko/internal/interp/interp_adapters"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultDirPerm is the umask-friendly directory permission.
	defaultDirPerm = 0o755

	// defaultFilePerm is the umask-friendly file permission.
	defaultFilePerm = 0o644
)

// Interpreter is the CLI-facing bytecode interpreter. It exposes the full Eval / Compile
// / Execute / bytecode / debug surface that the templater-facing InterpreterPort does not
// reach.
type Interpreter = interp_domain.Service

// Option configures an Interpreter at construction.
type Option = interp_domain.Option

// CompiledFunction is the compiled bytecode for a single Go function.
type CompiledFunction = interp_domain.CompiledFunction

// Debugger drives step debugging against a running Interpreter.
type Debugger = interp_domain.Debugger

// DebugSnapshot is a captured state of the running VM at a breakpoint or step stop.
type DebugSnapshot = interp_domain.DebugSnapshot

// StackFrame is one frame in a captured debugger stack trace.
type StackFrame = interp_domain.StackFrame

// VariableInfo describes a single in-scope variable in a frame.
type VariableInfo = interp_domain.VariableInfo

// BytecodeStorePort is the interface bytecode persistence stores implement.
type BytecodeStorePort = interp_domain.BytecodeStorePort

// Session is the stateful REPL-style evaluation handle. Constructed via
// Interpreter.NewSession; each Submit accumulates declarations and imports against the
// same backing service.
type Session = interp_domain.Session

// SessionOption configures a Session at construction.
type SessionOption = interp_domain.SessionOption

// SessionState is a Session.Inspect snapshot.
type SessionState = interp_domain.SessionState

// SessionDecl is a single declaration entry in a SessionState.
type SessionDecl = interp_domain.SessionDecl

// SessionDeclKind classifies a SessionDecl as var / func / type / const.
type SessionDeclKind = interp_domain.SessionDeclKind

const (
	// SessionDeclVar identifies a Session declaration introduced by a `var` statement.
	SessionDeclVar = interp_domain.SessionDeclVar

	// SessionDeclFunc identifies a Session declaration introduced by a `func` statement.
	SessionDeclFunc = interp_domain.SessionDeclFunc

	// SessionDeclType identifies a Session declaration introduced by a `type` statement.
	SessionDeclType = interp_domain.SessionDeclType

	// SessionDeclConst identifies a Session declaration introduced by a `const` statement.
	SessionDeclConst = interp_domain.SessionDeclConst
)

// NewInterpreter creates a bytecode interpreter pre-loaded with the stdlib and Piko
// symbol exports.
//
// Takes opts ([]Option variadic) which configure limits, dispatch mode, debugger
// attachment, environment, and bytecode store wiring.
//
// Returns an Interpreter ready for Eval / Compile / Execute calls.
func NewInterpreter(opts ...Option) *Interpreter {
	return NewProvider().NewCLIInterpreter(opts...)
}

// NewCLIInterpreter creates a CLI-facing bytecode interpreter.
//
// The interpreter is pre-loaded with the stdlib, Piko, and any extra symbols previously
// supplied via Provider.RegisterSymbols. Use this when the caller needs to inject
// additional host packages (e.g. net/http) before building the interpreter; otherwise
// NewInterpreter is the convenience entry point.
//
// Takes opts ([]Option variadic) which configure limits, dispatch mode, debugger
// attachment, environment, and bytecode store wiring.
//
// Returns an Interpreter ready for Eval / Compile / Execute calls.
func (p *Provider) NewCLIInterpreter(opts ...Option) *Interpreter {
	service := interp_domain.NewService(opts...)
	symbolProvider := p.NewSymbolProvider()
	if concrete, ok := symbolProvider.(*SymbolProvider); ok {
		concrete.applyToService(service)
	}
	return service
}

// NewDebugger constructs a fresh Debugger. Attach it to an Interpreter at construction
// with WithDebugger.
//
// Returns a Debugger ready to be attached to an Interpreter.
func NewDebugger() *Debugger {
	return interp_domain.NewDebugger()
}

// WithMaxExecutionTime caps total wall-clock time per evaluation.
//
// Takes maximum (time.Duration) which is the cap on wall-clock time.
//
// Returns an Option configuring the execution-time limit.
func WithMaxExecutionTime(maximum time.Duration) Option {
	return interp_domain.WithMaxExecutionTime(maximum)
}

// WithMaxAllocSize caps the element count of a single allocation.
//
// Takes maximum (int) which is the cap on element count.
//
// Returns an Option configuring the allocation-size limit.
func WithMaxAllocSize(maximum int) Option {
	return interp_domain.WithMaxAllocSize(maximum)
}

// WithMaxGoroutines caps concurrent goroutines spawned by scripts.
//
// Takes maximum (int32) which is the cap on live goroutines.
//
// Returns an Option configuring the goroutine limit.
func WithMaxGoroutines(maximum int32) Option {
	return interp_domain.WithMaxGoroutines(maximum)
}

// WithMaxCallDepth caps the call stack depth.
//
// Takes maximum (int) which is the cap on stack depth.
//
// Returns an Option configuring the call-depth limit.
func WithMaxCallDepth(maximum int) Option {
	return interp_domain.WithMaxCallDepth(maximum)
}

// WithMaxOutputSize caps the bytes print / println may write.
//
// Takes maximum (int) which is the cap on bytes written.
//
// Returns an Option configuring the output-size limit.
func WithMaxOutputSize(maximum int) Option {
	return interp_domain.WithMaxOutputSize(maximum)
}

// WithMaxSourceSize caps the source bytes accepted at compilation.
//
// Takes maximum (int) which is the cap on source bytes.
//
// Returns an Option configuring the source-size limit.
func WithMaxSourceSize(maximum int) Option {
	return interp_domain.WithMaxSourceSize(maximum)
}

// WithMaxStringSize caps the length of a runtime string concatenation.
//
// Takes maximum (int) which is the cap on string length.
//
// Returns an Option configuring the string-size limit.
func WithMaxStringSize(maximum int) Option {
	return interp_domain.WithMaxStringSize(maximum)
}

// WithMaxLiteralElements caps the element count in a composite literal.
//
// Takes maximum (int) which is the cap on literal element count.
//
// Returns an Option configuring the literal-element limit.
func WithMaxLiteralElements(maximum int) Option {
	return interp_domain.WithMaxLiteralElements(maximum)
}

// WithCostBudget enables instruction cost metering with a fixed budget.
//
// Takes budget (int64) which is the total instruction-cost budget.
//
// Returns an Option configuring the cost meter.
func WithCostBudget(budget int64) Option {
	return interp_domain.WithCostBudget(budget)
}

// WithYieldInterval enables cooperative yielding every n instructions.
//
// Takes interval (uint32) which is the instruction count between cooperative yields.
//
// Returns an Option configuring the yield interval.
func WithYieldInterval(interval uint32) Option {
	return interp_domain.WithYieldInterval(interval)
}

// CapabilityHook is the port consulted before every gated native operation (file open,
// network dial, process spawn, environment access, native function dispatch). Re-exported
// from interp_domain so external hosts can implement it without reaching into internal
// packages.
//
// A nil CapabilityHook preserves the pre-hook execution model. See
// interp_domain.CapabilityHook for the full contract.
type CapabilityHook = interp_domain.CapabilityHook

// WithCapabilityHook installs a CapabilityHook on the interpreter. The hook is consulted
// before every gated native operation; returning a non-nil error from any Check* method
// aborts the call with that error visible to interpreted code.
//
// Takes hook (CapabilityHook) which is the host policy.
//
// Returns an Option configuring the capability hook.
func WithCapabilityHook(hook CapabilityHook) Option {
	return interp_domain.WithCapabilityHook(hook)
}

// Clock is the time source interpreted code observes through time.Now, time.Since,
// time.Until, time.Sleep, time.NewTimer, time.NewTicker. Re-exported from interp_domain
// so hosts can implement it without reaching into internal packages.
//
// Distinct from wdk/clock.Clock: that interface is for piko's internal services and
// returns wrapped Timer/Ticker abstractions; Clock must return stdlib *time.Timer /
// *time.Ticker because interpreted code expects those exact types. Hosts that have a
// wdk/clock.Clock can adapt via FromWDKClock.
type Clock = interp_domain.Clock

var (
	// WallClock is the default Clock implementation; all methods delegate directly to the
	// stdlib time package. The interpreter uses WallClock when no override is configured.
	WallClock = interp_domain.WallClock
)

// WithClock installs a Clock that replaces direct stdlib time access in interpreted code.
// Used for deterministic replay, test-controllable time, or audited time access.
//
// Takes clock (Clock) which becomes the interpreted time source. Pass nil to keep the
// WallClock default.
//
// Returns an Option configuring the clock override.
func WithClock(clock Clock) Option {
	return interp_domain.WithClock(clock)
}

// WithBuildTags adds extra build tags for //go:build evaluation.
//
// Takes tags ([]string variadic) which are extra build tags applied at compilation.
//
// Returns an Option configuring the build-tag set.
func WithBuildTags(tags ...string) Option {
	return interp_domain.WithBuildTags(tags...)
}

// WithEnv overrides os.Getenv inside scripts.
//
// Takes env (map[string]string) which holds the script-visible environment variables.
//
// Returns an Option configuring the script environment.
func WithEnv(env map[string]string) Option {
	return interp_domain.WithEnv(env)
}

// WithForceGoDispatch forces the Go dispatch loop, disabling ASM.
//
// Returns an Option configuring the dispatch mode.
func WithForceGoDispatch() Option {
	return interp_domain.WithForceGoDispatch()
}

// WithDebugInfo enables source position info in compiled bytecode.
//
// Returns an Option enabling source-position metadata.
func WithDebugInfo() Option {
	return interp_domain.WithDebugInfo()
}

// WithDebugger attaches a Debugger; implies debug info and Go dispatch.
//
// Takes debugger (*Debugger) which receives breakpoint and step events.
//
// Returns an Option attaching the debugger to the Interpreter.
func WithDebugger(debugger *Debugger) Option {
	return interp_domain.WithDebugger(debugger)
}

// WithBytecodeStore wires a BytecodeStorePort so Interpreter.SaveCompiled and
// LoadCompiled can be used.
//
// Takes store (BytecodeStorePort) which persists compiled bytecode.
//
// Returns an Option wiring the bytecode store.
func WithBytecodeStore(store BytecodeStorePort) Option {
	return interp_domain.WithBytecodeStore(store)
}

// NewDirectoryBytecodeStore returns a BytecodeStorePort backed by the given directory.
// Keys map to files named "bytecode-<key>.bin".
//
// Takes directory (string) which is the on-disk root for the store.
//
// Returns the bytecode store and a nil error on success.
// Returns a nil store and an error when the directory is empty, cannot be created, or
// cannot be sandboxed.
func NewDirectoryBytecodeStore(directory string) (BytecodeStorePort, error) {
	if directory == "" {
		return nil, errors.New("bytecode store directory must not be empty")
	}
	if err := os.MkdirAll(directory, defaultDirPerm); err != nil {
		return nil, fmt.Errorf("creating bytecode store directory: %w", err)
	}
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("creating bytecode store sandbox: %w", err)
	}
	return interp_adapters.NewBytecodeStore(sandbox), nil
}

// SaveCompiledToFile saves cfs to the user-chosen path. The given Interpreter must have
// been constructed with WithBytecodeStore using NewDirectoryBytecodeStore rooted at the
// file's parent directory.
//
// Takes ctx (context.Context) which carries cancellation and deadlines through the
// underlying store.
// Takes interpreter (*Interpreter) which provides the configured bytecode store.
// Takes path (string) which is the user-chosen output path.
// Takes cfs (*CompiledFileSet) which holds the compiled bytecode to persist.
//
// Returns an error when saving or renaming fails; nil on success.
func SaveCompiledToFile(ctx context.Context, interpreter *Interpreter, path string, cfs *CompiledFileSet) error {
	directory, key, natural := bytecodePathParts(path)
	if err := interpreter.SaveCompiled(ctx, key, cfs); err != nil {
		return err
	}
	if natural == path {
		return nil
	}
	if err := os.Rename(filepath.Join(directory, natural), path); err != nil {
		return fmt.Errorf("renaming bytecode artefact: %w", err)
	}
	return nil
}

// LoadCompiledFromFile loads a CompiledFileSet from a user-chosen path. The given
// Interpreter must have been constructed with WithBytecodeStore using
// NewDirectoryBytecodeStore rooted at the file's parent directory.
//
// Takes ctx (context.Context) which carries cancellation through the underlying store.
// Takes interpreter (*Interpreter) which provides the configured bytecode store.
// Takes path (string) which is the user-chosen input path.
//
// Returns the deserialised CompiledFileSet on success.
// Returns an error when the file is missing, the schema version has changed, or
// reconstruction fails.
func LoadCompiledFromFile(ctx context.Context, interpreter *Interpreter, path string) (*CompiledFileSet, error) {
	directory, key, natural := bytecodePathParts(path)
	naturalPath := filepath.Join(directory, natural)

	if naturalPath != path {
		sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("creating bytecode load sandbox: %w", err)
		}
		defer func() { _ = sandbox.Close() }()
		data, err := sandbox.ReadFile(filepath.Base(path))
		if err != nil {
			return nil, fmt.Errorf("reading bytecode artefact: %w", err)
		}
		if err := sandbox.WriteFile(natural, data, defaultFilePerm); err != nil {
			return nil, fmt.Errorf("staging bytecode artefact: %w", err)
		}
		defer func() { _ = os.Remove(naturalPath) }()
	}
	return interpreter.LoadCompiled(ctx, key)
}

// FormatGoSource formats Go source the same way `gofmt` does (it wraps stdlib
// go/format.Source). Useful for the CLI's `fmt` subcommand without depending on the host
// toolchain.
//
// Takes source ([]byte) which is the raw Go source.
//
// Returns the formatted source bytes on success.
// Returns an error when parsing fails.
func FormatGoSource(source []byte) ([]byte, error) {
	return format.Source(source)
}

// bytecodePathParts splits a user-supplied bytecode path into the directory, the
// BytecodeStore key (filename stem), and the on-disk filename the underlying store will
// pick (bytecode-<key>.bin).
//
// Takes path (string) which is the user-chosen bytecode path.
//
// Returns directory which is the parent directory of path.
// Returns key which is the filename stem (extension trimmed).
// Returns natural which is the underlying store's on-disk filename for that key.
func bytecodePathParts(path string) (directory, key, natural string) {
	directory = filepath.Dir(path)
	base := filepath.Base(path)
	key = strings.TrimSuffix(base, filepath.Ext(base))
	natural = "bytecode-" + key + ".bin"
	return directory, key, natural
}
