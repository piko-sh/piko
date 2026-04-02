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
	"fmt"
	"os"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
)

// CLIDiagnosticOutput writes formatted diagnostics to stderr for use on the
// command line. It implements DiagnosticOutputPort and keeps ANSI colour codes
// for clear output in terminal windows.
type CLIDiagnosticOutput struct{}

// NewCLIDiagnosticOutput creates a new CLI diagnostic output adapter.
//
// Returns *CLIDiagnosticOutput which is ready for use.
func NewCLIDiagnosticOutput() *CLIDiagnosticOutput {
	return &CLIDiagnosticOutput{}
}

// OutputDiagnostics writes richly formatted diagnostics to
// stderr using ANSI colour codes.
//
// This provides excellent DX for developers working in terminal
// environments by grouping diagnostics by file, showing source
// code context with syntax highlighting, using colours to
// distinguish errors from warnings, and displaying precise line
// and column information.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains the
// diagnostic messages to display.
// Takes sourceContents (map[string][]byte) which maps file paths to
// their source code for context display.
func (*CLIDiagnosticOutput) OutputDiagnostics(
	diagnostics []*ast_domain.Diagnostic,
	sourceContents map[string][]byte,
	_ bool,
) {
	if len(diagnostics) == 0 {
		return
	}

	formattedDiags := annotator_domain.FormatAllDiagnostics(diagnostics, sourceContents)
	_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", formattedDiags)
}
