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

package db

import "time"

// User represents the database entity for a user.
// It has different fields from `api.User`. This tests name collision resolution.
type User struct {
	ID        int
	Email     string
	Password  string // This field is not in the API model.
	CreatedAt time.Time
}

// LoginEvent represents a record of a user login.
type LoginEvent struct {
	UserID    int
	Timestamp time.Time
}
