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

package email_provider_mailchimp_transactional

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/logger"
)

const (
	// maxMailchimpResponseBytes caps the bytes read from a Mailchimp
	// Transactional reply, guarding against hostile or runaway responses while
	// staying generous enough for legitimate per-recipient result arrays.
	maxMailchimpResponseBytes = 4 * 1024 * 1024

	// defaultCallsPerSecond is the default rate limit. Mailchimp Transactional
	// allows 10 concurrent connections per API key, so a conservative per-second
	// limit keeps usage well within bounds.
	defaultCallsPerSecond = 10.0

	// defaultBurst is the default burst size for rate limiting.
	defaultBurst = 20

	// defaultBaseURL is the Mailchimp Transactional (Mandrill) API base URL.
	defaultBaseURL = "https://mandrillapp.com/api/1.0"

	// httpClientTimeout is the safety-net timeout for the HTTP client.
	httpClientTimeout = 30 * time.Second

	// httpMaxIdleConns is the maximum number of idle HTTP connections to keep.
	// Matches the Mailchimp Transactional limit of 10 concurrent connections
	// per API key.
	httpMaxIdleConns = 10

	// httpIdleConnTimeout is how long idle connections remain open before
	// being closed.
	httpIdleConnTimeout = 90 * time.Second

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

	// optionKeyTags is the provider option key for message tags.
	optionKeyTags = "tags"

	// optionKeyTrackOpens is the option key for open tracking.
	optionKeyTrackOpens = "track_opens"

	// optionKeyTrackClicks is the option key for click tracking.
	optionKeyTrackClicks = "track_clicks"

	// optionKeyMetadata is the option key for message metadata.
	optionKeyMetadata = "metadata"

	// optionKeyImportant is the option key for marking emails as important.
	optionKeyImportant = "important"

	// optionKeyAutoText is the option key for auto-generating text from HTML.
	optionKeyAutoText = "auto_text"

	// optionKeyInlineCSS is the option key for inlining CSS styles.
	optionKeyInlineCSS = "inline_css"

	// optionKeyHeaders is the option key for custom email headers.
	optionKeyHeaders = "headers"

	// optionKeyAsync is the option key for asynchronous sending.
	optionKeyAsync = "async"

	// logKeyTo is the log attribute key for email recipient addresses.
	logKeyTo = "to"
)

var _ email.ProviderPort = (*MailchimpTransactionalProvider)(nil)

// ErrMailchimpResponseTooLarge indicates the Mailchimp Transactional response
// body exceeded the maximum allowed size and was truncated before parsing.
var ErrMailchimpResponseTooLarge = errors.New("mailchimp transactional response body exceeded maximum allowed size")

// MailchimpTransactionalProvider implements the EmailProviderPort interface for
// the Mailchimp Transactional (formerly Mandrill) email service.
type MailchimpTransactionalProvider struct {
	// httpClient sends requests to the Mailchimp Transactional API.
	httpClient *http.Client

	// rateLimiter controls how fast requests can be sent to the API.
	rateLimiter *email_domain.ProviderRateLimiter

	// apiKey is the Mailchimp Transactional API key for authentication.
	apiKey string

	// fromEmail is the default sender email address for outgoing messages.
	fromEmail string

	// baseURL is the API base URL, defaulting to https://mandrillapp.com/api/1.0
	// but overridable for testing.
	baseURL string
}

// MailchimpTransactionalProviderArgs contains configuration for creating a new
// Mailchimp Transactional email provider.
type MailchimpTransactionalProviderArgs struct {
	// APIKey is the Mailchimp Transactional API key for authentication; must not
	// be empty.
	APIKey string

	// FromEmail is the default sender email address; must not be empty.
	FromEmail string
}

// ProviderOption is a functional option for setting up the Mailchimp
// Transactional provider.
type ProviderOption = email_domain.ProviderOption

// mandrillSendRequest is the JSON body for the /messages/send endpoint.
type mandrillSendRequest struct {
	// Key is the Mailchimp Transactional API key for authentication.
	Key string `json:"key"`

	// Message holds the email content, recipients, and sending options.
	Message mandrillMessage `json:"message"`

	// Async requests asynchronous delivery when true.
	Async bool `json:"async,omitempty"`
}

