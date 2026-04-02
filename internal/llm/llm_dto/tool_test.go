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

func TestToolChoiceAuto(t *testing.T) {
	t.Parallel()

	tc := ToolChoiceAuto()
	assert.Equal(t, ToolChoiceTypeAuto, tc.Type)
	assert.Nil(t, tc.Function)
}

func TestToolChoiceNone(t *testing.T) {
	t.Parallel()

	tc := ToolChoiceNone()
	assert.Equal(t, ToolChoiceTypeNone, tc.Type)
	assert.Nil(t, tc.Function)
}

func TestToolChoiceRequired(t *testing.T) {
	t.Parallel()

	tc := ToolChoiceRequired()
	assert.Equal(t, ToolChoiceTypeRequired, tc.Type)
	assert.Nil(t, tc.Function)
}

func TestToolChoiceSpecific(t *testing.T) {
	t.Parallel()

	tc := ToolChoiceSpecific("search_web")
	assert.Equal(t, ToolChoiceTypeFunction, tc.Type)
	assert.NotNil(t, tc.Function)
	assert.Equal(t, "search_web", tc.Function.Name)
}

func TestNewFunctionTool(t *testing.T) {
	t.Parallel()

	params := &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"query": {Type: "string"},
		},
		Required: []string{"query"},
	}

	tool := NewFunctionTool("search", "Search the web", params)

	assert.Equal(t, "function", tool.Type)
	assert.Equal(t, "search", tool.Function.Name)
	assert.NotNil(t, tool.Function.Description)
	assert.Equal(t, "Search the web", *tool.Function.Description)
	assert.Equal(t, params, tool.Function.Parameters)
	assert.Nil(t, tool.Function.Strict)
}

func TestNewStrictFunctionTool(t *testing.T) {
	t.Parallel()

	params := &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"input": {Type: "string"},
		},
		Required: []string{"input"},
	}

	tool := NewStrictFunctionTool("process", "Process input", params)

	assert.Equal(t, "function", tool.Type)
	assert.Equal(t, "process", tool.Function.Name)
	assert.NotNil(t, tool.Function.Description)
	assert.Equal(t, "Process input", *tool.Function.Description)
	assert.Equal(t, params, tool.Function.Parameters)
	assert.NotNil(t, tool.Function.Strict)
	assert.True(t, *tool.Function.Strict)
}
