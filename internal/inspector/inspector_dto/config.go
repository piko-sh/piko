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

package inspector_dto

const (
	// AnalysisBuildTag is the Go build tag used to exclude physical dist files
	// from type-checking during LSP analysis. Generated dist files carry the
	// constraint "//go:build !piko_analysis", so passing this tag to the Go
	// toolchain makes them invisible to the inspector while overlay files
	// (which have no constraint) remain visible.
	AnalysisBuildTag = "piko_analysis"
)

// AnalysisBuildFlags is the set of build flags that activates the analysis
// build tag, causing the Go toolchain to skip physical dist files.
var AnalysisBuildFlags = []string{"-tags=" + AnalysisBuildTag}

// Config holds settings for the inspector package.
type Config struct {
	// MaxParseWorkers is the maximum number of workers that can parse files at the
	// same time. If nil, uses a default value.
	MaxParseWorkers *int

	// BaseDir is the root directory of the project being analysed.
	BaseDir string

	// ModuleName is the Go module path used to derive package import paths.
	ModuleName string

	// GOOS specifies the target operating system; empty uses the current OS.
	GOOS string

	// GOARCH specifies the target architecture; empty uses the current system.
	GOARCH string

	// GOCACHE is the path to the Go build cache folder; empty uses the default.
	GOCACHE string

	// GOMODCACHE is the path to the Go module cache folder.
	GOMODCACHE string

	// BuildFlags specifies extra flags to pass to the Go build system.
	BuildFlags []string

	// UseStandardLoader causes the inspector to use the standard
	// golang.org/x/tools/go/packages.Load instead of the faster
	// quickpackages.Load.
	//
	// This is slower but always stable, as it is maintained by the
	// Go team. Useful as a fallback when quickpackages encounters
	// issues with specific dependency configurations.
	UseStandardLoader bool
}
