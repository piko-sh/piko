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

package formatter

import "piko.sh/piko/internal/formatter/formatter_domain"

// FormatterService formats Piko template (.pk) files.
type FormatterService = formatter_domain.FormatterService

// FormatOptions configures the behaviour of the formatter.
type FormatOptions = formatter_domain.FormatOptions

var (
	// NewFormatterService creates a new formatter with default options.
	NewFormatterService = formatter_domain.NewFormatterService

	// NewFormatterServiceWithOptions creates a new formatter with the given options.
	NewFormatterServiceWithOptions = formatter_domain.NewFormatterServiceWithOptions

	// DefaultFormatOptions returns the default formatting configuration.
	DefaultFormatOptions = formatter_domain.DefaultFormatOptions
)
