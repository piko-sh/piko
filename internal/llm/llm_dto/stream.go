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

// StreamEvent represents an event in a streaming completion response.
type StreamEvent struct {
	// Error holds the error value for error events; nil for non-error events.
	Error error

	// Chunk contains the new content for chunk events.
	Chunk *StreamChunk

	// FinalResponse holds the full response when the stream ends.
	// Only set if the provider supports it and it was requested.
	FinalResponse *CompletionResponse

	// Type indicates the kind of event: chunk, done, or error.
	Type StreamEventType

	// Done indicates whether this is the final event in the stream.
	Done bool
}

// StreamEventType identifies the kind of event in a stream.
type StreamEventType string

const (
	// StreamEventChunk shows that the event contains a content chunk with new data.
	StreamEventChunk StreamEventType = "chunk"

	// StreamEventDone indicates the stream has finished successfully.
	StreamEventDone StreamEventType = "done"

	// StreamEventError indicates that an error occurred during streaming.
	StreamEventError StreamEventType = "error"
)

// StreamChunk contains a single chunk of streaming data.
type StreamChunk struct {
	// Delta contains the incremental content changes.
	Delta *MessageDelta

	// FinishReason indicates why generation stopped; only set on the final chunk.
	FinishReason *FinishReason

	// Usage holds token usage statistics. Only present in the final chunk if
	// requested via StreamOptions.IncludeUsage.
	Usage *Usage

	// ID is the unique identifier for this chunk's parent completion.
	ID string

	// Model is the name of the model generating this stream.
	Model string
}

// MessageDelta contains incremental changes to a message during streaming.
type MessageDelta struct {
	// Role is set in the first chunk to show the message role.
	Role *Role

	// Content holds the new text to add to the message.
	Content *string

	// ToolCalls holds partial tool call updates received during streaming.
	ToolCalls []ToolCallDelta
}

// ToolCallDelta contains incremental changes to a tool call during streaming.
type ToolCallDelta struct {
	// ID is the tool call identifier, set in the first delta for this tool call.
	ID *string

	// Type is the tool type, set in the first delta for this tool call.
	Type *string

	// Function contains partial function call data received so far.
	Function *FunctionCallDelta

	// Index identifies which tool call this delta applies to.
	Index int
}

// FunctionCallDelta contains incremental changes to a function call.
type FunctionCallDelta struct {
	// Name is the function name, set in the first delta for this function call.
	Name *string

	// Arguments contains the partial JSON arguments string to append.
	Arguments *string
}

// NewChunkEvent creates a new chunk stream event.
//
// Takes chunk (*StreamChunk) which contains the delta data.
//
// Returns StreamEvent which is set up as a chunk event.
func NewChunkEvent(chunk *StreamChunk) StreamEvent {
	return StreamEvent{
		Type:  StreamEventChunk,
		Chunk: chunk,
	}
}

// NewDoneEvent creates a new done stream event.
//
// Takes finalResponse (*CompletionResponse) which holds the complete response,
// or nil if no final response is needed.
//
// Returns StreamEvent set up as a done event.
func NewDoneEvent(finalResponse *CompletionResponse) StreamEvent {
	return StreamEvent{
		Type:          StreamEventDone,
		Done:          true,
		FinalResponse: finalResponse,
	}
}

// NewErrorEvent creates a new error stream event.
//
// Takes err (error) which is the error that occurred.
//
// Returns StreamEvent which is set up as an error event.
func NewErrorEvent(err error) StreamEvent {
	return StreamEvent{
		Type:  StreamEventError,
		Error: err,
	}
}
