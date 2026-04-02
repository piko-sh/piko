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

package annotator_dto

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// StaticAssetDependency represents a single, statically-known asset referenced
// from a component, along with its required transformations.
type StaticAssetDependency struct {
	// SourcePath is the path to the asset as written in the src attribute
	// (e.g. "lib/images/hero.jpg").
	SourcePath string

	// AssetType is the type of asset, taken from the HTML tag (e.g. "svg", "img").
	AssetType string

	// TransformationParams maps parameter names to their values, taken from
	// component attributes. For example: {"width": "300", "density": "2x"}.
	TransformationParams map[string]string

	// OriginComponentPath is the path of the .pk file that contains this
	// reference, for diagnostics.
	OriginComponentPath string

	// Location specifies where the tag appears in the source file, used for
	// error messages.
	Location ast_domain.Location
}

// FinalAssetDependency holds the details of an asset that has been processed
// and is ready for use, including its source location and any transformations
// applied.
type FinalAssetDependency struct {
	// TransformationParams holds key-value pairs for asset transformation
	// settings, such as image quality or video encoding options.
	TransformationParams map[string][]string

	// SourcePath is the file path to the source asset.
	SourcePath string

	// AssetType is the kind of asset, such as "img" for images.
	AssetType string
}
