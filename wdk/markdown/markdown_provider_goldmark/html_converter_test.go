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
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTMLConverter_Singleton(t *testing.T) {
	t.Run("SafeConverterIsSingleton", func(t *testing.T) {
		ResetConverters()

		conv1 := getSafeConverter()
		conv2 := getSafeConverter()

		assert.Same(t, conv1, conv2, "Should return the same instance")
	})

	t.Run("UnsafeConverterIsSingleton", func(t *testing.T) {
		ResetConverters()

		conv1 := getUnsafeConverter()
		conv2 := getUnsafeConverter()

		assert.Same(t, conv1, conv2, "Should return the same instance")
	})

	t.Run("SafeAndUnsafeAreDifferent", func(t *testing.T) {
		ResetConverters()

		safe := getSafeConverter()
		unsafe := getUnsafeConverter()

		assert.NotSame(t, safe, unsafe, "Safe and unsafe converters should be different instances")
	})
}

func TestHTMLConverter_ThreadSafety(t *testing.T) {
	t.Run("ConcurrentSafeAccess", func(t *testing.T) {
		ResetConverters()

		var wg sync.WaitGroup
		converters := make([]any, 100)

		for i := range 100 {
			index := i
			wg.Go(func() {
				converters[index] = getSafeConverter()
			})
		}

		wg.Wait()

		first := converters[0]
		for i := 1; i < 100; i++ {
			assert.Same(t, first, converters[i], "All goroutines should get the same instance")
		}
	})

	t.Run("ConcurrentUnsafeAccess", func(t *testing.T) {
		ResetConverters()

		var wg sync.WaitGroup
		converters := make([]any, 100)

		for i := range 100 {
			index := i
			wg.Go(func() {
				converters[index] = getUnsafeConverter()
			})
		}

		wg.Wait()

		first := converters[0]
		for i := 1; i < 100; i++ {
			assert.Same(t, first, converters[i], "All goroutines should get the same instance")
		}
	})
}

func TestHTMLConverter_Reset(t *testing.T) {
	t.Run("ResetCreatesFreshInstances", func(t *testing.T) {

		safe1 := getSafeConverter()
		unsafe1 := getUnsafeConverter()

		ResetConverters()

		safe2 := getSafeConverter()
		unsafe2 := getUnsafeConverter()

		assert.NotSame(t, safe1, safe2, "Safe converter should be a new instance after reset")
		assert.NotSame(t, unsafe1, unsafe2, "Unsafe converter should be a new instance after reset")
	})
}

func TestHTMLConverterOption_WithUnsafe(t *testing.T) {
	t.Run("WithUnsafeSetsFlag", func(t *testing.T) {
		opts := &HTMLConverterOptions{}

		WithUnsafe()(opts)

		assert.True(t, opts.Unsafe, "WithUnsafe should set Unsafe to true")
	})

	t.Run("DefaultIsSafe", func(t *testing.T) {
		opts := &HTMLConverterOptions{}

		assert.False(t, opts.Unsafe, "Default should be safe (Unsafe=false)")
	})
}

func TestToHTML_SafeMode(t *testing.T) {
	ResetConverters()

	t.Run("OmitsRawHTML", func(t *testing.T) {
		input := `Hello <script>alert('xss')</script> World`
		result, err := ToHTML(context.Background(), input)
		require.NoError(t, err)

		assert.NotContains(t, result, "<script>")
		assert.Contains(t, result, "<!-- raw HTML omitted -->")
		assert.Contains(t, result, "Hello")
		assert.Contains(t, result, "World")
	})

	t.Run("OmitsIframe", func(t *testing.T) {
		input := `Before <iframe src="evil.com"></iframe> After`
		result, err := ToHTML(context.Background(), input)
		require.NoError(t, err)

		assert.NotContains(t, result, "<iframe")
		assert.Contains(t, result, "<!-- raw HTML omitted -->")
		assert.Contains(t, result, "Before")
		assert.Contains(t, result, "After")
	})
}

func TestToHTML_UnsafeMode(t *testing.T) {
	ResetConverters()

	t.Run("PreservesRawHTML", func(t *testing.T) {
		input := `Hello <span class="highlight">styled</span> World`
		result, err := ToHTML(context.Background(), input, WithUnsafe())
		require.NoError(t, err)

		assert.Contains(t, result, `<span class="highlight">styled</span>`)
		assert.Contains(t, result, "Hello")
		assert.Contains(t, result, "World")
	})

	t.Run("PreservesScript", func(t *testing.T) {
		input := `Before <script>console.log('test')</script> After`
		result, err := ToHTML(context.Background(), input, WithUnsafe())
		require.NoError(t, err)

		assert.Contains(t, result, "<script>")
		assert.Contains(t, result, "console.log")
	})
}

func TestToHTMLBytes_SafeMode(t *testing.T) {
	ResetConverters()

	t.Run("OmitsRawHTML", func(t *testing.T) {
		input := []byte(`Hello <script>alert('xss')</script> World`)
		result, err := ToHTMLBytes(context.Background(), input)
		require.NoError(t, err)

		assert.NotContains(t, string(result), "<script>")
		assert.Contains(t, string(result), "<!-- raw HTML omitted -->")
	})
}

func TestToHTMLBytes_UnsafeMode(t *testing.T) {
	ResetConverters()

	t.Run("PreservesRawHTML", func(t *testing.T) {
		input := []byte(`Hello <span class="highlight">styled</span> World`)
		result, err := ToHTMLBytes(context.Background(), input, WithUnsafe())
		require.NoError(t, err)

		assert.Contains(t, string(result), `<span class="highlight">styled</span>`)
	})
}

func TestToHTML_InputLimits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		input   string
		options []HTMLConverterOption
	}{
		{name: "a document within the limits converts", input: "> quoted\n- item"},
		{name: "a document over the size limit is rejected", input: "too long", options: []HTMLConverterOption{WithHTMLMaxInputSize(4)}, wantErr: ErrInputTooLarge},
		{name: "a document over the default size is rejected", input: strings.Repeat("a", defaultMaxInputSize+1), wantErr: ErrInputTooLarge},
		{name: "deep nesting is rejected", input: strings.Repeat(">", 100_000) + " deep", wantErr: ErrNestingTooDeep},
		{name: "nesting past a custom limit is rejected", input: "> > > x", options: []HTMLConverterOption{WithHTMLMaxNestingDepth(2)}, wantErr: ErrNestingTooDeep},
		{name: "limits below one keep the defaults", input: "> > > x", options: []HTMLConverterOption{WithHTMLMaxInputSize(0), WithHTMLMaxNestingDepth(-1)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := ToHTML(context.Background(), tc.input, tc.options...)

			if tc.wantErr == nil {
				require.NoError(t, err)
				assert.Contains(t, result, "<blockquote>")
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			assert.Empty(t, result)
		})
	}
}

func TestToHTMLBytes_CancelledContext(t *testing.T) {
	t.Parallel()

	errStopped := errors.New("render abandoned")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errStopped)

	result, err := ToHTMLBytes(ctx, []byte("# Title"))

	require.ErrorIs(t, err, errStopped)
	assert.Nil(t, result)
}
