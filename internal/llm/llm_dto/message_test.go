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

func TestNewSystemMessage(t *testing.T) {
	t.Parallel()

	message := NewSystemMessage("you are helpful")
	assert.Equal(t, RoleSystem, message.Role)
	assert.Equal(t, "you are helpful", message.Content)
	assert.Nil(t, message.ToolCallID)
	assert.Empty(t, message.ContentParts)
}

func TestNewUserMessage(t *testing.T) {
	t.Parallel()

	message := NewUserMessage("hello")
	assert.Equal(t, RoleUser, message.Role)
	assert.Equal(t, "hello", message.Content)
	assert.Nil(t, message.ToolCallID)
	assert.Empty(t, message.ContentParts)
}

func TestNewAssistantMessage(t *testing.T) {
	t.Parallel()

	message := NewAssistantMessage("sure thing")
	assert.Equal(t, RoleAssistant, message.Role)
	assert.Equal(t, "sure thing", message.Content)
	assert.Nil(t, message.ToolCallID)
	assert.Empty(t, message.ContentParts)
}

func TestNewToolResultMessage(t *testing.T) {
	t.Parallel()

	message := NewToolResultMessage("call_123", `{"result": 42}`)
	assert.Equal(t, RoleTool, message.Role)
	assert.Equal(t, `{"result": 42}`, message.Content)
	assert.NotNil(t, message.ToolCallID)
	assert.Equal(t, "call_123", *message.ToolCallID)
}

func TestNewUserMessageWithImages(t *testing.T) {
	t.Parallel()

	t.Run("text with one image", func(t *testing.T) {
		t.Parallel()

		img := ImageURLPart("https://example.com/img.png")
		message := NewUserMessageWithImages("describe this", img)

		assert.Equal(t, RoleUser, message.Role)
		assert.Len(t, message.ContentParts, 2)
		assert.Equal(t, ContentPartTypeText, message.ContentParts[0].Type)
		assert.Equal(t, "describe this", *message.ContentParts[0].Text)
		assert.Equal(t, ContentPartTypeImageURL, message.ContentParts[1].Type)
		assert.Equal(t, "https://example.com/img.png", message.ContentParts[1].ImageURL.URL)
	})

	t.Run("text with multiple images", func(t *testing.T) {
		t.Parallel()

		img1 := ImageURLPart("https://example.com/a.png")
		img2 := ImageURLPart("https://example.com/b.png")
		message := NewUserMessageWithImages("compare these", img1, img2)

		assert.Equal(t, RoleUser, message.Role)
		assert.Len(t, message.ContentParts, 3)
		assert.Equal(t, ContentPartTypeText, message.ContentParts[0].Type)
		assert.Equal(t, ContentPartTypeImageURL, message.ContentParts[1].Type)
		assert.Equal(t, ContentPartTypeImageURL, message.ContentParts[2].Type)
	})

	t.Run("text only", func(t *testing.T) {
		t.Parallel()

		message := NewUserMessageWithImages("just text")
		assert.Len(t, message.ContentParts, 1)
		assert.Equal(t, ContentPartTypeText, message.ContentParts[0].Type)
	})
}

func TestNewUserMessageWithImageURL(t *testing.T) {
	t.Parallel()

	message := NewUserMessageWithImageURL("what is this?", "https://example.com/photo.jpg")

	assert.Equal(t, RoleUser, message.Role)
	assert.Len(t, message.ContentParts, 2)
	assert.Equal(t, ContentPartTypeText, message.ContentParts[0].Type)
	assert.Equal(t, "what is this?", *message.ContentParts[0].Text)
	assert.Equal(t, ContentPartTypeImageURL, message.ContentParts[1].Type)
	assert.Equal(t, "https://example.com/photo.jpg", message.ContentParts[1].ImageURL.URL)
}

func TestNewUserMessageWithImageData(t *testing.T) {
	t.Parallel()

	message := NewUserMessageWithImageData("describe", "image/png", "iVBORw0KGgo=")

	assert.Equal(t, RoleUser, message.Role)
	assert.Len(t, message.ContentParts, 2)
	assert.Equal(t, ContentPartTypeText, message.ContentParts[0].Type)
	assert.Equal(t, "describe", *message.ContentParts[0].Text)
	assert.Equal(t, ContentPartTypeImageData, message.ContentParts[1].Type)
	assert.Equal(t, "image/png", message.ContentParts[1].ImageData.MIMEType)
	assert.Equal(t, "iVBORw0KGgo=", message.ContentParts[1].ImageData.Data)
}
