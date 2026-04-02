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
	"fmt"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
)

// RecoverPanic recovers from a panic in a goroutine, logging the panic
// information and stack trace at error level and incrementing an OTel
// counter.
//
// This function must be called via defer. It is designed to work alongside
// other deferred calls such as wg.Done(). Because Go executes defers in
// LIFO order, declare RecoverPanic after wg.Done so that recovery happens
// first:
//
//	go func() {
//	    defer wg.Done()
//	    defer goroutine.RecoverPanic(ctx, "mypackage.myLoop")
//	    // ...
//	}()
//
// Takes ctx (context.Context) which carries trace/baggage for OTel metrics.
// Takes component (string) which identifies the goroutine for logging and
// diagnostics. Use the format "package.functionName" for consistency.
func RecoverPanic(ctx context.Context, component string) {
	r := recover() //nolint:revive // called via defer
	if r == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	stack := string(debug.Stack())

	l.Error("Goroutine panicked",
		logger_domain.String(logger_domain.FieldStrComponent, component),
		logger_domain.String("panic_info", fmt.Sprintf("%v", r)),
		logger_domain.String("stack_trace", stack),
	)

	PanicRecoveryCount.Add(ctx, 1)
}

// RecoverPanicToChannel recovers from a panic in a goroutine, logging the
// panic information at error level, incrementing an OTel counter, and
// sending the panic as an error on the provided channel.
//
// This is intended for goroutines that communicate failures via error
// channels, such as background server processes. The panic is converted
// to an error and sent on errCh. If the channel is full or nil, the error
// is logged but not sent, to avoid blocking the recovery.
//
// This function must be called via defer. Declare it after close(errCh) in
// the defer stack so that recovery and the channel send happen before the
// close:
//
//	go func() {
//	    defer close(errCh)
//	    defer goroutine.RecoverPanicToChannel(ctx, "daemon.mainProcess", errCh)
//	    // ...
//	}()
//
// Takes ctx (context.Context) which carries trace/baggage for OTel metrics.
// Takes component (string) which identifies the goroutine for logging.
// Takes errCh (chan<- error) which receives the panic converted to an
// error.
func RecoverPanicToChannel(ctx context.Context, component string, errCh chan<- error) {
	r := recover() //nolint:revive // called via defer
	if r == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	stack := string(debug.Stack())

	l.Error("Goroutine panicked",
		logger_domain.String(logger_domain.FieldStrComponent, component),
		logger_domain.String("panic_info", fmt.Sprintf("%v", r)),
		logger_domain.String("stack_trace", stack),
	)

	PanicRecoveryCount.Add(ctx, 1)

	if errCh == nil {
		return
	}

	panicErr := fmt.Errorf("panic in %s: %v", component, r)

	select {
	case errCh <- panicErr:
	default:
		l.Error("Failed to send panic error to channel (channel full or closed)",
			logger_domain.String(logger_domain.FieldStrComponent, component),
		)
	}
}
