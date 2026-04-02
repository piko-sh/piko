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

// ToolDefinition defines a tool/function that the model can call.
type ToolDefinition struct {
	// Function holds the function definition.
	Function FunctionDefinition

	// Type is the kind of tool. Currently only "function" is supported.
	Type string
}

// FunctionDefinition describes a function that the model can call.
type FunctionDefinition struct {
	// Description explains what the function does. The model uses this to
	// decide when to call the function.
	Description *string

	// Parameters is a JSON Schema that describes the function's input values.
	Parameters *JSONSchema

	// Strict enables strict schema adherence when true. The model will follow
	// the schema exactly, which is needed for structured outputs.
	Strict *bool

	// Name is the function name. Must contain only a-z, A-Z, 0-9, or underscores.
	Name string
}

// ToolCall represents a tool call made by the model.
type ToolCall struct {
	// ID is the unique identifier for this tool call.
	ID string

	// Type is the kind of tool called; currently only "function" is supported.
	Type string

	// Function holds the function name and arguments for this tool call.
	Function FunctionCall
}

// FunctionCall holds the name and arguments for a function to be called.
type FunctionCall struct {
	// Name is the name of the function to call.
	Name string

	// Arguments is a JSON string containing the function arguments.
	Arguments string
}

// ToolChoice controls how the model selects and uses available tools.
type ToolChoice struct {
	// Function specifies which function to use when Type is "function".
	Function *ToolChoiceFunction

	// Type specifies the choice mode: "auto", "none", "required", or "function".
	Type string
}

// ToolChoiceFunction specifies a function that the model must call.
type ToolChoiceFunction struct {
	// Name is the name of the function to call.
	Name string
}

const (
	// ToolChoiceTypeAuto allows the model to decide whether to call tools.
	ToolChoiceTypeAuto = "auto"

	// ToolChoiceTypeNone stops the model from calling any tools.
	ToolChoiceTypeNone = "none"

	// ToolChoiceTypeRequired makes the model call at least one tool.
	ToolChoiceTypeRequired = "required"

	// ToolChoiceTypeFunction forces the model to call a specific function.
	ToolChoiceTypeFunction = "function"
)

// NewFunctionTool creates a new function tool definition.
//
// Takes name (string) which is the function name.
// Takes description (string) which explains what the function does.
// Takes parameters (*JSONSchema) which describes the function parameters.
//
// Returns ToolDefinition configured as a function tool.
func NewFunctionTool(name, description string, parameters *JSONSchema) ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        name,
			Description: &description,
			Parameters:  parameters,
		},
	}
}

// NewStrictFunctionTool creates a new function tool with strict schema
// enforcement.
//
// Takes name (string) which is the function name.
// Takes description (string) which explains what the function does.
// Takes parameters (*JSONSchema) which describes the function parameters.
//
// Returns ToolDefinition configured as a strict function tool.
func NewStrictFunctionTool(name, description string, parameters *JSONSchema) ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        name,
			Description: &description,
			Parameters:  parameters,
			Strict:      new(true),
		},
	}
}

// DeepCopy returns an independent copy of the tool definition with all nested
// pointers duplicated.
//
// Returns ToolDefinition which is a deep copy of the receiver.
func (t ToolDefinition) DeepCopy() ToolDefinition {
	return ToolDefinition{
		Type:     t.Type,
		Function: t.Function.DeepCopy(),
	}
}

// DeepCopy returns an independent copy of the function definition with all
// nested pointers duplicated.
//
// Returns FunctionDefinition which is a deep copy of the receiver.
func (f FunctionDefinition) DeepCopy() FunctionDefinition {
	cp := FunctionDefinition{
		Name: f.Name,
	}
	cp.Description = copyPtr(f.Description)
	cp.Strict = copyPtr(f.Strict)
	if f.Parameters != nil {
		cp.Parameters = new(f.Parameters.DeepCopy())
	}
	return cp
}

// ToolChoiceAuto returns a ToolChoice that lets the model decide when to use
// tools.
//
// Returns *ToolChoice set up for automatic tool selection.
func ToolChoiceAuto() *ToolChoice {
	return &ToolChoice{Type: ToolChoiceTypeAuto}
}

// ToolChoiceNone returns a ToolChoice that stops the model from calling tools.
//
// Returns *ToolChoice which is set to disable all tool calls.
func ToolChoiceNone() *ToolChoice {
	return &ToolChoice{Type: ToolChoiceTypeNone}
}

// ToolChoiceRequired returns a ToolChoice that forces at least one tool call.
//
// Returns *ToolChoice which is set to require tool use.
func ToolChoiceRequired() *ToolChoice {
	return &ToolChoice{Type: ToolChoiceTypeRequired}
}

// ToolChoiceSpecific returns a ToolChoice that forces a specific function call.
//
// Takes name (string) which is the name of the function to call.
//
// Returns *ToolChoice configured to call the specified function.
func ToolChoiceSpecific(name string) *ToolChoice {
	return &ToolChoice{
		Type:     ToolChoiceTypeFunction,
		Function: &ToolChoiceFunction{Name: name},
	}
}
