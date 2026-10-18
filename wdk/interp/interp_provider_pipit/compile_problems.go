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

package interp_provider_pipit

import (
	"errors"
	"fmt"
	"strings"

	"pipit.sh/pipit"
)

const (
	// problemIndent indents a problem's message and hint under its numbered label.
	problemIndent = "     "

	// hintPrefix starts the hint line pipit or piko appends to a problem.
	hintPrefix = "hint: "

	// notePrefix starts the note piko adds for a piko project.
	notePrefix = "note: "
)

// problemClass associates a pipit compile sentinel with its label, fallback hint, and
// Piko-specific note.
type problemClass struct {
	// sentinel is the pipit error a problem of this class wraps.
	sentinel error

	// label names the class in the startup error.
	label string

	// hint supplies fallback advice when pipit provides none.
	hint string

	// note adds Piko-specific advice after the compiler message and hint.
	note string
}

// compileProblemsError lists all compile problems with hints and unwraps to the original
// pipit error.
type compileProblemsError struct {
	// cause is the compile error pipit returned.
	cause error

	// modulePath is the module whose program failed to compile.
	modulePath string

	// problems lists the separate problems pipit reported, in its order.
	problems []error
}

var (
	// problemClasses lists compile-error matches from most to least specific.
	problemClasses = []problemClass{
		{
			sentinel: pipit.ErrEmbedUnsupported,
			label:    "//go:embed in a package dev-i compiles from source",
			hint:     "",
			note: "dev-i runs from the project root, so a relative path in the //go:build interp " +
				"file, such as os.DirFS(\"content\"), is read from there",
		},
		{
			sentinel: pipit.ErrRegisterOverflow,
			label:    "function too large for the interpreter",
			hint: "if the function is a page's or partial's generated BuildAST, move part of its " +
				"template into a partial; otherwise split the function, or keep long-lived values " +
				"in a struct or slice",
			note: "",
		},
		{
			sentinel: pipit.ErrPackageNotInRegistry,
			label:    "package without a symbol table",
			hint:     "",
			note: "in a piko project the manifest is piko-symbols.yaml (piko extract discover finds " +
				"every missing package), and the generated table is registered with ssr.WithSymbols",
		},
		{
			sentinel: pipit.ErrTypeCheck,
			label:    "type error",
			hint: "dev-i type-checks the files the interp build tag selects; if go build accepts the " +
				"package, check that its //go:build interp files declare everything their " +
				"//go:build !interp counterparts do, and regenerate stale symbol tables with piko " +
				"extract generate",
			note: "",
		},
		{
			sentinel: pipit.ErrParse,
			label:    "syntax error",
			hint:     "",
			note:     "",
		},
		{
			sentinel: pipit.ErrCyclicImport,
			label:    "import cycle",
			hint:     "",
			note:     "",
		},
	}

	// otherProblemClass is the class of a problem no sentinel identifies.
	otherProblemClass = problemClass{sentinel: nil, label: "compile error", hint: "", note: ""}
)

// newCompileProblemsError splits a compile error from pipit into its separate problems.
//
// Takes modulePath (string) which is the module whose program failed to compile.
// Takes cause (error) which is pipit's compile error; it must not be nil.
//
// Returns *compileProblemsError which lists at least one problem.
func newCompileProblemsError(modulePath string, cause error) *compileProblemsError {
	return &compileProblemsError{
		cause:      cause,
		modulePath: modulePath,
		problems:   pipit.CompileProblems(cause),
	}
}

// Error renders a header naming the module and the problem count, then each problem
// numbered, labelled with its class, and followed by a hint and a note when they apply.
//
// Returns string which is the combined, multi-line message.
func (e *compileProblemsError) Error() string {
	var builder strings.Builder
	noun := "problems"
	if len(e.problems) == 1 {
		noun = "problem"
	}
	_, _ = fmt.Fprintf(&builder, "dev-i could not compile module %q (%d %s):", e.modulePath, len(e.problems), noun)
	for index, problem := range e.problems {
		class := classifyProblem(problem)
		message := problem.Error()
		_, _ = fmt.Fprintf(&builder, "\n\n  %d. %s\n", index+1, class.label)
		writeIndented(&builder, message)
		if class.hint != "" && !carriesHint(message) {
			builder.WriteString("\n\n")
			writeIndented(&builder, hintPrefix+class.hint)
		}
		if class.note != "" {
			builder.WriteString("\n\n")
			writeIndented(&builder, notePrefix+class.note)
		}
	}
	return builder.String()
}

// Unwrap returns pipit's compile error.
//
// Returns error which is the error the problems were split from.
func (e *compileProblemsError) Unwrap() error {
	return e.cause
}

// classifyProblem finds the class of one problem by the pipit sentinel it wraps.
//
// Takes problem (error) which is one problem of a compile error.
//
// Returns problemClass which is the first matching class, or otherProblemClass.
func classifyProblem(problem error) problemClass {
	for _, class := range problemClasses {
		if errors.Is(problem, class.sentinel) {
			return class
		}
	}
	return otherProblemClass
}

// carriesHint reports whether a problem's message already ends with a hint, as pipit
// appends to the problems it can advise on, so piko does not add a second one.
//
// Takes message (string) which is the problem's message.
//
// Returns bool which is true when a line of the message starts with "hint: ".
func carriesHint(message string) bool {
	return strings.HasPrefix(message, hintPrefix) || strings.Contains(message, "\n"+hintPrefix)
}

// writeIndented writes text with every non-empty line indented by problemIndent, so a
// problem's lines sit under its numbered label.
//
// Takes builder (*strings.Builder) which receives the text.
// Takes text (string) which may span several lines.
func writeIndented(builder *strings.Builder, text string) {
	for index, line := range strings.Split(text, "\n") {
		if index > 0 {
			builder.WriteByte('\n')
		}
		if line != "" {
			builder.WriteString(problemIndent)
			builder.WriteString(line)
		}
	}
}
