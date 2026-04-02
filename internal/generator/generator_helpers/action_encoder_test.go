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

package generator_helpers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestEncodeActionPayloadBytes_SimpleFunction(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args:     nil,
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, "handleClick", parsed.Function)
	assert.Nil(t, parsed.Args)
}

func TestEncodeActionPayloadBytes_WithArgs(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handleSubmit",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "test-value"},
			{Type: "s", Value: 42},
			{Type: "s", Value: true},
		},
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, "handleSubmit", parsed.Function)
	require.Len(t, parsed.Args, 3)
	assert.Equal(t, "s", parsed.Args[0].Type)
	assert.Equal(t, "test-value", parsed.Args[0].Value)
}

func TestEncodeActionPayloadBytes_EmptyArgs(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "noArgs",
		Args:     []templater_dto.ActionArgument{},
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, "noArgs", parsed.Function)
	assert.NotNil(t, parsed.Args)
	assert.Empty(t, parsed.Args)
}

func TestEncodeActionPayloadBytes_SpecialChars(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handle\"Test",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "line1\nline2\ttab\"quote\\backslash"},
		},
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, "handle\"Test", parsed.Function)
	require.Len(t, parsed.Args, 1)
	assert.Equal(t, "line1\nline2\ttab\"quote\\backslash", parsed.Args[0].Value)
}

func TestEncodeActionPayloadBytes_ControlChars(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handleData",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "control\x00\x01\x1f"},
		},
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, "handleData", parsed.Function)
	require.Len(t, parsed.Args, 1)
	assert.Equal(t, "control\x00\x01\x1f", parsed.Args[0].Value)
}

func TestEncodeActionPayloadBytes_NumericTypes(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handleNumbers",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: int(42)},
			{Type: "s", Value: int64(9223372036854775807)},
			{Type: "s", Value: float64(3.14159265358979)},
			{Type: "s", Value: float32(2.71828)},
			{Type: "s", Value: uint64(18446744073709551615)},
		},
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleNumbers", parsed["f"])
}

func TestEncodeActionPayloadBytes_NilValue(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handleEvent",
		Args: []templater_dto.ActionArgument{
			{Type: "e", Value: nil},
		},
	}

	bufferPointer := EncodeActionPayloadBytes(payload)
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, "handleEvent", parsed.Function)
	require.Len(t, parsed.Args, 1)
	assert.Equal(t, "e", parsed.Args[0].Type)

}

func warmupPool() {
	buffers := make([]*[]byte, 100)
	for i := range buffers {
		buffers[i] = ast_domain.GetByteBuf()
	}
	for _, buffer := range buffers {
		ast_domain.PutByteBuf(buffer)
	}
}

func TestEncodeActionPayloadBytes_ZeroAlloc(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "value"},
		},
	}

	warmupPool()

	for range 100 {
		bufferPointer := EncodeActionPayloadBytes(payload)
		require.NotNil(t, bufferPointer)
		ast_domain.PutByteBuf(bufferPointer)
	}
}

func TestEncodeActionPayloadBytes_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	payload := templater_dto.ActionPayload{
		Function: "delete_item",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "item_id_123"},
		},
	}

	warmupPool()

	const goroutines = 10
	const iterations = 100

	errChan := make(chan error, goroutines)

	for range goroutines {
		go func() {
			for range iterations {
				bufferPointer := EncodeActionPayloadBytes(payload)
				if bufferPointer == nil {
					errChan <- errors.New("got nil buffer")
					return
				}

				jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*bufferPointer))
				if err != nil {
					errChan <- fmt.Errorf("base64 decode error: %w (content: %s)", err, string(*bufferPointer))
					ast_domain.PutByteBuf(bufferPointer)
					return
				}

				if !bytes.Contains(jsonBytes, []byte(`"f":"delete_item"`)) {
					errChan <- fmt.Errorf("corrupted payload: expected delete_item, got: %s", string(jsonBytes))
					ast_domain.PutByteBuf(bufferPointer)
					return
				}

				ast_domain.PutByteBuf(bufferPointer)
			}
			errChan <- nil
		}()
	}

	for range goroutines {
		if err := <-errChan; err != nil {
			t.Fatal(err)
		}
	}
}

func TestEncodeActionPayloadBytes_FixedArityEquivalence(t *testing.T) {
	t.Parallel()

	warmupPool()

	testCases := []struct {
		name      string
		function  string
		arguments []templater_dto.ActionArgument
	}{
		{
			name:      "zero arguments",
			function:  "handleClick",
			arguments: []templater_dto.ActionArgument{},
		},
		{
			name:      "one argument event",
			function:  "handleSubmit",
			arguments: []templater_dto.ActionArgument{{Type: "e"}},
		},
		{
			name:      "one argument string",
			function:  "handleDelete",
			arguments: []templater_dto.ActionArgument{{Type: "s", Value: "item_123"}},
		},
		{
			name:     "two arguments",
			function: "handleUpdate",
			arguments: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: "s", Value: "test"},
			},
		},
		{
			name:     "three arguments",
			function: "handleCreate",
			arguments: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: "s", Value: "name"},
				{Type: "s", Value: "value"},
			},
		},
		{
			name:     "four arguments",
			function: "handleBatch",
			arguments: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: "s", Value: "a"},
				{Type: "s", Value: "b"},
				{Type: "s", Value: "c"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expectedBuf := EncodeActionPayloadBytes(templater_dto.ActionPayload{
				Function: tc.function,
				Args:     tc.arguments,
			})
			if expectedBuf == nil {
				t.Fatal("expected buffer is nil")
			}
			expected := string(*expectedBuf)
			ast_domain.PutByteBuf(expectedBuf)

			var actualBuf *[]byte
			switch len(tc.arguments) {
			case 0:
				actualBuf = EncodeActionPayloadBytes0(tc.function)
			case 1:
				actualBuf = EncodeActionPayloadBytes1(tc.function, tc.arguments[0])
			case 2:
				actualBuf = EncodeActionPayloadBytes2(tc.function, tc.arguments[0], tc.arguments[1])
			case 3:
				actualBuf = EncodeActionPayloadBytes3(tc.function, tc.arguments[0], tc.arguments[1], tc.arguments[2])
			case 4:
				actualBuf = EncodeActionPayloadBytes4(tc.function, tc.arguments[0], tc.arguments[1], tc.arguments[2], tc.arguments[3])
			default:
				t.Fatalf("unsupported argument count: %d", len(tc.arguments))
			}

			if actualBuf == nil {
				t.Fatal("actual buffer is nil")
			}
			actual := string(*actualBuf)
			ast_domain.PutByteBuf(actualBuf)

			if actual != expected {
				t.Errorf("output mismatch:\n  expected: %s\n  actual:   %s", expected, actual)
			}
		})
	}
}
