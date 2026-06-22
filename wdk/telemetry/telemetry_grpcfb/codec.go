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

package telemetry_grpcfb

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/encoding"
)

const (
	// CodecName is the gRPC content-subtype identifying the telemetry FlatBuffers codec.
	CodecName = "x-piko-telemetry-fb"
)

var (
	// ErrNotMessage is returned when a value handed to the codec does not implement the
	// FlatBuffers wire-message contract.
	ErrNotMessage = errors.New("telemetry_grpcfb: value is not a flatbuffers message")

	// ErrMalformedBuffer is returned when decoding recovers from a panic triggered by a
	// malformed buffer that slipped past the structural verifier.
	ErrMalformedBuffer = errors.New("telemetry_grpcfb: malformed buffer")
)

// fbMessage is implemented by every wire message (fb.go). Unmarshal MUST verify untrusted
// input (verify.go) before exposing any field.
type fbMessage interface {
	// Marshal serialises the wire message to its FlatBuffers byte encoding.
	//
	// Returns []byte which is the encoded message.
	// Returns error which is non-nil when encoding fails.
	Marshal() ([]byte, error)

	// Unmarshal decodes data into the wire message after verifying untrusted input.
	//
	// Takes data ([]byte) which is the untrusted FlatBuffers frame to decode.
	//
	// Returns error which is non-nil when the frame is invalid.
	Unmarshal(data []byte) error
}

// Codec is a gRPC codec that puts telemetry FlatBuffers on the wire.
//
// Decoding is hardened:
//   - a size cap + structural verifier run inside each message's Unmarshal, and
//   - a recover() backstop here converts any residual panic into an error so a malformed
//     buffer can never crash the process (defence in depth, not the primary defence,
//     which is the verifier).
type Codec struct{}

// Name reports the gRPC content-subtype this codec registers under (CodecName).
//
// Returns string which is the registered content-subtype (CodecName).
func (Codec) Name() string { return CodecName }

// Marshal serialises a FlatBuffers wire message to bytes. v must implement fbMessage.
//
// Takes v (any) which must implement fbMessage to be encodable.
//
// Returns []byte which is the encoded message.
// Returns error which is non-nil when v is not a message or encoding fails.
func (Codec) Marshal(v any) ([]byte, error) {
	m, ok := v.(fbMessage)
	if !ok {
		return nil, fmt.Errorf("telemetry_grpcfb: cannot marshal %T: %w", v, ErrNotMessage)
	}
	data, err := m.Marshal()
	if err != nil {
		return nil, fmt.Errorf("telemetry_grpcfb: marshal %T: %w", v, err)
	}
	return data, nil
}

// Unmarshal decodes a FlatBuffers wire message into v (which must implement fbMessage). A
// recover() backstop converts any residual panic from a malformed buffer into an error.
//
// Takes data ([]byte) which is the untrusted FlatBuffers frame to decode.
// Takes v (any) which must implement fbMessage to receive the decoded message.
//
// Returns err (error) which is non-nil when v is not a message or decoding fails.
func (Codec) Unmarshal(data []byte, v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("%w: recovered from %w", ErrMalformedBuffer, e)
			} else {
				err = fmt.Errorf("%w: recovered from %v", ErrMalformedBuffer, r)
			}
		}
	}()
	m, ok := v.(fbMessage)
	if !ok {
		return fmt.Errorf("telemetry_grpcfb: cannot unmarshal into %T: %w", v, ErrNotMessage)
	}
	if err := m.Unmarshal(data); err != nil {
		return fmt.Errorf("telemetry_grpcfb: unmarshal into %T: %w", v, err)
	}
	return nil
}

func init() { encoding.RegisterCodec(Codec{}) }
