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

type Node[T any] struct {
	Value    T          `json:"value"`
	Children []*Node[T] `json:"children,omitempty"`
	Parent   *Node[T]   `json:"parent,omitempty"`
}

func (n Node[T]) GetValue() T {
	return n.Value
}

func (n Node[T]) HasChildren() bool {
	return len(n.Children) > 0
}

func (n Node[T]) ChildCount() int {
	return len(n.Children)
}

func (n Node[T]) FirstChild() *Node[T] {
	if len(n.Children) > 0 {
		return n.Children[0]
	}
	return nil
}

func (n Node[T]) HasParent() bool {
	return n.Parent != nil
}

type LinkedList[T any] struct {
	Value T              `json:"value"`
	Next  *LinkedList[T] `json:"next,omitempty"`
}

func (l LinkedList[T]) GetValue() T {
	return l.Value
}

func (l LinkedList[T]) HasNext() bool {
	return l.Next != nil
}

func (l LinkedList[T]) GetNext() *LinkedList[T] {
	return l.Next
}
