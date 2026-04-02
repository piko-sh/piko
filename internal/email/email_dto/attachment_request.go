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

package email_dto

// EmailAssetRequest represents a declarative request from a rendered email
// template for an asset to be fetched and attached to the final email.
//
// This struct is populated during PML rendering when <pml-img> components
// encounter local asset references (e.g., "assets/logo.png"). The email
// builder uses these requests to query the registry, retrieve the appropriate
// transformed variant, and attach it with the specified Content-ID for CID
// embedding.
type EmailAssetRequest struct {
	// SourcePath is the original path to the asset in the project
	// (e.g., "assets/images/logo.png"). It is used to query the registry
	// for the corresponding artefact.
	SourcePath string

	// Profile specifies which transformation profile or variant to use
	// (e.g., "email-outlook", "email-default"). The registry returns the
	// variant with matching profile metadata.
	Profile string

	// Density is the requested pixel density for the asset variant (e.g., "x1",
	// "x2", "x3"). When responsive images use densities, this selects the highest
	// density variant for optimal display in email clients.
	Density string

	// CID is the Content-ID used for MIME embedding in emails. HTML templates
	// reference this value using the cid: scheme (e.g., <img src="cid:logo_abc123">).
	CID string

	// Width is the requested width for the asset variant in pixels; 0 means use
	// the profile default. Set it to select a specific width variant when
	// responsive images are used.
	Width int
}
