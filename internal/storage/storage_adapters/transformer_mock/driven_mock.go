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

package transformer_mock

import (
	"context"
	"fmt"
	"io"
	"strings"

	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// rot13Offset is the shift amount for ROT13 encoding (13 positions).
	rot13Offset = 13

	// alphabetSize is the number of letters in the English alphabet.
	alphabetSize = 26
)

// MockTransformer is a test implementation of StreamTransformerPort.
// It adds "MOCK:<name>:" to data on Transform and removes it on Reverse.
type MockTransformer struct {
	// name identifies this transformer; used in MOCK:<name>: prefixes.
	name string

	// priority is the value returned by the Priority method.
	priority int
}

var (
	_ storage_domain.StreamTransformerPort = (*MockTransformer)(nil)

	_ storage_domain.StreamTransformerPort = (*ConfigurableTransformer)(nil)

	_ storage_domain.StreamTransformerPort = (*ROT13Transformer)(nil)

	_ storage_domain.StreamTransformerPort = (*UppercaseTransformer)(nil)
)

// NewMockTransformer creates a new mock transformer.
//
// Takes name (string) which identifies the transformer.
// Takes priority (int) which sets the transformer's priority order.
//
// Returns *MockTransformer which is the configured mock transformer.
func NewMockTransformer(name string, priority int) *MockTransformer {
	return &MockTransformer{
		name:     name,
		priority: priority,
	}
}

// Name returns the transformer name.
//
// Returns string which is the name of this mock transformer.
func (m *MockTransformer) Name() string {
	return m.name
}

// Type returns the transformer type.
//
// Returns storage_dto.TransformerType which identifies this as a custom
// transformer.
func (*MockTransformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerCustom
}

// Priority returns the execution priority.
//
// Returns int which is the priority value for ordering transformers.
func (m *MockTransformer) Priority() int {
	return m.priority
}

// Transform wraps the input reader to prefix data with "MOCK:<name>:".
//
// Takes input (io.Reader) which provides the data to prefix.
//
// Returns io.Reader which provides the prefixed data stream.
// Returns error which is always nil for this transformer.
func (m *MockTransformer) Transform(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	prefix := fmt.Sprintf("MOCK:%s:", m.name)
	return &prefixReader{
		reader:     input,
		prefix:     []byte(prefix),
		prefixRead: false,
	}, nil
}

// Reverse wraps the input reader to strip the "MOCK:<name>:" prefix.
//
// Takes input (io.Reader) which provides the prefixed data to strip.
//
// Returns io.Reader which provides the data with the prefix removed.
// Returns error which is always nil for this transformer.
func (m *MockTransformer) Reverse(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	prefix := fmt.Sprintf("MOCK:%s:", m.name)
	return &stripPrefixReader{
		prefix:      []byte(prefix),
		reader:      input,
		prefixBuf:   make([]byte, 0, len(prefix)),
		prefixIndex: 0,
		afterPrefix: false,
	}, nil
}

// prefixReader implements io.Reader and adds a prefix before reading from
// the underlying reader.
type prefixReader struct {
	// reader is the source to read from after the prefix has been consumed.
	reader io.Reader

	// prefix is the bytes to add before the underlying reader content.
	prefix []byte

	// prefixRead indicates whether the prefix has already been read.
	prefixRead bool
}

// Read reads data with the prefix prepended.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the underlying reader returns an error.
func (r *prefixReader) Read(p []byte) (n int, err error) {
	if !r.prefixRead {
		n = copy(p, r.prefix)
		r.prefixRead = true
		if n < len(p) {
			n2, err := r.reader.Read(p[n:])
			return n + n2, err
		}
		return n, nil
	}

	return r.reader.Read(p)
}

// stripPrefixReader removes a prefix from the start of a stream.
// It implements io.Reader.
type stripPrefixReader struct {
	// prefix is the byte sequence to strip from the start of the stream.
	prefix []byte

	// reader is the source from which bytes are read.
	reader io.Reader

	// prefixBuf holds the prefix bytes being matched during reading.
	prefixBuf []byte

	// prefixIndex tracks where we are in the prefix during checking.
	prefixIndex int

	// afterPrefix indicates whether the prefix has been read and removed.
	afterPrefix bool
}

