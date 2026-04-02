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
	"fmt"
	"maps"
	"net/http"
	"time"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/premailer"
)

// baseEmailBuilder holds the shared state and settings for all email builders.
type baseEmailBuilder struct {
	// service provides email sending and dispatch operations.
	service *service

	// params holds the email settings as they are built.
	params *email_dto.SendParams

	// providerName specifies a non-default email provider; empty uses the default.
	providerName string

	// immediateSend skips the dispatcher queue when true.
	immediateSend bool
}

// EmailBuilder constructs emails using a fluent interface.
type EmailBuilder struct {
	*baseEmailBuilder
}

// TemplatedEmailBuilder composes emails using Piko templates with type-safe
// props.
type TemplatedEmailBuilder[PropsT any] struct {
	// templater renders email templates.
	templater TemplaterAdapterPort

	// assetResolver provides access to email template assets.
	assetResolver AssetResolverPort

	// templateProps holds the template properties used to generate the email.
	templateProps PropsT

	*baseEmailBuilder

	// premailerOptions sets options for CSS inlining; nil uses default settings.
	premailerOptions *premailer.Options

	// templateRequest is the HTTP request used to resolve template paths.
	templateRequest *http.Request

	// templatePath is the path to the email template file.
	templatePath string

	// useTemplate indicates whether to use a template for the email.
	useTemplate bool
}

// To adds recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) To(addresses ...string) *baseEmailBuilder {
	b.params.To = append(b.params.To, addresses...)
	return b
}

// Cc adds CC recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add as CC.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) Cc(addresses ...string) *baseEmailBuilder {
	b.params.Cc = append(b.params.Cc, addresses...)
	return b
}

// Bcc adds BCC recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add
// as BCC.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) Bcc(addresses ...string) *baseEmailBuilder {
	b.params.Bcc = append(b.params.Bcc, addresses...)
	return b
}

// From sets the sender email address.
//
// Takes address (string) which is the sender email address.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) From(address string) *baseEmailBuilder {
	b.params.From = &address
	return b
}

// Subject sets the email subject.
//
// Takes subject (string) which is the email subject line.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) Subject(subject string) *baseEmailBuilder {
	b.params.Subject = subject
	return b
}

// Attachment adds an attachment to the email.
//
// Takes filename (string) which specifies the attachment's display name.
// Takes mimeType (string) which specifies the MIME type of the content.
// Takes content ([]byte) which contains the attachment data.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) Attachment(filename, mimeType string, content []byte) *baseEmailBuilder {
	b.params.Attachments = append(b.params.Attachments, email_dto.Attachment{
		Filename: filename,
		MIMEType: mimeType,
		Content:  content,
	})
	return b
}

// Provider specifies a non-default provider for this email.
//
// Takes name (string) which identifies the provider to use.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) Provider(name string) *baseEmailBuilder {
	b.providerName = name
	return b
}

// Immediate forces the email to be sent immediately, bypassing the dispatcher
// queue.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) Immediate() *baseEmailBuilder {
	b.immediateSend = true
	return b
}

// ProviderOption sets a provider-specific option key/value to be passed through
// to the adapter.
//
// Takes key (string) which identifies the option name.
// Takes value (any) which provides the option value.
//
// Returns *baseEmailBuilder for method chaining.
func (b *baseEmailBuilder) ProviderOption(key string, value any) *baseEmailBuilder {
	if b.params.ProviderOptions == nil {
		b.params.ProviderOptions = make(map[string]any)
	}
	b.params.ProviderOptions[key] = value
	return b
}

// To adds recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) To(addresses ...string) *EmailBuilder {
	b.baseEmailBuilder.To(addresses...)
	return b
}

// Cc adds CC recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add as CC.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) Cc(addresses ...string) *EmailBuilder {
	b.baseEmailBuilder.Cc(addresses...)
	return b
}

// Bcc adds BCC recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add
// as BCC.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) Bcc(addresses ...string) *EmailBuilder {
	b.baseEmailBuilder.Bcc(addresses...)
	return b
}

