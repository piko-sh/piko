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
	"bytes"
	"context"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"strings"
)

// newSessionCompiler constructs a compiler that emits the current Submit's eval body into
// a fresh transient evalFunction.
//
// The compiler mounts on a persistent rootFunction (function table holder). funcTable and
// globalVariables are shared with the Session so previously registered names remain
// visible via the standard chain of locals, upvalues, globalVariables, and funcTable in
// compileIdent.
//
// Splitting root from evalFunction matters: the existing compileAndRunEvalBody writes
// bytecode into c.function, sets numRegisters on it, and hands it to vm.execute. If
// c.function were the persistent root, every Submit would overwrite the root's body and
// corrupt the next Submit's function table. Each Submit gets its own evalFunction; the
// root only ever grows its functions slice.
//
// Takes ctx (context.Context) for cancellation.
// Takes root (*CompiledFunction) which holds the persistent function table to grow.
// Takes evalFunction (*CompiledFunction) which is the fresh, per-Submit target into which
// the synthetic _eval_ body compiles.
// Takes info (*types.Info) which is the fresh type-check info for the current Submit.
// Takes globalVariables (map[string]globalVariableInfo) which is the persistent name to
// (index, kind) mapping.
// Takes funcTable (map[string]uint16) which is the persistent function-name to
// root-functions-index mapping.
//
// Returns *compiler configured for session-mode emission.
func (s *Service) newSessionCompiler(
	ctx context.Context,
	root *CompiledFunction,
	evalFunction *CompiledFunction,
	info *types.Info,
	globalVariables map[string]globalVariableInfo,
	funcTable map[string]uint16,
) *compiler {
	s.applyResourceLimits(root)
	c := &compiler{
		fileSet:            s.fileSet,
		info:               info,
		function:           evalFunction,
		rootFunction:       root,
		scopes:             newScopeStack("<session>"),
		funcTable:          funcTable,
		symbols:            s.symbols,
		globalVariables:    globalVariables,
		globals:            s.globals,
		debugEnabled:       s.config != nil && s.config.debugInfo,
		features:           s.features,
		maxLiteralElements: s.maxLiteralElements(),
		maxExpressionDepth: s.maxExpressionDepth(),
	}
	c.initDebugInfo(ctx, nil)
	return c
}

// declNamesIn enumerates every session-scope name introduced by a declaration, supporting
// redeclaration detection on grouped declarations such as `var (x int; y string)`.
//
// Takes declaration (ast.Decl) which is the declaration to walk.
//
// Returns []SessionDecl listing each (name, kind) introduced, in source order.
// Returns an empty slice for declarations with no session-scope effect.
func declNamesIn(declaration ast.Decl) []SessionDecl {
	switch typed := declaration.(type) {
	case *ast.FuncDecl:
		return funcDeclNames(typed)
	case *ast.GenDecl:
		return genDeclNames(typed)
	}
	return nil
}

// funcDeclNames returns the SessionDecl introduced by a function declaration, excluding
// the special init function which is run rather than registered as a named declaration.
//
// Takes functionDeclaration (*ast.FuncDecl) which is the function declaration to inspect.
//
// Returns []SessionDecl with one entry for a non-init named function, or nil for init or
// anonymous declarations.
func funcDeclNames(functionDeclaration *ast.FuncDecl) []SessionDecl {
	if functionDeclaration.Name == nil {
		return nil
	}

	if functionDeclaration.Name.Name == initFuncName && functionDeclaration.Recv == nil {
		return nil
	}
	return []SessionDecl{{Name: functionDeclaration.Name.Name, Kind: SessionDeclFunc}}
}

// genDeclNames returns the SessionDecls introduced by a generic declaration block
// (type/var/const). Imports are excluded because importPathsIn handles them.
//
// Takes genericDeclaration (*ast.GenDecl) which is the declaration block.
//
// Returns []SessionDecl in source order.
func genDeclNames(genericDeclaration *ast.GenDecl) []SessionDecl {
	var out []SessionDecl
	for _, spec := range genericDeclaration.Specs {
		switch typed := spec.(type) {
		case *ast.TypeSpec:
			if typed.Name != nil && typed.Name.Name != "_" {
				out = append(out, SessionDecl{Name: typed.Name.Name, Kind: SessionDeclType})
			}
		case *ast.ValueSpec:
			out = append(out, valueSpecNames(genericDeclaration.Tok, typed)...)
		}
	}
	return out
}

// valueSpecNames returns the SessionDecls introduced by a value spec, classified as const
// or var based on the parent GenDecl's keyword. Blank identifiers ("_") and nil names are
// skipped.
//
// Takes declarationToken (token.Token) which is the parent GenDecl's keyword token.
// Takes spec (*ast.ValueSpec) which lists the names.
//
// Returns []SessionDecl in source order.
func valueSpecNames(declarationToken token.Token, spec *ast.ValueSpec) []SessionDecl {
	kind := SessionDeclVar
	if declarationToken == token.CONST {
		kind = SessionDeclConst
	}
	out := make([]SessionDecl, 0, len(spec.Names))
	for _, name := range spec.Names {
		if name == nil || name.Name == "_" {
			continue
		}
		out = append(out, SessionDecl{Name: name.Name, Kind: kind})
	}
	return out
}

// isInitFuncDecl reports whether the declaration is a top-level init.
//
// True for `func init() { ... }` declarations. Methods named init (any receiver) are not
// package init functions and yield false.
//
// Takes declaration (ast.Decl) which is the declaration to test.
//
// Returns bool which is true for init function declarations.
func isInitFuncDecl(declaration ast.Decl) bool {
	function, ok := declaration.(*ast.FuncDecl)
	return ok && function.Name != nil && function.Name.Name == initFuncName && function.Recv == nil
}

// initDeclSignature returns a stable string fingerprint for an init function declaration
// so the session can recognise an init it has already compiled and avoid re-registering /
// re-running it on subsequent Submits. Uses go/printer with token.NewFileSet so the
// output is independent of source positions.
//
// Takes declaration (ast.Decl) which is expected to be an init function.
//
// Returns string which is the printer-rendered representation, or "" if rendering fails
// or declaration is not an init.
func initDeclSignature(declaration ast.Decl) string {
	if !isInitFuncDecl(declaration) {
		return ""
	}
	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, token.NewFileSet(), declaration); err != nil {
		return ""
	}
	return buffer.String()
}

// importPathsIn extracts the literal import paths from an import declaration, stripping
// surrounding quotes and returning an empty slice for non-import GenDecls.
//
// Takes declaration (ast.Decl) which may be an import declaration.
//
// Returns []string with one entry per imported path.
func importPathsIn(declaration ast.Decl) []string {
	generic, ok := declaration.(*ast.GenDecl)
	if !ok || generic.Tok != token.IMPORT {
		return nil
	}
	out := make([]string, 0, len(generic.Specs))
	for _, spec := range generic.Specs {
		importSpec, ok := spec.(*ast.ImportSpec)
		if !ok || importSpec.Path == nil {
			continue
		}
		out = append(out, strings.Trim(importSpec.Path.Value, "\"`"))
	}
	return out
}
