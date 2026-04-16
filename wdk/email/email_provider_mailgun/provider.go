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

package email_provider_mailgun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	mailgun "github.com/mailgun/mailgun-go/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/logger"
)

var _ email.ProviderPort = (*MailgunProvider)(nil)

const (
	// defaultCallsPerSecond is a conservative default rate limit for the
	// Mailgun API. Actual limits vary by plan.
	defaultCallsPerSecond = 50.0

	// defaultBurst is the default burst size for the token bucket rate limiter.
	defaultBurst = 100

	// metricKeyStatus is the metric attribute key for recording operation status.
	metricKeyStatus = "status"

	// metricKeySendType is the metric attribute key for the email send type.
	metricKeySendType = "send_type"

	// metricStatusSuccess is the metric attribute value for operations that
	// complete without error.
	metricStatusSuccess = "success"

	// metricStatusError is the metric attribute value for failed operations.
	metricStatusError = "error"

	// metricSendTypeSingle marks a single email send for metrics.
	metricSendTypeSingle = "single"

	// metricSendTypeBulk marks the metric as a bulk email send.
	metricSendTypeBulk = "bulk"

	// optionKeyTag is the provider option key for adding a tag.
	optionKeyTag = "tag"

	// optionKeyTracking is the option key for enabling or disabling tracking.
	optionKeyTracking = "tracking"

	// optionKeyTrackingClicks is the option key for click tracking.
	optionKeyTrackingClicks = "tracking_clicks"

	// optionKeyTrackingOpens is the option key for open tracking.
	optionKeyTrackingOpens = "tracking_opens"

	// optionKeyRequireTLS is the option key for requiring TLS delivery.
	optionKeyRequireTLS = "require_tls"

	// optionKeySkipVerification is the option key for skipping certificate
	// verification.
	optionKeySkipVerification = "skip_verification"

	// logKeyTo is the log attribute key for email recipient addresses.
	logKeyTo = "to"
)

// MailgunProvider implements the EmailProviderPort interface for the
// Mailgun email service.
type MailgunProvider struct {
	// client sends emails through the Mailgun API.
	client mailgun.Mailgun

	// rateLimiter controls how fast requests can be sent to the Mailgun API.
	rateLimiter *email_domain.ProviderRateLimiter

	// fromEmail is the default sender email address for outgoing messages.
	fromEmail string

	// domain is the Mailgun sending domain.
	domain string
}

// MailgunProviderArgs contains configuration for creating a new Mailgun email
// provider.
type MailgunProviderArgs struct {
	// Domain is the Mailgun sending domain (e.g. "mg.example.com"); must not
	// be empty.
	Domain string

	// APIKey is the Mailgun API key for authentication; must not be empty.
	APIKey string

	// FromEmail is the default sender email address; must not be empty.
	FromEmail string

	// APIBase is an optional override for the Mailgun API base URL, empty
	// for the default US endpoint.
	APIBase string
}

// ProviderOption is a functional option for setting up the Mailgun provider.
type ProviderOption = email_domain.ProviderOption

// NewMailgunProvider creates a new Mailgun email provider with the given
// settings.
//
// Takes arguments (MailgunProviderArgs) which specifies the Mailgun credentials,
// sending domain, and sender details.
// Takes opts (...ProviderOption) which provides optional rate limiting
// settings.
//
// Returns email.ProviderPort which is the configured provider ready for
// use.
// Returns error when the domain, API key, or from email is empty.
func NewMailgunProvider(ctx context.Context, arguments MailgunProviderArgs, opts ...ProviderOption) (email.ProviderPort, error) {
	if arguments.Domain == "" {
		return nil, errors.New("mailgun domain must not be empty")
	}
	if arguments.APIKey == "" {
		return nil, errors.New("mailgun API key must not be empty")
	}
	if arguments.FromEmail == "" {
		return nil, errors.New("mailgun from email must not be empty")
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "NewMailgunProvider"))
	contextLog.Internal("Creating Mailgun provider")

	mg := mailgun.NewMailgun(arguments.Domain, arguments.APIKey)
	if arguments.APIBase != "" {
		mg.SetAPIBase(arguments.APIBase)
	}

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &MailgunProvider{
		client:      mg,
		fromEmail:   arguments.FromEmail,
		domain:      arguments.Domain,
		rateLimiter: rateLimiter,
	}, nil
}

