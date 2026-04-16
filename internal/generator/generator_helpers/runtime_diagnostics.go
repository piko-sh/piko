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

package generator_helpers

import (
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/colour"
	"piko.sh/piko/internal/generator/generator_dto"
)

var (
	// errorColour is the ANSI colour used for runtime error severity labels.
	errorColour = colour.New(colour.FgRed, colour.Bold)

	// warningColour is the ANSI colour used for runtime warning severity labels.
	warningColour = colour.New(colour.FgYellow, colour.Bold)

	// infoColour is the ANSI colour used for runtime informational severity labels.
	infoColour = colour.New(colour.FgBlue, colour.Bold)

	// pathColour is the ANSI colour used for source file paths in runtime diagnostics.
	pathColour = colour.New(colour.FgCyan)

	// gutterColour is the ANSI colour used for line number gutters in runtime diagnostics.
	gutterColour = colour.New(colour.FgBlue, colour.Bold)

	// messageColour is the ANSI colour used for runtime diagnostic message text.
	messageColour = colour.New(colour.Bold)

	// rejectedPikoElementTags lists tag names that cannot be used as the resolved
	// target of a <piko:element :is="..."> at runtime.
	rejectedPikoElementTags = map[string]bool{
		"piko:partial": true,
		"piko:slot":    true,
		"piko:element": true,
	}
)

// AppendDiagnostic appends d to diagnostics and returns the updated slice.
// Uses return-style (like Go's built-in append) so the caller's slice
// variable stays stack-allocated.
//
// Takes diagnostics ([]*generator_dto.RuntimeDiagnostic) which is the slice
// to append to.
// Takes d (*generator_dto.RuntimeDiagnostic) which is the diagnostic to append.
//
// Returns []*generator_dto.RuntimeDiagnostic which is the updated slice.
func AppendDiagnostic(
	diagnostics []*generator_dto.RuntimeDiagnostic,
	d *generator_dto.RuntimeDiagnostic,
) []*generator_dto.RuntimeDiagnostic {
	return append(diagnostics, d)
}

// ValidatePikoElementTagName checks that a dynamically resolved tag name for
// <piko:element :is="..."> is valid. If the tag is empty or a rejected target,
// it appends a runtime diagnostic and returns "div" as a safe fallback.
//
// Takes tag (string) which is the resolved tag name.
// Takes diagnostics ([]*generator_dto.RuntimeDiagnostic) which collects
// runtime issues.
// Takes sourcePath (string) which identifies the source file.
// Takes expression (string) which is the original :is expression.
// Takes line (int) which is the line number.
// Takes column (int) which is the column number.
//
// Returns string which is the validated tag name or "div" if invalid.
// Returns []*generator_dto.RuntimeDiagnostic which is the updated diagnostics
// slice.
func ValidatePikoElementTagName(
	tag string,
	diagnostics []*generator_dto.RuntimeDiagnostic,
	sourcePath, expression string,
	line, column int,
) (string, []*generator_dto.RuntimeDiagnostic) {
	if tag == "" {
		diagnostics = AppendDiagnostic(diagnostics, &generator_dto.RuntimeDiagnostic{
			Severity:   generator_dto.Error,
			Message:    "<piko:element> resolved to an empty tag name",
			Code:       generator_dto.CodeEmptyElementTag,
			SourcePath: sourcePath,
			Expression: expression,
			Line:       line,
			Column:     column,
		})
		return "div", diagnostics
	}
	if rejectedPikoElementTags[tag] {
		diagnostics = AppendDiagnostic(diagnostics, &generator_dto.RuntimeDiagnostic{
			Severity:   generator_dto.Error,
			Message:    fmt.Sprintf("<piko:element> cannot target '%s'", tag),
			Code:       generator_dto.CodeRejectedElementTag,
			SourcePath: sourcePath,
			Expression: expression,
			Line:       line,
			Column:     column,
		})
		return "div", diagnostics
	}
	return tag, diagnostics
}

// FormatRuntimeDiagnostics formats runtime diagnostics into a rich,
// pretty-printed string suitable for console output.
//
// When the diagnostics slice is empty, returns an empty string. Handles
// invalid location data gracefully.
//
// Takes diagnostics ([]*generator_dto.RuntimeDiagnostic) which contains the
// runtime issues to format.
// Takes baseDir (string) which is the project root directory used to
// reconstruct absolute paths from relative paths for IDE navigation.
// Pass an empty string to display paths as-is.
//
// Returns string which is the formatted, coloured output for console display.
func FormatRuntimeDiagnostics(diagnostics []*generator_dto.RuntimeDiagnostic, baseDir string) string {
	if len(diagnostics) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(errorColour.Sprintf("--> Found %d runtime issue(s):\n\n", len(diagnostics)))

	for _, d := range diagnostics {
		var primaryColour colour.Style
		switch d.Severity {
		case generator_dto.Error:
			primaryColour = errorColour
		case generator_dto.Warning:
			primaryColour = warningColour
		case generator_dto.Info:
			primaryColour = infoColour
		default:
			primaryColour = messageColour
		}

		builder.WriteString(primaryColour.Sprintf(" ● %s", d.Severity))
		if d.Code != "" {
			builder.WriteString(messageColour.Sprintf(" [%s]", d.Code))
		}
		builder.WriteString(messageColour.Sprintf(": %s\n", d.Message))

		displayPath := d.SourcePath
		if baseDir != "" && d.SourcePath != "" && !filepath.IsAbs(d.SourcePath) {
			displayPath = filepath.Join(baseDir, d.SourcePath)
		}

		fmt.Fprintf(&builder, "   %s %s:%d:%d\n",
			gutterColour.Sprint("-->"),
			pathColour.Sprint(displayPath),
			d.Line,
			d.Column,
		)

		builder.WriteString(gutterColour.Sprint("    |") + "\n")

		if d.Expression != "" {
			fmt.Fprintf(&builder, "%s %s\n", gutterColour.Sprint("    |"), d.Expression)

			caretColumn := max(d.Column, 1)
			caretPadding := strings.Repeat(" ", caretColumn-1)

			highlightLen := len(d.Expression)
			if highlightLen == 0 {
				highlightLen = 1
			}
			highlight := strings.Repeat("^", highlightLen)

			fmt.Fprintf(&builder, "%s %s%s\n", gutterColour.Sprint("    |"), caretPadding, primaryColour.Sprint(highlight))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}
