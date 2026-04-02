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

package config_domain

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Base64Resolver decodes base64-encoded values from placeholders such as
// "base64:SGVsbG8gV29ybGQ=". It implements the Resolver interface.
type Base64Resolver struct{}

var _ = Resolver(&Base64Resolver{})

// GetPrefix returns the "base64:" prefix for base64-encoded values.
//
// Returns string which is the prefix that identifies base64-encoded values.
func (*Base64Resolver) GetPrefix() string {
	return "base64:"
}

// Resolve decodes a base64-encoded value.
//
// Takes value (string) which is the base64-encoded input to decode.
//
// Returns string which is the decoded content.
// Returns error when the value is not valid base64.
func (*Base64Resolver) Resolve(_ context.Context, value string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("invalid base64 string: %w", err)
	}
	return string(decoded), nil
}
