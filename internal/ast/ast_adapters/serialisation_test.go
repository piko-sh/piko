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

package ast_adapters

import (
	"context"
	"encoding/binary"
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema"
)

const (
	corruptionTestTemplate = `<div class="card" p-if="state.Visible == true" :title="state.Title">` +
		`<span p-on:click.prevent.stop="handle(1, 'a')">{{ state.Name }} and {{ state.Count }}</span>` +
		`<p p-for="(index, item) in state.Items" p-key="item.ID" :data-index="index">text {{ item.Label }}</p>` +
		`<img src="/a.png" alt="x"><input p-model="state.Query" :disabled="!state.Enabled">` +
		`<my-partial is="card" :item="{ 'a': 1, 'b': [1, 2, 3] }"></my-partial></div>`
	decodeAllocationBytesPerPayloadByte = 512
	decodeAllocationBaseBytes           = 4 << 20
)

func TestDecodeAST_CorruptData(t *testing.T) {
	encoded := encodeCorruptionTestTemplate(t)
	headerSize := len(encoded) - len(mustUnpackPayload(t, encoded))

	testCases := []struct {
		mutate func([]byte) []byte
		name   string
	}{
		{name: "schema hash only", mutate: func(data []byte) []byte { return data[:headerSize] }},
		{name: "payload shorter than a root offset", mutate: func(data []byte) []byte { return data[:headerSize+2] }},
		{name: "truncated to half", mutate: func(data []byte) []byte { return data[:len(data)/2] }},
		{name: "last byte removed", mutate: func(data []byte) []byte { return data[:len(data)-1] }},
		{name: "root offset beyond payload", mutate: func(data []byte) []byte {
			binary.LittleEndian.PutUint32(data[headerSize:], 0xfffffff0)
			return data
		}},
		{name: "root offset zero", mutate: func(data []byte) []byte {
			binary.LittleEndian.PutUint32(data[headerSize:], 0)
			return data
		}},
		{name: "every payload byte inverted", mutate: func(data []byte) []byte {
			for index := headerSize; index < len(data); index++ {
				data[index] = ^data[index]
			}
			return data
		}},
		{name: "flipped byte early in payload", mutate: flipByteAt(headerSize + 8)},
		{name: "flipped byte in the middle", mutate: func(data []byte) []byte { return flipByteAt(len(data) / 2)(data) }},
		{name: "flipped byte near the end", mutate: func(data []byte) []byte { return flipByteAt(len(data) - 3)(data) }},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			corrupt := testCase.mutate(append([]byte(nil), encoded...))

			for _, decode := range astDecoders() {
				ast, err := decode.fn(context.Background(), corrupt)
				if err != nil {
					assert.Nil(t, ast, decode.name)
					assert.NotContains(t, err.Error(), "goroutine", decode.name)
				}
			}
		})
	}
}

func TestDecodeAST_CorruptPayloadErrorsAreSentinel(t *testing.T) {
	encoded := encodeCorruptionTestTemplate(t)
	headerSize := len(encoded) - len(mustUnpackPayload(t, encoded))

	testCases := []struct {
		name string
		data []byte
	}{
		{name: "payload shorter than a root offset", data: encoded[:headerSize+2]},
		{name: "root offset beyond payload", data: func() []byte {
			corrupt := append([]byte(nil), encoded...)
			binary.LittleEndian.PutUint32(corrupt[headerSize:], 0xfffffff0)
			return corrupt
		}()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, decode := range astDecoders() {
				ast, err := decode.fn(context.Background(), testCase.data)
				require.Error(t, err, decode.name)
				assert.ErrorIs(t, err, errCorruptAST, decode.name)
				assert.Nil(t, ast, decode.name)
			}
		})
	}
}

