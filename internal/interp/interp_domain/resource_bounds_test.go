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
	"strings"
	"testing"
)

func TestParentContextPropagatesToChildVM(t *testing.T) {
	t.Parallel()

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	parentVM := newVM(parentCtx, newGlobalStore(), NewSymbolRegistry(nil))

	child := newVM(parentVM.ctx, parentVM.globals, parentVM.symbols)
	if child.ctx != parentCtx {
		t.Fatalf("child VM did not inherit parent context")
	}

	parentCancel()
	if err := child.ctx.Err(); err == nil {
		t.Fatalf("after parent cancel, child ctx.Err() == nil; want non-nil")
	}
}

func TestGoroutineLimitIsWiredThroughService(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxGoroutines(4))
	if service.limits.maxGoroutines != 4 {
		t.Fatalf("limits.maxGoroutines = %d, want 4", service.limits.maxGoroutines)
	}
}

func TestErrGoroutineLimitIsExported(t *testing.T) {
	t.Parallel()
	if errGoroutineLimit == nil {
		t.Fatalf("errGoroutineLimit unexported or nil")
	}
	wrapped := errors.Join(errors.New("wrapper"), errGoroutineLimit)
	if !errors.Is(wrapped, errGoroutineLimit) {
		t.Fatalf("errors.Is failed to unwrap errGoroutineLimit")
	}
	if !strings.Contains(errGoroutineLimit.Error(), "goroutine") {
		t.Fatalf("errGoroutineLimit message %q lacks 'goroutine'", errGoroutineLimit.Error())
	}
}

func TestWithCapabilityHookAndForceGoDispatchCompose(t *testing.T) {
	t.Parallel()
	hook := &recordingHook{}
	service := NewService(
		WithCapabilityHook(hook),
		WithForceGoDispatch(),
		WithMaxGoroutines(8),
		WithMaxCallDepth(1024),
	)
	if service.CapabilityHook() != hook {
		t.Fatalf("CapabilityHook not preserved across option list")
	}
	if service.limits.maxGoroutines != 8 {
		t.Fatalf("maxGoroutines = %d, want 8", service.limits.maxGoroutines)
	}
	if service.limits.maxCallDepth != 1024 {
		t.Fatalf("maxCallDepth = %d, want 1024", service.limits.maxCallDepth)
	}
}
