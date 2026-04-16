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

package lifecycle_adapters

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/internal/colour"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
)

const (
	// checkMark is the tick symbol used to indicate success in build summaries.
	checkMark = "✓"

	// arrowMarker is the arrow indicator used for artefact paths in failure output.
	arrowMarker = "-->"

	// timingArrow is the arrow used for timing lines in build summaries.
	timingArrow = "\u2192"

	// maxDiagnosticLineParts is the maximum number of line parts when splitting
	// diagnostic output for the error snippet extractor.
	maxDiagnosticLineParts = 4

	// minDiagnosticLineParts is the minimum number of parts expected when parsing
	// build error snippets with source location and caret information.
	minDiagnosticLineParts = 3

	// summaryLineFormat is the format string for individual summary lines.
	summaryLineFormat = "   %s %d %s\n"

	// summaryDurationRounding is the rounding granularity for durations in
	// build summaries. 10ms gives useful precision without false accuracy.
	summaryDurationRounding = 10 * time.Millisecond
)

var (
	// summaryHeaderColour is the ANSI colour used for build summary section headings.
	summaryHeaderColour = colour.New(colour.Bold)

	// summarySuccessColour is the ANSI colour used for success indicators in build summaries.
	summarySuccessColour = colour.New(colour.FgGreen, colour.Bold)

	// summaryFailureColour is the ANSI colour used for failure indicators in build summaries.
	summaryFailureColour = colour.New(colour.FgRed, colour.Bold)

	// summaryPathColour is the ANSI colour used for file paths in build failure output.
	summaryPathColour = colour.New(colour.FgCyan)

	// summaryDimColour is the ANSI colour used for secondary text in build summaries.
	summaryDimColour = colour.New(colour.FgHiBlack)

	// summaryGutterColour is the ANSI colour used for line number
	// gutters in build error snippets.
	summaryGutterColour = colour.New(colour.FgBlue, colour.Bold)

	// locationPattern matches esbuild-style location suffixes: (line N, col M).
	locationPattern = regexp.MustCompile(`\(line (\d+), col (\d+)\)\s*$`)
)

// GeneratorResult holds the counts produced by the code generation phase.
// It is used by FormatGeneratorSummary to render a coloured summary.
type GeneratorResult struct {
	// Pages is the number of page components generated.
	Pages int

	// Partials is the number of partial components generated.
	Partials int

	// Emails is the number of email templates generated.
	Emails int

	// PDFs is the number of PDF templates generated.
	PDFs int

	// SQLQueries is the number of SQL queries that had Go code generated.
	SQLQueries int

	// Artefacts is the total number of generated Go source files.
	Artefacts int

	// Duration is the wall-clock time spent on code generation.
	Duration time.Duration
}

// buildErrorSnippet holds extracted diagnostic parts from an error string that
// contains esbuild-style location info.
type buildErrorSnippet struct {
	// coreMessage is the innermost error message without wrapping.
	coreMessage string

	// sourceText is the source code line where the error occurred.
	sourceText string

	// caretLine marks the error column with caret characters.
	caretLine string

	// line is the one-based line number of the error location.
	line int

	// column is the zero-based column number of the error location.
	column int
}

// FormatBuildSummary renders a coloured build summary string following the
// AST diagnostic formatting conventions. The result is intended for writing
// to stderr.
//
// Takes result (*lifecycle_domain.BuildResult) which holds the build outcome.
//
// Returns string which contains the formatted summary with ANSI colour codes.
func FormatBuildSummary(result *lifecycle_domain.BuildResult) string {
	var builder strings.Builder

	builder.WriteString("\n")
	builder.WriteString(summaryHeaderColour.Sprint(" ● Build Summary"))
	builder.WriteString("\n")

	_, _ = fmt.Fprintf(&builder, "   %s %d tasks dispatched, completed in %s\n",
		summaryDimColour.Sprint(arrowMarker),
		result.TotalDispatched,
		result.Duration.Round(summaryDurationRounding).String())

	builder.WriteString("\n")

	_, _ = fmt.Fprintf(&builder, "   %s %d completed\n",
		summarySuccessColour.Sprint(checkMark),
		result.TotalCompleted)

	if result.TotalFailed > 0 {
		failDetail := fmt.Sprintf("%d failed", result.TotalFailed)
		if result.TotalFatalFailed > 0 {
			failDetail += fmt.Sprintf(" (%d fatal)", result.TotalFatalFailed)
		}
		_, _ = fmt.Fprintf(&builder, "   %s %s\n",
			summaryFailureColour.Sprint("✗"),
			summaryFailureColour.Sprint(failDetail))
	}

	if result.TotalRetried > 0 {
		_, _ = fmt.Fprintf(&builder, "   %s %d retried\n",
			summaryDimColour.Sprint("↻"),
			result.TotalRetried)
	}

	if result.TimedOut {
		_, _ = fmt.Fprintf(&builder, "\n   %s\n",
			summaryFailureColour.Sprint("Build timed out before all tasks could complete."))
	}

	for _, f := range result.Failures {
		builder.WriteString("\n")
		formatBuildFailure(&builder, &f)
	}

	builder.WriteString("\n")
	return builder.String()
}

