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

package interp_provider_pipit

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// errInterpreterPanic reports a panic raised inside the interpreter and recovered at the
	// provider boundary.
	errInterpreterPanic = errors.New("interpreter panicked")
)

// interpreterPanicError turns a value recovered from a panic inside the interpreter into
// an error.
//
// The stack is logged once at warning level through the context logger, and the returned
// error carries only the recovered value so callers never see internal stack frames.
//
// Takes operation (string) which names the interpreter call for the log and the error.
// Takes recovered (any) which is the value returned by recover.
//
// Returns error which wraps errInterpreterPanic.
func interpreterPanicError(ctx context.Context, operation string, recovered any) error {
	_, l := logger_domain.From(ctx, log)
	l.Warn("Interpreter panicked",
		logger_domain.String("operation", operation),
		logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
		logger_domain.String("stack", string(debug.Stack())),
	)
	return fmt.Errorf("%w while %s: %v", errInterpreterPanic, operation, recovered)
}
