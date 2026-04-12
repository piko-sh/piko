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
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"strings"
)

// ensureSessionTables initialises persistent compiler-state tables on the first Submit.
// Deferring this work out of NewSession keeps a constructed-and-discarded session from
// allocating the underlying CompiledFunction.
func (sess *Session) ensureSessionTables() {
	if sess.rootFunction == nil {
		sess.rootFunction = &CompiledFunction{name: "<session>"}
		sess.funcTable = make(map[string]uint16)
		sess.globalVariables = make(map[string]globalVariableInfo)
		sess.compiledDecls = make(map[string]bool)
		sess.executedInits = make(map[uint16]bool)
		sess.compiledInits = make(map[string]bool)
	}
}

// submit runs the Session pipeline: preprocess short-var, classify, redeclaration check,
// re-Check the combined source, compile only new declarations against the persistent
// compiler tables, execute new init / var-init code, run the transient _eval_ body. State
// mutation is gated on success.
//
// Type checking sets DisableUnusedImportCheck so that an import added in one Submit and
// first used in a later Submit does not raise an "imported and not used" error, and so
// the synthetic _eval_ body (which is a function body) does not reject session-scope
// imports for the same reason.
//
// Takes ctx (context.Context) for cancellation.
// Takes code (string) which is the user submission.
//
// Returns the result of the trailing expression (or nil) and any error encountered. On
// error the session state is unchanged.
func (sess *Session) submit(ctx context.Context, code string) (any, error) {
	sess.ensureSessionTables()

	preprocessed, err := preprocessShortVar(code)
	if err != nil {
		return nil, err
	}

	cl := classifyLines(strings.Split(preprocessed, newlineSep))

	candidateImports, err := sess.candidateImportPaths(cl.imports)
	if err != nil {
		return nil, err
	}
	candidateDecls, err := sess.parseCandidateDecls(cl.decls)
	if err != nil {
		return nil, err
	}
	if err := sess.detectRedeclaration(candidateDecls); err != nil {
		return nil, err
	}

	combined := sess.buildCombinedSource(candidateImports, cl.decls, cl.statements)

	file, err := parser.ParseFile(sess.service.fileSet, evalFileName, combined, 0)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}

	lastExpr, hasResult := sess.service.rewriteLastExprStmt(file)

	info := sess.service.newTypesInfo()
	conf := sess.service.newTypesConfig()
	conf.DisableUnusedImportCheck = true
	if _, err := conf.Check(mainPackageName, sess.service.fileSet, []*ast.File{file}, info); err != nil {
		return nil, fmt.Errorf(errChainFmt, errTypeCheck, sess.service.enrichTypeCheckError(err, []*ast.File{file}, nil))
	}
	if hasResult && !expressionYieldsValue(info, lastExpr) {
		hasResult = false
	}

	snapshot := sess.snapshot()
	result, err := sess.compileAndExecute(ctx, file, info, lastExpr, hasResult, candidateDecls)
	if err != nil {
		sess.restore(snapshot)
		return nil, err
	}

	sess.commit(candidateImports, candidateDecls)
	return result, nil
}

// candidateImportPaths returns the set of import paths in the submission's import block
// that are NOT already in the session's import set. Duplicates within the submission are
// also deduped.
//
// Takes importLines ([]string) which are the import lines as returned by classifyLines.
//
// Returns []string of newly-introduced paths in source order, and an error for malformed
// import declarations.
func (sess *Session) candidateImportPaths(importLines []string) ([]string, error) {
	if len(importLines) == 0 {
		return nil, nil
	}
	src := "package main\n" + strings.Join(importLines, newlineSep) + newlineSep
	file, err := parser.ParseFile(sess.service.fileSet, evalFileName, src, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}
	var out []string
	seen := map[string]struct{}{}
	for _, declaration := range file.Decls {
		for _, path := range importPathsIn(declaration) {
			if _, exists := sess.imports[path]; exists {
				continue
			}
			if _, dup := seen[path]; dup {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, path)
		}
	}
	return out, nil
}

