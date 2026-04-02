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

//go:build safe || (js && wasm)

package binder

import (
	"reflect"
)

// convertAndSetDirect sets a field value using reflection in safe mode.
// This delegates to convertAndSet without unsafe pointer operations,
// which is slower but works correctly under all conditions.
//
// Takes structVal (reflect.Value) which is the struct to modify.
// Takes value (string) which is the value to convert and set.
// Takes fullPath (string) which identifies the field for error messages.
// Takes fi (*fieldInfo) which describes the target field.
//
// Returns error when the value cannot be converted or set.
func (b *ASTBinder) convertAndSetDirect(structVal reflect.Value, value string, fullPath string, fi *fieldInfo) error {
	field := structVal.Field(fi.Index[0])
	return b.convertAndSet(field, value, fullPath, fi)
}
