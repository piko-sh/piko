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

package db_provider_seagull

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
)

// panicError turns a panic recovered from Seagull into an error.
//
// The stack is logged once at warning level; the caller receives an error without it.
// Call it from a deferred function that has just recovered a non-nil value.
//
// Takes operation (string) which names what was running, for the error and the log.
// Takes recovered (any) which is the value recover returned.
//
// Returns error which describes the panic.
func panicError(ctx context.Context, operation string, recovered any) error {
	slog.Default().WarnContext(ctx, "recovered a panic from the SQL toolchain",
		slog.String("operation", operation),
		slog.Any("panic", recovered),
		slog.String("stack", string(debug.Stack())),
	)
	return fmt.Errorf("%s: SQL toolchain panicked: %v", operation, recovered)
}
