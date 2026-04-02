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

package i18n_domain

import (
	"fmt"

	"piko.sh/piko/internal/json"
)

// FlattenTranslations converts nested translation maps into flat key-value
// pairs using dot notation. For example, {"user": {"name": "Name"}} becomes
// {"user.name": "Name"}.
//
// Takes data (map[string]any) which contains the nested translation map to
// flatten.
// Takes prefix (string) which specifies the key prefix for nested calls, or an
// empty string for the root call.
// Takes result (map[string]string) which collects the flattened key-value
// pairs.
func FlattenTranslations(data map[string]any, prefix string, result map[string]string) {
	for key, value := range data {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			FlattenTranslations(v, newKey, result)
		case string:
			result[newKey] = v
		default:
			result[newKey] = fmt.Sprintf("%v", v)
		}
	}
}

// ParseAndFlatten parses JSON translation data and returns a flattened map
// with dot-notation keys.
//
// Takes jsonData ([]byte) which contains the raw JSON translation data.
//
// Returns map[string]string which maps dot-notation keys to their translated
// values.
// Returns error when the JSON data cannot be parsed.
func ParseAndFlatten(jsonData []byte) (map[string]string, error) {
	var nested map[string]any
	if err := json.Unmarshal(jsonData, &nested); err != nil {
		return nil, fmt.Errorf("failed to unmarshal i18n JSON: %w", err)
	}

	flattened := make(map[string]string)
	FlattenTranslations(nested, "", flattened)
	return flattened, nil
}
