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

package fields

type Text string

func (t Text) String() string {
	return string(t)
}

type Wrapper[T any] struct {
	Inner T `json:"inner"`
}

func (w Wrapper[T]) Get() T {
	return w.Inner
}

type Container[T any] struct {
	Data T `json:"data"`
}

func (c Container[T]) Get() T {
	return c.Data
}

type Ref[T any] struct {
	ID   string `json:"id"`
	Item *T     `json:"item,omitempty"`
}

func (r Ref[T]) Get() T {
	if r.Item != nil {
		return *r.Item
	}
	var zero T
	return zero
}

func (r Ref[T]) HasItem() bool {
	return r.Item != nil
}
