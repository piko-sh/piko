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

package diagnostic_docs

import (
	"piko.sh/piko/internal/generator/generator_dto"
)

var (
	// renderingCodes explains the R0xx codes, reported by generated code while a page
	// renders.
	renderingCodes = []Entry{
		{
			Code: generator_dto.CodeRenderError, Name: "RenderError", Family: FamilyRendering,
			Title:      "Render returned an error",
			Severities: []Severity{SeverityError},
			Summary:    "A page or component's Render function returned an error.",
			Detail:     "The error is recorded and logged with the page's other render diagnostics, and the request is answered with an error page.",
			Example:    nil,
			Fix:        "Handle the cause in Render, or return a piko error with a safe message for the visitor.",
			Unused:     false,
		},
		{
			Code: generator_dto.CodePartialRenderError, Name: "PartialRenderError", Family: FamilyRendering,
			Title:      "Partial Render returned an error",
			Severities: []Severity{SeverityError},
			Summary:    "A partial's Render function returned an error.",
			Detail:     "The partial's error is recorded with the page's render diagnostics and logged.",
			Example:    nil,
			Fix:        "Handle the cause in the partial's Render function.",
			Unused:     false,
		},
		{
			Code: generator_dto.CodeNilPropertyAccess, Name: "NilPropertyAccess", Family: FamilyRendering,
			Title:      "Field read through nil",
			Severities: []Severity{SeverityWarning},
			Summary:    "A template read a field through a pointer that was nil.",
			Detail: "Templates guard field access through pointers that may be nil. Instead of panicking, the expression gives the field's zero value and this warning is " +
				"recorded. The build reports the risk ahead of time as T101.",
			Example: nil,
			Fix:     "Check the value before using it, such as p-if=\"state.User != nil\", or make sure Render sets it.",
			Unused:  false,
		},
		{
			Code: generator_dto.CodeNilIndexAccess, Name: "NilIndexAccess", Family: FamilyRendering,
			Title:      "Index into nil",
			Severities: []Severity{SeverityWarning},
			Summary:    "A template indexed a slice or map that was nil.",
			Detail:     "Instead of panicking, the expression gives the zero value and this warning is recorded.",
			Example:    nil,
			Fix:        "Make sure the slice or map is set before it is indexed, or check it first.",
			Unused:     false,
		},
		{
			Code: generator_dto.CodeIndexOutOfBounds, Name: "IndexOutOfBounds", Family: FamilyRendering,
			Title:      "Index out of range",
			Severities: []Severity{SeverityWarning},
			Summary:    "A template indexed a slice outside its length.",
			Detail:     "Instead of panicking, the expression gives the zero value and this warning is recorded.",
			Example:    nil,
			Fix:        "Check the length first, such as p-if=\"len(state.Items) > 2\", or loop over the slice with p-for.",
			Unused:     false,
		},
		{
			Code: generator_dto.CodeEmptyElementTag, Name: "EmptyElementTag", Family: FamilyRendering,
			Title:      "<piko:element> has no tag",
			Severities: []Severity{SeverityError},
			Summary:    "A <piko:element :is> expression resolved to an empty tag name.",
			Detail:     "The element cannot be rendered without a tag, so it is skipped.",
			Example:    nil,
			Fix:        "Make sure the is expression always gives a tag name.",
			Unused:     false,
		},
		{
			Code: generator_dto.CodeRejectedElementTag, Name: "RejectedElementTag", Family: FamilyRendering,
			Title:      "<piko:element> tag not allowed",
			Severities: []Severity{SeverityError},
			Summary:    "A <piko:element :is> expression resolved to a reserved tag.",
			Detail:     "<piko:element> cannot render piko:partial, piko:slot or piko:element, so the element is skipped.",
			Example:    nil,
			Fix:        "Use a plain HTML tag name, or use <piko:partial> for partials.",
			Unused:     false,
		},
	}
)
