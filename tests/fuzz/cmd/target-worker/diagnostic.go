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

package main

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// checkDiagnosticPair checks the valid and invalid paired-source diagnostics.
//
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkDiagnosticPair(ctx context.Context, _ *tape.Tape, savedCase *casefile.Case) error {
	expected := savedCase.Expected.Diagnostic
	if expected == nil {
		return property("model", "missing diagnostic pair")
	}
	if expected.SectionPath != "" {
		source, present := savedCase.Files[expected.SectionPath]
		if !present {
			return &propertyError{ID: "missing-source", Source: expected.SectionPath, Err: fmt.Errorf("missing section source %s", expected.SectionPath)}
		}
		blocks, err := sfcparser.Parse([]byte(source))
		if err != nil {
			return fmt.Errorf("parsing section source %s: %w", expected.SectionPath, err)
		}
		if err = checkSectionSpans(source, blocks, expected.Sections); err != nil {
			if failure, ok := errors.AsType[*propertyError](err); ok {
				failure.Source = expected.SectionPath
			}
			return err
		}
	}
	for _, stage := range []struct {
		id, path string
		valid    bool
	}{
		{id: "clean", path: expected.GoodPath, valid: true},
		{id: "edit", path: expected.BadPath, valid: false},
		{id: "repair", path: expected.GoodPath, valid: true},
	} {
		if err := checkDiagnosticStage(ctx, savedCase, expected, stage.id, stage.path, stage.valid); err != nil {
			return err
		}
	}
	return nil
}

// checkDiagnosticStage checks one stage of a diagnostic pair against its expectation.
//
// Takes savedCase (*casefile.Case) which holds the saved sources.
// Takes expected (*casefile.DiagnosticPair) which describes the expected diagnostic.
// Takes stage (string) which names the clean, edit or repair stage.
// Takes path (string) which is the source checked at this stage.
// Takes valid (bool) which reports whether the stage source must be diagnostic-free.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkDiagnosticStage(ctx context.Context, savedCase *casefile.Case, expected *casefile.DiagnosticPair, stage, path string, valid bool) error {
	source, present := savedCase.Files[path]
	if !present {
		return &propertyError{ID: "missing-source", Source: path, Err: fmt.Errorf("missing %s source %s", stage, path)}
	}
	blocks, err := sfcparser.Parse([]byte(source))
	if err != nil {
		return fmt.Errorf("parsing %s source %s: %w", stage, path, err)
	}
	position := ast_domain.Location{Line: blocks.TemplateContentLocation.Line, Column: blocks.TemplateContentLocation.Column, Offset: 0}
	tree, err := ast_domain.Parse(ctx, blocks.Template, path, &position)
	if err != nil {
		return fmt.Errorf("parsing %s template %s: %w", stage, path, err)
	}
	if tree == nil {
		return property("diagnostic-tree", "%s/%s: no parse tree", expected.ID, stage)
	}
	defer ast_domain.PutTree(tree)
	diagnostics := tree.Diagnostics
	count, match := countErrorDiagnostics(diagnostics, expected, path)
	if valid && count == 0 || !valid && match == 1 && count == 1 {
		return nil
	}
	failure := &propertyError{
		ID:     "diagnostic-" + stage,
		Source: "",
		Err: fmt.Errorf("%s expected %s at %d:%d (expression offset %d): %+v",
			expected.ID, expected.Code, expected.Line, expected.Column, expected.ExpressionOffset, diagnostics),
	}
	if expected.Column != 0 {
		failure.Source = path
	}
	return failure
}

// countErrorDiagnostics counts error diagnostics and those matching the expectation.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which lists the parser's diagnostics.
// Takes expected (*casefile.DiagnosticPair) which describes the expected diagnostic.
// Takes path (string) which is the source the diagnostic must name.
//
// Returns count (int) which is the number of error diagnostics.
// Returns match (int) which is the number of error diagnostics matching the expectation.
func countErrorDiagnostics(diagnostics []*ast_domain.Diagnostic, expected *casefile.DiagnosticPair, path string) (count, match int) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != ast_domain.Error {
			continue
		}
		count++
		exactPosition := expected.Column == 0 ||
			diagnostic.Location.Column == expected.Column &&
				diagnostic.Location.Offset == expected.ExpressionOffset &&
				diagnostic.Expression == expected.Expression
		if diagnostic.Code == expected.Code && diagnostic.Location.Line == expected.Line && diagnostic.SourcePath == path && exactPosition {
			match++
		}
	}
	return count, match
}
