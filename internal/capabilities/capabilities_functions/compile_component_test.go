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
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
)

type mockCompilerService struct {
	compileSFCBytesFunction func(ctx context.Context, sourceIdentifier string, rawSFC []byte) (*compiler_dto.CompiledArtefact, error)
}

func (m *mockCompilerService) CompileSingle(_ context.Context, _ string) (*compiler_dto.CompiledArtefact, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCompilerService) CompileSFCBytes(ctx context.Context, sourceIdentifier string, rawSFC []byte) (*compiler_dto.CompiledArtefact, error) {
	if m.compileSFCBytesFunction != nil {
		return m.compileSFCBytesFunction(ctx, sourceIdentifier, rawSFC)
	}
	return nil, errors.New("no mock configured")
}

func TestExtractSourcePath(t *testing.T) {
	t.Parallel()

	span := noop.Span{}

	t.Run("should return source path when present", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"sourcePath": "/path/to/component.vue"}
		path, err := extractSourcePath(context.Background(), params, span)
		require.NoError(t, err)
		assert.Equal(t, "/path/to/component.vue", path)
	})

	t.Run("should return error when sourcePath is missing", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		_, err := extractSourcePath(context.Background(), params, span)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourcePath")
	})

	t.Run("should return error when sourcePath is empty string", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"sourcePath": ""}
		_, err := extractSourcePath(context.Background(), params, span)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourcePath")
	})

	t.Run("should return error when params is nil", func(t *testing.T) {
		t.Parallel()
		_, err := extractSourcePath(context.Background(), nil, span)
		require.Error(t, err)
	})
}

func TestExtractOutput(t *testing.T) {
	t.Parallel()

	span := noop.Span{}

	t.Run("should return JS content when entrypoint exists", func(t *testing.T) {
		t.Parallel()
		artefact := &compiler_dto.CompiledArtefact{
			BaseJSPath: "output.js",
			Files: map[string]string{
				"output.js": "console.log('hello');",
			},
		}
		output, err := extractOutput(context.Background(), artefact, "test.vue", span)
		require.NoError(t, err)
		assert.Equal(t, "console.log('hello');", output)
	})

	t.Run("should return error when entrypoint file is missing", func(t *testing.T) {
		t.Parallel()
		artefact := &compiler_dto.CompiledArtefact{
			BaseJSPath: "missing.js",
			Files: map[string]string{
				"output.js": "console.log('hello');",
			},
		}
		_, err := extractOutput(context.Background(), artefact, "test.vue", span)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing.js")
		assert.Contains(t, err.Error(), "test.vue")
	})

	t.Run("should return error when files map is empty", func(t *testing.T) {
		t.Parallel()
		artefact := &compiler_dto.CompiledArtefact{
			BaseJSPath: "output.js",
			Files:      map[string]string{},
		}
		_, err := extractOutput(context.Background(), artefact, "test.vue", span)
		require.Error(t, err)
	})

	t.Run("should return empty string content when entrypoint exists with empty content", func(t *testing.T) {
		t.Parallel()
		artefact := &compiler_dto.CompiledArtefact{
			BaseJSPath: "output.js",
			Files: map[string]string{
				"output.js": "",
			},
		}
		output, err := extractOutput(context.Background(), artefact, "test.vue", span)
		require.NoError(t, err)
		assert.Empty(t, output)
	})
}

func TestReadInputData(t *testing.T) {
	t.Parallel()

	span := noop.Span{}

	t.Run("should read all input data", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("hello world")
		data, err := readInputData(context.Background(), span, input)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(data))
	})

	t.Run("should handle empty input", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("")
		data, err := readInputData(context.Background(), span, input)
		require.NoError(t, err)
		assert.Empty(t, data)
	})

	t.Run("should return error from failing reader", func(t *testing.T) {
		t.Parallel()
		input := &errorReader{Err: io.ErrUnexpectedEOF}
		_, err := readInputData(context.Background(), span, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "component source stream")
	})
}

