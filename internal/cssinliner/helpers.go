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

package cssinliner

import (
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/css_ast"
	es_logger "piko.sh/piko/internal/esbuild/logger"
)

// CleanCSSTree walks the CSS AST and removes any rules where Data is nil.
// This is needed because the esbuild minifier can leave empty rules in nested
// blocks (such as @media) that the printer cannot handle.
//
// Takes rules ([]css_ast.Rule) which is the slice of CSS rules to clean.
//
// Returns []css_ast.Rule which is the filtered slice with nil rules removed.
func CleanCSSTree(rules []css_ast.Rule) []css_ast.Rule {
	if rules == nil {
		return nil
	}

	n := 0
	for i := range len(rules) {
		rule := &rules[i]
		if rule.Data == nil {
			continue
		}

		switch r := rule.Data.(type) {
		case *css_ast.RAtKeyframes:
			for j := range r.Blocks {
				r.Blocks[j].Rules = CleanCSSTree(r.Blocks[j].Rules)
			}
		case *css_ast.RKnownAt:
			r.Rules = CleanCSSTree(r.Rules)
		case *css_ast.RAtMedia:
			r.Rules = CleanCSSTree(r.Rules)
		case *css_ast.RAtLayer:
			r.Rules = CleanCSSTree(r.Rules)
		case *css_ast.RSelector:
			r.Rules = CleanCSSTree(r.Rules)
		}

		rules[n] = *rule
		n++
	}
	return rules[:n]
}

// ConvertESBuildMessagesToDiagnostics converts esbuild log messages into
// diagnostic objects for the domain layer.
//
// When messages is empty, returns nil.
//
// Takes messages ([]es_logger.Msg) which contains the esbuild log messages to
// convert.
// Takes sourcePath (string) which identifies the source file for diagnostics.
// Takes startLocation (ast_domain.Location) which provides the offset for
// line and column values.
// Takes diagnosticCode (string) which is the code to assign to each
// diagnostic.
//
// Returns []*ast_domain.Diagnostic which contains the converted diagnostics.
func ConvertESBuildMessagesToDiagnostics(messages []es_logger.Msg, sourcePath string, startLocation ast_domain.Location, diagnosticCode string) []*ast_domain.Diagnostic {
	if len(messages) == 0 {
		return nil
	}

	diagnostics := make([]*ast_domain.Diagnostic, 0, len(messages))
	for _, message := range messages {
		var severity ast_domain.Severity
		switch message.Kind {
		case es_logger.Error:
			severity = ast_domain.Error
		case es_logger.Warning:
			severity = ast_domain.Warning
		case es_logger.Info:
			severity = ast_domain.Info
		default:
			continue
		}

		var finalLocation ast_domain.Location
		var expression string
		if message.Data.Location != nil {
			messageLocation := message.Data.Location
			line := messageLocation.Line + 1
			column := messageLocation.Column + 1

			if line == 1 {
				finalLocation.Line = startLocation.Line
				finalLocation.Column = startLocation.Column + column - 1
			} else {
				finalLocation.Line = startLocation.Line + line - 1
				finalLocation.Column = column
			}
			expression = messageLocation.LineText
		}

		diagnostics = append(diagnostics, &ast_domain.Diagnostic{
			Data:         nil,
			Message:      message.Data.Text,
			Expression:   expression,
			SourcePath:   sourcePath,
			Code:         diagnosticCode,
			RelatedInfo:  nil,
			Location:     finalLocation,
			SourceLength: 0,
			Severity:     severity,
		})
	}
	return diagnostics
}
