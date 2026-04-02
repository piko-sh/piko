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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func TestGetString(t *testing.T) {
	t.Parallel()

	t.Run("valid string", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{"key": "value"}
		value, err := getString(payload, "key")
		require.NoError(t, err)
		assert.Equal(t, "value", value)
	})

	t.Run("missing key", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{}
		_, err := getString(payload, "missing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required key")
	})

	t.Run("non-string value", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{"key": 42}
		_, err := getString(payload, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or empty")
	})

	t.Run("empty string value", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{"key": ""}
		_, err := getString(payload, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or empty")
	})
}

func TestParseCompilerPayload(t *testing.T) {
	t.Parallel()

	validPayload := map[string]any{
		"artefactID":         "art-123",
		"sourceVariantID":    "var-456",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-789",
	}

	t.Run("valid payload", func(t *testing.T) {
		t.Parallel()

		p, err := parseCompilerPayload(validPayload)
		require.NoError(t, err)
		assert.Equal(t, "art-123", p.ArtefactID)
		assert.Equal(t, "var-456", p.SourceVariantID)
		assert.Equal(t, "thumb", p.DesiredProfileName)
		assert.Equal(t, "resize", p.CapabilityToRun)
		assert.Equal(t, "task-789", p.TaskID)
		assert.NotNil(t, p.CapabilityParams)
	})

	t.Run("missing artefactID", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{
			"sourceVariantID":    "var-456",
			"desiredProfileName": "thumb",
			"capabilityToRun":    "resize",
			"taskID":             "task-789",
		}
		_, err := parseCompilerPayload(payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "artefactID")
	})

	t.Run("missing taskID", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{
			"artefactID":         "art-123",
			"sourceVariantID":    "var-456",
			"desiredProfileName": "thumb",
			"capabilityToRun":    "resize",
		}
		_, err := parseCompilerPayload(payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "taskID")
	})

	t.Run("with capability params", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{
			"artefactID":         "art-123",
			"sourceVariantID":    "var-456",
			"desiredProfileName": "thumb",
			"capabilityToRun":    "resize",
			"taskID":             "task-789",
			"capabilityParams":   map[string]any{"width": "100", "height": "100"},
		}
		p, err := parseCompilerPayload(payload)
		require.NoError(t, err)
		assert.Equal(t, "100", p.CapabilityParams["width"])
		assert.Equal(t, "100", p.CapabilityParams["height"])
	})
}

func TestParseCapabilityParams(t *testing.T) {
	t.Parallel()

	t.Run("missing key returns empty map", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("nil value returns empty map", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{"capabilityParams": nil}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("map[string]string", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{
			"capabilityParams": map[string]string{"width": "100"},
		}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Equal(t, "100", result["width"])
	})

	t.Run("map[string]any with strings", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{
			"capabilityParams": map[string]any{"width": "100", "quality": "high"},
		}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Equal(t, "100", result["width"])
		assert.Equal(t, "high", result["quality"])
	})

	t.Run("map[string]any skips non-strings", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{
			"capabilityParams": map[string]any{"width": "100", "count": 42},
		}
		result, err := parseCapabilityParams(payload)
		require.NoError(t, err)
		assert.Equal(t, "100", result["width"])
		assert.NotContains(t, result, "count")
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()

		payload := map[string]any{"capabilityParams": "not-a-map"}
		_, err := parseCapabilityParams(payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
	})
}

func TestFindVariantByID(t *testing.T) {
	t.Parallel()

	variants := []registry_dto.Variant{
		{VariantID: "v1"},
		{VariantID: "v2"},
		{VariantID: "v3"},
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		v := findVariantByID(variants, "v2")
		require.NotNil(t, v)
		assert.Equal(t, "v2", v.VariantID)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		v := findVariantByID(variants, "v99")
		assert.Nil(t, v)
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		v := findVariantByID(nil, "v1")
		assert.Nil(t, v)
	})
}

func TestReadCounter(t *testing.T) {
	t.Parallel()

	t.Run("tracks bytes read", func(t *testing.T) {
		t.Parallel()

		data := []byte("hello world")
		rc := &readCounter{Reader: bytes.NewReader(data)}

		buffer := make([]byte, 5)
		n, err := rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, int64(5), rc.Count)

		n, err = rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, int64(10), rc.Count)
	})

	t.Run("handles EOF", func(t *testing.T) {
		t.Parallel()

		data := []byte("hi")
		rc := &readCounter{Reader: bytes.NewReader(data)}

		buffer := make([]byte, 10)
		n, err := rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, int64(2), rc.Count)

		_, err = rc.Read(buffer)
		assert.ErrorIs(t, err, io.EOF)
	})
}

func TestBuildVariantStatusMap(t *testing.T) {
	t.Parallel()

	t.Run("builds map", func(t *testing.T) {
		t.Parallel()

		variants := []registry_dto.Variant{
			{VariantID: "thumb", Status: registry_dto.VariantStatusReady},
			{VariantID: "webp", Status: registry_dto.VariantStatusPending},
		}
		m := buildVariantStatusMap(variants)
		assert.Equal(t, registry_dto.VariantStatusReady, m["thumb"])
		assert.Equal(t, registry_dto.VariantStatusPending, m["webp"])
	})

	t.Run("empty variants", func(t *testing.T) {
		t.Parallel()

		m := buildVariantStatusMap(nil)
		assert.Empty(t, m)
	})
}

func TestIsProfileAlreadyReady(t *testing.T) {
	t.Parallel()

	variantStatus := map[string]registry_dto.VariantStatus{
		"thumb": registry_dto.VariantStatusReady,
		"webp":  registry_dto.VariantStatusPending,
	}

	assert.True(t, isProfileAlreadyReady(variantStatus, "thumb"))
	assert.False(t, isProfileAlreadyReady(variantStatus, "webp"))
	assert.False(t, isProfileAlreadyReady(variantStatus, "missing"))
}

func TestEventBusTypeName(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "nil", eventBusTypeName(nil))
	})
}