// From sets the sender email address.
//
// Takes address (string) which is the sender email address.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) From(address string) *EmailBuilder {
	b.baseEmailBuilder.From(address)
	return b
}

// Subject sets the email subject.
//
// Takes subject (string) which is the email subject line.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) Subject(subject string) *EmailBuilder {
	b.baseEmailBuilder.Subject(subject)
	return b
}

// BodyHTML sets the HTML body content.
//
// Takes html (string) which is the HTML content for the email body.
//
// Returns *EmailBuilder which allows method chaining.
func (b *EmailBuilder) BodyHTML(html string) *EmailBuilder {
	b.params.BodyHTML = html
	return b
}

// BodyPlain sets the plain text body content.
//
// Takes plain (string) which is the plain text content for the email
// body.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) BodyPlain(plain string) *EmailBuilder {
	b.params.BodyPlain = plain
	return b
}

// Attachment adds an attachment to the email.
//
// Takes filename (string) which specifies the attachment's display name.
// Takes mimeType (string) which specifies the MIME type of the content.
// Takes content ([]byte) which contains the attachment data.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) Attachment(filename, mimeType string, content []byte) *EmailBuilder {
	b.baseEmailBuilder.Attachment(filename, mimeType, content)
	return b
}

// Provider specifies a non-default provider for this email.
//
// Takes name (string) which identifies the provider to use.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) Provider(name string) *EmailBuilder {
	b.baseEmailBuilder.Provider(name)
	return b
}

// Immediate forces the email to be sent immediately, bypassing the dispatcher
// queue.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) Immediate() *EmailBuilder {
	b.baseEmailBuilder.Immediate()
	return b
}

// ProviderOption sets a provider-specific option key/value to be passed through
// to the adapter.
//
// Takes key (string) which identifies the option name.
// Takes value (any) which provides the option value.
//
// Returns *EmailBuilder for method chaining.
func (b *EmailBuilder) ProviderOption(key string, value any) *EmailBuilder {
	b.baseEmailBuilder.ProviderOption(key, value)
	return b
}

// To adds recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) To(addresses ...string) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.To(addresses...)
	return b
}

// Cc adds CC recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add as CC.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Cc(addresses ...string) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.Cc(addresses...)
	return b
}

// Bcc adds BCC recipient email addresses.
//
// Takes addresses (...string) which are the email addresses to add
// as BCC.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Bcc(addresses ...string) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.Bcc(addresses...)
	return b
}

// From sets the sender email address.
//
// Takes address (string) which is the sender email address.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) From(address string) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.From(address)
	return b
}

// Subject sets the email subject.
//
// Takes subject (string) which is the email subject line.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Subject(subject string) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.Subject(subject)
	return b
}

// Props sets the strongly-typed props data for the template.
//
// Takes props (PropsT) which provides the template data.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Props(props PropsT) *TemplatedEmailBuilder[PropsT] {
	b.templateProps = props
	return b
}

// Request sets the HTTP request context for the template rendering.
//
// Takes request (*http.Request) which provides the HTTP context for
// resolving template paths.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Request(request *http.Request) *TemplatedEmailBuilder[PropsT] {
	b.templateRequest = request
	return b
}

// BodyTemplate sets the template path to use for rendering the email body.
// The template will be rendered when Send() is called.
//
// Takes templatePath (string) which is the path to the email template
// file.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) BodyTemplate(templatePath string) *TemplatedEmailBuilder[PropsT] {
	b.templatePath = templatePath
	b.useTemplate = true
	return b
}

// BodyPlain sets a custom plain text body that overrides the template's
// generated plain text.
//
// Takes plain (string) which is the plain text content to use.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) BodyPlain(plain string) *TemplatedEmailBuilder[PropsT] {
	b.params.BodyPlain = plain
	return b
}

// PremailerOptions allows overriding the default premailer settings for this
// specific email.
//
// Takes opts (premailer.Options) which provides the CSS inlining
// settings to use.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) PremailerOptions(opts premailer.Options) *TemplatedEmailBuilder[PropsT] {
	b.premailerOptions = &opts
	return b
}

