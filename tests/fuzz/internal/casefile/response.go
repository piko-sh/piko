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

package casefile

// ResponseContract selects authored head fields and ordered semantic landmarks.
//
// Multiple values preserve duplicate head tags. Ordinary describes the complete list of
// unmarked controls and links when supplied. Preserve an explicit empty list in saved
// JSON since it asserts their absence. Elements select marked nodes.
type ResponseContract struct {
	// AppText holds visible application text when checked.
	AppText *string `json:",omitempty"`

	// Head holds selected head fields and duplicate values.
	Head map[string][]string `json:",omitempty"`

	// Ordinary holds the complete ordinary control and link sequence when supplied.
	Ordinary []PageElement `json:",omitzero"`

	// Elements holds the selected marked landmarks.
	Elements []PageElement `json:",omitempty"`
}

// PageElement holds a semantic landmark and its public attributes.
type PageElement struct {
	// Attributes holds the element's public attributes.
	Attributes map[string]string `json:",omitempty"`

	// Key identifies the marked landmark.
	Key string

	// Tag is the landmark's element name.
	Tag string

	// Name is the landmark's accessible name.
	Name string
}

// ModeResponse permits a difference on one specific request.
type ModeResponse struct {
	// Semantic holds the selected semantic response fields.
	Semantic ResponseContract

	// Status is the HTTP response status.
	Status int
}

// ResponseAsset specifies a linked resource's delivered bytes.
type ResponseAsset struct {
	// MediaType is the delivered media type.
	MediaType string

	// SHA256 identifies the delivered bytes.
	SHA256 string

	// Revalidate enables conditional-request checks.
	Revalidate bool `json:",omitempty"`
}