// mandrillMessage represents the message object in the Mandrill API request.
type mandrillMessage struct {
	// FromEmail is the sender email address for this message.
	FromEmail string `json:"from_email"`

	// Subject is the email subject line.
	Subject string `json:"subject"`

	// HTML is the HTML body content of the email.
	HTML string `json:"html,omitempty"`

	// Text is the plain-text body content of the email.
	Text string `json:"text,omitempty"`

	// To holds the list of recipients for this message.
	To []mandrillRecipient `json:"to"`

	// Headers holds custom email headers to include in the message.
	Headers map[string]string `json:"headers,omitempty"`

	// Metadata holds key-value pairs attached to the message for tracking.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Tags holds message tags used for filtering and reporting.
	Tags []string `json:"tags,omitempty"`

	// TrackOpens enables or disables open tracking for this message.
	TrackOpens *bool `json:"track_opens,omitempty"`

	// TrackClicks enables or disables click tracking for this message.
	TrackClicks *bool `json:"track_clicks,omitempty"`

	// Important marks the message as important in the recipient's inbox.
	Important *bool `json:"important,omitempty"`

	// AutoText enables automatic plain-text generation from the HTML body.
	AutoText *bool `json:"auto_text,omitempty"`

	// InlineCSS enables automatic inlining of CSS styles into the HTML body.
	InlineCSS *bool `json:"inline_css,omitempty"`

	// Attachments holds file attachments to include with the message.
	Attachments []mandrillAttachment `json:"attachments,omitempty"`

	// Images holds inline images referenced by Content-ID in the HTML body.
	Images []mandrillAttachment `json:"images,omitempty"`
}

// mandrillRecipient represents a single recipient in the Mandrill API.
type mandrillRecipient struct {
	// Email is the recipient's email address.
	Email string `json:"email"`

	// Name is the optional display name for the recipient.
	Name string `json:"name,omitempty"`

	// Type is the recipient type: "to", "cc", or "bcc".
	Type string `json:"type"`
}

// mandrillAttachment represents an attachment or inline image in the Mandrill
// API.
type mandrillAttachment struct {
	// Type is the MIME type of the attachment (e.g. "application/pdf").
	Type string `json:"type"`

	// Name is the filename or Content-ID for inline images.
	Name string `json:"name"`

	// Content is the base64-encoded attachment data.
	Content string `json:"content"`
}

// mandrillSendResponse represents a single recipient's send result.
type mandrillSendResponse struct {
	// Email is the recipient email address this result pertains to.
	Email string `json:"email"`

	// Status is the delivery status (e.g. "sent", "queued", "rejected").
	Status string `json:"status"`

	// ID is the Mailchimp Transactional message identifier.
	ID string `json:"_id"`
}

// mandrillErrorResponse represents an API error from Mandrill.
type mandrillErrorResponse struct {
	// Name is the error type identifier returned by the API.
	Name string `json:"name"`

	// Message is the human-readable error description.
	Message string `json:"message"`

	// Status is the error status string (e.g. "error").
	Status string `json:"status"`

	// Code is the numeric error code returned by the API.
	Code int `json:"code"`
}

// Send transmits a single email via Mailchimp Transactional.
//
// Takes params (*email_dto.SendParams) which specifies the email recipients,
// subject, and body content.
//
// Returns error when rate limiting fails, parameters are invalid, or the
// Mailchimp Transactional API request fails.
func (p *MailchimpTransactionalProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	startTime := time.Now()

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "MailchimpTransactional.Send"))
	contextLog.Trace("Sending email via Mailchimp Transactional",
		logger.Strings(logKeyTo, params.To),
		logger.String("subject", params.Subject))

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	if err := validateSendParams(params); err != nil {
		return err
	}

	request := p.buildMandrillRequest(params)
	results, err := p.doSendRequest(ctx, request)

	recordSendMetrics(ctx, startTime, err, metricSendTypeSingle)

	if err != nil {
		contextLog.ReportError(nil, err, "Failed to send email via Mailchimp Transactional",
			logger.Strings(logKeyTo, params.To),
			logger.String("subject", params.Subject))
		return fmt.Errorf("failed to send email via Mailchimp Transactional: %w", err)
	}

	var messageIDs []string
	for _, r := range results {
		if r.ID != "" {
			messageIDs = append(messageIDs, r.ID)
		}
	}

	contextLog.Trace("Email sent successfully via Mailchimp Transactional",
		logger.Strings("message_ids", messageIDs),
		logger.Strings(logKeyTo, params.To))

	return nil
}

