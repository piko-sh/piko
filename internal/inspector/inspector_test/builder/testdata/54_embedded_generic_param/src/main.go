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

// MyType has a method we expect to be promoted.
type MyType struct{}

func (m MyType) DoWork() string { return "done" }

// Wrapper is a generic type that will be instantiated and embedded.
type Wrapper[T any] struct {
	Value T
}

// Worker is a generic struct with a method.
type Worker[T any] struct {
	Data T
}

func (w Worker[T]) Process() T {
	return w.Data
}

// StringWorker embeds an *instantiated* generic type.
// THIS is the pattern that results in method promotion.
type StringWorker struct {
	Worker[string]
}
