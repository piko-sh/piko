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

package capabilities_functions

import (
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

// minifierJS holds the pre-configured JavaScript minifier instance.
var minifierJS *minify.M

// MinifyJavascript returns a capability function that minifies JavaScript
// content using the tdewolff/minify library.
//
// Returns capabilities_domain.CapabilityFunc which performs JavaScript
// minification when invoked.
func MinifyJavascript() capabilities_domain.CapabilityFunc {
	return createMinifyCapability(minifyConfig{
		minifier:    minifierJS,
		spanName:    "MinifyJavaScript",
		mimeType:    "application/javascript",
		contentType: "JavaScript",
	})
}

func init() {
	minifierJS = minify.New()
	minifierJS.AddFunc("application/javascript", js.Minify)
}
