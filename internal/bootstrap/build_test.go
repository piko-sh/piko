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

package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsErrorPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		filename   string
		wantStatus int
		wantMin    int
		wantMax    int
		wantIsErr  bool
		wantCatch  bool
	}{

		{name: "404 error page", filename: "!404.pk", wantIsErr: true, wantStatus: 404},
		{name: "500 error page", filename: "!500.pk", wantIsErr: true, wantStatus: 500},
		{name: "403 error page", filename: "!403.pk", wantIsErr: true, wantStatus: 403},
		{name: "401 error page", filename: "!401.pk", wantIsErr: true, wantStatus: 401},
		{name: "418 teapot page", filename: "!418.pk", wantIsErr: true, wantStatus: 418},
		{name: "100 lowest valid", filename: "!100.pk", wantIsErr: true, wantStatus: 100},
		{name: "599 highest valid", filename: "!599.pk", wantIsErr: true, wantStatus: 599},

		{name: "catch-all error page", filename: "!error.pk", wantIsErr: true, wantCatch: true},

		{name: "client error range", filename: "!400-499.pk", wantIsErr: true, wantMin: 400, wantMax: 499},
		{name: "server error range", filename: "!500-599.pk", wantIsErr: true, wantMin: 500, wantMax: 599},
		{name: "narrow range", filename: "!404-404.pk", wantIsErr: true, wantMin: 404, wantMax: 404},
		{name: "full range", filename: "!100-599.pk", wantIsErr: true, wantMin: 100, wantMax: 599},

		{name: "normal page", filename: "index.pk", wantIsErr: false},
		{name: "private partial", filename: "_private.pk", wantIsErr: false},
		{name: "exclamation in middle", filename: "page!404.pk", wantIsErr: false},

		{name: "non-numeric code", filename: "!abc.pk", wantIsErr: false},
		{name: "zero status", filename: "!0.pk", wantIsErr: false},
		{name: "below 100", filename: "!99.pk", wantIsErr: false},
		{name: "above 599", filename: "!600.pk", wantIsErr: false},
		{name: "negative code", filename: "!-1.pk", wantIsErr: false},
		{name: "no extension", filename: "!404", wantIsErr: false},
		{name: "wrong extension", filename: "!404.html", wantIsErr: false},
		{name: "just exclamation", filename: "!.pk", wantIsErr: false},
		{name: "empty string", filename: "", wantIsErr: false},
		{name: "inverted range min>max", filename: "!499-400.pk", wantIsErr: false},
		{name: "range below 100", filename: "!50-200.pk", wantIsErr: false},
		{name: "range above 599", filename: "!400-700.pk", wantIsErr: false},
		{name: "range non-numeric start", filename: "!abc-499.pk", wantIsErr: false},
		{name: "range non-numeric end", filename: "!400-abc.pk", wantIsErr: false},
		{name: "random word", filename: "!pancakes.pk", wantIsErr: false},
		{name: "hyphenated word", filename: "!hello-world.pk", wantIsErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isErrorPage(tt.filename)
			assert.Equal(t, tt.wantIsErr, result.isErrorPage, "isErrorPage mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantStatus, result.statusCode, "statusCode mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantMin, result.rangeMin, "rangeMin mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantMax, result.rangeMax, "rangeMax mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantCatch, result.isCatchAll, "isCatchAll mismatch for %q", tt.filename)
		})
	}
}
