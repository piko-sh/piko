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

import "testcase_67_slice_of_generics/models"

// A local generic type.
type Box[T any] struct {
	Content T
}

// A generic struct that contains a slice of its own type parameter.
// This is the direct replication of the bug.
type GenericResponse[T any] struct {
	Items []T
}

// A concrete struct that uses slices of instantiated generic types.
type Response struct {
	// A slice of a locally defined generic type.
	Boxes []Box[string]

	// A slice of a generic type from another package.
	Stores []models.Store[int]
}
