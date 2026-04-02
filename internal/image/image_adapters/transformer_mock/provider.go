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
	"io"
	"sync"

	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
)

const (
	// mockImageWidth is the width in pixels returned by the mock image provider.
	mockImageWidth = 800

	// mockImageHeight is the mock image height in pixels.
	mockImageHeight = 600
)

// TransformCall records the parameters passed to a single call of the Transform
// method, letting tests inspect what the service is asking the transformer to
// do.
type TransformCall struct {
	// InputData is a copy of the raw image bytes passed to the transformer.
	InputData []byte

	// Specification is the transformation that was requested.
	Specification image_dto.TransformationSpec
}

// Provider is a thread-safe, in-memory implementation of TransformerPort.
// It implements image_domain.TransformerPort and media.ImageTransformerPort
// for unit and integration testing, allowing call inspection and simulation
// of both successful transformations and errors.
type Provider struct {
	// errToReturn is the error that Transform and Transcode return; nil means
	// success.
	errToReturn error

	// mimeTypeToReturn is the MIME type to return when outputDataToReturn is set.
	mimeTypeToReturn string

	// transformCalls stores all calls made to Transform for test assertions.
	transformCalls []TransformCall

	// outputDataToReturn holds the data to write to the output writer
	// when set; nil means no data is returned.
	outputDataToReturn []byte

	// mu protects concurrent access to mock state.
	mu sync.RWMutex
}

var _ image_domain.TransformerPort = (*Provider)(nil)

// NewProvider creates a new mock image transformer for use in tests.
//
// Returns *Provider which is ready for use.
func NewProvider() *Provider {
	return &Provider{
		errToReturn:        nil,
		mimeTypeToReturn:   "",
		transformCalls:     make([]TransformCall, 0),
		outputDataToReturn: nil,
		mu:                 sync.RWMutex{},
	}
}

// Transform simulates an image transformation. It records the call
// and returns either a configured error/result or, by default, passes
// the input through to output.
//
// Takes input (io.Reader) which provides the source image data.
// Takes output (io.Writer) which receives the transformed image data.
// Takes spec (image_dto.TransformationSpec) which defines the
// transformation to apply.
//
// Returns string which is the MIME type of the output.
// Returns error when reading input or writing output fails.
//
// Safe for concurrent use; protected by a mutex.
func (p *Provider) Transform(_ context.Context, input io.Reader, output io.Writer, spec image_dto.TransformationSpec) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inBytes, err := io.ReadAll(input)
	if err != nil {
		return "", err
	}
	dataCopy := make([]byte, len(inBytes))
	copy(dataCopy, inBytes)
	p.transformCalls = append(p.transformCalls, TransformCall{InputData: dataCopy, Specification: spec})

	if p.errToReturn != nil {
		return "", p.errToReturn
	}

	if p.outputDataToReturn != nil {
		if _, err := output.Write(p.outputDataToReturn); err != nil {
			return "", err
		}
		return p.mimeTypeToReturn, nil
	}

	if _, err := output.Write(dataCopy); err != nil {
		return "", err
	}
	return "image/mock", nil
}

// GetSupportedFormats returns a full set of formats for testing purposes.
// The mock supports all formats to avoid breaking tests.
//
// Returns []string which contains all supported image format extensions.
func (*Provider) GetSupportedFormats() []string {
	return []string{"jpeg", "jpg", "png", "webp", "avif", "gif"}
}

// GetSupportedModifiers returns a full set of modifiers for testing purposes.
// The mock supports all modifiers to avoid breaking tests.
//
// Returns []string which contains all available image transformation modifiers.
func (*Provider) GetSupportedModifiers() []string {
	return []string{
		"greyscale", "blur", "sharpen", "rotate", "flip",
		"brightness", "contrast", "saturation",
		"hue", "tint", "gravity", "focus", "radius",
	}
}

// SetError configures the mock to return the specified error on the next
// Transform call.
//
// Takes err (error) which is the error to return from subsequent calls.
//
// Safe for concurrent use.
func (p *Provider) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errToReturn = err
}

// SetTransformResult configures the mock to return a specific successful result
// on the next Transform call.
//
// Takes data ([]byte) which is the transformed output to return.
// Takes mimeType (string) which is the MIME type of the output.
//
// Safe for concurrent use.
func (p *Provider) SetTransformResult(data []byte, mimeType string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outputDataToReturn = data
	p.mimeTypeToReturn = mimeType
}

// GetTransformCalls returns a copy of all recorded calls made to the
// Transform method. Returning a copy prevents the test from modifying the
// mock's internal state.
//
// Returns []TransformCall which contains all recorded Transform calls.
//
// Safe for concurrent use.
func (p *Provider) GetTransformCalls() []TransformCall {
	p.mu.RLock()
	defer p.mu.RUnlock()

	callsCopy := make([]TransformCall, len(p.transformCalls))
	copy(callsCopy, p.transformCalls)
	return callsCopy
}

// Reset clears all recorded calls and configured return values, preparing
// the mock for a new test case.
//
// Safe for concurrent use.
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.transformCalls = make([]TransformCall, 0)
	p.outputDataToReturn = nil
	p.mimeTypeToReturn = ""
	p.errToReturn = nil
}

// GetDimensions returns mock dimensions for testing purposes.
// Always returns 800x600 unless configured otherwise.
//
// Takes ctx (context.Context) which is ignored in the mock.
// Takes input (io.Reader) which is ignored in the mock.
//
// Returns width (int) which is mockImageWidth (800).
// Returns height (int) which is mockImageHeight (600).
// Returns error which is always nil in the mock.
func (*Provider) GetDimensions(_ context.Context, _ io.Reader) (width int, height int, err error) {
	return mockImageWidth, mockImageHeight, nil
}
