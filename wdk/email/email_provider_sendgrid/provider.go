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

package email_provider_sendgrid

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/email"
)

var _ email.ProviderPort = (*SendGridProvider)(nil)

const (
	// defaultCallsPerSecond is the SendGrid API rate limit in requests per second.
	defaultCallsPerSecond = 100.0

	// defaultBurst is the default burst size for the token bucket rate limiter.
	defaultBurst = 200

	// maxBatchSize is the largest number of emails that can be sent in one batch.
	maxBatchSize = 1000

	// httpStatusOK is the lowest HTTP status code that shows a successful response.
	httpStatusOK = 200

	// httpStatusError is the lowest HTTP status code that counts as an error.
	httpStatusError = 300

	// contentTypePlain is the MIME type for plain text email content.
	contentTypePlain = "text/plain"

	// contentTypeHTML is the MIME type for HTML email content.
	contentTypeHTML = "text/html"

	// dispositionInline is the content disposition for inline display.
	dispositionInline = "inline"

	// metricStatusSuccess is the metric attribute value for operations that succeed.
	metricStatusSuccess = "success"

	// metricStatusError is the status label for failed operations.
	metricStatusError = "error"

	// metricTypeSingle is the metric type for sending a single email.
	metricTypeSingle = "single"

	// metricTypeBulk is the metric type for bulk email operations.
	metricTypeBulk = "bulk"

	// logKeyMethod is the key used to store the method name in log entries.
	logKeyMethod = "method"
)

// SendGridProvider sends emails using the SendGrid API.
type SendGridProvider struct {
	// client is the SendGrid API client used to send emails.
	client *sendgrid.Client

	// rateLimiter controls how often API requests can be sent to SendGrid.
	rateLimiter *email_domain.ProviderRateLimiter

	// fromEmail is the default sender address used when no From field is given.
	fromEmail string
}

// SendGridProviderArgs holds settings for creating a SendGrid email provider.
type SendGridProviderArgs struct {
	// APIKey is the SendGrid API key for authentication; must not be empty.
	APIKey string

	// FromEmail is the sender email address; must not be empty.
	FromEmail string
}

// ProviderOption is a functional option for configuring the SendGrid provider.
type ProviderOption = email_domain.ProviderOption

// Close releases any resources held by the SendGrid provider.
//
// Returns error when resource cleanup fails. Currently always returns nil.
func (*SendGridProvider) Close(_ context.Context) error {
	return nil
}

// Send builds a SendGrid message and sends it using the client.
//
// Takes params (*email_dto.SendParams) which contains the email recipients,
// subject, and body content.
//
// Returns error when rate limiting fails, parameters are invalid, or the
// SendGrid API returns an error or a non-success status code.
func (p *SendGridProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	startTime := time.Now()

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String(logKeyMethod, "SendGrid.Send"))
	contextLog.Trace("Sending email via SendGrid",
		logger.Strings("to", params.To),
		logger.String("subject", params.Subject))

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	if err := validateSendParams(params); err != nil {
		return err
	}

	from := p.determineSender(params)
	message := buildSendGridMessage(from, params)

	response, sendErr := p.client.Send(message)
	duration := float64(time.Since(startTime).Milliseconds())

	var resultErr error
	if sendErr != nil {
		resultErr = fmt.Errorf("failed to send email via SendGrid: %w", sendErr)
	} else if !isSuccessStatusCode(response.StatusCode) {
		resultErr = fmt.Errorf("received non-success status code from SendGrid: %d, body: %s", response.StatusCode, response.Body)
	}

	recordSendMetrics(ctx, duration, resultErr, metricTypeSingle)
	return resultErr
}

// SupportsBulkSending reports whether SendGrid supports native bulk sending.
//
// Returns bool which is true because SendGrid supports bulk sending natively.
func (*SendGridProvider) SupportsBulkSending() bool {
	return true
}

