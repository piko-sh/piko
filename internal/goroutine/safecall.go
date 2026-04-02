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

package goroutine

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
)

// PanicError wraps a recovered panic value as a standard error,
// returned by SafeCall functions when user-provided code panics.
//
// Callers can use errors.As to distinguish panics from normal errors.
type PanicError struct {
	// Component identifies the provider or operation that panicked.
	Component string

	// Value is the recovered panic value.
	Value any

	// Stack is the stack trace captured at the point of recovery.
	Stack string
}

// Error returns a string describing the panic, including the
// component name and the panic value.
//
// Returns string which describes the panic.
func (e *PanicError) Error() string {
	return fmt.Sprintf("panic in %s: %v", e.Component, e.Value)
}

// ProviderTimeoutError indicates a provider returned a context
// deadline or cancellation error that did NOT originate from the
// caller's context.
//
// This means the provider created its own context with a timeout
// that expired, or cancelled its own context. Callers can use
// errors.As to distinguish provider-internal timeouts from caller
// timeouts.
type ProviderTimeoutError struct {
	// Err is the original error returned by the provider.
	Err error

	// Component identifies the provider or operation that timed out.
	Component string
}

// Error returns a string describing the provider timeout, including
// the component name and the original error.
//
// Returns string which describes the provider timeout.
func (e *ProviderTimeoutError) Error() string {
	return fmt.Sprintf("provider timeout in %s: %v", e.Component, e.Err)
}

// Unwrap returns the original error, preserving the error chain so
// that errors.Is(err, context.DeadlineExceeded) continues to work.
//
// Returns error which is the original provider error.
func (e *ProviderTimeoutError) Unwrap() error {
	return e.Err
}

// SafeCall calls operation and recovers from any panic, returning the panic as a
// *PanicError. Designed for provider methods that return error.
//
// If the provider returns a context deadline or cancellation error but the
// caller's context is still alive, the error is wrapped in a
// *ProviderTimeoutError to attribute the timeout to the provider.
//
// Takes ctx (context.Context) which carries trace/baggage for OTel metrics.
// Takes component (string) which identifies the provider for logging.
// Takes operation (func() error) which is the provider method to call.
//
// Returns error which wraps the panic as a *PanicError if operation panicked, or
// wraps a provider-internal timeout as a *ProviderTimeoutError.
func SafeCall(ctx context.Context, component string, operation func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanicRecovery(ctx, component, r)
		}
	}()

	err = operation()
	if err != nil {
		err = enrichProviderTimeout(ctx, component, err)
	}
	return err
}

// SafeCall1 calls operation and recovers from any panic, returning the panic as a
// *PanicError. Designed for provider methods that return (T, error).
//
// Takes ctx (context.Context) which carries trace/baggage for OTel metrics.
// Takes component (string) which identifies the provider for logging.
// Takes operation (func() (T, error)) which is the provider method to call.
//
// Returns T which is the zero value if operation panicked.
// Returns error which wraps the panic as a *PanicError if operation panicked.
func SafeCall1[T any](ctx context.Context, component string, operation func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanicRecovery(ctx, component, r)
		}
	}()

	result, err = operation()
	if err != nil {
		err = enrichProviderTimeout(ctx, component, err)
	}
	return result, err
}

// SafeCall2 calls operation and recovers from any panic, returning the panic as a
// *PanicError. Designed for provider methods that return (T1, T2, error).
//
// Takes ctx (context.Context) which carries trace/baggage for OTel metrics.
// Takes component (string) which identifies the provider for logging.
// Takes operation (func() (T1, T2, error)) which is the provider method to call.
//
// Returns T1 which is the zero value if operation panicked.
// Returns T2 which is the zero value if operation panicked.
// Returns error which wraps the panic as a *PanicError if operation panicked.
func SafeCall2[T1, T2 any](ctx context.Context, component string, operation func() (T1, T2, error)) (r1 T1, r2 T2, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanicRecovery(ctx, component, r)
		}
	}()

	r1, r2, err = operation()
	if err != nil {
		err = enrichProviderTimeout(ctx, component, err)
	}
	return r1, r2, err
}

// SafeCallValue calls operation and recovers from any panic, returning
// the zero value for provider methods with no error return.
//
// The panic is logged and counted but cannot be propagated as an
// error since the method signature has no error return.
//
// Takes ctx (context.Context) which carries trace/baggage for OTel metrics.
// Takes component (string) which identifies the provider for logging.
// Takes operation (func() T) which is the provider method to call.
//
// Returns T which is the zero value if operation panicked.
func SafeCallValue[T any](ctx context.Context, component string, operation func() T) (result T) {
	defer func() {
		if r := recover(); r != nil {
			_ = handlePanicRecovery(ctx, component, r)
		}
	}()

	return operation()
}

// enrichProviderTimeout checks whether an error from a provider is a
// context deadline or cancellation that originated inside the
// provider. If so, it wraps the error in a *ProviderTimeoutError
// for attribution.
//
// Takes component (string) which identifies the provider.
// Takes err (error) which is the error to inspect.
//
// Returns error which is either the original error or a wrapped
// *ProviderTimeoutError.
func enrichProviderTimeout(ctx context.Context, component string, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("provider %s returned error after context cancellation: %w", component, err)
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		ctx, l := logger_domain.From(ctx, log)
		l.Warn("Provider context error detected",
			logger_domain.String(logger_domain.FieldStrComponent, component),
			logger_domain.Error(err),
		)
		ProviderTimeoutCount.Add(ctx, 1)
		return &ProviderTimeoutError{Component: component, Err: err}
	}
	return fmt.Errorf("provider %s: %w", component, err)
}

// handlePanicRecovery logs the panic, increments the OTel counter,
// and returns a PanicError. This is shared by all SafeCall variants
// and by RecoverPanic.
//
// Takes component (string) which identifies the panicking provider.
// Takes r (any) which is the recovered panic value.
//
// Returns *PanicError which wraps the panic details.
func handlePanicRecovery(ctx context.Context, component string, r any) *PanicError {
	ctx, l := logger_domain.From(ctx, log)
	stack := string(debug.Stack())

	l.Error("Provider panicked",
		logger_domain.String(logger_domain.FieldStrComponent, component),
		logger_domain.String("panic_info", fmt.Sprintf("%v", r)),
		logger_domain.String("stack_trace", stack),
	)

	PanicRecoveryCount.Add(ctx, 1)

	return &PanicError{
		Component: component,
		Value:     r,
		Stack:     stack,
	}
}
