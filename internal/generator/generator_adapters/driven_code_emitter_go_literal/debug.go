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

package driven_code_emitter_go_literal

import (
	"fmt"
	goast "go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// computeRelativePath converts an absolute source path to a clean relative
// path relative to the project BaseDir, with no leading "./" prefix.
//
// Takes absPath (string) which is the absolute file path to convert.
//
// Returns string which is the relative path, or the original absolute path if
// conversion fails.
func (em *emitter) computeRelativePath(absPath string) string {
	relPath, err := filepath.Rel(em.config.BaseDir, absPath)
	if err != nil {
		return absPath
	}

	relPath = strings.TrimPrefix(relPath, "./")

	relPath = filepath.ToSlash(relPath)

	return relPath
}

// sourceMappingStmt creates a line directive statement for the
// given template node.
//
// When EnableDwarfLineDirectives is true, it emits "//line
// file:line" which the Go compiler embeds into DWARF debug info,
// enabling debuggers like Delve to map breakpoints and stack
// traces back to .pk files. When false (default), it emits
// "// line file:line" (with a space) which is a plain comment
// visible to humans but ignored by the compiler.
//
// Takes node (*ast_domain.TemplateNode) which provides the source
// location.
//
// Returns goast.Stmt which is an empty statement when the node
// has no source mapping data, or an expression statement with a
// line directive.
func (em *emitter) sourceMappingStmt(node *ast_domain.TemplateNode) goast.Stmt {
	if node == nil || node.Location.IsSynthetic() || node.GoAnnotations == nil || node.GoAnnotations.OriginalSourcePath == nil {
		return &goast.EmptyStmt{Implicit: true}
	}

	absPath := *node.GoAnnotations.OriginalSourcePath
	relPath := em.computeRelativePath(absPath)

	lineDirective := em.formatLineDirective(relPath, node.Location.Line, 0)

	return &goast.ExprStmt{
		X: &goast.BasicLit{
			Kind:  token.COMMENT,
			Value: lineDirective,
		},
	}
}

// directiveMappingStmt creates a line directive with column
// precision for a template directive (p-for, p-if, p-else-if).
//
// The column allows debuggers to distinguish between multiple
// directives on the same element line. Respects the
// EnableDwarfLineDirectives config; see formatLineDirective.
//
// Takes node (*ast_domain.TemplateNode) which provides the source file path.
// Takes dir (*ast_domain.Directive) which provides the directive's name
// location with line and column.
//
// Returns goast.Stmt which is an empty statement when the node or directive has
// no source mapping data, or an expression statement with a line directive.
func (em *emitter) directiveMappingStmt(node *ast_domain.TemplateNode, dir *ast_domain.Directive) goast.Stmt {
	if node == nil || dir == nil || node.GoAnnotations == nil || node.GoAnnotations.OriginalSourcePath == nil {
		return &goast.EmptyStmt{Implicit: true}
	}

	loc := dir.NameLocation
	if loc.IsSynthetic() {
		return &goast.EmptyStmt{Implicit: true}
	}

	absPath := *node.GoAnnotations.OriginalSourcePath
	relPath := em.computeRelativePath(absPath)

	lineDirective := em.formatLineDirective(relPath, loc.Line, loc.Column)

	return &goast.ExprStmt{
		X: &goast.BasicLit{
			Kind:  token.COMMENT,
			Value: lineDirective,
		},
	}
}

// formatLineDirective builds the line directive string.
//
// When EnableDwarfLineDirectives is true, it emits
// "//line file:line[:col]" (valid DWARF). When false, it emits
// "// line file:line[:col]" (plain comment). Column is included
// only when col > 0.
//
// Takes relPath (string) which is the relative source file path.
// Takes line (int) which is the source line number.
// Takes col (int) which is the source column number, or 0 to
// omit column.
//
// Returns string which is the formatted line directive.
func (em *emitter) formatLineDirective(relPath string, line int, col int) string {
	prefix := "// line"
	if em.config.EnableDwarfLineDirectives {
		prefix = "//line"
	}

	if col > 0 {
		return fmt.Sprintf("%s %s:%d:%d", prefix, relPath, line, col)
	}

	return fmt.Sprintf("%s %s:%d", prefix, relPath, line)
}

// sourceMappingCommentGroup creates a comment with source location data for
// attaching to declarations.
//
// Takes node (*ast_domain.TemplateNode) which provides the source location.
//
// Returns *goast.CommentGroup which holds the source mapping comment, or nil
// if the node has no valid location.
func (em *emitter) sourceMappingCommentGroup(node *ast_domain.TemplateNode) *goast.CommentGroup {
	if node == nil || node.Location.IsSynthetic() || node.GoAnnotations == nil || node.GoAnnotations.OriginalSourcePath == nil {
		return nil
	}

	absPath := *node.GoAnnotations.OriginalSourcePath
	relPath := em.computeRelativePath(absPath)

	commentText := fmt.Sprintf(" SOURCE: %s, LINE: %d ", relPath, node.Location.Line)

	return &goast.CommentGroup{
		List: []*goast.Comment{
			{Text: "//" + commentText},
		},
	}
}
