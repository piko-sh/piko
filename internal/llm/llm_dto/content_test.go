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

package llm_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextPart(t *testing.T) {
	t.Parallel()

	part := TextPart("hello world")
	assert.Equal(t, ContentPartTypeText, part.Type)
	assert.NotNil(t, part.Text)
	assert.Equal(t, "hello world", *part.Text)
	assert.Nil(t, part.ImageURL)
	assert.Nil(t, part.ImageData)
}

func TestImageURLPart(t *testing.T) {
	t.Parallel()

	t.Run("without detail", func(t *testing.T) {
		t.Parallel()

		part := ImageURLPart("https://example.com/img.png")
		assert.Equal(t, ContentPartTypeImageURL, part.Type)
		assert.NotNil(t, part.ImageURL)
		assert.Equal(t, "https://example.com/img.png", part.ImageURL.URL)
		assert.Nil(t, part.ImageURL.Detail)
		assert.Nil(t, part.Text)
		assert.Nil(t, part.ImageData)
	})

	t.Run("with detail", func(t *testing.T) {
		t.Parallel()

		part := ImageURLPart("https://example.com/img.png", "high")
		assert.Equal(t, ContentPartTypeImageURL, part.Type)
		assert.NotNil(t, part.ImageURL.Detail)
		assert.Equal(t, "high", *part.ImageURL.Detail)
	})

	t.Run("with empty detail", func(t *testing.T) {
		t.Parallel()

		part := ImageURLPart("https://example.com/img.png", "")
		assert.Nil(t, part.ImageURL.Detail)
	})
}

func TestImageDataPart(t *testing.T) {
	t.Parallel()

	part := ImageDataPart("image/jpeg", "base64data==")
	assert.Equal(t, ContentPartTypeImageData, part.Type)
	assert.NotNil(t, part.ImageData)
	assert.Equal(t, "image/jpeg", part.ImageData.MIMEType)
	assert.Equal(t, "base64data==", part.ImageData.Data)
	assert.Nil(t, part.Text)
	assert.Nil(t, part.ImageURL)
}

func TestImageDetailHelpers(t *testing.T) {
	t.Parallel()

	t.Run("auto", func(t *testing.T) {
		t.Parallel()

		d := ImageDetailAuto()
		assert.NotNil(t, d)
		assert.Equal(t, "auto", *d)
	})

	t.Run("low", func(t *testing.T) {
		t.Parallel()

		d := ImageDetailLow()
		assert.NotNil(t, d)
		assert.Equal(t, "low", *d)
	})

	t.Run("high", func(t *testing.T) {
		t.Parallel()

		d := ImageDetailHigh()
		assert.NotNil(t, d)
		assert.Equal(t, "high", *d)
	})
}