// Attachment adds an attachment to the email.
//
// Takes filename (string) which specifies the attachment's display name.
// Takes mimeType (string) which specifies the MIME type of the content.
// Takes content ([]byte) which contains the attachment data.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Attachment(filename, mimeType string, content []byte) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.Attachment(filename, mimeType, content)
	return b
}

// Provider specifies a non-default provider for this email.
//
// Takes name (string) which identifies the provider to use.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Provider(name string) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.Provider(name)
	return b
}

// Immediate forces the email to be sent immediately, bypassing the dispatcher
// queue.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) Immediate() *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.Immediate()
	return b
}

// ProviderOption sets a provider-specific option to be passed through to
// the adapter.
//
// Takes key (string) which identifies the option name.
// Takes value (any) which provides the option value.
//
// Returns *TemplatedEmailBuilder[PropsT] for method chaining.
func (b *TemplatedEmailBuilder[PropsT]) ProviderOption(key string, value any) *TemplatedEmailBuilder[PropsT] {
	b.baseEmailBuilder.ProviderOption(key, value)
	return b
}

// dispatchOrSend decides whether to queue or send an email.
// It queues the email through the dispatcher if one is set, or sends the email
// straight away through a provider.
//
// Returns error when queuing fails or the provider cannot send the email.
func (b *baseEmailBuilder) dispatchOrSend(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	s := b.service

	if s.dispatcher != nil && !b.immediateSend {
		if b.providerName != "" && b.providerName != s.registry.GetDefaultProvider() {
			l.Warn("Sending immediately: specific provider requested, which bypasses the default dispatcher queue",
				logger_domain.String("provider", b.providerName))
			return s.sendImmediateWithProvider(ctx, b.providerName, b.params)
		}
		return s.dispatcher.Queue(ctx, b.params)
	}

	provider, err := s.getProvider(ctx, b.providerName)
	if err != nil {
		return fmt.Errorf("resolving email provider %q: %w", b.providerName, err)
	}
	if err := goroutine.SafeCall(ctx, "email.Send", func() error { return provider.Send(ctx, b.params) }); err != nil {
		return fmt.Errorf("sending email via provider: %w", err)
	}
	return nil
}

// Do checks and sends the email that has been built.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// By default, it uses the dispatcher if one is set up. This can be changed
// with methods like Immediate and Provider.
//
// Returns error when the email is not valid or sending fails.
func (b *EmailBuilder) Do(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before sending email: %w", err)
	}
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "EmailBuilder.Do", func(spanCtx context.Context, _ logger_domain.Logger) error {
		start := time.Now()
		defer func() {
			builderSendDuration.Record(spanCtx, float64(time.Since(start).Milliseconds()))
		}()
		builderSendCount.Add(spanCtx, 1)

		sanitiseRecipients(b.params)
		if err := validateSingle(b.params, b.service.config); err != nil {
			builderSendErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("email parameter validation failed: %w", err)
		}

		if err := b.dispatchOrSend(spanCtx); err != nil {
			builderSendErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("dispatching or sending email: %w", err)
		}
		return nil
	})
}

// getPremailerOptions returns the premailer settings for template rendering.
//
// Returns premailer.Options which contains the configured settings, or
// sensible defaults if none were set.
func (b *TemplatedEmailBuilder[PropsT]) getPremailerOptions() premailer.Options {
	if b.premailerOptions != nil {
		return *b.premailerOptions
	}
	return premailer.Options{
		ExpandShorthands:      true,
		MakeLeftoverImportant: true,
		KeepBangImportant:     true,
		Theme:                 map[string]string{},
	}
}

// renderTemplate renders the email template and populates the params with
// the result.
//
// Returns error when the templating service is not configured or rendering
// fails.
func (b *TemplatedEmailBuilder[PropsT]) renderTemplate(ctx context.Context) error {
	if b.templater == nil {
		return errTemplaterNotConfigured
	}

	userProvidedPlainText := b.params.BodyPlain != ""

	rendered, err := b.templater.Render(ctx, b.templateRequest, b.templatePath, b.templateProps, new(b.getPremailerOptions()))
	if err != nil {
		return fmt.Errorf("failed to render email template '%s': %w", b.templatePath, err)
	}

	b.params.BodyHTML = rendered.HTML
	if !userProvidedPlainText {
		b.params.BodyPlain = rendered.PlainText
	}

	b.resolveAndAttachAssets(ctx, rendered.AttachmentRequests)
	return nil
}

