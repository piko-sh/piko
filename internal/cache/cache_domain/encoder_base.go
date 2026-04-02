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

package cache_domain

import (
	"fmt"
	"reflect"
)

// marshalFunc is a function type that converts a value of type V into bytes.
type marshalFunc[V any] func(value V) ([]byte, error)

// unmarshalFunc is a function type that converts bytes into a value of type V.
type unmarshalFunc[V any] func(data []byte, target *V) error

// baseEncoder is a generic encoder that implements AnyEncoder.
// It wraps marshal and unmarshal functions to create an encoder for a type.
type baseEncoder[V any] struct {
	// marshalFunction converts values to bytes for encoding.
	marshalFunction marshalFunc[V]

	// unmarshalFunction converts raw bytes into a value of type V.
	unmarshalFunction unmarshalFunc[V]

	// handlesType is the reflect.Type that this encoder can handle.
	handlesType reflect.Type
}

var _ EncoderPort[any] = (*baseEncoder[any])(nil)
var _ AnyEncoder = (*baseEncoder[any])(nil)

// Marshal converts a typed value to bytes using the configured marshal function.
//
// Takes value (V) which is the typed value to convert.
//
// Returns []byte which contains the encoded representation.
// Returns error when the marshal function fails.
func (s *baseEncoder[V]) Marshal(value V) ([]byte, error) {
	return s.marshalFunction(value)
}

// Unmarshal reconstructs a typed value from bytes using the configured
// unmarshal function.
//
// Takes data ([]byte) which contains the encoded bytes to decode.
// Takes target (*V) which receives the decoded value.
//
// Returns error when the unmarshal function fails to decode the data.
func (s *baseEncoder[V]) Unmarshal(data []byte, target *V) error {
	return s.unmarshalFunction(data, target)
}

// MarshalAny performs runtime type checking before delegating to the typed
// Marshal method.
//
// Takes value (any) which is the value to encode.
//
// Returns []byte which contains the encoded data.
// Returns error when the value's type does not match this encoder's target
// type or when encoding fails.
func (s *baseEncoder[V]) MarshalAny(value any) ([]byte, error) {
	v, ok := value.(V)
	if !ok {
		return nil, fmt.Errorf("encoder for type %s cannot handle value of type %T", s.handlesType.String(), value)
	}
	return s.Marshal(v)
}

// UnmarshalAny creates a zero value of type V, unmarshals the data into it,
// and returns it as any. The caller must type-assert the result back to V.
//
// Takes data ([]byte) which contains the encoded data to unmarshal.
//
// Returns any which is the unmarshalled value of type V.
// Returns error when unmarshalling fails.
func (s *baseEncoder[V]) UnmarshalAny(data []byte) (any, error) {
	var target V
	if err := s.Unmarshal(data, &target); err != nil {
		return nil, fmt.Errorf("unmarshalling value of type %s: %w", s.handlesType.String(), err)
	}
	return target, nil
}

// HandlesType returns the reflect.Type that this encoder operates on.
// This is used by the registry to match values to encoders at runtime.
//
// Returns reflect.Type which identifies the type this encoder handles.
func (s *baseEncoder[V]) HandlesType() reflect.Type {
	return s.handlesType
}

// NewEncoder creates a new encoder from marshal/unmarshal function pairs.
// This is the primary factory function for creating custom encoders.
//
// Takes marshal (marshalFunc[V]) which converts values to bytes.
// Takes unmarshal (unmarshalFunc[V]) which converts bytes back to values.
//
// Returns EncoderPort[V] which is the configured encoder ready for use.
func NewEncoder[V any](marshal marshalFunc[V], unmarshal unmarshalFunc[V]) EncoderPort[V] {
	var v V
	return &baseEncoder[V]{
		marshalFunction:   marshal,
		unmarshalFunction: unmarshal,
		handlesType:       reflect.TypeOf(v),
	}
}