// FormatGeneratorSummary renders a coloured summary of the code generation
// phase, showing how many pages, partials, and emails were produced. The
// result is intended for writing to stderr, matching the style of
// FormatBuildSummary.
//
// Takes result (*GeneratorResult) which holds the generation outcome.
//
// Returns string which contains the formatted summary with ANSI colour codes.
func FormatGeneratorSummary(result *GeneratorResult) string {
	var builder strings.Builder

	builder.WriteString("\n")
	builder.WriteString(summaryHeaderColour.Sprint(" ● Generator Summary"))
	builder.WriteString("\n")

	_, _ = fmt.Fprintf(&builder, "   %s %d artefacts generated in %s\n",
		summaryDimColour.Sprint(arrowMarker),
		result.Artefacts,
		result.Duration.Round(summaryDurationRounding).String())

	builder.WriteString("\n")

	_, _ = fmt.Fprintf(&builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		result.Pages,
		pluralise("page", result.Pages))

	_, _ = fmt.Fprintf(&builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		result.Partials,
		pluralise("partial", result.Partials))

	_, _ = fmt.Fprintf(&builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		result.Emails,
		pluralise("email", result.Emails))

	_, _ = fmt.Fprintf(&builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		result.PDFs,
		pluralise("pdf", result.PDFs))

	_, _ = fmt.Fprintf(&builder, "   %s %d sql %s\n",
		summarySuccessColour.Sprint(checkMark),
		result.SQLQueries,
		pluralise("query", result.SQLQueries))

	builder.WriteString("\n")
	return builder.String()
}

// FormatCombinedSummary renders a single summary block that merges the
// generator and build results. This is used when code emission and asset
// building run in parallel, so they can be reported together after both
// complete.
//
// Takes gen (*GeneratorResult) which holds the code generation outcome.
// Takes build (*lifecycle_domain.BuildResult) which holds the asset build
// outcome.
// Takes annotationDuration (time.Duration) which is the wall-clock time for
// inspection and annotation.
// Takes totalDuration (time.Duration) which is the wall-clock time for the
// entire build (annotation + parallel emission and assets).
//
// Returns string which contains the formatted summary with ANSI colour codes.
func FormatCombinedSummary(gen *GeneratorResult, build *lifecycle_domain.BuildResult, annotationDuration, totalDuration time.Duration) string {
	var builder strings.Builder

	builder.WriteString("\n")
	builder.WriteString(summaryHeaderColour.Sprint(" ● Build Summary"))
	builder.WriteString("\n")

	_, _ = fmt.Fprintf(&builder, "   %s inspection and annotation in %s\n",
		summaryDimColour.Sprint(timingArrow),
		annotationDuration.Round(summaryDurationRounding).String())
	_, _ = fmt.Fprintf(&builder, "   %s %d artefacts generated in %s\n",
		summaryDimColour.Sprint(timingArrow),
		gen.Artefacts,
		gen.Duration.Round(summaryDurationRounding).String())
	_, _ = fmt.Fprintf(&builder, "   %s %d asset tasks dispatched, completed in %s\n",
		summaryDimColour.Sprint(timingArrow),
		build.TotalDispatched,
		build.Duration.Round(summaryDurationRounding).String())
	_, _ = fmt.Fprintf(&builder, "   %s total: %s\n",
		summaryDimColour.Sprint(timingArrow),
		totalDuration.Round(summaryDurationRounding).String())
	builder.WriteString("\n")

	writeGeneratorSummaryLines(&builder, gen)
	writeBuildResultLines(&builder, build)

	builder.WriteString("\n")
	return builder.String()
}

