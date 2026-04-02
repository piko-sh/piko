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

package config_domain

import (
	"os"
	"reflect"
)

// applyDefaults sets default values on the struct fields within ptr.
//
// Takes ptr (any) which is a pointer to the struct to populate with defaults.
// Takes ctx (*LoadContext) which provides the loading context and options.
//
// Returns error when walking the struct fields fails.
func (l *Loader) applyDefaults(ptr any, ctx *LoadContext) error {
	state := &walkState{
		processor: processDefaults,
		ctx:       ctx,
		keyPrefix: "",
		source:    sourceDefault,
	}
	return l.walk(reflect.ValueOf(ptr), state)
}

// processDefaults sets a struct field's value from its "default" struct
// tag, expanding any environment variables in the tag value.
//
// Takes field (*reflect.StructField) which provides the struct tag to
// read the default value from.
// Takes value (reflect.Value) which is the field to set.
//
// Returns error when setting the field value fails.
func processDefaults(field *reflect.StructField, value reflect.Value, _, _ string) error {
	tagValue, ok := field.Tag.Lookup("default")
	if !ok {
		return nil
	}
	expandedValue := os.ExpandEnv(tagValue)
	return setField(value, expandedValue, field.Tag)
}
