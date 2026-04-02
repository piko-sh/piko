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

package render_domain

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockCMSMedia struct {
	variants map[string]*mockVariant
	url      string
	altText  string
	width    int
	height   int
}

func (m *mockCMSMedia) MediaURL() string {
	return m.url
}

func (m *mockCMSMedia) MediaWidth() int {
	return m.width
}

func (m *mockCMSMedia) MediaHeight() int {
	return m.height
}

func (m *mockCMSMedia) MediaAltText() string {
	return m.altText
}

func (m *mockCMSMedia) MediaVariant(name string) *mockVariant {
	if v, ok := m.variants[name]; ok {
		return v
	}
	return nil
}

func (m *mockCMSMedia) MediaVariants() map[string]*mockVariant {
	return m.variants
}

type mockVariant struct {
	url   string
	width int
	ready bool
}

func (v *mockVariant) VariantURL() string {
	return v.url
}

func (v *mockVariant) VariantWidth() int {
	return v.width
}

func (v *mockVariant) IsReady() bool {
	return v.ready
}

type mockMinimalMedia struct {
	url string
}

func (m *mockMinimalMedia) MediaURL() string {
	return m.url
}

type mockInvalidMedia struct{}

func (m *mockInvalidMedia) MediaURL(argument string) string {
	return argument
}

type mockNoMethods struct {
	Name string
}

func TestTryCMSMediaWrapper_NilInput(t *testing.T) {
	result := tryCMSMediaWrapper(nil)
	assert.Nil(t, result)
}

func TestTryCMSMediaWrapper_NoMediaURLMethod(t *testing.T) {
	result := tryCMSMediaWrapper(&mockNoMethods{Name: "test"})
	assert.Nil(t, result)
}

func TestTryCMSMediaWrapper_InvalidMediaURLSignature(t *testing.T) {
	result := tryCMSMediaWrapper(&mockInvalidMedia{})
	assert.Nil(t, result)
}

func TestTryCMSMediaWrapper_MinimalMedia(t *testing.T) {
	media := &mockMinimalMedia{url: "https://example.com/image.png"}
	wrapper := tryCMSMediaWrapper(media)

	assert.NotNil(t, wrapper)
	assert.Equal(t, "https://example.com/image.png", wrapper.MediaURL())

	assert.Equal(t, 0, wrapper.MediaWidth())
	assert.Equal(t, 0, wrapper.MediaHeight())
	assert.Equal(t, "", wrapper.MediaAltText())
	assert.Nil(t, wrapper.MediaVariant("test"))
	assert.Nil(t, wrapper.MediaVariants())
}

func TestTryCMSMediaWrapper_FullMedia(t *testing.T) {
	media := &mockCMSMedia{
		url:     "https://example.com/photo.jpg",
		width:   1920,
		height:  1080,
		altText: "Test photo",
		variants: map[string]*mockVariant{
			"thumb_200": {url: "https://example.com/photo_thumb.jpg", width: 200, ready: true},
			"w800":      {url: "https://example.com/photo_800.jpg", width: 800, ready: true},
		},
	}

	wrapper := tryCMSMediaWrapper(media)
	assert.NotNil(t, wrapper)

	assert.Equal(t, "https://example.com/photo.jpg", wrapper.MediaURL())
	assert.Equal(t, 1920, wrapper.MediaWidth())
	assert.Equal(t, 1080, wrapper.MediaHeight())
	assert.Equal(t, "Test photo", wrapper.MediaAltText())
}

func TestCMSMediaWrapper_MediaVariant(t *testing.T) {
	media := &mockCMSMedia{
		url: "https://example.com/image.png",
		variants: map[string]*mockVariant{
			"thumb_200": {url: "https://example.com/thumb.png", width: 200, ready: true},
			"pending":   {url: "https://example.com/pending.png", width: 400, ready: false},
		},
	}

	wrapper := tryCMSMediaWrapper(media)
	assert.NotNil(t, wrapper)

	variant := wrapper.MediaVariant("thumb_200")
	assert.NotNil(t, variant)
	assert.Equal(t, "https://example.com/thumb.png", variant.VariantURL())
	assert.Equal(t, 200, variant.VariantWidth())
	assert.True(t, variant.IsReady())

	pending := wrapper.MediaVariant("pending")
	assert.NotNil(t, pending)
	assert.False(t, pending.IsReady())

	missing := wrapper.MediaVariant("nonexistent")
	assert.Nil(t, missing)
}

func TestCMSMediaWrapper_MediaVariants(t *testing.T) {
	media := &mockCMSMedia{
		url: "https://example.com/image.png",
		variants: map[string]*mockVariant{
			"w320": {url: "https://example.com/320.png", width: 320, ready: true},
			"w640": {url: "https://example.com/640.png", width: 640, ready: true},
			"w960": {url: "https://example.com/960.png", width: 960, ready: true},
		},
	}

	wrapper := tryCMSMediaWrapper(media)
	assert.NotNil(t, wrapper)

	variants := wrapper.MediaVariants()
	assert.NotNil(t, variants)
	assert.Len(t, variants, 3)

	assert.NotNil(t, variants["w320"])
	assert.NotNil(t, variants["w640"])
	assert.NotNil(t, variants["w960"])

	assert.Equal(t, 320, variants["w320"].VariantWidth())
	assert.Equal(t, 640, variants["w640"].VariantWidth())
	assert.Equal(t, 960, variants["w960"].VariantWidth())
}

func TestCMSMediaWrapper_MediaVariants_Empty(t *testing.T) {
	media := &mockCMSMedia{
		url:      "https://example.com/image.png",
		variants: nil,
	}

	wrapper := tryCMSMediaWrapper(media)
	assert.NotNil(t, wrapper)

	variants := wrapper.MediaVariants()
	assert.Nil(t, variants)
}

func TestVariantWrapper_MissingMethods(t *testing.T) {

	media := &struct {
		URL string
	}{URL: "test"}

	wrapper := &variantWrapper{value: reflect.ValueOf(media)}

	assert.Equal(t, "", wrapper.VariantURL())
	assert.Equal(t, 0, wrapper.VariantWidth())
	assert.False(t, wrapper.IsReady())
}

func TestTryCMSMediaWrapper_StringType(t *testing.T) {

	result := tryCMSMediaWrapper("just a string")
	assert.Nil(t, result)
}

func TestTryCMSMediaWrapper_IntType(t *testing.T) {
	result := tryCMSMediaWrapper(42)
	assert.Nil(t, result)
}

func TestTryCMSMediaWrapper_MapType(t *testing.T) {
	result := tryCMSMediaWrapper(map[string]string{"url": "test"})
	assert.Nil(t, result)
}