// writeGeneratorSummaryLines writes the per-type generator counts (pages,
// partials, emails, PDFs, SQL queries) to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes gen (*GeneratorResult) which holds the generation counts to render.
func writeGeneratorSummaryLines(builder *strings.Builder, gen *GeneratorResult) {
	_, _ = fmt.Fprintf(builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		gen.Pages,
		pluralise("page", gen.Pages))

	_, _ = fmt.Fprintf(builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		gen.Partials,
		pluralise("partial", gen.Partials))

	_, _ = fmt.Fprintf(builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		gen.Emails,
		pluralise("email", gen.Emails))

	_, _ = fmt.Fprintf(builder, summaryLineFormat,
		summarySuccessColour.Sprint(checkMark),
		gen.PDFs,
		pluralise("pdf", gen.PDFs))

	_, _ = fmt.Fprintf(builder, "   %s %d sql %s\n",
		summarySuccessColour.Sprint(checkMark),
		gen.SQLQueries,
		pluralise("query", gen.SQLQueries))
}

// writeBuildResultLines writes the asset build outcome lines (completed,
// failed, retried, timed out, and individual failure details) to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes build (*lifecycle_domain.BuildResult) which holds the build outcome
// to render.
func writeBuildResultLines(builder *strings.Builder, build *lifecycle_domain.BuildResult) {
	_, _ = fmt.Fprintf(builder, "   %s %d asset %s completed\n",
		summarySuccessColour.Sprint(checkMark),
		build.TotalCompleted,
		pluralise("task", int(build.TotalCompleted)))

	if build.TotalFailed > 0 {
		failDetail := fmt.Sprintf("%d failed", build.TotalFailed)
		if build.TotalFatalFailed > 0 {
			failDetail += fmt.Sprintf(" (%d fatal)", build.TotalFatalFailed)
		}
		_, _ = fmt.Fprintf(builder, "   %s %s\n",
			summaryFailureColour.Sprint("✗"),
			summaryFailureColour.Sprint(failDetail))
	}

	if build.TotalRetried > 0 {
		_, _ = fmt.Fprintf(builder, "   %s %d retried\n",
			summaryDimColour.Sprint("↻"),
			build.TotalRetried)
	}

	if build.TimedOut {
		_, _ = fmt.Fprintf(builder, "\n   %s\n",
			summaryFailureColour.Sprint("Build timed out before all tasks could complete."))
	}

	for _, f := range build.Failures {
		builder.WriteString("\n")
		formatBuildFailure(builder, &f)
	}
}