// SendBulk sends multiple emails using SendGrid's bulk sending capabilities.
//
// Takes emails ([]*email_dto.SendParams) which contains the email messages to
// send.
//
// Returns error when any batch fails to send.
func (p *SendGridProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	startTime := time.Now()

	if len(emails) == 0 {
		return nil
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String(logKeyMethod, "SendGrid.SendBulk"))
	contextLog.Trace("Sending emails via SendGrid bulk API", logger.Int("count", len(emails)))

	batches := p.groupEmailsForBulkSending(emails)
	sendErr := p.sendBatches(ctx, batches)

	duration := float64(time.Since(startTime).Milliseconds())
	emailCount := int64(len(emails))
	recordBulkSendMetrics(ctx, duration, emailCount, sendErr)

	return sendErr
}

// determineSender returns the sender email address to use, preferring any
// override in the params.
//
// Takes params (*email_dto.SendParams) which may contain an override sender.
//
// Returns string which is the sender email address.
func (p *SendGridProvider) determineSender(params *email_dto.SendParams) string {
	if params.From != nil {
		return *params.From
	}
	return p.fromEmail
}

// sendBatches sends all batches and returns the first error encountered.
//
// Takes batches ([][]*email_dto.SendParams) which contains the grouped email
// parameters to send.
//
// Returns error when any batch fails to send.
func (p *SendGridProvider) sendBatches(ctx context.Context, batches [][]*email_dto.SendParams) error {
	for i, batch := range batches {
		if err := p.sendBulkBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to send bulk batch %d: %w", i, err)
		}
	}
	return nil
}

// groupEmailsForBulkSending splits emails into batches for efficient bulk
// sending.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to group.
//
// Returns [][]*email_dto.SendParams which contains batches of emails, each
// batch having at most maxBatchSize entries.
func (*SendGridProvider) groupEmailsForBulkSending(emails []*email_dto.SendParams) [][]*email_dto.SendParams {
	batches := make([][]*email_dto.SendParams, 0)

	for i := 0; i < len(emails); i += maxBatchSize {
		end := min(i+maxBatchSize, len(emails))
		batches = append(batches, emails[i:end])
	}

	return batches
}

// sendBulkBatch sends a batch of emails using SendGrid's personalisations
// feature when possible, falling back to individual sends when emails differ.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when the batch fails to send.
func (p *SendGridProvider) sendBulkBatch(ctx context.Context, emails []*email_dto.SendParams) error {
	if len(emails) == 0 {
		return nil
	}

	templateEmail := emails[0]

	from := p.fromEmail
	if templateEmail.From != nil {
		from = *templateEmail.From
	}

	message := mail.NewV3Mail()
	message.SetFrom(mail.NewEmail("", from))

	canUseBulkSending := p.canUseBulkSending(emails)

	if canUseBulkSending {
		return p.sendBulkWithPersonalisations(ctx, message, emails)
	}

	return p.sendIndividualEmailsInBatch(ctx, emails)
}

// canUseBulkSending checks if all emails can be sent as a true bulk operation
// by verifying they share the same subject, content, and sender.
//
// Takes emails ([]*email_dto.SendParams) which is the list of emails to check.
//
// Returns bool which is true if all emails have identical subjects, body
// content, and sender addresses; false if the list has one or fewer emails
// or if any email differs.
func (*SendGridProvider) canUseBulkSending(emails []*email_dto.SendParams) bool {
	if len(emails) <= 1 {
		return false
	}

	first := emails[0]
	for _, emailMessage := range emails[1:] {
		if emailMessage.Subject != first.Subject {
			return false
		}
		if emailMessage.BodyHTML != first.BodyHTML || emailMessage.BodyPlain != first.BodyPlain {
			return false
		}
		if (emailMessage.From == nil) != (first.From == nil) {
			return false
		}
		if emailMessage.From != nil && first.From != nil && *emailMessage.From != *first.From {
			return false
		}
	}
	return true
}

