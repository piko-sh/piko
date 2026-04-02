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

//go:build !(js && wasm)

package wasm_adapters

import (
	"context"
	"errors"

	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

// InterpreterAdapter is a stub for non-WASM builds.
// The actual implementation is in driven_interpreter.go with build tag js && wasm.
type InterpreterAdapter struct{}

var _ wasm_domain.InterpreterPort = (*InterpreterAdapter)(nil)

// InterpreterAdapterOption configures an InterpreterAdapter.
type InterpreterAdapterOption func(*InterpreterAdapter)

// NewInterpreterAdapter creates a stub interpreter adapter for non-WASM builds.
//
// Returns *InterpreterAdapter which is an empty adapter for non-WASM platforms.
func NewInterpreterAdapter(_ ...InterpreterAdapterOption) *InterpreterAdapter {
	return &InterpreterAdapter{}
}

// Interpret is not available in non-WASM builds.
//
// Returns *wasm_dto.InterpretResponse which is always nil in non-WASM builds.
// Returns error when called, as this feature requires WASM support.
func (*InterpreterAdapter) Interpret(_ context.Context, _ *wasm_dto.InterpretRequest) (*wasm_dto.InterpretResponse, error) {
	return nil, errors.New("interpreter not available in non-WASM builds")
}

// WithSymbolLoader is a no-op in non-WASM builds.
//
// Returns InterpreterAdapterOption which has no effect on the adapter.
func WithSymbolLoader(_ wasm_domain.SymbolLoaderPort) InterpreterAdapterOption {
	return func(_ *InterpreterAdapter) {}
}

// WithInterpreterFactory is a no-op in non-WASM builds.
//
// Returns InterpreterAdapterOption which has no effect.
func WithInterpreterFactory(_ wasm_domain.InterpreterFactoryPort) InterpreterAdapterOption {
	return func(_ *InterpreterAdapter) {}
}
