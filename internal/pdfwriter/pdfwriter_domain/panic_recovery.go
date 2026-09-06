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

package pdfwriter_domain

import (
	"context"
	"fmt"
	"runtime/debug"

	"piko.sh/piko/internal/logger/logger_domain"
)

// StorePanicAsError stores a value recovered from a panic in the PDF pipeline into err as
// an ordinary error.
//
// The stack is logged once at warn level so the inner fault can be diagnosed, while the
// stored error stays free of stack frames so it is safe to surface to callers. A nil
// recovered value means no panic occurred and leaves err untouched.
//
// Call it from the deferred function that recovers, passing recover() directly:
//
//	defer func() { StorePanicAsError(ctx, "layout", recover(), &err) }()
//
// so the logged stack still includes the frames that panicked.
//
// Takes operation (string) which names the pipeline stage being protected.
// Takes recovered (any) which is the value returned by recover.
// Takes err (*error) which receives the error describing the panic.
func StorePanicAsError(ctx context.Context, operation string, recovered any, err *error) {
	if recovered == nil {
		return
	}
	_, l := logger_domain.From(ctx, log)
	l.Warn("Recovered from a panic in the PDF pipeline",
		logger_domain.String("operation", operation),
		logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
		logger_domain.String("stack", string(debug.Stack())),
	)
	*err = fmt.Errorf("pdfwriter: panic during %s: %v", operation, recovered)
}
