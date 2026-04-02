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

import "reflect"

// EncoderPort defines the contract for encoding and decoding a specific type V.
// Implementations handle conversion between Go values and byte representations.
type EncoderPort[V any] interface {
	// Marshal converts a Go value into its byte form.
	//
	// Takes value (V) which is the Go value to convert.
	//
	// Returns []byte which contains the byte form of the value.
	// Returns error when the value cannot be converted.
	Marshal(value V) ([]byte, error)

	// Unmarshal reconstructs a Go value from its byte representation.
	//
	// Takes data ([]byte) which contains the encoded representation.
	// Takes target (*V) which must be non-nil and point to a value of type V.
	//
	// Returns error when the data is malformed or incompatible with the target.
	Unmarshal(data []byte, target *V) error
}

// AnyEncoder provides a type-agnostic interface for encoding.
//
// All typed encoders can be converted to this interface, so different
// encoders can be stored together in a registry. It links compile-time type
// safety with runtime flexibility.
type AnyEncoder interface {
	// MarshalAny takes a value of any type, checks if it matches the
	// encoder's type, and marshals it.
	//
	// Takes value (any) which is the value to marshal.
	//
	// Returns []byte which contains the encoded data.
	// Returns error when the type is incompatible or marshalling fails.
	MarshalAny(value any) ([]byte, error)

	// UnmarshalAny takes a byte slice and unmarshals it into a zero value of the
	// encoder's target type. The caller must type-assert the returned value.
	UnmarshalAny(data []byte) (any, error)

	// HandlesType returns the reflect.Type that this encoder handles.
	// This lets the registry match values to the correct encoder at runtime.
	//
	// Returns reflect.Type which is the type this encoder can process.
	HandlesType() reflect.Type
}
