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
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type recordingHook struct {
	denyFn func(call recordedHookCall) error
	calls  []recordedHookCall
	mu     sync.Mutex
}

type recordedHookCall struct {
	Fields     map[string]any
	Method     string
	ModulePath string
}

func (r *recordingHook) record(method, modulePath string, fields map[string]any) error {
	r.mu.Lock()
	call := recordedHookCall{Method: method, ModulePath: modulePath, Fields: fields}
	r.calls = append(r.calls, call)
	deny := r.denyFn
	r.mu.Unlock()
	if deny != nil {
		return deny(call)
	}
	return nil
}

func (r *recordingHook) CheckFunctionCall(_ context.Context, modulePath, fnPath string, args []reflect.Value) error {
	return r.record("CheckFunctionCall", modulePath, map[string]any{
		"fnPath":   fnPath,
		"argCount": len(args),
	})
}

func (r *recordingHook) CheckFileOpen(_ context.Context, modulePath, path string, flag int, mode os.FileMode) error {
	return r.record("CheckFileOpen", modulePath, map[string]any{
		"path": path,
		"flag": flag,
		"mode": mode,
	})
}

func (r *recordingHook) CheckFileWrite(_ context.Context, modulePath, path string) error {
	return r.record("CheckFileWrite", modulePath, map[string]any{"path": path})
}

func (r *recordingHook) CheckExec(_ context.Context, modulePath, name string, argv []string) error {
	return r.record("CheckExec", modulePath, map[string]any{"name": name, "argv": argv})
}

func (r *recordingHook) CheckNetDial(_ context.Context, modulePath, network, address string) error {
	return r.record("CheckNetDial", modulePath, map[string]any{"network": network, "address": address})
}

func (r *recordingHook) CheckNetListen(_ context.Context, modulePath, network, address string) error {
	return r.record("CheckNetListen", modulePath, map[string]any{"network": network, "address": address})
}

func (r *recordingHook) CheckGetenv(_ context.Context, modulePath, name string) error {
	return r.record("CheckGetenv", modulePath, map[string]any{"name": name})
}

func (r *recordingHook) CheckSetenv(_ context.Context, modulePath, name, value string) error {
	return r.record("CheckSetenv", modulePath, map[string]any{"name": name, "value": value})
}

func (r *recordingHook) CheckSubprocess(_ context.Context, modulePath, name string, argv []string) error {
	return r.record("CheckSubprocess", modulePath, map[string]any{"name": name, "argv": argv})
}

func (r *recordingHook) snapshot() []recordedHookCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]recordedHookCall, len(r.calls))
	copy(cp, r.calls)
	return cp
}

func TestCapabilityHookOption(t *testing.T) {
	t.Parallel()
	hook := &recordingHook{}
	service := NewService(WithCapabilityHook(hook))

	if got := service.CapabilityHook(); got != hook {
		t.Fatalf("Service.CapabilityHook() = %v, want recordingHook", got)
	}
	if service.limits.capabilityHook != hook {
		t.Fatalf("Service.limits.capabilityHook = %v, want recordingHook", service.limits.capabilityHook)
	}
}

func TestCapabilityHookSetter(t *testing.T) {
	t.Parallel()
	service := NewService()
	if service.CapabilityHook() != nil {
		t.Fatalf("default Service.CapabilityHook() = %v, want nil", service.CapabilityHook())
	}

	hook := &recordingHook{}
	service.SetCapabilityHook(hook)
	if service.CapabilityHook() != hook {
		t.Fatalf("after SetCapabilityHook, Service.CapabilityHook() = %v, want recordingHook", service.CapabilityHook())
	}
	if service.limits.capabilityHook != hook {
		t.Fatalf("after SetCapabilityHook, limits.capabilityHook = %v, want recordingHook", service.limits.capabilityHook)
	}

	service.SetCapabilityHook(nil)
	if service.CapabilityHook() != nil {
		t.Fatalf("after SetCapabilityHook(nil), Service.CapabilityHook() = %v, want nil", service.CapabilityHook())
	}
}