func TestRecordSuccess(t *testing.T) {
	t.Parallel()

	t.Run("should not panic", func(t *testing.T) {
		t.Parallel()
		span := noop.Span{}
		assert.NotPanics(t, func() {
			recordSuccess(context.Background(), span, "console.log('test');")
		})
	})

	t.Run("should not panic with empty output", func(t *testing.T) {
		t.Parallel()
		span := noop.Span{}
		assert.NotPanics(t, func() {
			recordSuccess(context.Background(), span, "")
		})
	})
}

func TestCompileComponent(t *testing.T) {
	t.Parallel()

	t.Run("should compile successfully", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{
			compileSFCBytesFunction: func(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
				return &compiler_dto.CompiledArtefact{
					BaseJSPath: "output.js",
					Files:      map[string]string{"output.js": "console.log('compiled');"},
				}, nil
			},
		}

		capabilityFunction := CompileComponent(compiler)
		params := capabilities_domain.CapabilityParams{"sourcePath": "/path/to/comp.vue"}
		result, err := capabilityFunction(context.Background(), strings.NewReader("<template>hello</template>"), params)
		require.NoError(t, err)
		require.NotNil(t, result)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, "console.log('compiled');", string(output))
	})

	t.Run("should return error when sourcePath is missing", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{}
		capabilityFunction := CompileComponent(compiler)

		_, err := capabilityFunction(context.Background(), strings.NewReader("content"), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourcePath")
	})

	t.Run("should return error when compiler fails", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{
			compileSFCBytesFunction: func(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
				return nil, errors.New("compilation failed")
			},
		}

		capabilityFunction := CompileComponent(compiler)
		params := capabilities_domain.CapabilityParams{"sourcePath": "/path/to/comp.vue"}
		_, err := capabilityFunction(context.Background(), strings.NewReader("content"), params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compiler service failed")
	})

	t.Run("should return error when entrypoint missing in artefact", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{
			compileSFCBytesFunction: func(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
				return &compiler_dto.CompiledArtefact{
					BaseJSPath: "missing.js",
					Files:      map[string]string{"other.js": "content"},
				}, nil
			},
		}

		capabilityFunction := CompileComponent(compiler)
		params := capabilities_domain.CapabilityParams{"sourcePath": "/path/to/comp.vue"}
		_, err := capabilityFunction(context.Background(), strings.NewReader("content"), params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing.js")
	})

	t.Run("should return error when context is cancelled", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{}
		capabilityFunction := CompileComponent(compiler)

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		params := capabilities_domain.CapabilityParams{"sourcePath": "/path/to/comp.vue"}
		_, err := capabilityFunction(ctx, strings.NewReader("content"), params)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should return error when input reader fails", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{}
		capabilityFunction := CompileComponent(compiler)

		params := capabilities_domain.CapabilityParams{"sourcePath": "/path/to/comp.vue"}
		_, err := capabilityFunction(context.Background(), &errorReader{Err: io.ErrUnexpectedEOF}, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "component source stream")
	})
}

func TestCompileSource(t *testing.T) {
	t.Parallel()

	t.Run("should compile successfully", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{
			compileSFCBytesFunction: func(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
				return &compiler_dto.CompiledArtefact{
					BaseJSPath: "out.js",
					Files:      map[string]string{"out.js": "compiled"},
				}, nil
			},
		}

		artefact, err := compileSource(context.Background(), compiler, "test.vue", []byte("source"))
		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "out.js", artefact.BaseJSPath)
	})

	t.Run("should return error when compiler fails", func(t *testing.T) {
		t.Parallel()
		compiler := &mockCompilerService{
			compileSFCBytesFunction: func(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
				return nil, errors.New("parse error")
			},
		}

		_, err := compileSource(context.Background(), compiler, "test.vue", []byte("bad source"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compiler service failed")
		assert.Contains(t, err.Error(), "test.vue")
	})
}
