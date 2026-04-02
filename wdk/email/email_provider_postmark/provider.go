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

package email_provider_postmark

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/postmark"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/email"
)

var _ email.ProviderPort = (*PostmarkProvider)(nil)

const (
	// defaultCallsPerSecond is the default Postmark API rate limit for standard
	// accounts. See Postmark support article 1008 for details.
	defaultCallsPerSecond = 100.0

	// defaultBurst is the default burst size for rate limiting.
	defaultBurst = 200

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

	// optionKeyTag is the provider option key for the email tag.
	optionKeyTag = "tag"

	// optionKeyTrackOpens is the option key for turning on open tracking.
	optionKeyTrackOpens = "track_opens"

	// optionKeyTrackLinks is the option key for turning on link tracking in emails.
	optionKeyTrackLinks = "track_links"

	// optionKeyMessageStream is the option key for setting the message stream.
	optionKeyMessageStream = "message_stream"

	// logKeyTo is the log attribute key for email recipient addresses.
	logKeyTo = "to"
)

// PostmarkProvider implements the EmailProviderPort interface for the
// Postmark email service.
type PostmarkProvider struct {
	// client sends emails through the Postmark API.
	client *postmark.Client

	// rateLimiter controls how fast requests can be sent to the Postmark API.
	rateLimiter *email_domain.ProviderRateLimiter

	// fromEmail is the default sender email address for outgoing messages.
	fromEmail string

	// serverToken is the Postmark server API token used to send emails.
	serverToken string

	// accountToken is the Postmark account API token used for authentication.
	accountToken string
}

// PostmarkProviderArgs contains configuration for creating a new Postmark email
// provider.
type PostmarkProviderArgs struct {
	// AccountToken is the Postmark account API token used for authentication.
	AccountToken string

	// ServerToken is the Postmark server API token for sending emails.
	ServerToken string

	// FromEmail is the default sender email address; must not be empty.
	FromEmail string
}

// ProviderOption is a functional option for setting up the Postmark provider.
type ProviderOption = email_domain.ProviderOption

// Send transmits a single email via Postmark.
//
// Takes params (*email_dto.SendParams) which specifies the email recipients,
// subject, and body content.
//
// Returns error when rate limiting fails, parameters are invalid, or the
// Postmark API request fails.
func (p *PostmarkProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	startTime := time.Now()

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "Postmark.Send"))
	contextLog.Trace("Sending email via Postmark",
		logger.Strings(logKeyTo, params.To),
		logger.String("subject", params.Subject))

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	if err := validateSendParams(params); err != nil {
		return err
	}

	message := p.buildPostmarkMessage(params)
	response, err := p.client.SendEmail(ctx, message)

	recordSendMetrics(ctx, startTime, err, metricSendTypeSingle)

	if err != nil {
		contextLog.ReportError(nil, err, "Failed to send email via Postmark",
			logger.Strings(logKeyTo, params.To),
			logger.String("subject", params.Subject))
		return fmt.Errorf("failed to send email via Postmark: %w", err)
	}

	contextLog.Trace("Email sent successfully via Postmark",
		logger.String("message_id", response.MessageID),
		logger.Strings(logKeyTo, params.To))

	return nil
}