func TestPermissiveHookAllowsEverything(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := permissiveHook
	type checkFn func() error
	checks := []checkFn{
		func() error { return h.CheckFunctionCall(ctx, "", "fmt.Println", nil) },
		func() error { return h.CheckFileOpen(ctx, "", "/etc/passwd", os.O_RDONLY, 0) },
		func() error { return h.CheckFileWrite(ctx, "", "/tmp/x") },
		func() error { return h.CheckExec(ctx, "", "ls", []string{"ls", "-la"}) },
		func() error { return h.CheckNetDial(ctx, "", "tcp", "1.2.3.4:80") },
		func() error { return h.CheckNetListen(ctx, "", "tcp", ":8080") },
		func() error { return h.CheckGetenv(ctx, "", "PATH") },
		func() error { return h.CheckSetenv(ctx, "", "FOO", "bar") },
		func() error { return h.CheckSubprocess(ctx, "", "sh", []string{"sh", "-c", "echo"}) },
	}
	for i, check := range checks {
		if err := check(); err != nil {
			t.Errorf("permissive hook check %d returned %v, want nil", i, err)
		}
	}
}

func TestEffectiveCapabilityHook(t *testing.T) {
	t.Parallel()
	if got := effectiveCapabilityHook(nil); got != permissiveHook {
		t.Fatalf("effectiveCapabilityHook(nil) = %v, want permissiveHook", got)
	}
	hook := &recordingHook{}
	if got := effectiveCapabilityHook(hook); got != hook {
		t.Fatalf("effectiveCapabilityHook(hook) = %v, want supplied hook", got)
	}
}

func TestWrapNativeFunctionAllows(t *testing.T) {
	t.Parallel()
	fn := func(s string) string { return strings.ToUpper(s) }
	wrapped := wrapNativeFunction(nil, reflect.ValueOf(fn), "test.ToUpper", nil)
	results := wrapped.Call([]reflect.Value{reflect.ValueOf("hello")})
	if len(results) != 1 || results[0].String() != "HELLO" {
		t.Fatalf("wrapped function returned %v, want [HELLO]", results)
	}
}

func TestWrapNativeFunctionDeniesWithErrorReturn(t *testing.T) {
	t.Parallel()
	fn := func(path string) ([]byte, error) {
		return []byte("data"), nil
	}
	denial := errors.New("denied by policy")
	gate := func(_ context.Context, _ CapabilityHook, _ string, _ []reflect.Value) error {
		return denial
	}
	hook := &recordingHook{}
	provider := &staticHookProvider{hook: hook, modulePath: "test/mod"}
	wrapped := wrapNativeFunction(provider, reflect.ValueOf(fn), "os.ReadFile", gate)

	results := wrapped.Call([]reflect.Value{reflect.ValueOf("/etc/passwd")})
	if len(results) != 2 {
		t.Fatalf("wrapped call returned %d results, want 2", len(results))
	}
	if !results[0].IsZero() || results[0].Len() != 0 {
		t.Fatalf("denied call leaked non-zero first result %v", results[0])
	}
	gotErr, ok := results[1].Interface().(error)
	if !ok || !errors.Is(gotErr, denial) {
		t.Fatalf("denied call returned err %v, want %v", gotErr, denial)
	}
}

func TestWrapNativeFunctionPanicsWithoutErrorReturn(t *testing.T) {
	t.Parallel()
	fn := func(s string) string { return s }
	denial := errors.New("nope")
	gate := func(_ context.Context, _ CapabilityHook, _ string, _ []reflect.Value) error {
		return denial
	}
	hook := &recordingHook{}
	provider := &staticHookProvider{hook: hook}
	wrapped := wrapNativeFunction(provider, reflect.ValueOf(fn), "no.errret", gate)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic for gated function without error return")
		}
		err, ok := recovered.(error)
		if !ok || !errors.Is(err, denial) {
			t.Fatalf("panic value %v does not wrap denial %v", recovered, denial)
		}
	}()
	wrapped.Call([]reflect.Value{reflect.ValueOf("x")})
}

func TestWrapNativeFunctionGenericGateRecordsCall(t *testing.T) {
	t.Parallel()
	fn := func(s string) string { return s }
	hook := &recordingHook{}
	provider := &staticHookProvider{hook: hook, modulePath: "mod/foo"}
	wrapped := wrapNativeFunction(provider, reflect.ValueOf(fn), "fmt.Sprintln", nil)

	wrapped.Call([]reflect.Value{reflect.ValueOf("x")})

	calls := hook.snapshot()
	if len(calls) != 1 {
		t.Fatalf("got %d hook calls, want 1: %#v", len(calls), calls)
	}
	if calls[0].Method != "CheckFunctionCall" {
		t.Fatalf("hook method %q, want CheckFunctionCall", calls[0].Method)
	}
	if calls[0].ModulePath != "mod/foo" {
		t.Fatalf("hook modulePath %q, want mod/foo", calls[0].ModulePath)
	}
	if calls[0].Fields["fnPath"] != "fmt.Sprintln" {
		t.Fatalf("hook fnPath %q, want fmt.Sprintln", calls[0].Fields["fnPath"])
	}
}