// parseCandidateDecls parses the submission's declaration block to extract the set of
// session-scope names introduced for redeclaration detection and commit-time bookkeeping.
//
// Takes declLines ([]string) which are the decl lines from classifyLines.
//
// Returns []declRecord (one per introduced name) plus any parse error.
func (sess *Session) parseCandidateDecls(declLines []string) ([]declRecord, error) {
	if len(declLines) == 0 {
		return nil, nil
	}
	src := "package main\n" + strings.Join(declLines, newlineSep) + newlineSep
	file, err := parser.ParseFile(sess.service.fileSet, evalFileName, src, 0)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}
	source := strings.Join(declLines, newlineSep)
	var out []declRecord
	for _, declaration := range file.Decls {
		for _, named := range declNamesIn(declaration) {
			out = append(out, declRecord{name: named.Name, kind: named.Kind, source: source})
		}
	}
	return out, nil
}

// detectRedeclaration walks the candidate declarations and returns the first
// redeclaration error, or nil if every name is fresh. Two names within the SAME
// submission also clash; the check uses a per-submit seen-set to enforce that.
//
// Takes candidates ([]declRecord) which are the names introduced by the candidate
// declarations.
//
// Returns error describing the first redeclaration, or nil on success.
func (sess *Session) detectRedeclaration(candidates []declRecord) error {
	inSubmission := map[string]struct{}{}
	for _, candidate := range candidates {
		if existing, exists := sess.declaredNames[candidate.name]; exists {
			return fmt.Errorf(
				"session: %s redeclared in this session (originally %s); use Reset to start over",
				candidate.name,
				existing,
			)
		}
		if _, dup := inSubmission[candidate.name]; dup {
			return fmt.Errorf(
				"session: %s redeclared within this submission",
				candidate.name,
			)
		}
		inSubmission[candidate.name] = struct{}{}
	}
	return nil
}

// buildCombinedSource assembles the synthetic Go source consumed by re-Check + compile.
// The shape matches what buildMixedSource emits for one-shot Eval, but stitches in the
// persistent session decls (imports + decls accumulated across prior Submits) so the
// type-checker sees the full session universe.
//
// Takes newImports ([]string) which are the import paths to add this Submit (already
// deduped against sess.imports).
// Takes newDeclLines ([]string) which are the decl lines from the current submission.
// Takes statements ([]string) which are the statement lines for _eval_.
//
// Returns string which is the synthetic combined source.
func (sess *Session) buildCombinedSource(newImports []string, newDeclLines []string, statements []string) string {
	var src strings.Builder
	src.WriteString("package main\n")
	for _, path := range sess.importOrder {
		src.WriteString("import \"")
		src.WriteString(path)
		src.WriteString("\"\n")
	}
	for _, path := range newImports {
		src.WriteString("import \"")
		src.WriteString(path)
		src.WriteString("\"\n")
	}
	for _, record := range sess.decls {
		src.WriteString(record.source)
		src.WriteString(newlineSep)
	}
	for _, line := range newDeclLines {
		src.WriteString(line)
		src.WriteString(newlineSep)
	}
	src.WriteString("func ")
	src.WriteString(evalFuncName)
	src.WriteString("() {\n")
	for _, line := range statements {
		src.WriteString(line)
		src.WriteString(newlineSep)
	}
	src.WriteString("}\n")
	return src.String()
}

// sessionSnapshot captures the persistent session/compiler tables before a
// compile/execute pass so an error can roll them back to a clean state. Held by submit()
// and applied via restore() when any step after type-check fails.
type sessionSnapshot struct {
	// funcTable is a clone of the persistent funcTable.
	funcTable map[string]uint16

	// globalVariables is a clone of the persistent globalVariables map.
	globalVariables map[string]globalVariableInfo

	// executedInits is a clone of the persistent executedInits set.
	executedInits map[uint16]bool

	// compiledDecls is a clone of the persistent compiledDecls set.
	compiledDecls map[string]bool

	// compiledInits is a clone of the persistent compiledInits set.
	compiledInits map[string]bool

	// funcsLen is the prior length of rootFunction.functions; on rollback the slice is
	// truncated to this length.
	funcsLen int
}

// snapshot captures the current state of the persistent compiler tables so a failed
// compile/execute pass can restore them.
//
// Returns sessionSnapshot which holds the deep-copy state.
func (sess *Session) snapshot() sessionSnapshot {
	return sessionSnapshot{
		funcsLen:        len(sess.rootFunction.functions),
		funcTable:       maps.Clone(sess.funcTable),
		globalVariables: maps.Clone(sess.globalVariables),
		executedInits:   maps.Clone(sess.executedInits),
		compiledDecls:   maps.Clone(sess.compiledDecls),
		compiledInits:   maps.Clone(sess.compiledInits),
	}
}

