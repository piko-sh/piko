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
	"go/ast"
	"go/token"
)

// activeDeclaration tracks a variable binding introduced by a
// declaring statement whose register must survive across subsequent
// statements.
type activeDeclaration struct {
	// name is the variable name introduced by the declaration.
	name string

	// location is the register allocation for this variable.
	location varLocation
}

// trackOrRestoreDeclarations updates the active declaration list
// when the statement is a declaring statement, or restores the
// register allocation watermark otherwise.
//
// Takes statement (ast.Stmt) which is the statement to inspect.
// Takes watermark ([NumRegisterKinds]uint32) which is the
// register allocation watermark to restore for non-declaring
// statements.
// Takes active ([]activeDeclaration) which is the current list
// of tracked declarations.
//
// Returns the updated active declaration list.
func (c *compiler) trackOrRestoreDeclarations(
	statement ast.Stmt,
	watermark [NumRegisterKinds]uint32,
	active []activeDeclaration,
) []activeDeclaration {
	if !isDeclaringStatement(statement) {
		c.scopes.restoreWatermark(watermark)
		return active
	}
	for _, name := range extractDeclaredNames(statement) {
		active = c.trackDeclaredName(name, active)
	}
	return active
}

// trackDeclaredName appends a new activeDeclaration for name if
// it is not already tracked and exists in the current scope.
//
// Takes name (string) which is the variable name to track.
// Takes active ([]activeDeclaration) which is the current list
// of tracked declarations.
//
// Returns the updated active declaration list.
func (c *compiler) trackDeclaredName(name string, active []activeDeclaration) []activeDeclaration {
	declarationLocation, ok := c.scopes.lookupVar(name)
	if !ok {
		return active
	}
	for _, existing := range active {
		if existing.name == name {
			return active
		}
	}
	return append(active, activeDeclaration{name: name, location: declarationLocation})
}

// recycleDeadDeclarations removes declarations whose last use
// index has been reached and recycles their registers.
//
// Takes active ([]activeDeclaration) which is the current list
// of tracked declarations.
// Takes lastUseIndices (map[string]int) which maps variable
// names to the index of their last use.
// Takes currentIndex (int) which is the index of the current
// statement being processed.
//
// Returns the filtered active declaration list with dead
// entries removed.
func (c *compiler) recycleDeadDeclarations(
	active []activeDeclaration,
	lastUseIndices map[string]int,
	currentIndex int,
) []activeDeclaration {
	if lastUseIndices == nil {
		return active
	}
	remaining := active[:0]
	for _, declaration := range active {
		if c.shouldRetainDeclaration(declaration, lastUseIndices, currentIndex) {
			remaining = append(remaining, declaration)
		}
	}
	return remaining
}

// shouldRetainDeclaration reports whether the declaration's
// register should be kept alive past the current statement index.
//
// Takes declaration (activeDeclaration) which is the declaration
// to evaluate.
// Takes lastUseIndices (map[string]int) which maps variable
// names to the index of their last use.
// Takes currentIndex (int) which is the index of the current
// statement being processed.
//
// Returns true if the declaration should be retained.
func (c *compiler) shouldRetainDeclaration(
	declaration activeDeclaration,
	lastUseIndices map[string]int,
	currentIndex int,
) bool {
	lastUse, tracked := lastUseIndices[declaration.name]
	if !tracked || lastUse > currentIndex {
		return true
	}
	currentLocation, found := c.scopes.lookupVar(declaration.name)
	if !found || currentLocation.isUpvalue || currentLocation.isCaptured || currentLocation.isIndirect {
		return true
	}
	c.scopes.alloc.recycleRegister(currentLocation.kind, currentLocation.register)
	return false
}

// isDeclaringStatement reports whether a statement introduces
// new variable bindings whose registers must survive across
// subsequent statements. Short variable declarations (:=),
// var/const/type declarations, and labelled wrappers around
// declarations all qualify.
//
// Takes statement (ast.Stmt) which is the statement to inspect.
//
// Returns true if the statement is a declaring statement.
func isDeclaringStatement(statement ast.Stmt) bool {
	switch s := statement.(type) {
	case *ast.AssignStmt:
		return s.Tok == token.DEFINE
	case *ast.DeclStmt:
		return true
	case *ast.LabeledStmt:
		return isDeclaringStatement(s.Stmt)
	default:
		return false
	}
}