// Do checks and sends the email using the template.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// It renders the template just before sending. If a custom BodyPlain has been
// set, it will not be replaced by the template's plain text.
//
// Returns error when template rendering fails, parameter checking fails, or
// the email cannot be sent.
func (b *TemplatedEmailBuilder[PropsT]) Do(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before sending templated email: %w", err)
	}
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "TemplatedEmailBuilder.Do", func(spanCtx context.Context, _ logger_domain.Logger) error {
		start := time.Now()
		defer func() {
			builderSendDuration.Record(spanCtx, float64(time.Since(start).Milliseconds()))
		}()
		builderSendCount.Add(spanCtx, 1)

		if b.useTemplate {
			if err := b.renderTemplate(spanCtx); err != nil {
				builderSendErrorCount.Add(spanCtx, 1)
				return fmt.Errorf("rendering email template: %w", err)
			}
		}

		if err := spanCtx.Err(); err != nil {
			return fmt.Errorf("context cancelled before validating templated email: %w", err)
		}

		sanitiseRecipients(b.params)
		if err := validateSingle(b.params, b.service.config); err != nil {
			builderSendErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("email parameter validation failed: %w", err)
		}

		if err := b.dispatchOrSend(spanCtx); err != nil {
			builderSendErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("dispatching or sending templated email: %w", err)
		}
		return nil
	})
}

// resolveAndAttachAssets fetches email assets and attaches them to the email
// params. This helper method reduces nesting in the Send method.
//
// Takes requests ([]*email_dto.EmailAssetRequest) which lists the assets to
// fetch and attach.
func (b *TemplatedEmailBuilder[PropsT]) resolveAndAttachAssets(ctx context.Context, requests []*email_dto.EmailAssetRequest) {
	ctx, l := logger_domain.From(ctx, log)
	if len(requests) == 0 || b.assetResolver == nil {
		return
	}

	l.Trace("Resolving email assets for CID embedding",
		logger_domain.Int("asset_request_count", len(requests)))

	attachments, resolvedErrors := b.assetResolver.ResolveAssets(ctx, requests)

	failureCount := b.logAssetResolutionErrors(ctx, requests, resolvedErrors)

	b.attachResolvedAssets(ctx, attachments, failureCount)
}

// logAssetResolutionErrors logs any errors from asset resolution and returns
// the count of failures.
//
// Takes requests ([]*email_dto.EmailAssetRequest) which provides source paths
// and identifiers for error context.
// Takes errors ([]error) which contains resolution errors to check and log.
//
// Returns int which is the number of errors found.
func (*TemplatedEmailBuilder[PropsT]) logAssetResolutionErrors(ctx context.Context, requests []*email_dto.EmailAssetRequest, errors []error) int {
	ctx, l := logger_domain.From(ctx, log)
	failureCount := 0
	for i, err := range errors {
		if err != nil {
			failureCount++
			l.Warn("Failed to resolve email asset",
				logger_domain.String("source_path", requests[i].SourcePath),
				logger_domain.String("profile", requests[i].Profile),
				logger_domain.String("cid", requests[i].CID),
				logger_domain.Error(err),
			)
		}
	}
	return failureCount
}

// attachResolvedAssets adds resolved attachments to the email and logs the
// result.
//
// Takes attachments ([]*email_dto.Attachment) which contains the resolved
// assets to add.
// Takes failureCount (int) which tracks how many asset resolutions failed.
func (b *TemplatedEmailBuilder[PropsT]) attachResolvedAssets(ctx context.Context, attachments []*email_dto.Attachment, failureCount int) {
	ctx, l := logger_domain.From(ctx, log)
	if len(attachments) > 0 {
		for _, attachment := range attachments {
			if attachment != nil {
				b.params.Attachments = append(b.params.Attachments, *attachment)
			}
		}
		l.Trace("Successfully resolved and attached email assets",
			logger_domain.Int("success_count", len(attachments)),
			logger_domain.Int("failure_count", failureCount),
		)
		return
	}

	if failureCount > 0 {
		l.Warn("All email asset resolutions failed, email will be sent without embedded images",
			logger_domain.Int("total_failures", failureCount))
	}
}

