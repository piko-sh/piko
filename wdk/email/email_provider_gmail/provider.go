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

package email_provider_gmail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wneessen/go-mail"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/wdk/logger"
)

const (
	// gmailHost is the Gmail SMTP server address.
	gmailHost = "smtp.gmail.com"

	// gmailPort is the SMTP port for Gmail with STARTTLS.
	gmailPort = 587

	// defaultCallsPerSecond is the default rate limit for Gmail SMTP to avoid
	// hitting sending quotas.
	defaultCallsPerSecond = 5.0

	// defaultBurst is the most API calls that can happen at once.
	defaultBurst = 10

	// metricAttrStatus is the metric attribute key for the operation status.
	metricAttrStatus = "status"

	// metricAttrSendType is the metric attribute key for the type of send operation.
	metricAttrSendType = "send_type"

	// metricStatusSuccess is the value used to mark a successful send in metrics.
	metricStatusSuccess = "success"

	// metricStatusError is the metric status value for failed operations.
	metricStatusError = "error"

	// metricSendTypeSingle is the metric label for single email send operations.
	metricSendTypeSingle = "single"

	// metricSendTypeBulk is the metric attribute value for bulk send operations.
	metricSendTypeBulk = "bulk"
)

// GmailProvider sends emails through Gmail SMTP and implements
// EmailProviderPort.
type GmailProvider struct {
	// rateLimiter controls how often emails are sent to avoid exceeding quotas.
	rateLimiter *email_domain.ProviderRateLimiter

	// username is the email address used for SMTP login.
	username string

	// password is the SMTP password for Gmail authentication.
	password string

	// fromEmail is the sender email address used when no override is given.
	fromEmail string
}

var _ email_domain.EmailProviderPort = (*GmailProvider)(nil)
var _ provider_domain.ProviderMetadata = (*GmailProvider)(nil)

// GmailProviderArgs holds the credentials needed to set up a Gmail provider.
type GmailProviderArgs struct {
	// Username is the Gmail account email address used for SMTP login.
	Username string

	// Password is the Gmail password or app-specific password; must not be empty.
	Password string

	// FromEmail is the sender address; defaults to the username if empty.
	FromEmail string
}

// NewGmailProvider creates a new Gmail email provider with the given settings.
//
// Takes arguments (GmailProviderArgs) which holds the Gmail
// username, password, and sender details.
// Takes opts (...email_domain.ProviderOption) which sets rate limiting options.
//
// Returns *GmailProvider which is ready to send emails via Gmail SMTP.
// Returns error when the username or password is empty.
func NewGmailProvider(ctx context.Context, arguments GmailProviderArgs, opts ...email_domain.ProviderOption) (*GmailProvider, error) {
	if arguments.Username == "" {
		return nil, errors.New("gmail username must not be empty")
	}
	if arguments.Password == "" {
		return nil, errors.New("gmail password must not be empty")
	}

	fromEmail := arguments.FromEmail
	if fromEmail == "" {
		fromEmail = arguments.Username
	}

	_, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "NewGmailProvider"))
	contextLog.Internal("Creating Gmail provider")

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
		Clock:          nil,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &GmailProvider{
		username:    arguments.Username,
		password:    arguments.Password,
		fromEmail:   fromEmail,
		rateLimiter: rateLimiter,
	}, nil
}

// GetProviderType returns the type identifier for this email provider.
//
// Returns string which is "gmail".
func (*GmailProvider) GetProviderType() string {
	return "gmail"
}

// GetProviderMetadata returns metadata about this Gmail email provider.
//
// Returns map[string]any which describes the provider configuration.
func (p *GmailProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"host":       gmailHost,
		"port":       gmailPort,
		"from_email": p.fromEmail,
	}
}

