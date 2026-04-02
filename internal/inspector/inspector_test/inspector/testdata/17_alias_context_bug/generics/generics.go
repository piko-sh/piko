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

package generics

// This package also imports 'models' with its default name.
import "testproject_alias_bug/models"

// A generic container.
type Box[T any] struct {
	Value T
}

// A function returning an instantiated generic. The inspector must resolve
// the type argument `models.User` within this package's context.
func GetUserModelBox() Box[models.User] {
	return Box[models.User]{}
}
