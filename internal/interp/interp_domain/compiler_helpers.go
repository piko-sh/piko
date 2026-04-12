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

// activeDeclaration tracks a variable binding introduced by a declaring statement whose
// register must survive across subsequent statements.
type activeDeclaration struct {
	// name is the variable name introduced by the declaration.
	name string

	// location is the register allocation for the variable.
	location varLocation
}

// trackOrRestoreDeclarations appends each name introduced by a declaring statement to
// active, or restores the register-allocation watermark when statement does not declare
// any name.
//
// Takes statement (ast.Stmt) which is the statement being processed for declarations.
// Takes watermark ([NumRegisterKinds]uint32) which is the register-allocation watermark
// to restore when no declaration is present.
// Takes active ([]activeDeclaration) which is the current list of tracked active
// declarations.
//
// Returns the updated list of active declarations.
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

// trackDeclaredName appends a new activeDeclaration for name when it is visible in the
// current scope and not already tracked.
//
// Takes name (string) which is the declared variable name to track.
// Takes active ([]activeDeclaration) which is the current list of tracked declarations.
//
// Returns the updated list of active declarations.
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

// recycleDeadDeclarations drops declarations whose last-use statement has been compiled
// and recycles their registers, returning a filtered list (or active unchanged when
// lastUseIndices is nil).
//
// Takes active ([]activeDeclaration) which is the current list of tracked declarations.
// Takes lastUseIndices (map[string]int) which maps each name to the index of its last
// use.
// Takes currentIndex (int) which is the index of the statement just compiled.
//
// Returns the filtered list of declarations whose registers must survive.
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

// shouldRetainDeclaration reports whether declaration's register must survive past
// currentIndex. Upvalue, captured, and indirect locations are always retained; otherwise
// the decision follows the recorded last-use index.
//
// Takes declaration (activeDeclaration) which is the declaration whose retention is being
// decided.
// Takes lastUseIndices (map[string]int) which maps each name to the index of its last
// use.
// Takes currentIndex (int) which is the index of the statement just compiled.
//
// Returns true when the declaration's register must survive past currentIndex.
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

// isDeclaringStatement reports whether statement introduces new variable bindings whose
// registers must survive across subsequent statements. Short variable declarations (:=),
// var/const/type declarations, and labelled wrappers around them all qualify.
//
// Takes statement (ast.Stmt) which is the candidate AST statement.
//
// Returns true when the statement introduces a new variable binding.
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

// extractDeclaredNames returns the variable names introduced by a declaring statement (:=
// or var/const), or nil for non-declaring statements. The blank identifier is filtered
// out.
//
// Takes statement (ast.Stmt) which is the AST statement to inspect.
//
// Returns the slice of declared non-blank variable names, or nil when none.
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

// extractShortVarDeclNames returns the non-blank identifiers introduced by a short
// variable declaration (:=), or nil when statement is not a := assignment.
//
// Takes statement (*ast.AssignStmt) which is the AST assignment statement to inspect.
//
// Returns the slice of non-blank declared names, or nil when not a :=.
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

// extractDeclStmtNames returns the non-blank names introduced by a var or const
// declaration statement, or nil when statement contains no value specs.
//
// Takes statement (*ast.DeclStmt) which is the AST declaration statement to inspect.
//
// Returns the slice of non-blank declared names, or nil when no value specs.
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

// computeLastUseIndices pre-scans statements to find the index of the last reference to
// each locally declared name.
//
// Takes statements ([]ast.Stmt) which is the block of statements to scan.
//
// Returns a map from declared name to last-use index, or nil when no declarations exist
// or goto/label statements invalidate the forward-only liveness assumption.
func computeLastUseIndices(statements []ast.Stmt) map[string]int {
	declared, hasGotoOrLabel := collectDeclaredNamesAndLabels(statements)
	if len(declared) == 0 || hasGotoOrLabel {
		return nil
	}
	return scanLastUsePerVariable(statements, declared)
}

// collectDeclaredNamesAndLabels scans statements and returns the set of locally declared
// names together with a flag that is true when any goto or labelled statement appears.
//
// Takes statements ([]ast.Stmt) which is the block of statements to scan.
//
// Returns the set of locally declared names and a boolean that is true when any goto or
// labelled statement appears in the block.
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

// statementHasGotoOrLabel reports whether statement is a goto branch or a labelled
// statement.
//
// Takes statement (ast.Stmt) which is the AST statement to inspect.
//
// Returns true when the statement is a goto branch or labelled statement.
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

// scanLastUsePerVariable walks each statement and records the index of the last statement
// that references each name in declared.
//
// Takes statements ([]ast.Stmt) which is the block of statements to scan.
// Takes declared (map[string]struct{}) which is the set of locally declared names to
// track.
//
// Returns a map from declared name to the index of its last use.
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