// Send constructs and sends an email via Gmail SMTP.
//
// Takes params (*email_dto.SendParams) which specifies the email recipients,
// subject, and body content.
//
// Returns error when rate limiting fails, parameters are invalid, the message
// cannot be built, or sending fails.
func (p *GmailProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	ctx, l := logger.From(ctx, log)
	startTime := time.Now()

	contextLog := l.With(logger.String("method", "Gmail.Send"))
	contextLog.Trace("Sending email via Gmail SMTP",
		logger.Strings("to", params.To),
		logger.String("subject", params.Subject),
	)

	if err := p.rateLimiter.Wait(ctx); err != nil {
		duration := float64(time.Since(startTime).Milliseconds())
		recordSendMetrics(ctx, duration, metricStatusError, metricSendTypeSingle)
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	if err := validateSendParams(params); err != nil {
		duration := float64(time.Since(startTime).Milliseconds())
		recordSendMetrics(ctx, duration, metricStatusError, metricSendTypeSingle)
		return fmt.Errorf("validating Gmail send parameters: %w", err)
	}

	message, err := p.buildMailMessage(params)
	if err != nil {
		duration := float64(time.Since(startTime).Milliseconds())
		recordSendMetrics(ctx, duration, metricStatusError, metricSendTypeSingle)
		return fmt.Errorf("failed to build mail message: %w", err)
	}

	if err := p.dialAndSend(message); err != nil {
		duration := float64(time.Since(startTime).Milliseconds())
		recordSendMetrics(ctx, duration, metricStatusError, metricSendTypeSingle)
		return fmt.Errorf("failed to send email via Gmail: %w", err)
	}

	duration := float64(time.Since(startTime).Milliseconds())
	recordSendMetrics(ctx, duration, metricStatusSuccess, metricSendTypeSingle)

	contextLog.Trace("Email sent successfully via Gmail SMTP")
	return nil
}

// Close does nothing as the SMTP client does not keep a connection open.
//
// Returns error which is always nil.
func (*GmailProvider) Close(_ context.Context) error {
	return nil
}

// SupportsBulkSending reports whether Gmail SMTP has a native bulk sending API.
//
// Returns bool which is false as Gmail does not support bulk sending.
func (*GmailProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by calling Send individually and collecting
// any errors.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when one or more emails fail to send. The error is a
// MultiError containing details of each failed email.
func (p *GmailProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	startTime := time.Now()

	if len(emails) == 0 {
		return nil
	}

	ctx, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "Gmail.SendBulk"))
	contextLog.Trace("Sending emails via Gmail SMTP (individual sends)", logger.Int("count", len(emails)))

	var multiError *email_domain.MultiError
	for i, email := range emails {
		if err := p.Send(ctx, email); err != nil {
			contextLog.ReportError(nil, err, "Failed to send email in bulk operation",
				logger.Int("email_index", i),
				logger.String("subject", email.Subject))

			emailError := &email_domain.EmailError{
				Email:       *email,
				Error:       err,
				Attempt:     1,
				LastAttempt: time.Now(),
			}

			if multiError == nil {
				multiError = &email_domain.MultiError{}
			}
			multiError.Add(emailError)
		}
	}

	duration := float64(time.Since(startTime).Milliseconds())
	status := metricStatusSuccess
	if multiError != nil && multiError.HasErrors() {
		status = metricStatusError
	}

	recordBulkSendMetrics(ctx, duration, len(emails), status)

	if multiError != nil && multiError.HasErrors() {
		return multiError
	}
	return nil
}

// Name returns the display name of this provider.
//
// Returns string which is the human-readable name for this provider.
func (*GmailProvider) Name() string {
	return "EmailProvider (Gmail)"
}

