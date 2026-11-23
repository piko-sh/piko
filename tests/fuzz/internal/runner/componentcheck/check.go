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

// Package componentcheck runs the production PKC compiler inside the selected worker.
package componentcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// sourceMax bounds the PKC source read for one component, in bytes.
	sourceMax = 4 << 20

	// componentDirectory is the project directory holding component sources.
	componentDirectory = "components"
)

var (
	// errSourceOversize reports a component source above sourceMax bytes.
	errSourceOversize = errors.New("component source exceeds byte limit")
)

// Verify compiles the selected PKC and validates its generated JavaScript.
//
// Takes tag (string) which identifies the custom element.
//
// Returns error which reports an invalid component name, unreadable or oversized source,
// or failed PKC compilation joining every error diagnostic, or nil on success.
func Verify(ctx context.Context, tag string) error {
	if tag == "" {
		return nil
	}
	if filepath.Base(tag) != tag {
		return fmt.Errorf("invalid component name %q", tag)
	}
	name := filepath.Join(componentDirectory, tag+".pkc")
	source, err := readSource(componentDirectory, tag+".pkc")
	if err != nil {
		return err
	}
	compiler := compiler_domain.NewCompilerOrchestrator(nil, nil, compiler_domain.WithOrchestratorModuleName(casefile.Module))
	result, err := compiler.CompileSFCBytes(ctx, name, source)
	if err != nil {
		return err
	}
	if result == nil || result.TagName != tag {
		return fmt.Errorf("compiler did not produce <%s>", tag)
	}
	if result.BaseJSPath == "" || strings.TrimSpace(result.Files[result.BaseJSPath]) == "" {
		return fmt.Errorf("missing main JavaScript for <%s>", tag)
	}
	return diagnosticErrors(result.Diagnostics)
}

// diagnosticErrors joins every error-severity compiler diagnostic.
//
// Takes diagnostics ([]compiler_dto.CompilationDiagnostic) which holds the compiler's
// diagnostics.
//
// Returns error which joins one error per error diagnostic, or nil when there is none.
func diagnosticErrors(diagnostics []compiler_dto.CompilationDiagnostic) error {
	var failures []error
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == compiler_dto.DiagnosticSeverityError {
			failures = append(failures, errors.New(diagnostic.Message))
		}
	}
	return errors.Join(failures...)
}

// readSource reads one component source inside its directory, refusing one above
// sourceMax bytes.
//
// Takes directory (string) which is the directory the source must stay within.
// Takes name (string) which is the source file name within directory.
//
// Returns []byte which holds the source bytes.
// Returns error which reports an unreadable or escaping source, or wraps
// errSourceOversize.
func readSource(directory, name string) (source []byte, err error) {
	file, err := os.OpenInRoot(directory, name)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	source, err = io.ReadAll(io.LimitReader(file, sourceMax+1))
	if err != nil {
		return nil, err
	}
	if len(source) > sourceMax {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", errSourceOversize, name, sourceMax)
	}
	return source, nil
}
