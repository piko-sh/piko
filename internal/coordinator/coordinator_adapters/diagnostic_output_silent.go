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

package coordinator_adapters

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// SilentDiagnosticOutput implements DiagnosticOutputPort for cases where
// diagnostics are sent by other means, such as LSP mode where writing to
// stderr would break the JSON-RPC stream.
type SilentDiagnosticOutput struct{}

// NewSilentDiagnosticOutput creates a new silent diagnostic output adapter.
//
// Returns *SilentDiagnosticOutput which discards all diagnostic messages.
func NewSilentDiagnosticOutput() *SilentDiagnosticOutput {
	return &SilentDiagnosticOutput{}
}

// OutputDiagnostics does nothing by design.
// Diagnostics are sent through the LSP protocol's publishDiagnostics message
// in the LSP workspace layer instead.
func (*SilentDiagnosticOutput) OutputDiagnostics(
	_ []*ast_domain.Diagnostic,
	_ map[string][]byte,
	_ bool,
) {
}
