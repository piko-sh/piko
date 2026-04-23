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

// Package interp_link exposes the sentinel type used to bridge Go
// generic functions into Piko's interpreter.
//
// Go monomorphises generics at compile time: the machine code for
// GetData[Post] only exists if the Go compiler saw that instantiation.
// When Piko's interpreter executes a .pk file that calls a generic
// function with a user-defined type, the specialised code is absent.
// A source-level //piko:link directive declares a non-generic sibling
// that accepts the instantiated types as prepended reflect.Type
// arguments; the extract tool wraps the sibling in a [LinkedFunction]
// during symbol generation, and the interpreter dispatches through it.
//
// This package is imported by generated gen_*.go symbol files, so it
// lives in a public path rather than under internal/.
package interp_link
