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

package orchestrator_adapters

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestNewCompilerExecutor(t *testing.T) {
	t.Parallel()

	executor := NewCompilerExecutor(nil, nil)
	require.NotNil(t, executor)

	ce, ok := executor.(*compilerExecutor)
	require.True(t, ok)
	assert.Nil(t, ce.registryService)
	assert.Nil(t, ce.capabilityService)
}

func TestExecutorNameArtefactCompiler(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "artefact.compiler", ExecutorNameArtefactCompiler)
}

func TestGetString_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("valid string returns value", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"name": "hello"}
		value, err := getString(payload, "name")
		require.NoError(t, err)
		assert.Equal(t, "hello", value)
	})

	t.Run("missing key returns error", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{}
		_, err := getString(payload, "name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required key 'name'")
	})

	t.Run("nil payload returns error", func(t *testing.T) {
		t.Parallel()
		_, err := getString(nil, "name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required key")
	})

	t.Run("integer value returns error", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"name": 123}
		_, err := getString(payload, "name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or empty")
	})

	t.Run("bool value returns error", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"name": true}
		_, err := getString(payload, "name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or empty")
	})

	t.Run("nil value returns error", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"name": nil}
		_, err := getString(payload, "name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or empty")
	})

	t.Run("empty string returns error", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"name": ""}
		_, err := getString(payload, "name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or empty")
	})
}

func TestParseCompilerPayload_MissingFields(t *testing.T) {
	t.Parallel()

	t.Run("missing sourceVariantID", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"artefactID":         "art-1",
			"desiredProfileName": "thumb",
			"capabilityToRun":    "resize",
			"taskID":             "task-1",
		}
		_, err := parseCompilerPayload(payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourceVariantID")
	})

	t.Run("missing desiredProfileName", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"artefactID":      "art-1",
			"sourceVariantID": "var-1",
			"capabilityToRun": "resize",
			"taskID":          "task-1",
		}
		_, err := parseCompilerPayload(payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "desiredProfileName")
	})

	t.Run("missing capabilityToRun", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"artefactID":         "art-1",
			"sourceVariantID":    "var-1",
			"desiredProfileName": "thumb",
			"taskID":             "task-1",
		}
		_, err := parseCompilerPayload(payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "capabilityToRun")
	})

	t.Run("empty payload", func(t *testing.T) {
		t.Parallel()
		_, err := parseCompilerPayload(map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "artefactID")
	})
}

func TestParseCompilerPayload_WithAllParams(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"artefactID":         "art-123",
		"sourceVariantID":    "var-456",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-789",
		"capabilityParams":   map[string]string{"width": "100", "height": "200"},
	}

	p, err := parseCompilerPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "art-123", p.ArtefactID)
	assert.Equal(t, "var-456", p.SourceVariantID)
	assert.Equal(t, "thumb", p.DesiredProfileName)
	assert.Equal(t, "resize", p.CapabilityToRun)
	assert.Equal(t, "task-789", p.TaskID)
	assert.Equal(t, "100", p.CapabilityParams["width"])
	assert.Equal(t, "200", p.CapabilityParams["height"])
}

func TestParseCapabilityParams_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("empty map of strings", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"capabilityParams": map[string]string{},
		}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("empty map of any", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"capabilityParams": map[string]any{},
		}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("map with mixed types keeps only strings", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"capabilityParams": map[string]any{
				"width":   "100",
				"count":   42,
				"enabled": true,
				"ratio":   3.14,
				"empty":   "",
			},
		}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Equal(t, "100", result["width"])
		assert.Equal(t, "", result["empty"])
		assert.NotContains(t, result, "count")
		assert.NotContains(t, result, "enabled")
		assert.NotContains(t, result, "ratio")
	})

	t.Run("integer type is invalid", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"capabilityParams": 42}
		_, err := parseCapabilityParams(payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
	})

	t.Run("bool type is invalid", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"capabilityParams": true}
		_, err := parseCapabilityParams(payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
	})

	t.Run("slice type is invalid", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"capabilityParams": []string{"a", "b"}}
		_, err := parseCapabilityParams(payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
	})
}

