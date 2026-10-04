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

package markdown_provider_goldmark

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// HTMLConverterOptions holds configuration for HTML conversion.
type HTMLConverterOptions struct {
	// MaxInputSize is the largest markdown document, in bytes, that is converted; larger
	// documents fail with ErrInputTooLarge. Defaults to 16 MiB.
	MaxInputSize int

	// MaxNestingDepth is the most blockquote and list markers that may open on one line;
	// deeper lines fail with ErrNestingTooDeep. Defaults to 1000.
	MaxNestingDepth int

	// Unsafe enables rendering of raw HTML embedded in markdown. When false (default), raw
	// HTML is stripped for security.
	Unsafe bool
}

// HTMLConverterOption configures HTML conversion behaviour.
type HTMLConverterOption func(*HTMLConverterOptions)

var (
	// getSafeConverter returns the lazily initialised safe Goldmark converter with GFM and
	// footnote extensions enabled but raw HTML stripped.
	getSafeConverter = newSafeConverterOnce()

	// getUnsafeConverter returns the lazily initialised unsafe Goldmark converter with raw
	// HTML rendering enabled.
	getUnsafeConverter = newUnsafeConverterOnce()
)

// WithUnsafe enables rendering of raw HTML embedded in markdown.
//
// Returns HTMLConverterOption which configures the converter to allow raw HTML.
//
// WARNING: Only use with fully trusted content. Enabling this with user-generated content
// exposes your application to XSS attacks.
func WithUnsafe() HTMLConverterOption {
	return func(opts *HTMLConverterOptions) {
		opts.Unsafe = true
	}
}

// WithHTMLMaxInputSize sets the largest markdown document, in bytes, that ToHTML and
// ToHTMLBytes convert; larger documents fail with ErrInputTooLarge. The default is 16
// MiB.
//
// Takes size (int) which is the limit in bytes; values below one keep the default.
//
// Returns HTMLConverterOption which applies the limit.
func WithHTMLMaxInputSize(size int) HTMLConverterOption {
	return func(opts *HTMLConverterOptions) {
		if size > 0 {
			opts.MaxInputSize = size
		}
	}
}

// WithHTMLMaxNestingDepth sets the most blockquote and list markers that may open on a
// single line; deeper lines fail with ErrNestingTooDeep before conversion. The default is
// 1000.
//
// Takes depth (int) which is the limit; values below one keep the default.
//
// Returns HTMLConverterOption which applies the limit.
func WithHTMLMaxNestingDepth(depth int) HTMLConverterOption {
	return func(opts *HTMLConverterOptions) {
		if depth > 0 {
			opts.MaxNestingDepth = depth
		}
	}
}

// ToHTML converts a Markdown string to an HTML string.
//
// By default, raw HTML in the Markdown is removed for security. Use WithUnsafe() to allow
// raw HTML rendering for trusted content only. Documents beyond the size or nesting
// limits are rejected before conversion.
//
// Takes markdown (string) which is the Markdown content to convert.
// Takes opts (...HTMLConverterOption) which sets options for the conversion.
//
// Returns string which is the rendered HTML output.
// Returns error which wraps ErrInputTooLarge or ErrNestingTooDeep when markdown is beyond
// the limits, the cancellation cause when ctx is done, or the conversion failure.
func ToHTML(ctx context.Context, markdown string, opts ...HTMLConverterOption) (string, error) {
	rendered, err := ToHTMLBytes(ctx, []byte(markdown), opts...)
	if err != nil {
		return "", err
	}
	return string(rendered), nil
}

// ToHTMLBytes converts markdown bytes to HTML bytes.
//
// This variant avoids string conversion overhead when working with byte slices.
//
// By default, raw HTML embedded in the markdown is stripped for security. Use
// WithUnsafe() to enable raw HTML rendering for trusted content only. Documents beyond
// the size or nesting limits are rejected before conversion.
//
// Takes markdown ([]byte) which is the markdown content to convert.
// Takes opts (...HTMLConverterOption) which configures the conversion.
//
// Returns []byte which is the rendered HTML output.
// Returns error which wraps ErrInputTooLarge or ErrNestingTooDeep when markdown is beyond
// the limits, the cancellation cause when ctx is done, or the conversion failure.
func ToHTMLBytes(ctx context.Context, markdown []byte, opts ...HTMLConverterOption) ([]byte, error) {
	options := newHTMLConverterOptions(opts)
	if ctx.Err() != nil {
		return nil, fmt.Errorf("converting markdown to HTML: %w", context.Cause(ctx))
	}
	if err := checkInputLimits(markdown, options.MaxInputSize, options.MaxNestingDepth); err != nil {
		return nil, err
	}

	var converter goldmark.Markdown
	if options.Unsafe {
		converter = getUnsafeConverter()
	} else {
		converter = getSafeConverter()
	}

	var buffer bytes.Buffer
	if err := converter.Convert(markdown, &buffer); err != nil {
		return nil, fmt.Errorf("converting markdown to HTML: %w", err)
	}

	return buffer.Bytes(), nil
}

// ResetConverters resets the singleton converters to their initial state. This is only
// intended for testing purposes.
func ResetConverters() {
	getSafeConverter = newSafeConverterOnce()
	getUnsafeConverter = newUnsafeConverterOnce()
}

// newSafeConverterOnce creates a fresh sync.OnceValue for the safe converter.
//
// Returns func() goldmark.Markdown which lazily initialises the safe Goldmark instance on
// first call.
func newSafeConverterOnce() func() goldmark.Markdown {
	return sync.OnceValue(func() goldmark.Markdown {
		return goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				extension.Footnote,
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
				parser.WithAttribute(),
			),
		)
	})
}

// newUnsafeConverterOnce creates a fresh sync.OnceValue for the unsafe converter.
//
// Returns func() goldmark.Markdown which lazily initialises the unsafe Goldmark instance
// with raw HTML rendering on first call.
func newUnsafeConverterOnce() func() goldmark.Markdown {
	return sync.OnceValue(func() goldmark.Markdown {
		return goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				extension.Footnote,
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
				parser.WithAttribute(),
			),
			goldmark.WithRendererOptions(
				html.WithUnsafe(),
			),
		)
	})
}

// newHTMLConverterOptions applies opts over the default HTML conversion settings.
//
// Takes opts ([]HTMLConverterOption) which adjust the defaults.
//
// Returns HTMLConverterOptions which holds the resulting settings.
func newHTMLConverterOptions(opts []HTMLConverterOption) HTMLConverterOptions {
	options := HTMLConverterOptions{
		MaxInputSize:    defaultMaxInputSize,
		MaxNestingDepth: defaultMaxNestingDepth,
		Unsafe:          false,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}
