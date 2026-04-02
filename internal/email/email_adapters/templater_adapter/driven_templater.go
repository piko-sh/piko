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

package templater_adapter

import (
	"context"
	"fmt"
	"net/http"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/templater/templater_domain"
)

// Adapter implements email_domain.TemplaterAdapterPort by wrapping the
// templater_domain.EmailTemplateService. This keeps the email and templater
// domains separate while allowing integration through dependency injection.
type Adapter struct {
	// implementation is the template service that does the actual rendering.
	implementation templater_domain.EmailTemplateService
}

var _ email_domain.TemplaterAdapterPort = (*Adapter)(nil)

// Render delegates template rendering to the underlying EmailTemplateService
// and converts the result to the email domain's RenderedEmail type.
//
// Takes request (*http.Request) which provides the HTTP context for rendering.
// Takes templatePath (string) which specifies the path to the email template.
// Takes props (any) which contains the data to pass to the template.
// Takes opts (*premailer.Options) which configures CSS inlining behaviour.
//
// Returns *email_domain.RenderedEmail which contains the rendered HTML and
// plain text versions of the email.
// Returns error when the underlying service fails to render the template.
func (a *Adapter) Render(ctx context.Context, request *http.Request, templatePath string, props any, opts *premailer.Options) (*email_domain.RenderedEmail, error) {
	rendered, err := a.implementation.Render(ctx, request, templatePath, props, opts, false)
	if err != nil {
		return nil, fmt.Errorf("rendering email template %q: %w", templatePath, err)
	}
	return &email_domain.RenderedEmail{
		HTML:               rendered.HTML,
		PlainText:          rendered.PlainText,
		AttachmentRequests: rendered.AttachmentRequests,
	}, nil
}

// New creates a templater adapter that wraps the given EmailTemplateService.
//
// Takes implementation (EmailTemplateService) which provides the
// template service to wrap.
//
// Returns *Adapter which wraps the service for use in the application.
func New(implementation templater_domain.EmailTemplateService) *Adapter {
	return &Adapter{implementation: implementation}
}