// Send transmits a single email via Mailgun.
//
// Takes params (*email_dto.SendParams) which specifies the email recipients,
// subject, and body content.
//
// Returns error when rate limiting fails, parameters are invalid, or the
// Mailgun API request fails.
func (p *MailgunProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	startTime := time.Now()

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "Mailgun.Send"))
	contextLog.Trace("Sending email via Mailgun",
		logger.Strings(logKeyTo, params.To),
		logger.String("subject", params.Subject))

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	if err := validateSendParams(params); err != nil {
		return err
	}

	message, err := p.buildMailgunMessage(params)
	if err != nil {
		return fmt.Errorf("building mailgun message: %w", err)
	}
	_, _, err = p.client.Send(ctx, message)

	recordSendMetrics(ctx, startTime, err, metricSendTypeSingle)

	if err != nil {
		contextLog.ReportError(nil, err, "Failed to send email via Mailgun",
			logger.Strings(logKeyTo, params.To),
			logger.String("subject", params.Subject))
		return fmt.Errorf("failed to send email via Mailgun: %w", err)
	}

	contextLog.Trace("Email sent successfully via Mailgun",
		logger.Strings(logKeyTo, params.To))

	return nil
}

// SupportsBulkSending reports whether the provider supports native bulk sending.
// Mailgun does not have a native bulk sending API for varied messages, so bulk
// sends fall back to individual Send calls.
//
// Returns bool which is always false for this provider.
func (*MailgunProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by calling Send for each email individually.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when any sends fail, as a MultiError containing all failures.
func (p *MailgunProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	if len(emails) == 0 {
		return nil
	}

	startTime := time.Now()
	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "Mailgun.SendBulk"))
	contextLog.Trace("Sending emails via Mailgun (individual sends)",
		logger.Int("count", len(emails)))

	multiError := p.sendEmailsIndividually(ctx, emails)

	recordBulkMetrics(ctx, startTime, len(emails), multiError)

	if multiError != nil && multiError.HasErrors() {
		return multiError
	}

	contextLog.Trace("Bulk email send completed successfully",
		logger.Int("total", len(emails)))

	return nil
}

// Close releases resources held by the provider. The Mailgun client is
// stateless, so this method does nothing.
//
// Returns error when cleanup fails, though this always returns nil.
func (*MailgunProvider) Close(_ context.Context) error {
	return nil
}

// buildMailgunMessage converts email parameters to a Mailgun API message.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// convert.
//
// Returns *mailgun.Message which is the formatted message ready for the Mailgun
// API.
// Returns error when provider options cannot be applied.
func (p *MailgunProvider) buildMailgunMessage(params *email_dto.SendParams) (*mailgun.Message, error) {
	from := p.fromEmail
	if params.From != nil {
		from = *params.From
	}

	message := mailgun.NewMessage(from, params.Subject, params.BodyPlain, params.To...)

	if params.BodyHTML != "" {
		message.SetHTML(params.BodyHTML)
	}

	addRecipients(message, params)
	addAttachments(message, params.Attachments)

	if err := applyProviderOptions(message, params.ProviderOptions); err != nil {
		return nil, err
	}

	return message, nil
}

// sendEmailsIndividually sends each email separately and collects any errors.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns *email_domain.MultiError which contains all send failures, or nil if
// all emails were sent successfully.
func (p *MailgunProvider) sendEmailsIndividually(ctx context.Context, emails []*email_dto.SendParams) *email_domain.MultiError {
	ctx, l := logger.From(ctx, log)
	var multiError *email_domain.MultiError

	for i, emailMessage := range emails {
		if err := p.Send(ctx, emailMessage); err != nil {
			l.ReportError(nil, err, "Failed to send email in bulk operation",
				logger.Int("email_index", i),
				logger.String("subject", emailMessage.Subject),
				logger.Strings(logKeyTo, emailMessage.To))

			now := time.Now()
			emailError := &email_domain.EmailError{
				Email:        *emailMessage,
				Error:        err,
				Attempt:      1,
				FirstAttempt: now,
				LastAttempt:  now,
				NextRetry:    time.Time{},
			}

			if multiError == nil {
				multiError = &email_domain.MultiError{}
			}
			multiError.Add(emailError)
		}
	}

	return multiError
}

