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

package render_dto

// ComponentMetadata holds static information about a component type,
// including its JavaScript and CSS dependencies.
type ComponentMetadata struct {
	// TagName is the HTML tag name used to reference this component.
	TagName string

	// BaseJSPath is the URL path to the component's base JavaScript module.
	BaseJSPath string

	// DefaultCSS is the default CSS styling for the component.
	DefaultCSS string

	// SRIHash is the Subresource Integrity hash for the component's JS module.
	// Empty when SRI is disabled or the hash has not been computed.
	SRIHash string

	// RequiredModules lists the module paths this component needs.
	RequiredModules []string
}