func TestFindVariantByID_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("first element", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "v1", SizeBytes: 100},
			{VariantID: "v2", SizeBytes: 200},
		}
		v := findVariantByID(variants, "v1")
		require.NotNil(t, v)
		assert.Equal(t, "v1", v.VariantID)
		assert.Equal(t, int64(100), v.SizeBytes)
	})

	t.Run("last element", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "v1"},
			{VariantID: "v2"},
			{VariantID: "v3", SizeBytes: 300},
		}
		v := findVariantByID(variants, "v3")
		require.NotNil(t, v)
		assert.Equal(t, "v3", v.VariantID)
		assert.Equal(t, int64(300), v.SizeBytes)
	})

	t.Run("returns pointer to slice element not copy", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "v1", SizeBytes: 100},
		}
		v := findVariantByID(variants, "v1")
		require.NotNil(t, v)
		v.SizeBytes = 999
		assert.Equal(t, int64(999), variants[0].SizeBytes)
	})

	t.Run("single element found", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "only"},
		}
		v := findVariantByID(variants, "only")
		require.NotNil(t, v)
		assert.Equal(t, "only", v.VariantID)
	})

	t.Run("single element not found", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "only"},
		}
		v := findVariantByID(variants, "other")
		assert.Nil(t, v)
	})
}

func TestReadCounter_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("multiple reads accumulate", func(t *testing.T) {
		t.Parallel()
		data := []byte("hello world, this is a longer string for testing")
		rc := &readCounter{Reader: bytes.NewReader(data)}

		buffer := make([]byte, 5)
		total := int64(0)
		for {
			n, err := rc.Read(buffer)
			total += int64(n)
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
		}
		assert.Equal(t, total, rc.Count)
		assert.Equal(t, int64(len(data)), rc.Count)
	})

	t.Run("empty reader", func(t *testing.T) {
		t.Parallel()
		rc := &readCounter{Reader: bytes.NewReader([]byte{})}
		buffer := make([]byte, 10)
		_, err := rc.Read(buffer)
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, int64(0), rc.Count)
	})

	t.Run("read exactly one byte at a time", func(t *testing.T) {
		t.Parallel()
		data := []byte("abc")
		rc := &readCounter{Reader: bytes.NewReader(data)}
		buffer := make([]byte, 1)

		n, err := rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, int64(1), rc.Count)
		assert.Equal(t, byte('a'), buffer[0])

		n, err = rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, int64(2), rc.Count)
		assert.Equal(t, byte('b'), buffer[0])

		n, err = rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, int64(3), rc.Count)
		assert.Equal(t, byte('c'), buffer[0])
	})

	t.Run("starts at zero count", func(t *testing.T) {
		t.Parallel()
		rc := &readCounter{Reader: bytes.NewReader([]byte("x"))}
		assert.Equal(t, int64(0), rc.Count)
	})
}

func TestRecordCompilationSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	startTime := time.Now().Add(-100 * time.Millisecond)

	variant := &registry_dto.Variant{
		VariantID:  "thumb",
		StorageKey: "generated/image_abc12345.webp",
		SizeBytes:  12345,
	}

	result, err := recordCompilationSuccess(ctx, startTime, variant)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "SUCCESS", result["status"])
	assert.Equal(t, "thumb", result["variantId"])
	assert.Equal(t, "generated/image_abc12345.webp", result["storageKey"])
	assert.Equal(t, int64(12345), result["sizeBytes"])
}

func TestRecordCompilationSuccess_ZeroSizeVariant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	startTime := time.Now()

	variant := &registry_dto.Variant{
		VariantID:  "empty",
		StorageKey: "generated/empty.bin",
		SizeBytes:  0,
	}

	result, err := recordCompilationSuccess(ctx, startTime, variant)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", result["status"])
	assert.Equal(t, "empty", result["variantId"])
	assert.Equal(t, int64(0), result["sizeBytes"])
}

func TestCompilerPayload_Fields(t *testing.T) {
	t.Parallel()

	p := &compilerPayload{
		ArtefactID:         "art-1",
		SourceVariantID:    "source",
		DesiredProfileName: "thumb",
		CapabilityToRun:    "resize",
		TaskID:             "task-1",
		CapabilityParams:   map[string]string{"width": "100"},
	}

	assert.Equal(t, "art-1", p.ArtefactID)
	assert.Equal(t, "source", p.SourceVariantID)
	assert.Equal(t, "thumb", p.DesiredProfileName)
	assert.Equal(t, "resize", p.CapabilityToRun)
	assert.Equal(t, "task-1", p.TaskID)
	assert.Equal(t, "100", p.CapabilityParams["width"])
}
