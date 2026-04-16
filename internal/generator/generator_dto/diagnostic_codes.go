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

package generator_dto

const (
	// CodeRenderError indicates the component's Render function returned an
	// error at runtime.
	CodeRenderError = "R001"

	// CodePartialRenderError indicates a partial's Render function returned an
	// error at runtime.
	CodePartialRenderError = "R002"

	// CodeNilPropertyAccess indicates an attempt to access a property on a nil
	// value during template rendering.
	CodeNilPropertyAccess = "R003"

	// CodeNilIndexAccess indicates an attempt to index into a nil slice or map
	// during template rendering.
	CodeNilIndexAccess = "R004"

	// CodeIndexOutOfBounds indicates a slice index was outside the valid range
	// during template rendering.
	CodeIndexOutOfBounds = "R005"

	// CodeEmptyElementTag indicates a <piko:element :is="..."> resolved to an
	// empty string at runtime.
	CodeEmptyElementTag = "R006"

	// CodeRejectedElementTag indicates a <piko:element :is="..."> resolved to
	// a reserved tag name (piko:partial, piko:slot, piko:element).
	CodeRejectedElementTag = "R007"
)