// Build creates and returns a copy of the email parameters.
//
// Use it to create an email that can be changed before being added to a
// bulk send, without affecting the original builder.
//
// Returns email_dto.SendParams which contains the complete email settings.
func (b *EmailBuilder) Build() email_dto.SendParams {
	return buildParamsCopy(b.params)
}

// Clone creates a deep copy of the EmailBuilder, allowing for the creation
// of an email template that can be modified and used multiple times.
// This is the recommended approach for composing bulk emails.
//
// Returns *EmailBuilder which is an independent copy of the builder.
func (b *EmailBuilder) Clone() *EmailBuilder {
	return &EmailBuilder{
		baseEmailBuilder: &baseEmailBuilder{
			service:       b.service,
			params:        new(buildParamsCopy(b.params)),
			providerName:  b.providerName,
			immediateSend: b.immediateSend,
		},
	}
}

// Build returns the configured email parameters without rendering the template.
//
// The template is only rendered when Send is called on the returned parameters.
//
// Returns email_dto.SendParams which is a copy of the configured parameters.
func (b *TemplatedEmailBuilder[PropsT]) Build() email_dto.SendParams {
	return buildParamsCopy(b.params)
}

// Clone creates a deep copy of the TemplatedEmailBuilder, allowing for the
// creation of an email template that can be modified and used multiple times.
//
// Returns *TemplatedEmailBuilder[PropsT] which is an independent copy
// of the builder.
func (b *TemplatedEmailBuilder[PropsT]) Clone() *TemplatedEmailBuilder[PropsT] {
	var premailerOptsCopy *premailer.Options
	if b.premailerOptions != nil {
		premailerOptsCopy = new(*b.premailerOptions)
	}

	return &TemplatedEmailBuilder[PropsT]{
		baseEmailBuilder: &baseEmailBuilder{
			service:       b.service,
			params:        new(buildParamsCopy(b.params)),
			providerName:  b.providerName,
			immediateSend: b.immediateSend,
		},
		templater:        b.templater,
		assetResolver:    b.assetResolver,
		premailerOptions: premailerOptsCopy,
		templatePath:     b.templatePath,
		templateProps:    b.templateProps,
		templateRequest:  b.templateRequest,
		useTemplate:      b.useTemplate,
	}
}

// buildParamsCopy performs a deep copy of SendParams.
//
// Takes params (*email_dto.SendParams) which provides the source parameters
// to copy.
//
// Returns email_dto.SendParams which is an independent copy of the input.
func buildParamsCopy(params *email_dto.SendParams) email_dto.SendParams {
	paramsCopy := email_dto.SendParams{
		Subject:   params.Subject,
		BodyHTML:  params.BodyHTML,
		BodyPlain: params.BodyPlain,
	}

	if params.From != nil {
		paramsCopy.From = new(*params.From)
	}

	paramsCopy.To = make([]string, len(params.To))
	copy(paramsCopy.To, params.To)

	paramsCopy.Cc = make([]string, len(params.Cc))
	copy(paramsCopy.Cc, params.Cc)

	paramsCopy.Bcc = make([]string, len(params.Bcc))
	copy(paramsCopy.Bcc, params.Bcc)

	paramsCopy.Attachments = make([]email_dto.Attachment, len(params.Attachments))
	copy(paramsCopy.Attachments, params.Attachments)

	if params.ProviderOptions != nil {
		paramsCopy.ProviderOptions = make(map[string]any, len(params.ProviderOptions))
		maps.Copy(paramsCopy.ProviderOptions, params.ProviderOptions)
	}

	return paramsCopy
}
