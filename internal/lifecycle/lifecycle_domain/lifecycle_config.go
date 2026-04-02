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

package lifecycle_domain

// LifecyclePathsConfig holds the resolved path values needed by the lifecycle
// service. All fields are value types; pointer-to-value conversion is performed
// in the bootstrap layer.
type LifecyclePathsConfig struct {
	// BaseDir is the root directory of the website project.
	BaseDir string

	// PagesSourceDir is the directory for page definition files.
	PagesSourceDir string

	// PartialsSourceDir is the directory for partial definition files.
	PartialsSourceDir string

	// ComponentsSourceDir is the directory for component files.
	ComponentsSourceDir string

	// EmailsSourceDir is the directory for email template files.
	EmailsSourceDir string

	// AssetsSourceDir is the directory for asset files.
	AssetsSourceDir string

	// I18nSourceDir is the directory for locale and translation files.
	I18nSourceDir string
}
