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

package templater_dto

import (
	"piko.sh/piko/internal/email/email_dto"
)

// RenderedEmailContent holds the output of a rendered email template.
// It includes both HTML and plain text versions, along with any needed assets.
type RenderedEmailContent struct {
	// HTML contains the rendered email content with CSS styles applied inline.
	HTML string

	// PlainText is the plain text version of the email content.
	PlainText string

	// CSS contains the stylesheet content that was extracted or inlined.
	CSS string

	// AttachmentRequests holds the list of assets to attach to the email.
	AttachmentRequests []*email_dto.EmailAssetRequest
}