// extractDeclaredNames returns the variable names introduced by
// a declaring statement (:= or var/const).
//
// Takes statement (ast.Stmt) which is the statement to inspect.
//
// Returns the declared names, or nil for non-declaring
// statements.
func extractDeclaredNames(statement ast.Stmt) []string {
	switch s := statement.(type) {
	case *ast.AssignStmt:
		return extractShortVarDeclNames(s)
	case *ast.DeclStmt:
		return extractDeclStmtNames(s)
	case *ast.LabeledStmt:
		return extractDeclaredNames(s.Stmt)
	default:
		return nil
	}
}

// extractShortVarDeclNames returns the variable names introduced
// by a short variable declaration (:=).
//
// Takes statement (*ast.AssignStmt) which is the assignment
// statement to inspect.
//
// Returns the declared names, or nil if the statement is not a
// short variable declaration.
func extractShortVarDeclNames(statement *ast.AssignStmt) []string {
	if statement.Tok != token.DEFINE {
		return nil
	}
	var names []string
	for _, leftHandSide := range statement.Lhs {
		if identifier, ok := leftHandSide.(*ast.Ident); ok && identifier.Name != blankIdentName {
			names = append(names, identifier.Name)
		}
	}
	return names
}

// extractDeclStmtNames returns the variable names introduced by
// a var or const declaration statement.
//
// Takes statement (*ast.DeclStmt) which is the declaration
// statement to inspect.
//
// Returns the declared names, or nil if no value specs are
// found.
func extractDeclStmtNames(statement *ast.DeclStmt) []string {
	generalDeclaration, ok := statement.Decl.(*ast.GenDecl)
	if !ok {
		return nil
	}
	var names []string
	for _, spec := range generalDeclaration.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok {
			for _, name := range valueSpec.Names {
				if name.Name != blankIdentName {
					names = append(names, name.Name)
				}
			}
		}
	}
	return names
}

// computeLastUseIndices pre-scans a statement list to determine
// the index of the last statement that references each locally
// declared variable.
//
// Takes statements ([]ast.Stmt) which is the list of statements
// to scan.
//
// Returns a map from variable name to last-use index, or nil
// when no declarations exist or when the list contains
// goto/label statements that invalidate forward-only liveness
// analysis.
func computeLastUseIndices(statements []ast.Stmt) map[string]int {
	declared, hasGotoOrLabel := collectDeclaredNamesAndLabels(statements)
	if len(declared) == 0 || hasGotoOrLabel {
		return nil
	}
	return scanLastUsePerVariable(statements, declared)
}

// collectDeclaredNamesAndLabels scans statements for locally
// declared variable names and for goto/label statements that
// invalidate forward-only liveness analysis.
//
// Takes statements ([]ast.Stmt) which is the list of statements
// to scan.
//
// Returns the set of declared names and whether any goto or
// label statements were found.
func collectDeclaredNamesAndLabels(statements []ast.Stmt) (map[string]struct{}, bool) {
	declared := make(map[string]struct{})
	hasGotoOrLabel := false
	for _, statement := range statements {
		for _, name := range extractDeclaredNames(statement) {
			declared[name] = struct{}{}
		}
		if !hasGotoOrLabel {
			hasGotoOrLabel = statementHasGotoOrLabel(statement)
		}
	}
	return declared, hasGotoOrLabel
}

// statementHasGotoOrLabel reports whether a statement is a goto
// branch or a labelled statement.
//
// Takes statement (ast.Stmt) which is the statement to inspect.
//
// Returns true if the statement is a goto or label.
func statementHasGotoOrLabel(statement ast.Stmt) bool {
	switch s := statement.(type) {
	case *ast.BranchStmt:
		return s.Tok == token.GOTO
	case *ast.LabeledStmt:
		_ = s
		return true
	default:
		return false
	}
}

// scanLastUsePerVariable walks each statement and records the
// index of the last statement that references each declared
// variable name.
//
// Takes statements ([]ast.Stmt) which is the list of statements
// to walk.
// Takes declared (map[string]struct{}) which is the set of
// variable names to track.
//
// Returns a map from variable name to the index of the last
// statement that references it.
func scanLastUsePerVariable(statements []ast.Stmt, declared map[string]struct{}) map[string]int {
	lastUse := make(map[string]int, len(declared))
	for i, statement := range statements {
		ast.Inspect(statement, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok {
				if _, isDeclared := declared[identifier.Name]; isDeclared {
					lastUse[identifier.Name] = i
				}
			}
			return true
		})
	}
	return lastUse
}
