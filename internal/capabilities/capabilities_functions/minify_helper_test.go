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

package capabilities_functions

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
)

func TestCreateMinifyCapability(t *testing.T) {
	t.Parallel()

	testMinifier := minify.New()
	testMinifier.AddFunc("application/javascript", js.Minify)

	t.Run("should return a non-nil function", func(t *testing.T) {
		t.Parallel()
		config := minifyConfig{
			minifier:    testMinifier,
			spanName:    "TestMinify",
			mimeType:    "application/javascript",
			contentType: "JavaScript",
		}
		capabilityFunction := createMinifyCapability(config)
		require.NotNil(t, capabilityFunction)
	})

	t.Run("should minify JavaScript content", func(t *testing.T) {
		t.Parallel()
		config := minifyConfig{
			minifier:    testMinifier,
			spanName:    "TestMinifyJS",
			mimeType:    "application/javascript",
			contentType: "JavaScript",
		}
		capabilityFunction := createMinifyCapability(config)

		input := strings.NewReader("var  x  =  1 ;")
		stream, err := capabilityFunction(context.Background(), input, nil)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Equal(t, "var x=1", string(output))
	})

	t.Run("should handle empty input", func(t *testing.T) {
		t.Parallel()
		config := minifyConfig{
			minifier:    testMinifier,
			spanName:    "TestMinifyEmpty",
			mimeType:    "application/javascript",
			contentType: "JavaScript",
		}
		capabilityFunction := createMinifyCapability(config)

		input := strings.NewReader("")
		stream, err := capabilityFunction(context.Background(), input, nil)
		require.NoError(t, err)
		require.NotNil(t, stream)

		output, err := io.ReadAll(stream)
		require.NoError(t, err)
		assert.Empty(t, output)
	})

	t.Run("should fail on cancelled context", func(t *testing.T) {
		t.Parallel()
		config := minifyConfig{
			minifier:    testMinifier,
			spanName:    "TestMinifyCancelledCtx",
			mimeType:    "application/javascript",
			contentType: "JavaScript",
		}
		capabilityFunction := createMinifyCapability(config)

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		result, err := capabilityFunction(ctx, strings.NewReader("var x = 1;"), nil)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should fail on expired context", func(t *testing.T) {
		t.Parallel()
		config := minifyConfig{
			minifier:    testMinifier,
			spanName:    "TestMinifyExpiredCtx",
			mimeType:    "application/javascript",
			contentType: "JavaScript",
		}
		capabilityFunction := createMinifyCapability(config)

		ctx, cancel := context.WithTimeoutCause(context.Background(), -1*time.Second, fmt.Errorf("test: simulating expired deadline"))
		defer cancel()

		result, err := capabilityFunction(ctx, strings.NewReader("var x = 1;"), nil)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}
