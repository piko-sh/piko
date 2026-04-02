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

//go:build integration

package llm_integration_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestStructuredOutput_JSONSchema(t *testing.T) {
	service, ctx := createZoltaiService(t)

	schema := llm_dto.ObjectSchema(map[string]*llm_dto.JSONSchema{
		"message": {Type: "string"},
		"count":   {Type: "integer"},
	}, []string{"message", "count"})

	response, err := service.NewCompletion().
		User("Give me a fortune").
		ResponseFormat(llm_dto.ResponseFormatStructured("fortune", schema)).
		Do(ctx)
	require.NoError(t, err)

	content := response.Content()
	assert.NotEmpty(t, content)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(content), &result))
	assert.Contains(t, result, "message")
	assert.Contains(t, result, "count")
}

func TestStructuredOutput_JSONObject(t *testing.T) {
	service, ctx := createZoltaiService(t)

	response, err := service.NewCompletion().
		User("Give me a fortune").
		ResponseFormat(llm_dto.ResponseFormatJSON()).
		Do(ctx)
	require.NoError(t, err)

	content := response.Content()
	assert.NotEmpty(t, content)

	var result map[string]string
	require.NoError(t, json.Unmarshal([]byte(content), &result))
	assert.NotEmpty(t, result["response"])
}