// SupportsBulkSending reports whether the provider supports native bulk sending.
// Postmark does not have a native bulk sending API, so bulk sends fall back to
// individual Send calls.
//
// Returns bool which is always false for this provider.
func (*PostmarkProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by calling Send for each email individually.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when any sends fail, as a MultiError containing all failures.
func (p *PostmarkProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	if len(emails) == 0 {
		return nil
	}

	startTime := time.Now()
	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "Postmark.SendBulk"))
	contextLog.Trace("Sending emails via Postmark (individual sends)",
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

// Close releases resources held by the provider by closing idle HTTP
// connections in the underlying transport.
//
// Returns error when cleanup fails, though this always returns nil.
func (p *PostmarkProvider) Close(_ context.Context) error {
	if p.client != nil && p.client.HTTPClient != nil {
		p.client.HTTPClient.CloseIdleConnections()
	}
	return nil
}

// buildPostmarkMessage converts email parameters to Postmark API format.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// convert.
//
// Returns postmark.Email which is the formatted message ready for the Postmark
// API.
func (p *PostmarkProvider) buildPostmarkMessage(params *email_dto.SendParams) postmark.Email {
	from := p.fromEmail
	if params.From != nil {
		from = *params.From
	}

	message := postmark.Email{
		From:          from,
		To:            strings.Join(params.To, ","),
		Cc:            "",
		Bcc:           "",
		Subject:       params.Subject,
		Tag:           "",
		HTMLBody:      params.BodyHTML,
		TextBody:      params.BodyPlain,
		ReplyTo:       "",
		Headers:       nil,
		TrackOpens:    true,
		TrackLinks:    "",
		Attachments:   nil,
		Metadata:      nil,
		MessageStream: "",
		InlineCSS:     false,
	}

	addRecipients(&message, params)
	addAttachments(&message, params.Attachments)
	applyProviderOptions(&message, params.ProviderOptions)

	return message
}

// sendEmailsIndividually sends each email separately and collects any errors.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns *email_domain.MultiError which contains all send failures, or nil if
// all emails were sent successfully.
func (p *PostmarkProvider) sendEmailsIndividually(ctx context.Context, emails []*email_dto.SendParams) *email_domain.MultiError {
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

// NewPostmarkProvider creates a new Postmark email provider with the given
// settings.
//
// Takes arguments (PostmarkProviderArgs) which specifies the Postmark credentials
// and sender details.
// Takes opts (...ProviderOption) which provides optional rate limiting
// settings.
//
// Returns email.ProviderPort which is the configured provider ready for
// use.
// Returns error when the server token or from email is empty.
func NewPostmarkProvider(ctx context.Context, arguments PostmarkProviderArgs, opts ...ProviderOption) (email.ProviderPort, error) {
	if arguments.ServerToken == "" {
		return nil, errors.New("postmark server token must not be empty")
	}
	if arguments.FromEmail == "" {
		return nil, errors.New("postmark from email must not be empty")
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "NewPostmarkProvider"))
	contextLog.Internal("Creating Postmark provider")

	client := postmark.NewClient(arguments.ServerToken, arguments.AccountToken)

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &PostmarkProvider{
		client:       client,
		fromEmail:    arguments.FromEmail,
		serverToken:  arguments.ServerToken,
		accountToken: arguments.AccountToken,
		rateLimiter:  rateLimiter,
	}, nil
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
// Takes message (*postmark.Email) which is the email to modify.
// Takes params (*email_dto.SendParams) which contains the CC and BCC lists.
func addRecipients(message *postmark.Email, params *email_dto.SendParams) {
	if len(params.Cc) > 0 {
		message.Cc = strings.Join(params.Cc, ",")
	}
	if len(params.Bcc) > 0 {
		message.Bcc = strings.Join(params.Bcc, ",")
	}
}

// addAttachments converts email attachments to Postmark format and adds them
// to the message.
//
// Takes message (*postmark.Email) which receives the converted attachments.
// Takes attachments ([]email_dto.Attachment) which provides the attachments
// to convert.
func addAttachments(message *postmark.Email, attachments []email_dto.Attachment) {
	if len(attachments) == 0 {
		return
	}

	message.Attachments = make([]postmark.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		contentID := ""
		if attachment.ContentID != "" {
			contentID = "cid:" + attachment.ContentID
		}

		pmAttachment := postmark.Attachment{
			Name:        attachment.Filename,
			Content:     base64.StdEncoding.EncodeToString(attachment.Content),
			ContentType: attachment.MIMEType,
			ContentID:   contentID,
		}

		message.Attachments = append(message.Attachments, pmAttachment)
	}
}

// applyProviderOptions applies Postmark-specific options to the message.
//
// Takes message (*postmark.Email) which is the email to set up.
// Takes options (map[string]any) which holds provider-specific settings.
func applyProviderOptions(message *postmark.Email, options map[string]any) {
	if options == nil {
		return
	}

	if tag, ok := options[optionKeyTag].(string); ok {
		message.Tag = tag
	}
	if trackOpens, ok := options[optionKeyTrackOpens].(bool); ok {
		message.TrackOpens = trackOpens
	}
	if trackLinks, ok := options[optionKeyTrackLinks].(string); ok {
		message.TrackLinks = trackLinks
	}
	if messageStream, ok := options[optionKeyMessageStream].(string); ok {
		message.MessageStream = messageStream
	}
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
