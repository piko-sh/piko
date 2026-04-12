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
	"runtime"
)

// consultCapabilityHookForNativeCall gates a native call through the hook.
//
// Called by the native dispatch path before reflect.Call invokes the underlying function.
// Fast no-op when no hook is installed (the hot case for hosts that have not opted into
// capability gating); when a hook is installed it resolves the dotted symbol path lazily,
// caches it on the call site, and forwards (ctx, modulePath, fnPath, args) to
// CheckFunctionCall.
//
// Takes vm (*VM) which carries the live hook and execution context.
// Takes site (*callSite) which holds the cached symbol path.
// Takes reflectedFunction (reflect.Value) which is the function about to be dispatched.
// Takes arguments ([]reflect.Value) which is the prepared argument slice the dispatcher
// will pass to reflect.Call.
//
// Returns error from the hook's CheckFunctionCall, or nil to allow the call to proceed.
func consultCapabilityHookForNativeCall(vm *VM, site *callSite, reflectedFunction reflect.Value, arguments []reflect.Value) error {
	hook := vm.limits.capabilityHook
	if hook == nil {
		return nil
	}
	if site.nativeFunctionPath == "" {
		site.nativeFunctionPath = resolveNativeFunctionPath(reflectedFunction)
	}
	ctx := capabilityHookContext(vm)
	return hook.CheckFunctionCall(ctx, vm.modulePath, site.nativeFunctionPath, arguments)
}

// resolveNativeFunctionPath returns the dotted symbol identifier.
//
// Uses runtime.FuncForPC, which is suitable for cold-path use only. The result is
// intended to be cached on the call site so subsequent dispatches skip the lookup.
// Example outputs: "net/http.Get", "(*bytes.Buffer).Write".
//
// Takes reflectedFunction (reflect.Value) which holds the function.
//
// Returns string which is the resolved symbol path, or "" when resolution fails (zero
// value, non-func kind, or no debug symbol).
func resolveNativeFunctionPath(reflectedFunction reflect.Value) string {
	if !reflectedFunction.IsValid() || reflectedFunction.Kind() != reflect.Func {
		return ""
	}
	fn := runtime.FuncForPC(reflectedFunction.Pointer())
	if fn == nil {
		return ""
	}
	return fn.Name()
}

// capabilityHookContext returns the context associated with the VM for hook consultation.
// Falls back to context.Background() so hook implementations can assume a non-nil
// context.
//
// Takes vm (*VM) which may carry a per-execution context.
//
// Returns context.Context that's safe to pass to hook methods.
func capabilityHookContext(vm *VM) context.Context {
	if vm == nil || vm.ctx == nil {
		return context.Background()
	}
	return vm.ctx
}
