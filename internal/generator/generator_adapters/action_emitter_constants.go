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

package generator_adapters

const (
	// actionPikoPackagePath is the import path for the Piko package.
	actionPikoPackagePath = "piko.sh/piko"

	// actionJSONPackagePath is the import path for the JSON abstraction package.
	actionJSONPackagePath = "piko.sh/piko/wdk/json"

	// actionJSONPackageAlias is the import alias for the JSON package.
	actionJSONPackageAlias = "pikojson"

	// actionReflectPackagePath is the import path for the reflect package.
	actionReflectPackagePath = "reflect"

	// actionLoggerPackagePath is the import path for the Piko logger package.
	actionLoggerPackagePath = "piko.sh/piko/wdk/logger"

	// actionMultipartPackagePath is the import path for the multipart package.
	actionMultipartPackagePath = "mime/multipart"

	// actionBinderPackagePath is the import path for the Piko binder package.
	actionBinderPackagePath = "piko.sh/piko/wdk/binder"

	// actionBinderPackageAlias is the import alias for the binder package,
	// avoiding conflict with the internal binder package.
	actionBinderPackageAlias = "pikobinder"

	// actionGeneratedPackageName is the package name for generated action files.
	actionGeneratedPackageName = "actions"
)
