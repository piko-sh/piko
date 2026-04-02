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

package utils

import "strings"

type ButtonVariant string

const (
	ButtonVariantPrimary ButtonVariant = "primary"

	ButtonVariantSecondary ButtonVariant = "secondary"

	ButtonVariantDanger ButtonVariant = "danger"
)

func FormatButtonLabel(label string) string {
	if label == "" {
		return "Click Here"
	}
	return strings.ToUpper(label[:1]) + strings.ToLower(label[1:])
}

func GetButtonClass(variant ButtonVariant) string {
	if variant == "" {
		variant = ButtonVariantPrimary
	}
	return "btn btn-" + string(variant)
}