// Read reads data with the prefix stripped from the beginning.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the prefix does not match or the underlying
// reader fails.
func (r *stripPrefixReader) Read(p []byte) (n int, err error) {
	if r.afterPrefix {
		return r.reader.Read(p)
	}

	for r.prefixIndex < len(r.prefix) {
		buffer := make([]byte, 1)
		_, err := r.reader.Read(buffer)
		if err != nil {
			return 0, fmt.Errorf("failed to read expected prefix: %w", err)
		}

		if buffer[0] != r.prefix[r.prefixIndex] {
			return 0, fmt.Errorf("prefix mismatch: expected %q, got %q at position %d",
				string(r.prefix), string(buffer[0]), r.prefixIndex)
		}

		r.prefixIndex++
	}

	r.afterPrefix = true
	return r.reader.Read(p)
}

// ConfigurableTransformer is a mock transformer that implements
// StreamTransformerPort with configurable behaviour for testing.
type ConfigurableTransformer struct {
	// transformFunc is a custom transform function; nil uses default behaviour.
	transformFunc func(io.Reader) io.Reader

	// reverseFunc is a custom function for reverse operations; nil uses
	// default behaviour.
	reverseFunc func(io.Reader) io.Reader

	// name is the transformer's name returned by the Name method.
	name string

	// errorMessage is the text returned as an error when shouldError is true.
	errorMessage string

	// priority is the execution order; lower values run first.
	priority int

	// shouldError indicates whether Transform and Reverse should return an error.
	shouldError bool
}

// NewConfigurableTransformer creates a transformer with custom behaviour for
// testing.
//
// Takes name (string) which identifies the transformer.
// Takes priority (int) which sets the transformer's execution order.
//
// Returns *ConfigurableTransformer which is ready for configuration via its
// setter methods.
func NewConfigurableTransformer(name string, priority int) *ConfigurableTransformer {
	return &ConfigurableTransformer{
		transformFunc: nil,
		reverseFunc:   nil,
		name:          name,
		errorMessage:  "",
		priority:      priority,
		shouldError:   false,
	}
}

// Name returns the transformer name.
//
// Returns string which is the transformer's configured name.
func (c *ConfigurableTransformer) Name() string {
	return c.name
}

