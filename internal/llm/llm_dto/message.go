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

// Role represents who sent a message in a conversation.
type Role string

const (
	// RoleSystem is a system message that sets the behaviour or context for the
	// assistant.
	RoleSystem Role = "system"

	// RoleUser is a message role that marks the message as coming from the user.
	RoleUser Role = "user"

	// RoleAssistant marks a message as coming from the AI model.
	RoleAssistant Role = "assistant"

	// RoleTool indicates a message that contains tool or function call results.
	RoleTool Role = "tool"
)

// Message represents a single message in a conversation with an LLM.
type Message struct {
	// Name is an optional participant name for multi-participant conversations.
	Name *string

	// ToolCallID identifies which tool call this message responds to.
	// Only set for tool role messages.
	ToolCallID *string

	// Role identifies who sent the message (system, user, assistant, or tool).
	Role Role

	// Content is the text content of the message.
	Content string

	// ContentParts holds multi-modal content such as text and images.
	// When set, takes priority over Content for providers that support
	// vision.
	ContentParts []ContentPart

	// ToolCalls holds tool or function calls made by the assistant.
	// Only filled for assistant messages that include tool calls.
	ToolCalls []ToolCall
}

// NewSystemMessage creates a new system message with the given content.
//
// Takes content (string) which sets the system prompt or context for the
// conversation.
//
// Returns Message which is set up with the system role.
func NewSystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

// NewUserMessage creates a new user message with the given content.
//
// Takes content (string) which contains the text of the user's input.
//
// Returns Message which is set up with the user role.
func NewUserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

// NewAssistantMessage creates a new assistant message with the given content.
//
// Takes content (string) which contains the assistant's response.
//
// Returns Message configured with RoleAssistant.
func NewAssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

// NewToolResultMessage creates a new tool result message.
//
// Takes toolCallID (string) which identifies the tool call being answered.
// Takes content (string) which holds the result from the tool.
//
// Returns Message set up with RoleTool.
func NewToolResultMessage(toolCallID, content string) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		ToolCallID: &toolCallID,
	}
}

// NewUserMessageWithImages creates a new user message with text and image
// content parts. This enables vision and multi-modal requests where images
// are sent alongside text.
//
// Takes text (string) which is the text content of the message.
// Takes images (...ContentPart) which are additional image content parts.
//
// Returns Message configured with RoleUser and ContentParts.
func NewUserMessageWithImages(text string, images ...ContentPart) Message {
	parts := make([]ContentPart, 0, len(images)+1)
	parts = append(parts, TextPart(text))
	parts = append(parts, images...)
	return Message{
		Role:         RoleUser,
		ContentParts: parts,
	}
}

// NewUserMessageWithImageURL creates a new user message with text and an image
// URL. This is a convenience method for simple vision requests with a single
// image.
//
// Takes text (string) which is the text content of the message.
// Takes imageURL (string) which is the URL of the image to include.
//
// Returns Message configured with RoleUser and ContentParts.
func NewUserMessageWithImageURL(text, imageURL string) Message {
	return NewUserMessageWithImages(text, ImageURLPart(imageURL))
}

// NewUserMessageWithImageData creates a user message with text and inline
// image data. This is a convenience method for simple vision requests with a
// single inline image.
//
// Takes text (string) which is the text content of the message.
// Takes mimeType (string) which specifies the MIME type (e.g. "image/png").
// Takes base64Data (string) which is the base64-encoded image data.
//
// Returns Message configured with RoleUser and ContentParts.
func NewUserMessageWithImageData(text, mimeType, base64Data string) Message {
	return NewUserMessageWithImages(text, ImageDataPart(mimeType, base64Data))
}