// pluralise returns the word with an "s" suffix when count is not exactly one.
//
// Takes word (string) which is the singular form of the word.
// Takes count (int) which determines whether to pluralise.
//
// Returns string which is the word with an "s" appended when count is not one.
func pluralise(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

// formatBuildFailure renders a single failure entry in diagnostic style,
// matching the AST diagnostic format when the error contains source location
// information.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes f (*lifecycle_domain.BuildFailure) which holds the failure details.
func formatBuildFailure(builder *strings.Builder, f *lifecycle_domain.BuildFailure) {
	cleaned := cleanBuildError(f.Error)
	snippet := extractBuildErrorSnippet(cleaned)

	severity := "error"
	if f.IsFatal {
		severity = "fatal"
	}

	if snippet != nil {
		builder.WriteString(summaryFailureColour.Sprintf(" ● %s", severity))
		builder.WriteString(summaryHeaderColour.Sprintf(": %s\n", snippet.coreMessage))

		if f.ArtefactID != "" {
			_, _ = fmt.Fprintf(builder, "   %s %s:%d:%d\n",
				summaryGutterColour.Sprint("-->"),
				summaryPathColour.Sprint(f.ArtefactID),
				snippet.line, snippet.column)
		}

		attemptInfo := fmt.Sprintf("%s, attempt %d", f.Executor, f.Attempt)
		if f.IsFatal {
			attemptInfo += " (fatal - not retried)"
		}
		_, _ = fmt.Fprintf(builder, "   %s\n", summaryDimColour.Sprint(attemptInfo))

		formatSourceSnippet(builder, snippet)
		return
	}

	builder.WriteString(summaryFailureColour.Sprintf(" ● %s", severity))
	builder.WriteString(summaryHeaderColour.Sprintf(": %s failed\n", f.Executor))

	if f.ArtefactID != "" {
		_, _ = fmt.Fprintf(builder, "   %s %s\n",
			summaryDimColour.Sprint(arrowMarker),
			summaryPathColour.Sprint(f.ArtefactID))
	}

	attemptInfo := fmt.Sprintf("attempt %d", f.Attempt)
	if f.IsFatal {
		attemptInfo += " (fatal - not retried)"
	}
	_, _ = fmt.Fprintf(builder, "   %s\n", summaryDimColour.Sprint(attemptInfo))

	if cleaned != "" {
		for line := range strings.SplitSeq(cleaned, "\n") {
			_, _ = fmt.Fprintf(builder, "   %s\n", line)
		}
	}
}

// extractBuildErrorSnippet parses a cleaned error string looking for an
// esbuild-style diagnostic with (line N, col M) and source snippet. Returns
// nil when the error does not contain diagnostic information.
//
// Takes cleaned (string) which is the error string to parse for diagnostic
// information.
//
// Returns *buildErrorSnippet which holds the extracted diagnostic parts, or
// nil when no location information is found.
func extractBuildErrorSnippet(cleaned string) *buildErrorSnippet {
	lines := strings.SplitN(cleaned, "\n", maxDiagnosticLineParts)
	firstLine := lines[0]

	match := locationPattern.FindStringSubmatchIndex(firstLine)
	if match == nil {
		return nil
	}

	line, _ := strconv.Atoi(firstLine[match[2]:match[3]])
	column, _ := strconv.Atoi(firstLine[match[4]:match[5]])

	coreMessage := extractCoreMessage(firstLine[:match[0]])

	snippet := &buildErrorSnippet{
		coreMessage: coreMessage,
		line:        line,
		column:      column,
	}

	if len(lines) >= 2 {
		snippet.sourceText = extractGutterContent(lines[1])
	}
	if len(lines) >= minDiagnosticLineParts {
		snippet.caretLine = extractGutterContent(lines[2])
	}

	return snippet
}

// extractCoreMessage finds the innermost error message from a wrapping chain.
// Error chains use ": " as a separator; the core message is the last segment
// that starts with an uppercase letter or a quote.
//
// Takes chain (string) which is the full error message chain to unwrap.
//
// Returns string which is the innermost meaningful error message.
func extractCoreMessage(chain string) string {
	best := chain
	for i := len(chain) - 2; i >= 1; i-- {
		if chain[i-1] == ':' && chain[i] == ' ' && i+1 < len(chain) {
			character := chain[i+1]
			if (character >= 'A' && character <= 'Z') || character == '"' || character == '\'' {
				best = chain[i+1:]
				break
			}
		}
	}
	return strings.TrimSpace(best)
}

// extractGutterContent strips the existing "    N | " or "      | " prefix
// produced by formatParserError, returning just the content after the pipe.
//
// Takes line (string) which is the diagnostic line to strip the gutter from.
//
// Returns string which is the content after the pipe separator, or the trimmed
// line if no pipe is found.
func extractGutterContent(line string) string {
	index := strings.Index(line, "| ")
	if index >= 0 {
		return line[index+2:]
	}
	index = strings.Index(line, "|")
	if index >= 0 && index == len(strings.TrimRight(line, " \t"))-1 {
		return ""
	}
	return strings.TrimSpace(line)
}

// formatSourceSnippet writes a diagnostic-style source snippet with coloured
// gutters.
//
// Takes builder (*strings.Builder) which receives the formatted snippet output.
// Takes s (*buildErrorSnippet) which holds the source text and caret position.
func formatSourceSnippet(builder *strings.Builder, s *buildErrorSnippet) {
	lineNumString := strconv.Itoa(s.line)
	gutterWidth := len(lineNumString)
	padding := strings.Repeat(" ", gutterWidth)

	gutterEmpty := summaryGutterColour.Sprintf("   %s | ", padding)
	gutterWithNum := summaryGutterColour.Sprintf("   %s | ", lineNumString)

	builder.WriteString(gutterEmpty + "\n")
	builder.WriteString(gutterWithNum + s.sourceText + "\n")

	if s.caretLine != "" {
		caretIndex := strings.Index(s.caretLine, "^")
		if caretIndex >= 0 {
			caretPadding := strings.Repeat(" ", caretIndex)
			carets := s.caretLine[caretIndex:]
			_, _ = fmt.Fprintf(builder, "%s%s%s\n", gutterEmpty, caretPadding, summaryFailureColour.Sprint(carets))
		} else {
			builder.WriteString(gutterEmpty + s.caretLine + "\n")
		}
	}
}

// cleanBuildError strips implementation-detail sentinel suffixes from the
// error string so that the build summary shows only the meaningful message.
//
// Takes raw (string) which is the original error string to clean.
//
// Returns string which is the error with sentinel suffixes removed.
func cleanBuildError(raw string) string {
	s := raw
	s = strings.TrimSuffix(s, ": orchestrator: fatal error")
	s = strings.TrimSuffix(s, ": capability: fatal error")
	return s
}