// restore reverts the persistent compiler tables to a previous snapshot. Used when a
// compile/execute pass fails after the type check succeeded.
//
// Takes snap (sessionSnapshot) which is the prior state to restore.
func (sess *Session) restore(snap sessionSnapshot) {
	sess.rootFunction.functions = sess.rootFunction.functions[:snap.funcsLen]
	sess.funcTable = snap.funcTable
	sess.globalVariables = snap.globalVariables
	sess.executedInits = snap.executedInits
	sess.compiledDecls = snap.compiledDecls
	sess.compiledInits = snap.compiledInits
}

// commit applies a successful Submit's candidate state to the session: imports merged,
// declarations appended, submitCount incremented.
//
// Takes newImports ([]string) which are the newly introduced import paths.
// Takes candidates ([]declRecord) which are the newly introduced declarations.
func (sess *Session) commit(newImports []string, candidates []declRecord) {
	for _, path := range newImports {
		if _, exists := sess.imports[path]; exists {
			continue
		}
		sess.imports[path] = struct{}{}
		sess.importOrder = append(sess.importOrder, path)
	}
	for _, candidate := range candidates {
		sess.decls = append(sess.decls, candidate)
		sess.declaredNames[candidate.name] = candidate.kind
		sess.compiledDecls[candidate.name] = true
	}
	sess.submitCount++
}

// compileAndExecute compiles new declarations and runs the eval body against the
// persistent session compiler.
//
// Splits file.Decls into already-compiled (skipped) and new declarations that are
// registered, with var-init executed, function bodies compiled, and init functions
// executed. Init declaration signatures are recorded in compiledInits before compilation,
// and the snapshot/restore pair captures that map so a later compile failure rolls them
// back. Then runs the synthetic _eval_ function and returns its result. The eval frame's
// functions slice aliases sess.rootFunction.functions during execution.
//
// Takes ctx (context.Context) for cancellation.
// Takes file (*ast.File) which is the combined synthetic file (all decls + eval body).
// Takes info (*types.Info) which holds type-check results.
// Takes lastExpr (ast.Expr) the trailing expression, for return-value extraction.
// Takes hasResult (bool) which is true when the trailing statement yields a value.
// Takes candidates ([]declRecord) which are the declarations introduced this Submit; used
// to derive the "new decls" subset.
//
// Returns the eval result and any error from compile or execute.
func (sess *Session) compileAndExecute(
	ctx context.Context,
	file *ast.File,
	info *types.Info,
	lastExpr ast.Expr,
	hasResult bool,
	candidates []declRecord,
) (any, error) {
	newNames := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		newNames[candidate.name] = struct{}{}
	}

	newDecls := filterDeclsByName(file.Decls, newNames, sess.compiledInits)
	for _, declaration := range newDecls {
		if isInitFuncDecl(declaration) {
			sess.compiledInits[initDeclSignature(declaration)] = true
		}
	}

	evalFunction := &CompiledFunction{name: "<session-eval>"}

	c := sess.service.newSessionCompiler(ctx, sess.rootFunction, evalFunction, info, sess.globalVariables, sess.funcTable)

	c.registerPackageLevelVarsFromDecls(ctx, newDecls)

	newFile := &ast.File{Name: file.Name, Decls: newDecls}
	if err := sess.runSessionVarInits(ctx, c, newFile); err != nil {
		return nil, fmt.Errorf(errFmtEvaluatingFile, err)
	}

	if err := c.compileNonEvalFuncDecls(ctx, newDecls); err != nil {
		return nil, fmt.Errorf(errFmtEvaluatingFile, err)
	}

	evalFunction.functions = sess.rootFunction.functions

	for _, initIndex := range c.initFunctionIndices {
		if sess.executedInits[initIndex] {
			continue
		}
		if err := sess.service.executeInitFunc(ctx, sess.rootFunction, sess.rootFunction.functions[initIndex]); err != nil {
			return nil, fmt.Errorf(errFmtEvaluatingFile, err)
		}
		sess.executedInits[initIndex] = true
	}

	return c.compileAndRunEvalBody(ctx, sess.service, file, info, lastExpr, hasResult)
}

