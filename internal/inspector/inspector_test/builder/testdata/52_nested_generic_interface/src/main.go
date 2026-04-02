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

// Getter is a generic interface. Any type that implements this
// can be used as a constraint.
type Getter[T any] interface {
	Get() T
}

// Container is a generic struct that holds any type 'G'
// as long as 'G' satisfies the Getter[E] interface.
type Container[E any, G Getter[E]] struct {
	Source G
}

// Box is a concrete generic type that will be used to implement Getter.
type Box[T any] struct {
	Value T
}

// Box[T] now implements Getter[T].
func (b Box[T]) Get() T {
	return b.Value
}

// Report uses the complex Container type.
type Report struct {
	// This is a valid instantiation:
	// - E is 'string'
	// - G is 'Box[string]'
	// This is legal because Box[string] implements the interface Getter[string].
	StringSource Container[string, Box[string]]
}