func TestDecodeAST_EveryByteCorruptedStaysBounded(t *testing.T) {
	encoded := encodeCorruptionTestTemplate(t)
	headerSize := len(encoded) - len(mustUnpackPayload(t, encoded))
	allocationLimit := uint64(decodeAllocationBaseBytes + decodeAllocationBytesPerPayloadByte*len(encoded))

	replacements := []byte{0x00, 0x7f, 0x80, 0xff}
	stride := 1
	if testing.Short() {
		stride = 3
	}

	var corruptErrors, otherErrors, decoded int
	corrupt := make([]byte, len(encoded))
	for offset := headerSize; offset < len(encoded); offset += stride {
		for _, replacement := range replacements {
			copy(corrupt, encoded)
			corrupt[offset] = replacement

			for _, decode := range astDecoders() {
				allocated, err := measureDecodeAllocation(decode.fn, corrupt)
				require.LessOrEqualf(t, allocated, allocationLimit,
					"%s allocated %d bytes for a %d byte buffer with byte %d set to %#x",
					decode.name, allocated, len(corrupt), offset, replacement)

				switch {
				case err == nil:
					decoded++
				case isCorruptASTError(err):
					corruptErrors++
				default:
					otherErrors++
				}
			}
		}
	}

	assert.Positive(t, corruptErrors)
	assert.Positive(t, decoded)
	t.Logf("payload=%d bytes decoded=%d corrupt=%d other=%d", len(encoded), decoded, corruptErrors, otherErrors)
}

func TestDecodeAST_SchemaVersionMismatch(t *testing.T) {
	encoded := encodeCorruptionTestTemplate(t)
	encoded[0] ^= 0xff

	for _, decode := range astDecoders() {
		ast, err := decode.fn(context.Background(), encoded)
		require.ErrorIs(t, err, errASTSchemaVersionMismatch, decode.name)
		assert.NotErrorIs(t, err, errCorruptAST, decode.name)
		assert.Nil(t, ast, decode.name)
	}
}

func TestDecoder_ReserveElements(t *testing.T) {
	testCases := []struct {
		name          string
		requests      []int
		payloadLength int
		wantErrorAt   int
	}{
		{name: "fits exactly", payloadLength: 40, requests: []int{4, 6}, wantErrorAt: -1},
		{name: "single request too large", payloadLength: 40, requests: []int{11}, wantErrorAt: 0},
		{name: "cumulative requests too large", payloadLength: 40, requests: []int{6, 5}, wantErrorAt: 1},
		{name: "negative length", payloadLength: 40, requests: []int{-1}, wantErrorAt: 0},
		{name: "empty payload", payloadLength: 0, requests: []int{1}, wantErrorAt: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			d := newDecoder(testCase.payloadLength, false)
			for index, request := range testCase.requests {
				err := d.reserveElements(request)
				if index == testCase.wantErrorAt {
					require.ErrorIs(t, err, errCorruptAST)
					return
				}
				require.NoError(t, err)
			}
			assert.Equal(t, -1, testCase.wantErrorAt)
		})
	}
}

func TestDecodeAST_ParsedTemplateRoundTrip(t *testing.T) {
	encoded := encodeCorruptionTestTemplate(t)

	for _, decode := range astDecoders() {
		ast, err := decode.fn(context.Background(), encoded)
		require.NoError(t, err, decode.name)
		require.NotNil(t, ast, decode.name)
		require.Len(t, ast.RootNodes, 1, decode.name)
		assert.Equal(t, "div", ast.RootNodes[0].TagName, decode.name)
	}
}

type astDecoder struct {
	fn   func(context.Context, []byte) (*ast_domain.TemplateAST, error)
	name string
}

func astDecoders() []astDecoder {
	return []astDecoder{
		{name: "DecodeAST", fn: DecodeAST},
		{name: "DecodeASTForRender", fn: DecodeASTForRender},
	}
}

func encodeCorruptionTestTemplate(t *testing.T) []byte {
	t.Helper()

	tree, err := ast_domain.Parse(context.Background(), corruptionTestTemplate, "corrupt.pk", nil)
	require.NoError(t, err)

	encoded, err := EncodeAST(tree)
	require.NoError(t, err)
	return encoded
}

func mustUnpackPayload(t *testing.T, encoded []byte) []byte {
	t.Helper()

	payload, err := ast_schema.Unpack(encoded)
	require.NoError(t, err)
	return payload
}

func flipByteAt(offset int) func([]byte) []byte {
	return func(data []byte) []byte {
		data[offset] ^= 0xff
		return data
	}
}

func measureDecodeAllocation(decode func(context.Context, []byte) (*ast_domain.TemplateAST, error), data []byte) (uint64, error) {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := decode(context.Background(), data)
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc, err
}

func isCorruptASTError(err error) bool {
	return err != nil && errors.Is(err, errCorruptAST)
}
