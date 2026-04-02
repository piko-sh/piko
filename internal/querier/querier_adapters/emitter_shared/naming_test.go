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

package emitter_shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnakeToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"id", "ID"},
		{"user_id", "UserID"},
		{"name", "Name"},
		{"first_name", "FirstName"},
		{"created_at", "CreatedAt"},
		{"user_api_key", "UserAPIKey"},
		{"http_url", "HTTPURL"},
		{"json_data", "JSONData"},
		{"html_content", "HTMLContent"},
		{"ip_address", "IPAddress"},
		{"uuid", "UUID"},
		{"user_uuid", "UserUUID"},
		{"a_b_c", "ABC"},
		{"", ""},
		{"single", "Single"},
		{"ALREADY_UPPER", "AlreadyUpper"},
		{"user_ids", "UserIDs"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := SnakeToPascalCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSnakeToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"id", "id"},
		{"user_id", "userID"},
		{"get_user", "getUser"},
		{"get_user_by_id", "getUserByID"},
		{"list_users", "listUsers"},
		{"name", "name"},
		{"created_at", "createdAt"},
		{"", ""},
		{"single", "single"},
		{"get_http_url", "getHTTPURL"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := SnakeToCamelCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