// Check performs a health probe and implements healthprobe_domain.Probe.
//
// When checkType is liveness, reports healthy since Gmail is a managed
// service. When checkType is readiness, verifies that credentials are
// configured.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which contains the health state and details.
func (p *GmailProvider) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   "Gmail provider operational",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	if p.username == "" || p.password == "" {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Gmail credentials not configured",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Gmail provider configured and ready",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// buildMailMessage constructs a new go-mail message from the given
// parameters.
//
// Takes params (*email_dto.SendParams) which provides the email
// content and recipients.
//
// Returns *mail.Msg which is the constructed message ready for
// sending.
// Returns error when addresses are invalid or attachments cannot be
// added.
func (p *GmailProvider) buildMailMessage(params *email_dto.SendParams) (*mail.Msg, error) {
	message := mail.NewMsg()

	if err := p.setMailAddresses(message, params); err != nil {
		return nil, fmt.Errorf("setting Gmail message addresses: %w", err)
	}

	message.Subject(params.Subject)
	setMailBody(message, params)

	if err := addMailAttachments(message, params.Attachments); err != nil {
		return nil, fmt.Errorf("adding Gmail message attachments: %w", err)
	}

	return message, nil
}

// setMailAddresses sets the from, to, cc, and bcc addresses on the message.
//
// Takes message (*mail.Message) which is the email message to configure.
// Takes params (*email_dto.SendParams) which provides the address fields.
//
// Returns error when any address is invalid.
func (p *GmailProvider) setMailAddresses(message *mail.Msg, params *email_dto.SendParams) error {
	from := p.fromEmail
	if params.From != nil {
		from = *params.From
	}

	if err := message.From(from); err != nil {
		return fmt.Errorf("set from address: %w", err)
	}
	if err := message.To(params.To...); err != nil {
		return fmt.Errorf("set to addresses: %w", err)
	}
	if len(params.Cc) > 0 {
		if err := message.Cc(params.Cc...); err != nil {
			return fmt.Errorf("set cc addresses: %w", err)
		}
	}
	if len(params.Bcc) > 0 {
		if err := message.Bcc(params.Bcc...); err != nil {
			return fmt.Errorf("set bcc addresses: %w", err)
		}
	}
	return nil
}

// dialAndSend creates a new SMTP client, connects to Gmail, and sends the
// message.
//
// Takes message (*mail.Message) which contains the email to send.
//
// Returns error when client creation or sending fails.
func (p *GmailProvider) dialAndSend(message *mail.Msg) error {
	client, err := mail.NewClient(gmailHost,
		mail.WithPort(gmailPort),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithUsername(p.username),
		mail.WithPassword(p.password),
	)
	if err != nil {
		return fmt.Errorf("create Gmail client: %w", err)
	}
	defer func() { _ = client.Close() }()

	return client.DialAndSend(message)
}

// setMailBody sets the body content of the email message.
//
// When both plain text and HTML bodies are provided, creates a multipart
// message with both formats. Otherwise uses whichever format is available.
//
// Takes message (*mail.Message) which is the email message to set up.
// Takes params (*email_dto.SendParams) which holds the body content.
func setMailBody(message *mail.Msg, params *email_dto.SendParams) {
	switch {
	case params.BodyHTML != "" && params.BodyPlain != "":
		message.SetBodyString(mail.TypeTextPlain, params.BodyPlain)
		message.AddAlternativeString(mail.TypeTextHTML, params.BodyHTML)
	case params.BodyHTML != "":
		message.SetBodyString(mail.TypeTextHTML, params.BodyHTML)
	default:
		message.SetBodyString(mail.TypeTextPlain, params.BodyPlain)
	}
}

// addMailAttachments adds attachments to the message, handling CID-embedded
// images separately.
//
// Takes message (*mail.Message) which is the email message to add attachments to.
// Takes attachments ([]email_dto.Attachment) which contains the files to add.
//
// Returns error when an attachment cannot be embedded or attached.
func addMailAttachments(message *mail.Msg, attachments []email_dto.Attachment) error {
	for _, attachment := range attachments {
		contentReader := bytes.NewReader(attachment.Content)
		if attachment.ContentID != "" {
			if err := message.EmbedReader(attachment.Filename, contentReader, mail.WithFileContentID(attachment.ContentID)); err != nil {
				return fmt.Errorf("embed attachment %q: %w", attachment.Filename, err)
			}
		} else {
			if err := message.AttachReader(attachment.Filename, contentReader); err != nil {
				return fmt.Errorf("attach file %q: %w", attachment.Filename, err)
			}
		}
	}
	return nil
}

// recordSendMetrics records metrics for a single email send.
//
// Takes duration (float64) which is the send time in milliseconds.
// Takes status (string) which is the outcome of the send.
// Takes sendType (string) which is the type of email sent.
func recordSendMetrics(ctx context.Context, duration float64, status string, sendType string) {
	sendTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendType),
	))
	sendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendType),
	))
}

// recordBulkSendMetrics records metrics for a bulk email send operation.
//
// Takes duration (float64) which is the time taken in milliseconds.
// Takes count (int) which is the number of emails sent.
// Takes status (string) which shows whether the send succeeded or failed.
func recordBulkSendMetrics(ctx context.Context, duration float64, count int, status string) {
	sendTotal.Add(ctx, int64(count), metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, metricSendTypeBulk),
	))
	sendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, metricSendTypeBulk),
	))
}

// validateSendParams checks that email send parameters are valid.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// check.
//
// Returns error when no recipients are provided or when both BodyHTML and
// BodyPlain are empty.
func validateSendParams(params *email_dto.SendParams) error {
	if len(params.To) == 0 {
		return email_domain.ErrRecipientRequired
	}
	if params.BodyHTML == "" && params.BodyPlain == "" {
		return email_domain.ErrBodyRequired
	}
	return nil
}
