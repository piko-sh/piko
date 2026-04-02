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
	"fmt"
	"reflect"
	"strconv"
)

// processorFunc is the signature for functions that handle a field during a
// walk. It is the main contract for any action done by the recursive walker.
type processorFunc func(field *reflect.StructField, value reflect.Value, prefix, keyPath string) error

// fieldPathSetter is an interface for types that need to know their location
// in the config. Types can use this path for logging or audit purposes.
type fieldPathSetter interface {
	// SetFieldPath sets the field path for use in error messages.
	//
	// Takes path (string) which is the location within a structure.
	SetFieldPath(path string)
}

// walkState tracks the current position and context during struct traversal.
type walkState struct {
	// processor handles field processing; nil skips processing.
	processor processorFunc

	// ctx tracks which fields have been set and their sources; nil skips tracking.
	ctx *LoadContext

	// keyPrefix is the dot-separated path to the current field in nested structs.
	keyPrefix string

	// source identifies which loading pass set this field's value.
	source string
}

// walk traverses a struct value and processes each field that can be set.
//
// Takes value (reflect.Value) which is the struct to traverse.
// Takes state (*walkState) which tracks the current path and context.
//
// Returns error when field processing fails.
func (l *Loader) walk(value reflect.Value, state *walkState) error {
	value = reflect.Indirect(value)
	if value.Kind() != reflect.Struct {
		return nil
	}
	for fieldType, fieldValue := range value.Fields() {
		if !fieldValue.CanSet() {
			continue
		}

		currentKey := fieldType.Name
		if state.keyPrefix != "" {
			currentKey = state.keyPrefix + "." + currentKey
		}

		subState := &walkState{
			keyPrefix: currentKey,
			ctx:       state.ctx,
			source:    state.source,
			processor: state.processor,
		}

		if err := l.dispatchAndProcessField(fieldValue, &fieldType, subState); err != nil {
			return fmt.Errorf("field %q: %w", currentKey, err)
		}
	}
	return nil
}

// dispatchAndProcessField routes field processing to the appropriate handler
// based on type and updates source tracking when values change.
//
// Takes fieldValue (reflect.Value) which is the field to process.
// Takes fieldType (*reflect.StructField) which describes the field metadata.
// Takes state (*walkState) which holds the current traversal context.
//
// Returns error when field processing fails.
func (l *Loader) dispatchAndProcessField(fieldValue reflect.Value, fieldType *reflect.StructField, state *walkState) error {
	if state.processor == nil {
		return nil
	}

	var before reflect.Value
	if state.ctx != nil && fieldValue.IsValid() {
		before = reflect.ValueOf(fieldValue.Interface())
	}

	var err error
	if isUnmarshaler(fieldValue) {
		err = processUnmarshalerField(fieldType, fieldValue, state)
	} else {
		switch fieldValue.Kind() {
		case reflect.Struct:
			err = l.processStructField(fieldValue, state)
		case reflect.Pointer:
			err = l.processPointerField(fieldType, fieldValue, state)
		default:
			err = processPrimitiveField(fieldType, fieldValue, state)
		}
	}

	if err != nil {
		return fmt.Errorf("dispatching field at %q: %w", state.keyPrefix, err)
	}

	if state.source != "" && state.ctx != nil && before.IsValid() && fieldValue.IsValid() && !reflect.DeepEqual(before.Interface(), fieldValue.Interface()) {
		state.ctx.FieldSources[state.keyPrefix] = state.source
	}

	return nil
}

// processStructField processes a struct field by walking its address.
//
// Takes fieldValue (reflect.Value) which is the field to process.
// Takes state (*walkState) which tracks the current walk context.
//
// Returns error when walking the field address fails.
func (l *Loader) processStructField(fieldValue reflect.Value, state *walkState) error {
	return l.walk(fieldValue.Addr(), state)
}

// processPointerField handles a pointer-typed struct field during config
// walking.
//
// Takes fieldType (*reflect.StructField) which provides the field metadata
// including struct tags.
// Takes fieldValue (reflect.Value) which is the pointer field to process.
// Takes state (*walkState) which holds the current traversal context.
//
// Returns error when walking the pointed-to struct fails or when processing
// the field fails.
func (l *Loader) processPointerField(fieldType *reflect.StructField, fieldValue reflect.Value, state *walkState) error {
	if fieldValue.Type().Elem().Kind() == reflect.Struct {
		if fieldValue.IsNil() {
			if noinit, _ := strconv.ParseBool(fieldType.Tag.Get("noinit")); noinit {
				return nil
			}
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}
		return l.walk(fieldValue, state)
	}
	return state.processor(fieldType, fieldValue, "", state.keyPrefix)
}

// processUnmarshalerField handles a field that uses a custom unmarshaler.
//
// Takes fieldType (*reflect.StructField) which describes the struct field.
// Takes fieldValue (reflect.Value) which is the value to process.
// Takes state (*walkState) which tracks the current walk context.
//
// Returns error when the processor returns an error.
func processUnmarshalerField(fieldType *reflect.StructField, fieldValue reflect.Value, state *walkState) error {
	if err := state.processor(fieldType, fieldValue, "", state.keyPrefix); err != nil {
		return fmt.Errorf("processing unmarshaler field at %q: %w", state.keyPrefix, err)
	}

	if fieldValue.CanAddr() {
		if setter, ok := reflect.TypeAssert[fieldPathSetter](fieldValue.Addr()); ok {
			setter.SetFieldPath(state.keyPrefix)
		}
	}

	return nil
}

// processPrimitiveField handles a primitive struct field by calling the state
// processor.
//
// Takes fieldType (*reflect.StructField) which provides the field metadata.
// Takes fieldValue (reflect.Value) which holds the field value to process.
// Takes state (*walkState) which contains the processor and key prefix.
//
// Returns error when the processor returns an error.
func processPrimitiveField(fieldType *reflect.StructField, fieldValue reflect.Value, state *walkState) error {
	return state.processor(fieldType, fieldValue, "", state.keyPrefix)
}
