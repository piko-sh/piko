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

package config

const (
	// PikoInternalPath is the directory within BaseDir for Piko's internal
	// cache, state, and build artefacts.
	PikoInternalPath = ".piko"

	// RegistryPath is the directory within BaseDir for the build output
	// registry (compiled manifests, pages, etc.).
	RegistryPath = ".out"

	// LibFilesystemPath is the path to internal library files served by the
	// framework runtime.
	LibFilesystemPath = "web/lib"

	// DistFilesystemPath is the path to frontend distribution files (bundled
	// JS/CSS from the frontend build step).
	DistFilesystemPath = "frontend/dist"

	// CompiledPagesTargetDir is the output directory within the registry for
	// compiled page artefacts.
	CompiledPagesTargetDir = "dist/pages"

	// CompiledPartialsTargetDir is the output directory within the registry
	// for compiled partial template artefacts.
	CompiledPartialsTargetDir = "dist/partials"

	// CompiledEmailsTargetDir is the output directory within the registry for
	// compiled email template artefacts.
	CompiledEmailsTargetDir = "dist/emails"

	// CompiledPdfsTargetDir is the output directory within the registry for
	// compiled PDF template artefacts.
	CompiledPdfsTargetDir = "dist/pdfs"

	// CompiledAssetsTargetDir is the output directory within the registry for
	// compiled assets (JS/CSS).
	CompiledAssetsTargetDir = "dist/assets"

	// L2CacheDirName is the name of the directory within PikoInternalPath
	// for the persistent on-disk (L2) AST cache.
	L2CacheDirName = "ast_cache"

	// DefaultL1CacheCapacity is the default maximum number of compiled ASTs
	// held in the fast in-memory (L1) cache.
	DefaultL1CacheCapacity = 1000

	// DefaultL1CacheTTLMinutes is the default time-to-live in minutes for
	// items in the in-memory (L1) cache.
	DefaultL1CacheTTLMinutes = 15

	// ManifestFormat is the file format used for compiled manifests.
	ManifestFormat = "flatbuffers"

	// CacheStrategy is the caching strategy used for compiled artefacts.
	CacheStrategy = "flatbuffers"

	// CompilerDebugLogDir is the directory where per-component debug log files
	// are written. It is a subdirectory inside PikoInternalPath.
	CompilerDebugLogDir = ".piko/logs"

	// CompilerDefaultLogLevel is the default verbosity for per-component
	// compilation logs.
	CompilerDefaultLogLevel = "warn"

	// CompilerEnableDebugLogFiles controls whether the compiler writes a
	// detailed debug log for each component it processes. Default: true.
	CompilerEnableDebugLogFiles = true

	// CompilerVerifyGeneratedCode controls whether the compiler parses
	// generated Go code as a sanity check. Default: true.
	CompilerVerifyGeneratedCode = true

	// CompilerEnableStaticHoisting controls whether static template nodes are
	// hoisted to package-level variables. Default: true.
	CompilerEnableStaticHoisting = true
)
