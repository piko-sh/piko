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
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/gcexportdata"

	"piko.sh/piko/internal/modules/modules_domain"
)

// bridgeBytecodeModule bridges a bytecode-loaded module's exports.
//
// Mirrors Service.bridgePackageExports but for a module whose CompiledFileSet was
// reconstructed from a serialised [modules_domain.ModuleBundle] rather than produced by a
// fresh source compile. Constants and vars are not registered as reflect.Values because
// the bytecode wire format does not carry them; the decoded types.Package already carries
// constant values, and var refs from bytecode-loaded modules to package-level vars are
// not supported in v1 (uncommon in practice since modules are mostly functions). The
// runtime export set is built from CompiledFileSet's entrypoints map (one reflect.Value
// per exported function closure), and external methods are republished so cross-module
// method dispatch resolves.
//
// Idempotent under repeated load of the same bundle: the symbol registry's
// RegisterPackage / RegisterTypesPackage both last-write-wins.
//
// Takes entry (modules_domain.TypesExportEntry) which carries the serialised
// types-package data and per-package funcTable.
// Takes cfs (*CompiledFileSet) which is the reconstructed file set.
//
// Returns error when decoding or registration fails.
func (s *Service) bridgeBytecodeModule(entry modules_domain.TypesExportEntry, cfs *CompiledFileSet) error {
	if cfs == nil || cfs.root == nil {
		return errors.New("interp_domain: bridgeBytecodeModule needs a non-nil CompiledFileSet root")
	}

	typesPkg, err := s.decodeTypesExportEntry(entry)
	if err != nil {
		return fmt.Errorf("interp_domain: decoding TypesExport for %q: %w", entry.ImportPath, err)
	}

	exports := buildExportsFromFuncTable(cfs, entry.FuncTable)

	for _, pending := range s.pendingVarBridges {
		if pending.importPath != entry.ImportPath {
			continue
		}
		if exports == nil {
			exports = make(map[string]reflect.Value, len(pending.vars))
		}
		for _, v := range pending.vars {
			exports[v.name] = v.storage
		}
	}

	s.symbols.RegisterPackage(entry.ImportPath, exports)
	s.symbols.RegisterTypesPackage(entry.ImportPath, typesPkg)
	publishExternalMethods(s.globals, cfs.root)
	return nil
}

// decodeTypesExportEntry rebuilds a [go/types.Package].
//
// Seeds the gcexportdata import cache with packages already registered in the Service's
// SymbolRegistry so cross-module type references resolve to the same *types.Package
// instances rather than distinct stubs.
//
// Takes entry (modules_domain.TypesExportEntry) which carries the serialised
// types-package data.
//
// Returns *types.Package which is the rebuilt go/types package.
// Returns error when the entry is empty or decoding fails.
func (s *Service) decodeTypesExportEntry(entry modules_domain.TypesExportEntry) (*types.Package, error) {
	if len(entry.Data) == 0 {
		return nil, errors.New("empty TypesExport data")
	}
	imports := s.symbols.SnapshotTypesPackages()
	fset := token.NewFileSet()
	pkg, err := gcexportdata.Read(bytes.NewReader(entry.Data), fset, imports, entry.ImportPath)
	if err != nil {
		return nil, err
	}
	return pkg, nil
}

// buildExportsFromFuncTable constructs the exports map for one package.
//
// Uses the bundle's shared CompiledFileSet and the per-package funcTable carried in the
// [modules_domain.TypesExportEntry]. Filters out unexported names and synthetic dispatch
// keys (those containing a "."), mirroring collectPackageExports's rules for
// source-compiled packages.
//
// Takes cfs (*CompiledFileSet) which is the reconstructed file set.
// Takes funcTable (map[string]uint16) which maps each exported name to its index in
// cfs.root.functions.
//
// Returns map[string]reflect.Value which is the exports map keyed by name.
func buildExportsFromFuncTable(cfs *CompiledFileSet, funcTable map[string]uint16) map[string]reflect.Value {
	if cfs == nil || cfs.root == nil {
		return nil
	}
	exports := make(map[string]reflect.Value, len(funcTable))
	for name, index := range funcTable {
		if !ast.IsExported(name) || strings.Contains(name, ".") {
			continue
		}
		if int(index) >= len(cfs.root.functions) {
			continue
		}
		closure := &runtimeClosure{
			function:     cfs.root.functions[index],
			rootFunction: cfs.root,
		}
		exports[name] = reflect.ValueOf(closure)
	}
	return exports
}