// SupportsBulkSending reports whether the provider supports native bulk
// sending. Mailchimp Transactional does not have a separate bulk sending API,
// so bulk sends fall back to individual Send calls.
//
// Returns bool which is always false for this provider.
func (*MailchimpTransactionalProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by calling Send for each email individually.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when any sends fail, as a MultiError containing all failures.
func (p *MailchimpTransactionalProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	if len(emails) == 0 {
		return nil
	}

	startTime := time.Now()
	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "MailchimpTransactional.SendBulk"))
	contextLog.Trace("Sending emails via Mailchimp Transactional (individual sends)",
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

// Close releases resources held by the provider. The HTTP client is stateless,
// so the call does nothing.
//
// Returns error when cleanup fails, though this always returns nil.
func (*MailchimpTransactionalProvider) Close(_ context.Context) error {
	return nil
}

// NewMailchimpTransactionalProvider creates a new Mailchimp Transactional email
// provider with the given settings.
//
// Takes arguments (MailchimpTransactionalProviderArgs) which specifies the API
// key and sender details.
// Takes opts (...ProviderOption) which provides optional rate limiting
// settings.
//
// Returns email.ProviderPort which is the configured provider ready for use.
// Returns error when the API key or from email is empty.
func NewMailchimpTransactionalProvider(ctx context.Context, arguments MailchimpTransactionalProviderArgs, opts ...ProviderOption) (email.ProviderPort, error) {
	if arguments.APIKey == "" {
		return nil, errors.New("mailchimp transactional API key must not be empty")
	}
	if arguments.FromEmail == "" {
		return nil, errors.New("mailchimp transactional from email must not be empty")
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "NewMailchimpTransactionalProvider"))
	contextLog.Internal("Creating Mailchimp Transactional provider")

	httpClient := &http.Client{
		Timeout: httpClientTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        httpMaxIdleConns,
			MaxIdleConnsPerHost: httpMaxIdleConns,
			IdleConnTimeout:     httpIdleConnTimeout,
		},
	}

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &MailchimpTransactionalProvider{
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
		apiKey:      arguments.APIKey,
		fromEmail:   arguments.FromEmail,
		baseURL:     defaultBaseURL,
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

// buildMandrillRequest converts email parameters to a Mandrill API request.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// convert.
//
// Returns *mandrillSendRequest which is the formatted request ready for the
// Mailchimp Transactional API.
func (p *MailchimpTransactionalProvider) buildMandrillRequest(params *email_dto.SendParams) *mandrillSendRequest {
	from := p.fromEmail
	if params.From != nil {
		from = *params.From
	}

	attachments, images := buildAttachmentsAndImages(params.Attachments)

	message := mandrillMessage{
		FromEmail:   from,
		Subject:     params.Subject,
		HTML:        params.BodyHTML,
		Text:        params.BodyPlain,
		To:          buildRecipients(params),
		Attachments: attachments,
		Images:      images,
	}

	request := &mandrillSendRequest{
		Key:     p.apiKey,
		Message: message,
	}

	applyProviderOptions(&request.Message, request, params.ProviderOptions)

	return request
}

// buildRecipients converts To, Cc, and Bcc address lists into a unified
// Mandrill recipient array with type discriminators.
//
// Takes params (*email_dto.SendParams) which provides the recipient lists.
//
// Returns []mandrillRecipient which contains all recipients with their types.
func buildRecipients(params *email_dto.SendParams) []mandrillRecipient {
	recipients := make([]mandrillRecipient, 0, len(params.To)+len(params.Cc)+len(params.Bcc))

	for _, addr := range params.To {
		recipients = append(recipients, mandrillRecipient{Email: addr, Type: "to"})
	}
	for _, addr := range params.Cc {
		recipients = append(recipients, mandrillRecipient{Email: addr, Type: "cc"})
	}
	for _, addr := range params.Bcc {
		recipients = append(recipients, mandrillRecipient{Email: addr, Type: "bcc"})
	}

	return recipients
}

// buildAttachmentsAndImages converts email attachments to Mandrill format,
// splitting them into regular attachments and inline images based on whether
// a ContentID is present.
//
// Takes attachments ([]email_dto.Attachment) which provides the attachments to
// convert.
//
// Returns []mandrillAttachment for regular attachments and []mandrillAttachment
// for inline images.
func buildAttachmentsAndImages(attachments []email_dto.Attachment) (regular []mandrillAttachment, images []mandrillAttachment) {
	if len(attachments) == 0 {
		return nil, nil
	}

	for _, att := range attachments {
		converted := mandrillAttachment{
			Type:    att.MIMEType,
			Name:    att.Filename,
			Content: base64.StdEncoding.EncodeToString(att.Content),
		}

		if att.ContentID != "" {
			converted.Name = att.ContentID
			images = append(images, converted)
		} else {
			regular = append(regular, converted)
		}
	}

	return regular, images
}

// applyProviderOptions applies Mailchimp Transactional-specific options to the
// message and request.
//
// Takes message (*mandrillMessage) which is the message to configure.
// Takes request (*mandrillSendRequest) which is the request to configure.
// Takes options (map[string]any) which holds provider-specific settings.
func applyProviderOptions(message *mandrillMessage, request *mandrillSendRequest, options map[string]any) {
	if options == nil {
		return
	}

	if tags, ok := options[optionKeyTags].([]string); ok {
		message.Tags = tags
	}
	if trackOpens, ok := options[optionKeyTrackOpens].(bool); ok {
		message.TrackOpens = &trackOpens
	}
	if trackClicks, ok := options[optionKeyTrackClicks].(bool); ok {
		message.TrackClicks = &trackClicks
	}
	if metadata, ok := options[optionKeyMetadata].(map[string]string); ok {
		message.Metadata = metadata
	}
	if important, ok := options[optionKeyImportant].(bool); ok {
		message.Important = &important
	}
	if autoText, ok := options[optionKeyAutoText].(bool); ok {
		message.AutoText = &autoText
	}
	if inlineCSS, ok := options[optionKeyInlineCSS].(bool); ok {
		message.InlineCSS = &inlineCSS
	}
	if headers, ok := options[optionKeyHeaders].(map[string]string); ok {
		message.Headers = headers
	}
	if async, ok := options[optionKeyAsync].(bool); ok {
		request.Async = async
	}
}

// doSendRequest sends the email via the Mailchimp Transactional API.
//
// Takes request (*mandrillSendRequest) which contains the API request body.
//
// Returns []mandrillSendResponse which contains per-recipient results.
// Returns error when the HTTP request fails, the response cannot be parsed, or
// the API returns an error status.
func (p *MailchimpTransactionalProvider) doSendRequest(ctx context.Context, request *mandrillSendRequest) ([]mandrillSendResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := p.baseURL + "/messages/send"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "piko-email-provider/1.0")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, httpResp.Body)
		_ = httpResp.Body.Close()
	}()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxMailchimpResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if len(respBody) > maxMailchimpResponseBytes {
		return nil, fmt.Errorf("mailchimp transactional response truncated at %d bytes: %w", maxMailchimpResponseBytes, ErrMailchimpResponseTooLarge)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(respBody, httpResp.StatusCode)
	}

	var results []mandrillSendResponse
	if err := json.Unmarshal(respBody, &results); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if err := checkRecipientStatuses(results); err != nil {
		return results, err
	}

	return results, nil
}

