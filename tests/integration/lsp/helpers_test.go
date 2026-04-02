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

package lsp_stress_test

import (
	"fmt"
	"strings"
)

func generateModifiedTemplate(originalContent string, iteration int) string {

	marker := fmt.Sprintf("<!-- edit %d -->", iteration)

	idx := strings.Index(originalContent, "<template>")
	if idx == -1 {
		return originalContent
	}

	insertAt := idx + len("<template>")
	return originalContent[:insertAt] + "\n" + marker + originalContent[insertAt:]
}

func generateModifiedScript(originalContent string, iteration int) string {
	marker := fmt.Sprintf("// script edit %d", iteration)

	idx := strings.Index(originalContent, `package main`)
	if idx == -1 {
		return originalContent
	}

	insertAt := idx + len("package main")
	return originalContent[:insertAt] + "\n" + marker + originalContent[insertAt:]
}