// validateSendParams checks that the required fields in send parameters are
// present.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// check.
//
// Returns error when no recipients are provided or when both body fields are
// empty.
func validateSendParams(params *email_dto.SendParams) error {
	if len(params.To) == 0 {
		return email_domain.ErrRecipientRequired
	}
	if params.BodyHTML == "" && params.BodyPlain == "" {
		return email_domain.ErrBodyRequired
	}
	return nil
}

// addRecipients adds CC and BCC recipients to the email message.
//
// Takes message (*mailgun.Message) which is the email to modify.
// Takes params (*email_dto.SendParams) which contains the CC and BCC lists.
func addRecipients(message *mailgun.Message, params *email_dto.SendParams) {
	for _, cc := range params.Cc {
		message.AddCC(cc)
	}
	for _, bcc := range params.Bcc {
		message.AddBCC(bcc)
	}
}

// addAttachments converts email attachments to Mailgun format and adds them
// to the message.
//
// Takes message (*mailgun.Message) which receives the converted attachments.
// Takes attachments ([]email_dto.Attachment) which provides the attachments
// to convert.
func addAttachments(message *mailgun.Message, attachments []email_dto.Attachment) {
	for _, attachment := range attachments {
		if attachment.ContentID != "" {
			message.AddReaderInline(attachment.ContentID, io.NopCloser(bytes.NewReader(attachment.Content)))
		} else {
			message.AddBufferAttachment(attachment.Filename, attachment.Content)
		}
	}
}

// applyProviderOptions applies Mailgun-specific options to the message.
//
// Takes message (*mailgun.Message) which is the email to set up.
// Takes options (map[string]any) which holds provider-specific settings.
//
// Returns error when a provider option cannot be applied.
func applyProviderOptions(message *mailgun.Message, options map[string]any) error {
	if options == nil {
		return nil
	}

	if tag, ok := options[optionKeyTag].(string); ok {
		if err := message.AddTag(tag); err != nil {
			return fmt.Errorf("adding tag: %w", err)
		}
	}
	if tracking, ok := options[optionKeyTracking].(bool); ok {
		message.SetTracking(tracking)
	}
	if trackingClicks, ok := options[optionKeyTrackingClicks].(bool); ok {
		message.SetTrackingClicks(trackingClicks)
	}
	if trackingOpens, ok := options[optionKeyTrackingOpens].(bool); ok {
		message.SetTrackingOpens(trackingOpens)
	}
	if requireTLS, ok := options[optionKeyRequireTLS].(bool); ok {
		message.SetRequireTLS(requireTLS)
	}
	if skipVerification, ok := options[optionKeySkipVerification].(bool); ok {
		message.SetSkipVerification(skipVerification)
	}
	return nil
}

// recordSendMetrics records metrics for a send operation.
//
// Takes startTime (time.Time) which is when the send operation began.
// Takes err (error) which shows whether the operation failed.
// Takes sendType (string) which names the type of send operation.
func recordSendMetrics(ctx context.Context, startTime time.Time, err error, sendType string) {
	duration := float64(time.Since(startTime).Milliseconds())
	status := metricStatusSuccess
	count := int64(1)

	if err != nil {
		status = metricStatusError
	}

	attrs := metric.WithAttributes(
		attribute.String(metricKeyStatus, status),
		attribute.String(metricKeySendType, sendType),
	)

	SendTotal.Add(ctx, count, attrs)
	SendDuration.Record(ctx, duration, attrs)
}

// recordBulkMetrics records metrics for bulk email send operations.
//
// Takes startTime (time.Time) which marks when the bulk operation began.
// Takes emailCount (int) which is the total number of emails in the batch.
// Takes multiError (*email_domain.MultiError) which holds any errors from the
// bulk operation.
func recordBulkMetrics(ctx context.Context, startTime time.Time, emailCount int, multiError *email_domain.MultiError) {
	duration := float64(time.Since(startTime).Milliseconds())
	status := metricStatusSuccess

	if multiError != nil && multiError.HasErrors() {
		status = metricStatusError
	}

	attrs := metric.WithAttributes(
		attribute.String(metricKeyStatus, status),
		attribute.String(metricKeySendType, metricSendTypeBulk),
	)

	SendTotal.Add(ctx, int64(emailCount), attrs)
	SendDuration.Record(ctx, duration, attrs)
}
