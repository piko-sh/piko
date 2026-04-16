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

//go:build js && wasm

package runtime

import (
	"errors"

	"piko.sh/piko/internal/templater/templater_dto"
)

// errSearchNotAvailable is returned when search is called in a WASM build
// where search functionality is not supported.
var errSearchNotAvailable = errors.New("search functionality is not available in WASM")

// SearchCollection is not available in WASM builds.
//
// Returns []SearchResult[T] which is always nil in WASM builds.
// Returns error which always indicates search is unavailable.
func SearchCollection[T any](
	_ *templater_dto.RequestData,
	_ string,
	_ string,
	_ ...SearchOption,
) ([]SearchResult[T], error) {
	return nil, errSearchNotAvailable
}

// QuickSearch is not available in WASM builds.
//
// Returns []T which is always nil in WASM builds.
// Returns error when called, as search is unavailable.
func QuickSearch[T any](_ *templater_dto.RequestData, _ string, _ string) ([]T, error) {
	return nil, errSearchNotAvailable
}
