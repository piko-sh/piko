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

package services

// We import 'models' with its default package name.
import "testproject_alias_bug/models"

// GetUserFunc is a package-level function. The inspector must resolve its
// return type using THIS file's import context.
func GetUserFunc() *models.User {
	return nil
}

// UserService demonstrates a method on a type.
type UserService struct{}

// GetUser is a method whose return type must also be resolved
// using the 'services' package's import context.
func (s *UserService) GetUser() *models.User {
	return nil
}
