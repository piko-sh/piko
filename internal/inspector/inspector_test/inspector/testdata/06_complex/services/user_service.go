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

import (
	"context"

	// Internal dependencies
	"testproject_complex/api"
	"testproject_complex/db"

	// External dependency
	"go.opentelemetry.io/otel/trace"
)

// LoginEvent is an embedded struct to test field lookups through embedding.
type LoginEvent struct {
	db.LoginEvent
	TraceID trace.TraceID
}

// UserService defines an interface.
type UserService interface {
	GetUser(ctx context.Context, id int) (*api.User, error)
}

// UserServiceImpl is a concrete implementation of the interface.
type UserServiceImpl struct {
	DB *db.User // A field with a name-colliding type.
}

// GetUser implements the UserService interface.
func (s *UserServiceImpl) GetUser(ctx context.Context, id int) (*api.User, error) {
	return &api.User{ID: "1", Email: "test@example.com"}, nil
}