// sendBulkWithPersonalisations sends emails using SendGrid's personalisations
// feature to batch multiple recipients into a single API request.
//
// Takes message (*mail.SGMailV3) which is the base message to configure.
// Takes emails ([]*email_dto.SendParams) which contains the recipients and
// their personalisation data.
//
// Returns error when the SendGrid API call fails or returns a non-success
// status code.
func (p *SendGridProvider) sendBulkWithPersonalisations(ctx context.Context, message *mail.SGMailV3, emails []*email_dto.SendParams) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before sending bulk email: %w", err)
	}

	templateEmail := emails[0]
	message.Subject = templateEmail.Subject

	addContent(message, templateEmail)
	addBulkPersonalisations(message, emails)
	addAttachments(message, templateEmail)

	response, err := p.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send bulk email via SendGrid: %w", err)
	}

	if !isSuccessStatusCode(response.StatusCode) {
		return fmt.Errorf("received non-success status code from SendGrid bulk: %d, body: %s", response.StatusCode, response.Body)
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String(logKeyMethod, "sendBulkWithPersonalisations"))
	contextLog.Trace("Successfully sent bulk email batch", logger.Int("count", len(emails)))
	return nil
}

// sendIndividualEmailsInBatch sends emails individually when bulk sending is
// not possible.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when one or more emails fail to send.
func (p *SendGridProvider) sendIndividualEmailsInBatch(ctx context.Context, emails []*email_dto.SendParams) error {
	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String(logKeyMethod, "sendIndividualEmailsInBatch"))
	var multiError *email_domain.MultiError

	for i, emailMessage := range emails {
		if err := p.Send(ctx, emailMessage); err != nil {
			contextLog.ReportError(nil, err, "Failed to send email in bulk batch",
				logger.Int("email_index", i),
				logger.String("subject", emailMessage.Subject))

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

	if multiError != nil && multiError.HasErrors() {
		return multiError
	}

	return nil
}

// NewSendGridProvider creates a new SendGrid email provider with the given
// configuration.
//
// It validates the API key and 'From' email address, sets up the client, and
// configures rate limiting. SendGrid has a default rate limit of 100 calls per
// second based on their API limits.
//
// Takes arguments (SendGridProviderArgs) which provides the API key and sender
// email address.
// Takes opts (...ProviderOption) which allows overriding the
// default rate limit settings.
//
// Returns email.ProviderPort which is the configured provider ready for use.
// Returns error when the API key or 'From' email address is empty.
func NewSendGridProvider(ctx context.Context, arguments SendGridProviderArgs, opts ...ProviderOption) (email.ProviderPort, error) {
	if arguments.APIKey == "" {
		return nil, errors.New("SendGrid API key must not be empty")
	}
	if arguments.FromEmail == "" {
		return nil, errors.New("SendGrid from email must not be empty")
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String(logKeyMethod, "NewSendGridProvider"))
	contextLog.Internal("Creating SendGrid provider")

	sgClient := sendgrid.NewSendClient(arguments.APIKey)

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &SendGridProvider{
		client:      sgClient,
		fromEmail:   arguments.FromEmail,
		rateLimiter: rateLimiter,
	}, nil
}

// validateSendParams checks the send parameters for required fields.
//
// Takes params (*email_dto.SendParams) which contains the email parameters to
// check.
//
// Returns error when no recipients are given or when both body fields are
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

// buildSendGridMessage builds a SendGrid message from send parameters.
//
// Takes from (string) which specifies the sender email address.
// Takes params (*email_dto.SendParams) which contains the email details
// including recipients, subject, content, and attachments.
//
// Returns *mail.SGMailV3 which is the SendGrid message ready to send.
func buildSendGridMessage(from string, params *email_dto.SendParams) *mail.SGMailV3 {
	message := mail.NewV3Mail()
	message.SetFrom(mail.NewEmail("", from))
	message.Subject = params.Subject

	addRecipients(message, params)
	addContent(message, params)
	addAttachments(message, params)

	return message
}

// addRecipients adds To, Cc, and Bcc recipients to the message.
//
// Takes message (*mail.SGMailV3) which is the SendGrid message to change.
// Takes params (*email_dto.SendParams) which holds the recipient lists.
func addRecipients(message *mail.SGMailV3, params *email_dto.SendParams) {
	personalization := mail.NewPersonalization()

	for _, recipient := range params.To {
		personalization.AddTos(mail.NewEmail("", recipient))
	}
	for _, recipient := range params.Cc {
		personalization.AddCCs(mail.NewEmail("", recipient))
	}
	for _, recipient := range params.Bcc {
		personalization.AddBCCs(mail.NewEmail("", recipient))
	}

	message.AddPersonalizations(personalization)
}

// addContent adds plain text and HTML content to the email message.
//
// Takes message (*mail.SGMailV3) which is the email to add content to.
// Takes params (*email_dto.SendParams) which holds the plain and HTML body text.
func addContent(message *mail.SGMailV3, params *email_dto.SendParams) {
	if params.BodyPlain != "" {
		message.AddContent(mail.NewContent(contentTypePlain, params.BodyPlain))
	}
	if params.BodyHTML != "" {
		message.AddContent(mail.NewContent(contentTypeHTML, params.BodyHTML))
	}
}

// addAttachments adds file attachments to the email message.
//
// Attachments with a content ID are set as inline, which allows them to be
// shown within the email body rather than as separate files.
//
// Takes message (*mail.SGMailV3) which is the email to add attachments to.
// Takes params (*email_dto.SendParams) which contains the attachments to add.
func addAttachments(message *mail.SGMailV3, params *email_dto.SendParams) {
	for _, attachment := range params.Attachments {
		sgAttachment := mail.NewAttachment()
		sgAttachment.SetFilename(attachment.Filename)
		sgAttachment.SetType(attachment.MIMEType)
		sgAttachment.SetContent(base64.StdEncoding.EncodeToString(attachment.Content))

		if attachment.ContentID != "" {
			sgAttachment.SetContentID(attachment.ContentID)
			sgAttachment.SetDisposition(dispositionInline)
		}

		message.AddAttachment(sgAttachment)
	}
}

// isSuccessStatusCode checks if the HTTP status code shows success.
//
// Takes statusCode (int) which is the HTTP status code to check.
//
// Returns bool which is true if the status code is in the 2xx range.
func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= httpStatusOK && statusCode < httpStatusError
}

