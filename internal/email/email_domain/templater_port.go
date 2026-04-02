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

package email_domain

import (
	"context"
	"net/http"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/premailer"
)

// RenderedEmail represents the HTML and PlainText bodies produced by the templater
// for use in email sending. This keeps the email domain decoupled from the
// templater_domain package types.
type RenderedEmail struct {
	// HTML is the rendered HTML version of the email body.
	HTML string

	// PlainText is the plain text version of the email body.
	PlainText string

	// AttachmentRequests holds asset requests from <pml-img> tags for CID embedding.
	AttachmentRequests []*email_dto.EmailAssetRequest
}

// TemplaterAdapterPort provides the email templating capability needed by the
// email domain. Adapter implementations live in internal/email/email_adapters.
type TemplaterAdapterPort interface {
	// Render generates an email from a template with the given properties.
	//
	// Takes request (*http.Request) which provides the HTTP context for rendering.
	// Takes templatePath (string) which specifies the path to the email template.
	// Takes props (any) which contains the data to pass to the template.
	// Takes premailerOptions (*premailer.Options) which configures CSS inlining.
	//
	// Returns *RenderedEmail which contains the rendered email content.
	// Returns error when template rendering or processing fails.
	Render(ctx context.Context, request *http.Request, templatePath string, props any, premailerOptions *premailer.Options) (*RenderedEmail, error)
}
