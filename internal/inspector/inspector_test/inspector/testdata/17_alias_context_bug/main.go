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

package main

import (
	// Here is the conflict: we import 'models' with a different alias.
	// The bug would cause the inspector to incorrectly use `dbmodels` when
	// inspecting types inside the `services` or `layer2` packages.
	dbmodels "testproject_alias_bug/models"

	"testproject_alias_bug/generics"
	"testproject_alias_bug/layer1"
	"testproject_alias_bug/services"
)

// Response uses types from all other packages to ensure the inspector
// loads the entire dependency tree.
type Response struct {
	// This field is defined in `main`, so using `dbmodels` is correct here.
	MainUser dbmodels.User

	// This field's type is defined in a different package, triggering the test.
	NestedInfo layer1.Layer1Response

	// A generic type to test resolution of type arguments.
	GenericBox generics.Box[dbmodels.User]

	// A field to ensure the services package is loaded.
	Service services.UserService
}

// Dummy function to make the file a valid main package.
func main() {}