// Type returns the transformer type.
//
// Returns storage_dto.TransformerType which is always TransformerCustom.
func (*ConfigurableTransformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerCustom
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's priority value.
func (c *ConfigurableTransformer) Priority() int {
	return c.priority
}

// SetTransformFunc sets a custom transform function.
//
// Takes f (func(io.Reader) io.Reader) which transforms input during
// storage operations.
func (c *ConfigurableTransformer) SetTransformFunc(f func(io.Reader) io.Reader) {
	c.transformFunc = f
}

// SetReverseFunc sets a custom reverse function.
//
// Takes f (func(io.Reader) io.Reader) which transforms the reader during
// reverse operations.
func (c *ConfigurableTransformer) SetReverseFunc(f func(io.Reader) io.Reader) {
	c.reverseFunc = f
}

// SetError configures the transformer to return an error.
//
// Takes shouldError (bool) which enables or disables error behaviour.
// Takes message (string) which specifies the error message to return.
func (c *ConfigurableTransformer) SetError(shouldError bool, message string) {
	c.shouldError = shouldError
	c.errorMessage = message
}

// Transform applies the configured transformation or returns an error.
//
// Takes input (io.Reader) which provides the data to transform.
//
// Returns io.Reader which provides the transformed data stream.
// Returns error when shouldError is true.
func (c *ConfigurableTransformer) Transform(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	if c.shouldError {
		return nil, fmt.Errorf("%s", c.errorMessage)
	}
	if c.transformFunc != nil {
		return c.transformFunc(input), nil
	}
	return input, nil
}

// Reverse applies the configured reverse transformation or returns an error.
//
// Takes input (io.Reader) which provides the data to reverse-transform.
//
// Returns io.Reader which provides the reverse-transformed data stream.
// Returns error when shouldError is true.
func (c *ConfigurableTransformer) Reverse(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	if c.shouldError {
		return nil, fmt.Errorf("%s", c.errorMessage)
	}
	if c.reverseFunc != nil {
		return c.reverseFunc(input), nil
	}
	return input, nil
}

// ROT13Transformer is a test transformer that applies ROT13 encoding.
// It implements StreamTransformerPort and is reversible since ROT13 is its
// own inverse.
type ROT13Transformer struct {
	// name is the display name returned by the Name method.
	name string

	// priority is the execution order; lower values run first.
	priority int
}

// NewROT13Transformer creates a new ROT13 transformer.
//
// Takes name (string) which identifies this transformer.
// Takes priority (int) which sets the processing order.
//
// Returns *ROT13Transformer which is the configured transformer.
func NewROT13Transformer(name string, priority int) *ROT13Transformer {
	return &ROT13Transformer{
		name:     name,
		priority: priority,
	}
}

// Name returns the transformer name.
//
// Returns string which is the display name of this transformer.
func (r *ROT13Transformer) Name() string {
	return r.name
}

// Type returns the transformer type.
//
// Returns storage_dto.TransformerType which identifies this as a custom
// transformer.
func (*ROT13Transformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerCustom
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's priority value.
func (r *ROT13Transformer) Priority() int {
	return r.priority
}

// Transform applies ROT13 encoding.
//
// Takes input (io.Reader) which provides the data to encode.
//
// Returns io.Reader which provides the ROT13-encoded data stream.
// Returns error which is always nil for this transformer.
func (*ROT13Transformer) Transform(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	return &rot13Reader{reader: input}, nil
}

// Reverse applies ROT13 decoding (which is the same as encoding).
//
// Takes input (io.Reader) which provides the data to decode.
//
// Returns io.Reader which provides the ROT13-decoded data stream.
// Returns error which is always nil for this transformer.
func (*ROT13Transformer) Reverse(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	return &rot13Reader{reader: input}, nil
}

// rot13Reader wraps an io.Reader and applies ROT13 transformation to data
// read from it.
type rot13Reader struct {
	// reader is the source that provides data for ROT13 transformation.
	reader io.Reader
}

// Read reads data with ROT13 transformation applied.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the underlying reader returns an error.
func (r *rot13Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	for i := range n {
		if p[i] >= 'A' && p[i] <= 'Z' {
			p[i] = 'A' + (p[i]-'A'+rot13Offset)%alphabetSize
		} else if p[i] >= 'a' && p[i] <= 'z' {
			p[i] = 'a' + (p[i]-'a'+rot13Offset)%alphabetSize
		}
	}
	return n, err
}

// UppercaseTransformer implements StreamTransformerPort to convert text to
// uppercase. It is used for testing stream transformation behaviour.
type UppercaseTransformer struct {
	// name is the display name of this transformer.
	name string

	// priority sets the execution order for transformers.
	priority int
}

// NewUppercaseTransformer creates a new uppercase transformer.
//
// Takes name (string) which identifies the transformer.
// Takes priority (int) which sets the processing order.
//
// Returns *UppercaseTransformer which is the configured transformer.
func NewUppercaseTransformer(name string, priority int) *UppercaseTransformer {
	return &UppercaseTransformer{
		name:     name,
		priority: priority,
	}
}

// Name returns the transformer name.
//
// Returns string which is the display name of this transformer.
func (u *UppercaseTransformer) Name() string {
	return u.name
}

// Type returns the transformer type.
//
// Returns storage_dto.TransformerType which identifies this as a custom
// transformer.
func (*UppercaseTransformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerCustom
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's execution order.
func (u *UppercaseTransformer) Priority() int {
	return u.priority
}

// Transform converts to uppercase.
//
// Takes input (io.Reader) which provides the data to convert.
//
// Returns io.Reader which provides the uppercased data stream.
// Returns error which is always nil for this transformer.
func (*UppercaseTransformer) Transform(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	return &caseReader{reader: input, toUpper: true}, nil
}

// Reverse converts to lowercase.
//
// Takes input (io.Reader) which provides the data to convert.
//
// Returns io.Reader which provides the lowercased data stream.
// Returns error which is always nil for this transformer.
func (*UppercaseTransformer) Reverse(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	return &caseReader{reader: input, toUpper: false}, nil
}

// caseReader wraps an io.Reader to convert text to upper or lower case.
type caseReader struct {
	// reader is the source to read from before case conversion.
	reader io.Reader

	// toUpper indicates whether to convert bytes to uppercase; false means lowercase.
	toUpper bool
}

// Read reads data with case transformation applied.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the underlying reader returns an error.
func (r *caseReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	for i := range n {
		if r.toUpper {
			p[i] = byte(strings.ToUpper(string(p[i]))[0])
		} else {
			p[i] = byte(strings.ToLower(string(p[i]))[0])
		}
	}
	return n, err
}
