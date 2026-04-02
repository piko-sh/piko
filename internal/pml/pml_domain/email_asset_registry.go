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

package pml_domain

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
	"piko.sh/piko/internal/email/email_dto"
)

// EmailAssetRegistry collects requests for static assets to be embedded in
// emails during PML rendering. When <pml-img> components reference local
// assets, they register a request here, which is later fulfilled by the email
// builder querying the registry and attaching the transformed image data.
//
// This registry is created for email rendering contexts and attached to the
// TransformationContext so that all components can access it during the render
// tree walk.
type EmailAssetRegistry struct {
	// Requests holds all asset requests gathered during rendering.
	Requests []*email_dto.EmailAssetRequest
}

// NewEmailAssetRegistry creates a new, empty asset registry for email rendering.
//
// Returns *EmailAssetRegistry which is an empty registry ready to collect
// asset requests.
func NewEmailAssetRegistry() *EmailAssetRegistry {
	return &EmailAssetRegistry{
		Requests: make([]*email_dto.EmailAssetRequest, 0),
	}
}

// RegisterAsset records an asset to be fetched and embedded in the email.
//
// It returns a unique Content-ID (CID) for use in HTML as <img src="cid:...">.
// The same asset, profile, width, and density combination always returns the
// same CID.
//
// Takes sourcePath (string) which is the original path to the asset.
// Takes profile (string) which is the transformation profile to use.
// Takes width (int) which is the requested width, or 0 for profile default.
// Takes density (string) which is the pixel density, or empty for default.
//
// Returns string which is the unique CID without the "cid:" prefix.
func (r *EmailAssetRegistry) RegisterAsset(sourcePath, profile string, width int, density string) string {
	cid := r.generateCID(sourcePath, profile, width, density)

	for _, request := range r.Requests {
		if request.CID == cid {
			return cid
		}
	}

	r.Requests = append(r.Requests, &email_dto.EmailAssetRequest{
		SourcePath: sourcePath,
		Profile:    profile,
		Width:      width,
		Density:    density,
		CID:        cid,
	})

	return cid
}

// generateCID creates a unique, deterministic Content-ID from the given asset
// properties using xxhash. The CID is unique within an email and stable for
// the same inputs.
//
// Takes sourcePath (string) which specifies the path to the asset file.
// Takes profile (string) which identifies the image profile to use.
// Takes width (int) which specifies the target width in pixels.
// Takes density (string) which indicates the pixel density variant.
//
// Returns string which is the generated Content-ID.
func (*EmailAssetRegistry) generateCID(sourcePath, profile string, width int, density string) string {
	input := fmt.Sprintf("%s|%s|%d|%s", sourcePath, profile, width, density)
	hash := xxhash.Sum64String(input)
	return fmt.Sprintf("asset_%016x", hash)
}
