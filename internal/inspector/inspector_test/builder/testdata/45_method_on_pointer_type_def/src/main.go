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

// User is a plain struct.
type User struct {
	Name string
}

// UserPtr is a new type definition on a pointer to User.
type UserPtr *User

// GetName is a method defined directly on the UserPtr type.
// The receiver `p` has type UserPtr, which is already a pointer.
func (p UserPtr) GetName() string {
	if p != nil && (*p).Name != "" {
		return (*p).Name
	}
	return "guest"
}