func hookTestSymbols() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"hooktest": {
			"Upper":   reflect.ValueOf(func(s string) string { return strings.ToUpper(s) }),
			"Sprintf": reflect.ValueOf(func(format, value string) string { return format + ":" + value }),
		},
	})
}

func TestCapabilityHookFiresOnNativeCall(t *testing.T) {
	t.Parallel()
	hook := &recordingHook{}
	service := NewService(WithCapabilityHook(hook), WithForceGoDispatch())
	service.UseSymbols(hookTestSymbols())

	result, err := service.Eval(context.Background(), `
		import "hooktest"
		hooktest.Upper("hello")
	`)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if result != "HELLO" {
		t.Fatalf("Eval returned %v, want HELLO", result)
	}

	calls := hook.snapshot()
	if len(calls) == 0 {
		t.Fatalf("expected at least one hook call, got none")
	}
	var foundUpper bool
	for _, call := range calls {
		if call.Method != "CheckFunctionCall" {
			continue
		}
		fnPath, ok := call.Fields["fnPath"].(string)
		if !ok {
			continue
		}

		if fnPath != "" {
			foundUpper = true
		}
	}
	if !foundUpper {
		t.Fatalf("did not see CheckFunctionCall with resolved fnPath; got %d calls: %#v", len(calls), calls)
	}
}

func TestCapabilityHookDeniesNativeCall(t *testing.T) {
	t.Parallel()
	hook := &recordingHook{
		denyFn: func(call recordedHookCall) error {
			if call.Method == "CheckFunctionCall" {
				return errors.New("call denied by policy")
			}
			return nil
		},
	}
	service := NewService(WithCapabilityHook(hook), WithForceGoDispatch())
	service.UseSymbols(hookTestSymbols())

	_, err := service.Eval(context.Background(), `
		import "hooktest"
		hooktest.Upper("hello")
	`)
	if err == nil {
		t.Fatalf("expected Eval to fail when hook denies, got nil")
	}
	if !strings.Contains(err.Error(), "denied by policy") {
		t.Fatalf("error %q does not contain denial message", err.Error())
	}
}

func TestWrapNativeFunctionDeniesViaSpecificGate(t *testing.T) {
	t.Parallel()
	fn := func(path string) (*os.File, error) { return nil, nil }
	hook := &recordingHook{
		denyFn: func(call recordedHookCall) error {
			if call.Method == "CheckFileOpen" && call.Fields["path"] == "/etc/shadow" {
				return errors.New("path denied")
			}
			return nil
		},
	}
	provider := &staticHookProvider{hook: hook, modulePath: "mod/secret"}
	gate := func(ctx context.Context, h CapabilityHook, modulePath string, args []reflect.Value) error {
		return h.CheckFileOpen(ctx, modulePath, args[0].String(), os.O_RDONLY, 0)
	}
	wrapped := wrapNativeFunction(provider, reflect.ValueOf(fn), "os.Open", gate)

	results := wrapped.Call([]reflect.Value{reflect.ValueOf("/etc/shadow")})
	err, ok := results[1].Interface().(error)
	if !ok || err == nil {
		t.Fatalf("expected error for denied path, got nil")
	}

	results = wrapped.Call([]reflect.Value{reflect.ValueOf("/tmp/allowed")})
	if err, ok := results[1].Interface().(error); ok && err != nil {
		t.Fatalf("unexpected error for allowed path: %v", err)
	}

	calls := hook.snapshot()
	if len(calls) != 2 {
		t.Fatalf("expected 2 hook calls, got %d", len(calls))
	}
	for _, call := range calls {
		if call.Method != "CheckFileOpen" {
			t.Errorf("unexpected hook method %q", call.Method)
		}
		if call.ModulePath != "mod/secret" {
			t.Errorf("unexpected hook modulePath %q", call.ModulePath)
		}
	}
}
