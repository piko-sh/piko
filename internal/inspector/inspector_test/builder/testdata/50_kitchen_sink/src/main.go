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
	"context"
	"io"

	"testcase_50_kitchen_sink/models"

	"github.com/google/uuid"
)

// Event is another simple generic type.
type Event[T any] struct {
	Payload T
}

// Report is the "kitchen sink" type that combines multiple complex features.
type Report[T any] struct {
	// 1. Embed an instantiated generic from another package.
	models.Auditable[uuid.UUID]

	// 2. A complex field: map of slices of pointers to another generic type.
	EventsByTopic map[string][]*Event[T]

	// 3. A field that is a pointer to a standard library interface.
	Output *io.Writer
}

// 4. A method with a complex signature using the generic parameter and external types.
func (r *Report[T]) Summarise(ctx context.Context) (T, error) {
	var zero T
	return zero, nil
}