// parseErrorResponse extracts error details from a non-200 API response.
//
// Takes body ([]byte) which contains the raw response body.
// Takes statusCode (int) which is the HTTP status code.
//
// Returns error which describes the API failure.
func parseErrorResponse(body []byte, statusCode int) error {
	var apiErr mandrillErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("mailchimp transactional API returned status %d: %s",
			statusCode, string(body))
	}
	return fmt.Errorf("mailchimp transactional API error: %s - %s (code: %d)",
		apiErr.Name, apiErr.Message, apiErr.Code)
}

// checkRecipientStatuses inspects per-recipient results and returns an error if
// any recipients were rejected or invalid.
//
// Takes results ([]mandrillSendResponse) which contains the per-recipient send
// results.
//
// Returns error when any recipient has a "rejected" or "invalid" status.
func checkRecipientStatuses(results []mandrillSendResponse) error {
	var failed []string
	for _, r := range results {
		if r.Status == "rejected" || r.Status == "invalid" {
			failed = append(failed, fmt.Sprintf("%s (%s)", r.Email, r.Status))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("mailchimp transactional: recipients failed: %s",
			strings.Join(failed, ", "))
	}
	return nil
}

// sendEmailsIndividually sends each email separately and collects any errors.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns *email_domain.MultiError which contains all send failures, or nil if
// all emails were sent successfully.
func (p *MailchimpTransactionalProvider) sendEmailsIndividually(ctx context.Context, emails []*email_dto.SendParams) *email_domain.MultiError {
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
