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
	"reflect"
)

// CapabilityGate is the per-function policy that translates raw reflect arguments into a
// hook consultation. Each gated stdlib entry (os.Open, net.Dial, exec.Command, ...) gets
// its own gate that knows which argument carries the path, network, address, etc., and
// which Check* method to call.
//
// A gate returning a non-nil error aborts the wrapped call; the error is returned through
// the wrapped function's last error result if it has one, or panicked otherwise.
type CapabilityGate func(ctx context.Context, hook CapabilityHook, modulePath string, args []reflect.Value) error

// hookProvider is the minimal interface a gated-function wrapper needs from the live VM
// at call time: the current hook (which may have been changed via SetCapabilityHook
// between calls) and the module-path stack so the hook can scope decisions.
//
// The package-level wrapNativeFunction returns a closure that captures a hookProvider
// rather than a snapshot of (hook, modulePath) so the wrapped function follows the
// Service's live state. A nil provider means "no gating", used in tests and for code
// paths that wrap a function before the Service is fully wired.
type hookProvider interface {
	// Hook returns the live capability hook.
	//
	// Returns CapabilityHook which is the hook configured on the provider, or nil when no
	// hook is set.
	Hook() CapabilityHook

	// ModulePath returns the active module path for scoping decisions.
	//
	// Returns string which is the canonical module path, or empty for main-program code.
	ModulePath() string

	// Context returns the live execution context.
	//
	// Returns context.Context which is the active context, never nil.
	Context() context.Context
}

// staticHookProvider implements hookProvider with constant values. Used in tests and for
// stdlib bindings that are wrapped once at Service construction time and don't change.
type staticHookProvider struct {
	// hook holds the constant capability hook returned by Hook.
	hook CapabilityHook

	// ctx holds the constant execution context returned by Context.
	ctx context.Context

	// modulePath holds the constant module path returned by ModulePath.
	modulePath string
}

// Hook returns the stored hook.
//
// Returns CapabilityHook which is the stored hook.
func (s *staticHookProvider) Hook() CapabilityHook { return s.hook }

// ModulePath returns the stored module path.
//
// Returns string which is the stored module path.
func (s *staticHookProvider) ModulePath() string { return s.modulePath }

// Context returns the stored context.
//
// Returns context.Context which is the stored context.
func (s *staticHookProvider) Context() context.Context { return s.ctx }

// wrapNativeFunction wraps a stdlib function value with a CapabilityGate. The returned
// reflect.Value has the same type as fn; when invoked, it consults the supplied hook
// through provider before dispatching to the underlying function.
//
// On gate denial the wrapper either sets the last error-typed return value to the denial
// error when the wrapped function's signature ends with an error return, or panics with
// the denial error when there is no error return. The panic path matches Go's behaviour
// for unrecoverable native failures and is recoverable by interpreted code via the
// existing raiseNativePanicAsInterpreted machinery.
//
// Takes provider (hookProvider) which supplies the live hook and module path at call
// time. Pass nil for tests that don't need gating.
// Takes fn (reflect.Value) which is the stdlib function to wrap.
// Takes fnPath (string) which is the dotted identifier used for CheckFunctionCall and for
// error messages.
// Takes gate (CapabilityGate) which is the per-function policy. Pass nil to use a generic
// CheckFunctionCall gate.
//
// Returns reflect.Value which is a function of the same type as fn.
func wrapNativeFunction(provider hookProvider, fn reflect.Value, fnPath string, gate CapabilityGate) reflect.Value {
	if !fn.IsValid() || fn.Kind() != reflect.Func {
		return fn
	}
	fnType := fn.Type()
	hasErrorReturn := fnType.NumOut() > 0 && fnType.Out(fnType.NumOut()-1) == errorType
	return reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		hook, modulePath, ctx := capabilityCallContext(provider)
		var gateErr error
		if hook != nil {
			if gate != nil {
				gateErr = gate(ctx, hook, modulePath, args)
			} else {
				gateErr = hook.CheckFunctionCall(ctx, modulePath, fnPath, args)
			}
		}
		if gateErr != nil {
			return handleGateDenial(fnType, hasErrorReturn, gateErr)
		}
		return fn.Call(args)
	})
}

// capabilityCallContext extracts the (hook, modulePath, ctx) tuple from a provider, with
// nil-safe defaults.
//
// Takes provider (hookProvider) which may be nil.
//
// Returns the live hook (nil when provider is nil or hook is unset), module path, and
// context.
func capabilityCallContext(provider hookProvider) (CapabilityHook, string, context.Context) {
	if provider == nil {
		return nil, "", context.Background()
	}
	hook := provider.Hook()
	ctx := provider.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return hook, provider.ModulePath(), ctx
}

// handleGateDenial builds the return slice for a denied call: either a slice ending with
// the denial error, or a panic.
//
// Takes fnType (reflect.Type) which describes the function signature.
// Takes hasErrorReturn (bool) which is true when the last return type is error.
// Takes denial (error) which is the gate's rejection.
//
// Returns []reflect.Value matching fnType's return list.
//
// Panics with denial when hasErrorReturn is false. The wrapper relies on reflect's normal
// panic semantics; interpreted code that wraps the call can recover.
func handleGateDenial(fnType reflect.Type, hasErrorReturn bool, denial error) []reflect.Value {
	if !hasErrorReturn {
		panic(denial)
	}
	results := make([]reflect.Value, fnType.NumOut())
	for i := range fnType.NumOut() - 1 {
		results[i] = reflect.Zero(fnType.Out(i))
	}
	results[fnType.NumOut()-1] = reflect.ValueOf(&denial).Elem()
	return results
}

var (
	// errorType is the reflect.Type of the error interface, used to detect functions that
	// surface failures through an error return.
	errorType = reflect.TypeFor[error]()
)
