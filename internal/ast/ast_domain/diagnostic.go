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

package ast_domain

// Defines diagnostic types and formatting utilities for presenting compilation errors, warnings, and informational messages.
// Provides severity levels, source location tracking, and coloured output formatting with source context for developer-friendly error reporting.

import (
	"cmp"
	"fmt"
	gohtml "html"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/colour"
)

const (
	// enableDiagnosticColour controls whether diagnostic output uses ANSI colour
	// codes.
	enableDiagnosticColour = true

	// tabWidth is the number of spaces used to display a tab character. This
	// matches the standard terminal default of 8.
	tabWidth = 8

	// spaceChar is a single space used for padding in diagnostic output.
	spaceChar = " "
)

// Severity represents how serious a diagnostic message is. It implements
// fmt.Stringer.
type Severity int

const (
	// Debug is for detailed information used to find and fix problems.
	Debug Severity = iota

	// Info indicates an informational diagnostic message.
	Info

	// Warning indicates a diagnostic that does not prevent compilation but
	// should be addressed.
	Warning

	// Error indicates a diagnostic that prevents successful compilation.
	Error
)

// String returns the severity level as a lowercase string.
//
// Returns string which is the name of the severity, such as "debug", "info",
// "warning", or "error". Returns "unknown" for unrecognised values.
func (s Severity) String() string {
	switch s {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// CodeString returns the capitalised string representation of the severity
// for use in code generation.
//
// Returns string which is the severity name with initial capital letter,
// or "unknown" for undefined severity values.
func (s Severity) CodeString() string {
	switch s {
	case Debug:
		return "Debug"
	case Info:
		return "Info"
	case Warning:
		return "Warning"
	case Error:
		return "Error"
	default:
		return "unknown"
	}
}

// Location represents a position within a source file.
type Location struct {
	// Line is the 1-based line number in the source file; 0 means no position.
	Line int

	// Column is the column number within the line, starting from 1.
	Column int

	// Offset is the byte position from the start of the source, starting at 0.
	Offset int
}

// Range represents a span in source code between two positions.
type Range struct {
	// Start is the beginning position of the range.
	Start Location

	// End is the position just after the last character.
	End Location
}

// IsSynthetic returns true if this location is synthetic (not from actual
// source).
//
// Returns bool which is true when the location has no real source position.
func (l Location) IsSynthetic() bool {
	return l.Line == 0
}

// Add combines two locations into one, useful for tracking compound
// expressions.
//
// Takes other (Location) which is the location to add to this one.
//
// Returns Location which is the combined result.
func (l Location) Add(other Location) Location {
	if other.Line == 0 {
		return l
	}
	if l.Line == 0 {
		return other
	}

	finalLine := l.Line + other.Line - 1
	var finalCol int

	if other.Line == 1 {
		finalCol = l.Column + other.Column - 1
	} else {
		finalCol = other.Column
	}

	return Location{
		Line:   finalLine,
		Column: finalCol,
		Offset: 0,
	}
}

// IsBefore reports whether this location comes before another location in the
// source file.
//
// Takes other (Location) which is the location to compare against.
//
// Returns bool which is true if this location comes before the other location.
func (l Location) IsBefore(other Location) bool {
	if l.Line < other.Line {
		return true
	}
	if l.Line == other.Line && l.Column < other.Column {
		return true
	}
	return false
}

// DiagnosticRelatedInfo represents additional locations and context for a
// diagnostic. It provides multi-location error information, such as showing
// both the usage site and the declaration site of a symbol.
type DiagnosticRelatedInfo struct {
	// Message is the text that explains the related information to the user.
	Message string

	// Location specifies the file position linked to this diagnostic.
	Location Location
}

// Diagnostic represents a compiler message such as an error, warning, or hint.
// It implements the error interface and holds structured metadata to support
// LSP quick fixes and IDE features.
type Diagnostic struct {
	// Data holds extra details for quick fixes and IDE features.
	// Passed directly to the LSP protocol and used by quick fix handlers.
	Data map[string]any

	// Message is a clear description of the issue for the user to read.
	Message string

	// Expression is the code fragment where the issue was found.
	Expression string

	// SourcePath is the path to the file where the diagnostic was found.
	SourcePath string

	// Code is the identifier for this diagnostic, used by quick fix handlers.
	Code string

	// RelatedInfo holds extra locations and messages that give context for this
	// diagnostic, such as where a symbol is declared or other related errors.
	RelatedInfo []DiagnosticRelatedInfo

	// Location specifies the line and column where the issue was found.
	Location Location

	// SourceLength is the byte count of the problematic expression in source.
	// A value of 0 means len(Expression) is used instead.
	SourceLength int

	// Severity indicates how serious this diagnostic is (Error or Warning).
	Severity Severity
}

// NewDiagnostic creates a new diagnostic message with the given
// details.
//
// The SourceLength will default to 0, which falls back to
// len(Expression) when needed.
//
// Takes sev (Severity) which specifies the severity level of
// the diagnostic.
// Takes message (string) which provides the diagnostic message
// text.
// Takes expression (string) which contains the source
// expression being diagnosed.
// Takes location (Location) which specifies where the
// diagnostic occurred.
// Takes sourcePath (string) which identifies the source file
// path.
//
// Returns *Diagnostic which is the constructed diagnostic
// instance.
func NewDiagnostic(sev Severity, message, expression string, location Location, sourcePath string) *Diagnostic {
	return &Diagnostic{
		Severity:     sev,
		Message:      message,
		Expression:   expression,
		Location:     location,
		SourcePath:   sourcePath,
		SourceLength: 0,
		Code:         "",
		RelatedInfo:  nil,
		Data:         nil,
	}
}

// NewDiagnosticForExpression creates a diagnostic with accurate
// source length from an expression's metadata.
//
// Takes sev (Severity) which specifies the diagnostic severity
// level.
// Takes message (string) which provides the diagnostic message
// text.
// Takes expression (Expression) which supplies source length
// metadata.
// Takes location (Location) which indicates where the
// diagnostic occurred.
// Takes sourcePath (string) which identifies the source file
// path.
//
// Returns *Diagnostic which is configured with the
// expression's source length.
func NewDiagnosticForExpression(sev Severity, message string, expression Expression, location Location, sourcePath string) *Diagnostic {
	return &Diagnostic{
		Severity:     sev,
		Message:      message,
		Expression:   expression.String(),
		Location:     location,
		SourcePath:   sourcePath,
		SourceLength: expression.GetSourceLength(),
		Code:         "",
		RelatedInfo:  nil,
		Data:         nil,
	}
}

// NewDiagnosticWithCode creates a diagnostic with a
// machine-readable code for LSP quick fixes.
//
// Takes sev (Severity) which specifies the severity level of
// the diagnostic.
// Takes message (string) which provides the human-readable
// diagnostic message.
// Takes expression (string) which contains the source
// expression being diagnosed.
// Takes code (string) which provides the machine-readable code
// for quick fixes.
// Takes location (Location) which specifies where the
// diagnostic occurs.
// Takes sourcePath (string) which identifies the source file
// path.
//
// Returns *Diagnostic which is the configured diagnostic ready
// for use.
func NewDiagnosticWithCode(sev Severity, message, expression, code string, location Location, sourcePath string) *Diagnostic {
	return &Diagnostic{
		Severity:     sev,
		Message:      message,
		Expression:   expression,
		Location:     location,
		SourcePath:   sourcePath,
		SourceLength: 0,
		Code:         code,
		RelatedInfo:  nil,
		Data:         make(map[string]any),
	}
}

// NewDiagnosticWithData creates a diagnostic with structured
// metadata for LSP quick fixes.
//
// Takes sev (Severity) which specifies the severity level of
// the diagnostic.
// Takes message (string) which provides the human-readable
// diagnostic message.
// Takes expression (string) which contains the source
// expression being diagnosed.
// Takes code (string) which identifies the diagnostic rule or
// check.
// Takes location (Location) which specifies the position in
// the source file.
// Takes sourcePath (string) which provides the path to the
// source file.
// Takes data (map[string]any) which contains structured
// metadata for quick fixes.
//
// Returns *Diagnostic which is the configured diagnostic with
// the given data.
func NewDiagnosticWithData(sev Severity, message, expression, code string, location Location, sourcePath string, data map[string]any) *Diagnostic {
	return &Diagnostic{
		Severity:     sev,
		Message:      message,
		Expression:   expression,
		Location:     location,
		SourcePath:   sourcePath,
		SourceLength: 0,
		Code:         code,
		RelatedInfo:  nil,
		Data:         data,
	}
}

// GetEffectiveSourceLength returns the SourceLength if set, otherwise falls
// back to len(Expression). This preserves backward compatibility for diagnostics
// created before SourceLength was added.
//
// Returns int which is the effective length of the source text.
func (d *Diagnostic) GetEffectiveSourceLength() int {
	if d.SourceLength > 0 {
		return d.SourceLength
	}
	return len(d.Expression)
}

// Error implements the error interface, returning a formatted diagnostic
// message.
//
// Returns string which contains the severity, source path, line, column, and
// message.
func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s in %s at line %d, col %d: %s", d.Severity, d.SourcePath, d.Location.Line, d.Location.Column, d.Message)
}

var (
	// errorColour is the ANSI colour used for error severity labels and highlights.
	errorColour = colour.New(colour.FgRed, colour.Bold)

	// warningColour is the ANSI colour used for warning severity labels and highlights.
	warningColour = colour.New(colour.FgYellow, colour.Bold)

	// infoColour is the ANSI colour used for informational severity labels.
	infoColour = colour.New(colour.FgBlue, colour.Bold)

	// debugColour is the ANSI colour used for debug severity labels.
	debugColour = colour.New(colour.FgHiBlack, colour.Bold)

	// pathColour is the ANSI colour used for source file paths in diagnostic output.
	pathColour = colour.New(colour.FgCyan)

	// gutterColour is the ANSI colour used for line number gutters in diagnostic output.
	gutterColour = colour.New(colour.FgBlue, colour.Bold)

	// messageColour is the ANSI colour used for diagnostic message text.
	messageColour = colour.New(colour.Bold)
)

// Clone creates a shallow copy of the diagnostic.
//
// Returns *Diagnostic which is the copied diagnostic, or nil if the receiver
// is nil.
func (d *Diagnostic) Clone() *Diagnostic {
	if d == nil {
		return nil
	}
	return new(*d)
}

// HasErrors reports whether any diagnostic in the slice has Error severity.
//
// Takes diagnostics ([]*Diagnostic) which is the slice of diagnostics to check.
//
// Returns bool which is true if at least one diagnostic has Error severity.
func HasErrors(diagnostics []*Diagnostic) bool {
	for _, d := range diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// HasDiagnostics reports whether the slice contains any diagnostics.
//
// Takes diagnostics ([]*Diagnostic) which is the slice to check.
//
// Returns bool which is true when at least one diagnostic exists.
func HasDiagnostics(diagnostics []*Diagnostic) bool {
	return len(diagnostics) > 0
}

// DeduplicateDiagnostics removes duplicate diagnostics from a slice based on
// their identity (source path, location, severity, and message). This stops
// the same warning or error from appearing many times when a partial is used
// by more than one page, as each page's expansion checks the partial nodes
// again.
//
// The function keeps the order of items, keeping the first of each unique
// diagnostic.
//
// Takes diagnostics ([]*Diagnostic) which is the slice of diagnostics to check.
//
// Returns []*Diagnostic which contains only unique diagnostics.
func DeduplicateDiagnostics(diagnostics []*Diagnostic) []*Diagnostic {
	if len(diagnostics) == 0 {
		return diagnostics
	}

	type diagKey struct {
		SourcePath string
		Message    string
		Line       int
		Column     int
		Severity   Severity
	}

	seen := make(map[diagKey]struct{}, len(diagnostics))
	result := make([]*Diagnostic, 0, len(diagnostics))

	for _, d := range diagnostics {
		key := diagKey{
			SourcePath: d.SourcePath,
			Line:       d.Location.Line,
			Column:     d.Location.Column,
			Severity:   d.Severity,
			Message:    d.Message,
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, d)
	}

	return result
}

// FormatDiagnostics renders diagnostics with coloured source context.
// It adjusts the caret column position to account for the difference in
// character count between the raw source (with HTML entities) and the clean
// version shown to the user.
//
// When the diagnostics slice is empty, returns an empty string.
//
// Takes sourcePath (string) which specifies the file path to show in output.
// Takes sourceCode (string) which provides the raw source text.
// Takes diagnostics ([]*Diagnostic) which contains the diagnostics to format.
//
// Returns string which contains the formatted output with line numbers,
// source snippets, and coloured markers.
func FormatDiagnostics(sourcePath string, sourceCode string, diagnostics []*Diagnostic) string {
	if len(diagnostics) == 0 {
		return ""
	}

	lines := strings.Split(sourceCode, "\n")
	var builder strings.Builder

	builder.WriteString(errorColour.Sprintf("--> Found %d issue(s) in %s\n\n", len(diagnostics), sourcePath))

	maxGutterWidth := calculateMaxGutterWidth(diagnostics)

	for _, d := range diagnostics {
		formatSingleDiagnostic(&builder, d, lines, sourcePath, maxGutterWidth)
	}

	return builder.String()
}

// calculateMaxGutterWidth finds the width needed to display the largest line
// number from the given diagnostics.
//
// Takes diagnostics ([]*Diagnostic) which contains the diagnostics to check.
//
// Returns int which is the number of digits in the largest line number.
func calculateMaxGutterWidth(diagnostics []*Diagnostic) int {
	maxLineNum := 0
	for _, d := range diagnostics {
		if d.Location.Line > maxLineNum {
			maxLineNum = d.Location.Line
		}
	}
	return len(fmt.Sprintf("%d", maxLineNum))
}

// formatSingleDiagnostic writes a formatted diagnostic message to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes d (*Diagnostic) which contains the diagnostic to format.
// Takes lines ([]string) which holds the source file lines for context display.
// Takes sourcePath (string) which is the file path shown in the output.
// Takes maxGutterWidth (int) which sets the width of the line number column.
func formatSingleDiagnostic(builder *strings.Builder, d *Diagnostic, lines []string, sourcePath string, maxGutterWidth int) {
	lineIndex := d.Location.Line - 1
	if lineIndex < 0 || lineIndex >= len(lines) {
		_, _ = fmt.Fprintf(builder, " ● %s: %s (at invalid location L%d:%d)\n\n",
			d.Severity, d.Message, d.Location.Line, d.Location.Column)
		return
	}

	originalLine := lines[lineIndex]
	displayLine, adjustedColumn := calculateDisplayPosition(originalLine, d.Location.Column)
	highlightLen := calculateHighlightLength(d.Expression)
	primaryColour := colourForSeverity(d.Severity)

	lineNumString := fmt.Sprintf("%d", lineIndex+1)
	gutterPadding := strings.Repeat(spaceChar, maxGutterWidth-len(lineNumString))
	gutterWithNum := gutterColour.Sprintf("%s%s | ", gutterPadding, lineNumString)
	gutterEmpty := gutterColour.Sprintf("%s%s | ", gutterPadding, strings.Repeat(spaceChar, len(lineNumString)))

	builder.WriteString(primaryColour.Sprintf(" ● %s", d.Severity))
	if d.Code != "" {
		builder.WriteString(messageColour.Sprintf(" [%s]", d.Code))
	}
	builder.WriteString(messageColour.Sprintf(": %s\n", d.Message))
	_, _ = fmt.Fprintf(builder, "   %s %s:%d:%d\n", gutterColour.Sprint("-->"), pathColour.Sprint(sourcePath), d.Location.Line, d.Location.Column)
	builder.WriteString(gutterEmpty + "\n")
	builder.WriteString(gutterWithNum + displayLine + "\n")

	caretPadding := ""
	if adjustedColumn > 1 {
		caretPadding = strings.Repeat(spaceChar, adjustedColumn-1)
	}
	highlight := strings.Repeat("^", highlightLen)
	_, _ = fmt.Fprintf(builder, "%s%s%s\n\n", gutterEmpty, caretPadding, primaryColour.Sprint(highlight))
}

// calculateDisplayPosition converts an HTML-escaped line to its display form
// and adjusts the error column to match the new format. It changes HTML
// entities (such as &lt; becoming <) and expands tabs to spaces.
//
// Takes originalLine (string) which is the HTML-escaped source line.
// Takes errorColumn (int) which is the 1-based column in the escaped line.
//
// Returns displayLine (string) which is the line with HTML entities converted
// and tabs replaced with spaces.
// Returns adjustedColumn (int) which is the corresponding visual column
// position.
func calculateDisplayPosition(originalLine string, errorColumn int) (displayLine string, adjustedColumn int) {
	unescapedLine := gohtml.UnescapeString(originalLine)

	var originalPrefix string
	if len(originalLine) >= errorColumn {
		originalPrefix = originalLine[:errorColumn-1]
	} else {
		originalPrefix = originalLine
	}
	unescapedPrefix := gohtml.UnescapeString(originalPrefix)
	targetBytePosition := len(unescapedPrefix)

	var builder strings.Builder
	visualColumn := 1
	targetVisualColumn := 1
	bytePosition := 0

	for _, r := range unescapedLine {
		if bytePosition == targetBytePosition {
			targetVisualColumn = visualColumn
		}

		if r == '\t' {
			spacesToNextStop := tabWidth - ((visualColumn - 1) % tabWidth)
			builder.WriteString(strings.Repeat(spaceChar, spacesToNextStop))
			visualColumn += spacesToNextStop
		} else {
			_, _ = builder.WriteRune(r)
			visualColumn++
		}

		bytePosition += utf8.RuneLen(r)
	}

	if targetBytePosition >= len(unescapedLine) {
		targetVisualColumn = visualColumn
	}

	return builder.String(), targetVisualColumn
}

// calculateHighlightLength returns the display length of an HTML expression.
//
// Takes expression (string) which is the HTML-escaped text to measure.
//
// Returns int which is the length in characters, with a minimum of one.
func calculateHighlightLength(expression string) int {
	return cmp.Or(len(gohtml.UnescapeString(expression)), 1)
}

// colourForSeverity returns the display colour for a given severity level.
//
// Takes severity (Severity) which is the level to get a colour for.
//
// Returns colour.Style which is the colour used when showing the severity.
func colourForSeverity(severity Severity) colour.Style {
	switch severity {
	case Error:
		return errorColour
	case Warning:
		return warningColour
	case Info:
		return infoColour
	case Debug:
		return debugColour
	default:
		return messageColour
	}
}

func init() {
	colour.SetEnabled(enableDiagnosticColour)
}