// recordSendMetrics records metrics for a send operation.
//
// Takes duration (float64) which is how long the send took.
// Takes err (error) which sets the status to success or error.
// Takes sendType (string) which is the kind of send operation.
func recordSendMetrics(ctx context.Context, duration float64, err error, sendType string) {
	status := metricStatusSuccess
	if err != nil {
		status = metricStatusError
	}

	attrs := metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("send_type", sendType),
	)

	if sendType == metricTypeSingle {
		SendTotal.Add(ctx, 1, attrs)
	}
	SendDuration.Record(ctx, duration, attrs)
}

// recordBulkSendMetrics records metrics for a bulk send operation.
//
// Takes duration (float64) which is the time taken in seconds.
// Takes emailCount (int64) which is the number of emails sent.
// Takes err (error) which sets the status to success or error.
func recordBulkSendMetrics(ctx context.Context, duration float64, emailCount int64, err error) {
	status := metricStatusSuccess
	if err != nil {
		status = metricStatusError
	}

	attrs := metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("send_type", metricTypeBulk),
	)

	SendTotal.Add(ctx, emailCount, attrs)
	SendDuration.Record(ctx, duration, attrs)
}

// addBulkPersonalisations adds recipient details to each email in a bulk send.
//
// Takes message (*mail.SGMailV3) which is the message to add recipients to.
// Takes emails ([]*email_dto.SendParams) which contains the recipient details
// for each email in the batch.
func addBulkPersonalisations(message *mail.SGMailV3, emails []*email_dto.SendParams) {
	for _, emailMessage := range emails {
		personalization := mail.NewPersonalization()

		for _, recipient := range emailMessage.To {
			personalization.AddTos(mail.NewEmail("", recipient))
		}
		for _, recipient := range emailMessage.Cc {
			personalization.AddCCs(mail.NewEmail("", recipient))
		}
		for _, recipient := range emailMessage.Bcc {
			personalization.AddBCCs(mail.NewEmail("", recipient))
		}

		message.AddPersonalizations(personalization)
	}
}
