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

	"piko.sh/piko/internal/compiler/compiler_dto"
)

// InputReaderPort defines the interface for reading SFC source files.
type InputReaderPort interface {
	// ReadSFC reads the source file content for the given identifier.
	//
	// Takes sourceIdentifier (string) which specifies the source to read.
	//
	// Returns []byte which contains the file content.
	// Returns error when the source cannot be read.
	ReadSFC(ctx context.Context, sourceIdentifier string) ([]byte, error)
}

// TransformationPort defines the interface for post-compilation transformations.
type TransformationPort interface {
	// Transform applies a transformation to the compiled artefact.
	//
	// Takes artefact (*compiler_dto.CompiledArtefact) which is the artefact to
	// transform.
	//
	// Returns *compiler_dto.CompiledArtefact which is the transformed artefact.
	// Returns error when the transformation fails.
	Transform(ctx context.Context, artefact *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error)
}

// CSSPreProcessorPort resolves CSS @import statements before CSS is embedded
// into compiled component output by inlining external CSS references.
type CSSPreProcessorPort interface {
	// InlineImports resolves @import statements in the given CSS content,
	// reads the imported files, and returns a single merged CSS string.
	//
	// Takes cssContent (string) which is the raw CSS with potential @import
	// rules.
	// Takes sourcePath (string) which identifies the source file for resolving
	// relative imports.
	//
	// Returns string which is the CSS with all imports inlined.
	// Returns error when import resolution or file reading fails.
	InlineImports(ctx context.Context, cssContent string, sourcePath string) (string, error)
}

// CompilerService defines the interface for compiling single-file components.
type CompilerService interface {
	// CompileSingle compiles a single source file and returns the result.
	//
	// Takes sourceIdentifier (string) which identifies the source to compile.
	//
	// Returns *CompiledArtefact which contains the compiled output.
	// Returns error when compilation fails.
	CompileSingle(ctx context.Context, sourceIdentifier string) (*compiler_dto.CompiledArtefact, error)

	// CompileSFCBytes compiles a single-file component from raw bytes.
	//
	// Takes sourceIdentifier (string) which identifies the source for error
	// reporting.
	// Takes rawSFC ([]byte) which contains the raw single-file component content.
	//
	// Returns *CompiledArtefact which contains the compiled output.
	// Returns error when compilation fails.
	CompileSFCBytes(ctx context.Context, sourceIdentifier string, rawSFC []byte) (*compiler_dto.CompiledArtefact, error)
}
