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

package pikotest_dto

import (
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// BuildASTFunc is the signature of the generated BuildAST function in compiled
// components. It takes a RequestData and props, and returns the AST, metadata,
// and any runtime diagnostics.
type BuildASTFunc func(
	r *templater_dto.RequestData,
	propsData any,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*RuntimeDiagnostic)

// RuntimeDiagnostic represents a runtime error or warning from component
// execution. This is a type alias to generator_dto.RuntimeDiagnostic for
// compatibility with the generated BuildAST functions.
type RuntimeDiagnostic = generator_dto.RuntimeDiagnostic
