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

//go:build !integration

package pdfwriter_domain

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLocaTable_ShortFormat(t *testing.T) {
	t.Parallel()

	numberOfGlyphs := 3
	data := make([]byte, (numberOfGlyphs+1)*2)
	binary.BigEndian.PutUint16(data[0:], 0)
	binary.BigEndian.PutUint16(data[2:], 50)
	binary.BigEndian.PutUint16(data[4:], 100)
	binary.BigEndian.PutUint16(data[6:], 150)

	offsets, err := parseLocaTable(data, 0, numberOfGlyphs)
	require.NoError(t, err)

	require.Len(t, offsets, numberOfGlyphs+1)
	assert.Equal(t, uint32(0), offsets[0])
	assert.Equal(t, uint32(100), offsets[1])
	assert.Equal(t, uint32(200), offsets[2])
	assert.Equal(t, uint32(300), offsets[3])
}

func TestParseLocaTable_ShortFormat_TooShort(t *testing.T) {
	t.Parallel()

	data := make([]byte, 4)
	_, err := parseLocaTable(data, 0, 3)
	require.Error(t, err)
}

func TestParseLocaTable_LongFormat_TooShort(t *testing.T) {
	t.Parallel()

	data := make([]byte, 8)
	_, err := parseLocaTable(data, 1, 3)
	require.Error(t, err)
}

func TestExtractCompositeComponents_WithComponents(t *testing.T) {
	t.Parallel()

	data := make([]byte, 30)

	binary.BigEndian.PutUint16(data[0:], 0xFFFF)

	flags1 := compositeFlagMoreComponents | compositeFlagArg1And2AreWords
	binary.BigEndian.PutUint16(data[10:], flags1)
	binary.BigEndian.PutUint16(data[12:], 42)

	binary.BigEndian.PutUint16(data[18:], 0)
	binary.BigEndian.PutUint16(data[20:], 99)

	components := extractCompositeComponents(data)
	require.Len(t, components, 2)
	assert.Equal(t, uint16(42), components[0])
	assert.Equal(t, uint16(99), components[1])
}

func TestExtractCompositeComponents_TooShort(t *testing.T) {
	t.Parallel()

	data := make([]byte, 5)
	components := extractCompositeComponents(data)
	assert.Nil(t, components)
}

func TestExtractCompositeComponents_WithScaleTransform(t *testing.T) {
	t.Parallel()

	data := make([]byte, 30)
	binary.BigEndian.PutUint16(data[0:], 0xFFFF)

	flags := compositeFlagMoreComponents | compositeFlagWeHaveAScale
	binary.BigEndian.PutUint16(data[10:], flags)
	binary.BigEndian.PutUint16(data[12:], 10)

	offset := 10 + compositeMinComponentBytes + compositeByteArgBytes + compositeScaleBytes

	binary.BigEndian.PutUint16(data[offset:], 0)
	binary.BigEndian.PutUint16(data[offset+2:], 20)

	components := extractCompositeComponents(data)
	require.Len(t, components, 2)
	assert.Equal(t, []uint16{10, 20}, components)
}

func TestAdvancePastCompositeTransform_NoTransform(t *testing.T) {
	t.Parallel()

	result := advancePastCompositeTransform(0, 10)
	assert.Equal(t, 10, result)
}

func TestAdvancePastCompositeTransform_Scale(t *testing.T) {
	t.Parallel()

	result := advancePastCompositeTransform(compositeFlagWeHaveAScale, 10)
	assert.Equal(t, 10+compositeScaleBytes, result)
}

func TestAdvancePastCompositeTransform_XYScale(t *testing.T) {
	t.Parallel()

	result := advancePastCompositeTransform(compositeFlagWeHaveAnXAndYScale, 10)
	assert.Equal(t, 10+compositeXYScaleBytes, result)
}

func TestAdvancePastCompositeTransform_TwoByTwo(t *testing.T) {
	t.Parallel()

	result := advancePastCompositeTransform(compositeFlagWeHaveATwoByTwo, 10)
	assert.Equal(t, 10+compositeTwoByTwoBytes, result)
}

func TestResolveCompositeGlyphs_AddsComponents(t *testing.T) {
	t.Parallel()

	glyfData := make([]byte, 100)

	binary.BigEndian.PutUint16(glyfData[0:], 0xFFFF)

	binary.BigEndian.PutUint16(glyfData[10:], 0)
	binary.BigEndian.PutUint16(glyfData[12:], 2)

	offsets := []uint32{0, 0, 24, 50}

	glyphSet := map[uint16]bool{
		0: true,
		1: true,
	}

	resolveCompositeGlyphs(glyfData, offsets, glyphSet, 3)

	assert.True(t, glyphSet[2], "expected glyph 2 to be added as a composite component dependency")
}

func TestCompositeComponentsForGlyph_SimpleGlyph(t *testing.T) {
	t.Parallel()

	glyfData := make([]byte, 20)
	binary.BigEndian.PutUint16(glyfData[0:], 1)

	offsets := []uint32{0, 20, 20}
	result := compositeComponentsForGlyph(glyfData, offsets, 0)
	assert.Nil(t, result)
}

func TestCompositeComponentsForGlyph_EmptyGlyph(t *testing.T) {
	t.Parallel()

	offsets := []uint32{0, 0, 0}
	result := compositeComponentsForGlyph(nil, offsets, 0)
	assert.Nil(t, result)
}
