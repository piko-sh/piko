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

package pikotest_domain

// containsSubstring checks if s contains substr.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns bool which is true if s contains substr, false otherwise.
func containsSubstring(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && searchString(s, substr)
}

// searchString checks whether substr appears within s.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns bool which is true if substr is found in s.
func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
