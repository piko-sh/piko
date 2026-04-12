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

package interp_domain

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
)

// braceTracker walks lines of source and tracks the cumulative brace depth at the START
// of the next line. Used by preprocessShortVar to decide whether the current line is at
// package scope.
type braceTracker struct {
	// depth is the cumulative brace nesting depth at the start of the next line.
	depth int
}

// atTopLevel reports whether the next line starts at brace depth zero, i.e. outside any
// function/struct/interface body.
//
// Returns bool which is true when the tracker is at package scope.
func (b *braceTracker) atTopLevel() bool {
	return b.depth == 0
}

// advance consumes one source line, updating brace depth based on `{` and `}` tokens.
// Strings and comments are skipped via the Go scanner so braces inside string literals do
// not affect depth.
//
// Takes line (string) which is the source line to consume.
func (b *braceTracker) advance(line string) {
	var sc scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("line.go", fset.Base(), len(line))
	sc.Init(file, []byte(line), nil, scanner.ScanComments)
	for {
		_, tok, _ := sc.Scan()
		if tok == token.EOF {
			return
		}
		switch tok {
		case token.LBRACE:
			b.depth++
		case token.RBRACE:
			if b.depth > 0 {
				b.depth--
			}
		default:
		}
	}
}

// preprocessShortVar rewrites a top-level short-var as a session-scope var.
//
// Only submissions whose first non-blank top-level statement is a single-identifier ":="
// form are touched; statements inside braces, ":=" in if/for/switch headers, multi-value
// ":=", and blank-ident ":=" are left for downstream paths. A line such as "a := 1"
// becomes "var a = 1"; an indented or control-flow header form passes through unchanged;
// a multi-value or blank-ident form returns an error directing the user at the equivalent
// "var" form.
//
// Takes code (string) which is the verbatim user submission.
//
// Returns the rewritten source, or the input unchanged when no rewrite applied.
// Returns an error when an unsupported short-var shape was found at top level.
func preprocessShortVar(code string) (string, error) {
	lines := strings.Split(code, "\n")
	tracker := newBraceTracker()

	for index, line := range lines {
		if tracker.atTopLevel() {
			rewritten, rewrote, err := tryRewriteShortVarLine(line)
			if err != nil {
				return "", err
			}
			if rewrote {
				lines[index] = rewritten
				tracker.advance(rewritten)
				continue
			}
		}
		tracker.advance(line)
	}
	return strings.Join(lines, "\n"), nil
}

// tryRewriteShortVarLine rewrites a column-zero "name := expr" form to "var name = expr".
//
// Lines that begin with a control-flow keyword or any leading whitespace are passed
// through unchanged; multi-value and blank-ident shapes return an error.
//
// Takes line (string) which is the original source line.
//
// Returns the rewritten line, true when a rewrite happened (false otherwise), and an
// error when an unsupported shape was detected.
func tryRewriteShortVarLine(line string) (string, bool, error) {
	stripped := strings.TrimRight(line, " \t\r")
	if stripped == "" {
		return line, false, nil
	}
	if stripped[0] == ' ' || stripped[0] == '\t' {
		return line, false, nil
	}
	if startsWithControlKeyword(stripped) {
		return line, false, nil
	}
	if !containsTopLevelDefine(stripped) {
		return line, false, nil
	}

	statement, ok := parseSingleStatement(stripped)
	if !ok {
		return line, false, nil
	}
	assign, ok := statement.(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE {
		return line, false, nil
	}

	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return "", false, fmt.Errorf(
			"session: short-variable declaration with multiple names is not supported; use `var %s = %s` instead",
			renderExprList(assign.Lhs),
			renderExprList(assign.Rhs),
		)
	}
	ident, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return line, false, nil
	}
	if ident.Name == "_" {
		return "", false, errors.New(
			"session: blank identifier on the left of `:=` has no observable effect; use the right-hand expression directly",
		)
	}

	rhs := strings.TrimSpace(stripped[strings.Index(stripped, ":=")+2:])
	return "var " + ident.Name + " = " + rhs, true, nil
}

// startsWithControlKeyword reports whether a trimmed line begins with a control-flow
// keyword.
//
// The check covers keywords that introduce a non-statement context where ":=" is valid
// but must not be lifted to package scope, such as "if x := f();" or "for i := 0; ...". A
// trailing space or tab is required so "if(" and similar are not misclassified.
//
// Takes line (string) which is the candidate top-level line.
//
// Returns true when the line begins with a recognised control-flow keyword.
func startsWithControlKeyword(line string) bool {
	keywords := []string{"if ", "if\t", "for ", "for\t", "switch ", "switch\t", "select ", "select\t", "go ", "go\t", "defer ", "defer\t", "return ", "return\t"}
	for _, prefix := range keywords {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// containsTopLevelDefine reports whether the line contains a `:=` token outside of any
// nested braces, parens, or brackets. Strings and comments are tokenised so `s := "{:=
// }"` is correctly seen as a top-level define.
//
// Takes line (string) which is the source line.
//
// Returns bool which is true when a top-level `:=` was found.
func containsTopLevelDefine(line string) bool {
	var sc scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("line.go", fset.Base(), len(line))
	sc.Init(file, []byte(line), nil, scanner.ScanComments)
	depth := 0
	for {
		_, tok, _ := sc.Scan()
		if tok == token.EOF {
			return false
		}
		switch tok {
		case token.LBRACE, token.LPAREN, token.LBRACK:
			depth++
		case token.RBRACE, token.RPAREN, token.RBRACK:
			if depth > 0 {
				depth--
			}
		case token.DEFINE:
			if depth == 0 {
				return true
			}
		default:
		}
	}
}

// parseSingleStatement parses source as a single Go statement by wrapping it in a
// synthetic function body.
//
// Takes source (string) which is the candidate statement.
//
// Returns the parsed statement on success (nil otherwise) and true on success or false on
// any parse error (the caller treats false as "leave alone").
func parseSingleStatement(source string) (ast.Stmt, bool) {
	wrapped := "package main\nfunc _shortvarprobe_() {\n" + source + "\n}"
	file, err := parser.ParseFile(token.NewFileSet(), "shortvar.go", wrapped, 0)
	if err != nil {
		return nil, false
	}
	if len(file.Decls) == 0 {
		return nil, false
	}
	funcDecl, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || funcDecl.Body == nil || len(funcDecl.Body.List) != 1 {
		return nil, false
	}
	return funcDecl.Body.List[0], true
}

// renderExprList stringifies a list of expressions as comma-separated source text
// suitable for human-facing error messages. Uses the printer-free path so we do not
// depend on go/printer just for diags.
//
// Takes expressions ([]ast.Expr) which is the list to render.
//
// Returns string which is the comma-separated text.
func renderExprList(expressions []ast.Expr) string {
	parts := make([]string, 0, len(expressions))
	for _, expression := range expressions {
		parts = append(parts, exprAsString(expression))
	}
	return strings.Join(parts, ", ")
}

// exprAsString renders a single expression as source text without requiring go/printer.
// Only handles the shapes that appear on the LHS / RHS of a short-var rewrite; falls back
// to "<expr>" for anything more complex.
//
// Takes expression (ast.Expr) which is the expression to render.
//
// Returns string which is the rendered source.
func exprAsString(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.BasicLit:
		return typed.Value
	default:
		return "<expr>"
	}
}

// newBraceTracker returns a tracker positioned at depth zero, i.e. the start of the
// source.
//
// Returns *braceTracker ready to consume lines.
func newBraceTracker() *braceTracker {
	return &braceTracker{}
}
