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

// Package publiccontracts lists versioned, repository-owned consumer contracts.
package publiccontracts

var (
	// LocalExamples lists local browser recipes in replay-stable order.
	LocalExamples = []string{
		"001_hello_world", "003_reactive_counter", "004_product_catalogue", "005_blog_with_layout",
		"006_data_table", "007_todo_app", "008_event_bus_chat", "009_form_wizard",
	}
)

// Receipt records completed public-contract assertions.
type Receipt struct {
	// Contract identifies the checked public contract.
	Contract string

	// SourceSHA256 is the hex digest of the contract source bytes that ran.
	SourceSHA256 string

	// Steps counts completed interaction steps.
	Steps int

	// Assertions counts completed assertions.
	Assertions int

	// HTTPStatusVerified reports whether an HTTP status assertion completed.
	HTTPStatusVerified bool

	// BrowserVisited reports whether a browser visit completed.
	BrowserVisited bool
}