// runSessionVarInits compiles and executes the variable-initialisation function for the
// supplied file.
//
// Unlike Service.compileAndRunVarInits, it wires vm.functions and vm.rootFunction to the
// session's persistent root before executing. This wiring lets opMakeClosure inside a var
// initialiser, for example `var f = func() int { return a }`, resolve its function index
// against the session function table.
//
// Takes ctx (context.Context) for cancellation.
// Takes c (*compiler) which is the session-mode compiler.
// Takes newFile (*ast.File) containing the new var declarations.
//
// Returns error when compilation or execution fails.
func (sess *Session) runSessionVarInits(ctx context.Context, c *compiler, newFile *ast.File) error {
	variableInit, err := c.compileVariableInitFunction(ctx, []*ast.File{newFile})
	if err != nil {
		return fmt.Errorf("compiling variable initialisers: %w", err)
	}
	if variableInit == nil {
		return nil
	}
	vm := newVM(ctx, sess.service.globals, sess.service.symbols)
	vm.limits = sess.service.limits
	sess.service.applyVMOutputs(vm)
	vm.functions = sess.rootFunction.functions
	vm.rootFunction = sess.rootFunction
	vm.ensureCallStack()
	defer vm.releaseArena()
	defer vm.finishWatcher()
	vm.pushFrame(variableInit)
	if _, err := vm.runGuarded(0); err != nil {
		return fmt.Errorf(errVarinitFmt, err)
	}
	vm.globals.materialiseStrings(vm.arena)
	return nil
}

// filterDeclsByName returns the subset of decls whose introduced name is in the keepNames
// set. init() function decls are always kept (they have no session-scope name but must be
// compiled in the new-decl pass so executedInits can track them).
//
// Takes declarations ([]ast.Decl) which is the full declaration list.
// Takes keepNames (map[string]struct{}) which is the set of names to retain.
//
// Returns []ast.Decl with only the retained declarations.
func filterDeclsByName(declarations []ast.Decl, keepNames map[string]struct{}, compiledInits map[string]bool) []ast.Decl {
	out := make([]ast.Decl, 0, len(keepNames))
	for _, declaration := range declarations {
		if shouldKeepDecl(declaration, keepNames, compiledInits) {
			out = append(out, declaration)
		}
	}
	return out
}

// shouldKeepDecl reports whether a single declaration should be retained by
// filterDeclsByName.
//
// Init function declarations are retained only when their signature has not been
// compiled. Import declarations are always skipped. Named declarations are retained when
// at least one introduced name appears in keepNames.
//
// Takes declaration (ast.Decl) which is the candidate declaration.
// Takes keepNames (map[string]struct{}) which is the set of names to retain.
// Takes compiledInits (map[string]bool) which records init signatures already compiled in
// earlier Submits.
//
// Returns bool which is true when the declaration should be kept.
func shouldKeepDecl(declaration ast.Decl, keepNames map[string]struct{}, compiledInits map[string]bool) bool {
	if isInitFuncDecl(declaration) {
		return !compiledInits[initDeclSignature(declaration)]
	}
	if isImportDecl(declaration) {
		return false
	}
	named := declNamesIn(declaration)
	if len(named) == 0 {
		return false
	}
	return anyNameMatches(named, keepNames)
}

// anyNameMatches reports whether at least one SessionDecl name appears in the keepNames
// set.
//
// Takes named ([]SessionDecl) which is the list to test.
// Takes keepNames (map[string]struct{}) which is the membership set.
//
// Returns bool which is true on the first match.
func anyNameMatches(named []SessionDecl, keepNames map[string]struct{}) bool {
	for _, entry := range named {
		if _, ok := keepNames[entry.Name]; ok {
			return true
		}
	}
	return false
}

// isImportDecl reports whether the declaration is an import block.
//
// Takes declaration (ast.Decl) which is the declaration to test.
//
// Returns bool which is true for import declarations.
func isImportDecl(declaration ast.Decl) bool {
	gen, ok := declaration.(*ast.GenDecl)
	return ok && gen.Tok == token.IMPORT
}
