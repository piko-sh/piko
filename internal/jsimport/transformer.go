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

package jsimport

import (
	"path"
	"strings"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/esbuild/ast"
)

const (
	// extensionTS is the TypeScript file extension.
	extensionTS = ".ts"

	// extensionJS is the JavaScript file extension.
	extensionJS = ".js"

	// aliasPrefix is the import path prefix that triggers module alias
	// resolution.
	aliasPrefix = "@/"
)

// NormaliseExtension ensures an import path has a .js extension when it refers
// to a JavaScript or TypeScript source file, leaving other extensions unchanged.
//
// Takes importPath (string) which is the import path to normalise.
//
// Returns string which is the path with a .js extension where appropriate.
func NormaliseExtension(importPath string) string {
	ext := path.Ext(importPath)
	if ext == "" {
		return importPath + extensionJS
	}
	if base, ok := strings.CutSuffix(importPath, extensionTS); ok {
		return base + extensionJS
	}
	return importPath
}

// IsTransformable reports whether the given import path uses the @/ alias and
// can be rewritten to a served asset path.
//
// Takes importPath (string) which is the path to check.
//
// Returns bool which is true if the path starts with @/.
func IsTransformable(importPath string) bool {
	return strings.HasPrefix(importPath, aliasPrefix)
}

// ResolveModuleAlias rewrites an @/ prefixed import path to the corresponding
// served asset URL. The @/ prefix is replaced with
// /_piko/assets/{moduleName}/ and the extension is normalised to .js.
//
// For example, with moduleName "github.com/org/repo":
//
//	@/lib/utils     -> /_piko/assets/github.com/org/repo/lib/utils.js
//	@/lib/utils.ts  -> /_piko/assets/github.com/org/repo/lib/utils.js
//	@/lib/utils.js  -> /_piko/assets/github.com/org/repo/lib/utils.js
//
// Takes importPath (string) which is the @/ prefixed import path.
// Takes moduleName (string) which is the Go module name.
//
// Returns string which is the fully resolved served asset path.
func ResolveModuleAlias(importPath, moduleName string) string {
	subpath := strings.TrimPrefix(importPath, aliasPrefix)
	return NormaliseExtension(assetpath.DefaultServePath + "/" + moduleName + "/" + subpath)
}

// ResolveModulePath returns the module-qualified path (without the serve
// prefix) for an @/ import. This is used by the compiler to build dependency
// records that track which artefacts a component depends on.
//
// For example, with moduleName "github.com/org/repo":
//
//	@/lib/utils -> github.com/org/repo/lib/utils.js
//
// Takes importPath (string) which is the @/ prefixed import path.
// Takes moduleName (string) which is the Go module name.
//
// Returns string which is the module-qualified resolved path.
func ResolveModulePath(importPath, moduleName string) string {
	subpath := strings.TrimPrefix(importPath, aliasPrefix)
	return NormaliseExtension(moduleName + "/" + subpath)
}

// RewriteImportRecords rewrites import paths in parsed esbuild AST import
// records in place, handling alias resolution, .ts-to-.js conversion, and
// extensionless relative import normalisation.
//
// Takes records ([]ast.ImportRecord) which are the parsed import records to
// modify in place.
// Takes moduleName (string) which is the Go module name for @/ resolution,
// or empty to skip alias resolution.
func RewriteImportRecords(records []ast.ImportRecord, moduleName string) {
	for i := range records {
		importPath := records[i].Path.Text
		if importPath == "" {
			continue
		}

		if IsTransformable(importPath) && moduleName != "" {
			records[i].Path.Text = ResolveModuleAlias(importPath, moduleName)
			continue
		}

		if base, ok := strings.CutSuffix(importPath, extensionTS); ok {
			records[i].Path.Text = base + extensionJS
			continue
		}

		if isRelativePath(importPath) && path.Ext(importPath) == "" {
			records[i].Path.Text = importPath + extensionJS
		}
	}
}

// isRelativePath reports whether a path starts with ./ or ../ and therefore
// refers to a local file rather than a bare module specifier or URL.
//
// Takes importPath (string) which is the import path to check.
//
// Returns bool which is true if the path is a relative file reference.
func isRelativePath(importPath string) bool {
	return strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
}
