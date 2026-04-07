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

package compiler_domain

import (
	"context"
	"strings"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// TransformJSImportPath transforms a JS import path if it uses the @/ alias.
// The @/ prefix is expanded to the module name from context, and the path is
// rewritten to the asset serve URL.
//
// For example:
// @/lib/utils.js -> /_piko/assets/mymodule/lib/utils.js
// This matches how the build system registers assets (with module-qualified IDs).
//
// Takes importPath (string) which is the original import path from the PKC.
//
// Returns string which is the transformed path, or original if no transform
// needed.
// Returns *compiler_dto.JSDependency which is the dependency if the path was
// transformed, or nil otherwise.
func TransformJSImportPath(ctx context.Context, importPath string, moduleName string) (string, *compiler_dto.JSDependency) {
	ctx, l := logger_domain.From(ctx, log)
	if !strings.HasPrefix(importPath, "@/") {
		return importPath, nil
	}

	if moduleName == "" {
		l.Warn("Cannot transform @/ import: no module name in context",
			logger_domain.String("importPath", importPath))
		return importPath, nil
	}

	subpath := strings.TrimPrefix(importPath, "@/")
	resolvedPath := moduleName + "/" + subpath

	servedPath := assetpath.DefaultServePath + "/" + resolvedPath

	dependency := &compiler_dto.JSDependency{
		OriginalPath: importPath,
		ResolvedPath: resolvedPath,
		ServedPath:   servedPath,
	}

	l.Trace("Transformed JS import path",
		logger_domain.String("original", importPath),
		logger_domain.String("resolved", resolvedPath),
		logger_domain.String("served", servedPath),
	)

	return servedPath, dependency
}

// isJSImportTransformable reports whether the given import path can be
// changed. Paths that should not be changed include external URLs (http://,
// https://), data URIs (data:), absolute paths (/), and paths not starting
// with @/.
//
// Takes importPath (string) which is the path to check.
//
// Returns bool which is true if the path starts with @/.
func isJSImportTransformable(importPath string) bool {
	return strings.HasPrefix(importPath, "@/")
}
