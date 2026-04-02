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

package email_provider_smtp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
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
	// maxSMTPPort is the highest valid TCP port number.
	maxSMTPPort = 65535

	// defaultCallsPerSecond is the conservative default rate limit for SMTP servers.
	// This is a safe starting point for most general-purpose SMTP servers.
	defaultCallsPerSecond = 10.0

	// defaultBurst is the maximum number of SMTP calls allowed in a single burst.
	defaultBurst = 20

	// statusSuccess indicates the operation finished without error.
	statusSuccess = "success"

	// statusError indicates that the operation has failed.
	statusError = "error"
)

// SMTPProvider wraps an SMTP client and manages a persistent connection.
// It reuses the connection for multiple sends and implements
// email.ProviderPort and io.Closer.
type SMTPProvider struct {
	// rateLimiter controls the rate of email sending to avoid exceeding
	// provider limits.
	rateLimiter *email_domain.ProviderRateLimiter

	// client is the cached SMTP connection; nil when not connected.
	client *mail.Client

	// host is the SMTP server hostname or IP address.
	host string

	// username is the SMTP authentication username.
	username string

	// password is the SMTP authentication password.
	password string

	// fromEmail is the default sender address for outgoing messages.
	fromEmail string

	// port is the TCP port number for the SMTP server.
	port int

	// mu guards access to the client field during connection handling.
	mu sync.Mutex
}

// SMTPProviderArgs holds the settings needed to create a new SMTPProvider.
type SMTPProviderArgs struct {
	// Host is the SMTP server hostname or IP address; must not be empty.
	Host string

	// Username is the SMTP authentication username.
	Username string

	// Password is the SMTP authentication password.
	Password string

	// FromEmail is the sender email address.
	FromEmail string

	// Port is the TCP port number for the SMTP server.
	Port int
}

var _ email_domain.EmailProviderPort = (*SMTPProvider)(nil)
var _ provider_domain.ProviderMetadata = (*SMTPProvider)(nil)

// NewSMTPProvider creates a new SMTP email provider with the given settings.
//
// The provider is prepared but does not establish a connection until the
// first Send call. A conservative default rate limit is applied, suitable
// for most shared SMTP servers.
//
// Takes arguments (SMTPProviderArgs) which specifies the SMTP server settings.
// Takes opts (...email_domain.ProviderOption) which provides optional
// rate limit configuration.
//
// Returns *SMTPProvider which is the configured provider ready for use.
// Returns error when the host is empty or port is invalid.
func NewSMTPProvider(_ context.Context, arguments SMTPProviderArgs, opts ...email_domain.ProviderOption) (*SMTPProvider, error) {
	if arguments.Host == "" || arguments.Port <= 0 || arguments.Port > maxSMTPPort {
		return nil, errors.New("invalid SMTP host or port provided")
	}

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
		Clock:          nil,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &SMTPProvider{
		host:        arguments.Host,
		port:        arguments.Port,
		username:    arguments.Username,
		password:    arguments.Password,
		fromEmail:   arguments.FromEmail,
		rateLimiter: rateLimiter,
	}, nil
}

// GetProviderType returns the type identifier for this email provider.
//
// Returns string which is "smtp".
func (*SMTPProvider) GetProviderType() string {
	return "smtp"
}

// GetProviderMetadata returns metadata about this SMTP email provider.
//
// Returns map[string]any which describes the provider configuration.
func (p *SMTPProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"host":       p.host,
		"port":       p.port,
		"from_email": p.fromEmail,
	}
}

// Send constructs and sends an email over a persistent SMTP connection.
// If the connection is not active, it will be established automatically.
//
// Takes params (*email_dto.SendParams) which specifies the email recipients,
// subject, and body content.
//
// Returns error when rate limiting fails, message building fails, the SMTP
// client cannot connect, or the email transmission fails.
func (p *SMTPProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	ctx, l := logger.From(ctx, log)
	startTime := time.Now()
	contextLog := l.With(logger.String("method", "SMTP.Send"))

	status := statusSuccess
	defer func() {
		duration := float64(time.Since(startTime).Milliseconds())
		attrs := metric.WithAttributes(
			attribute.String("status", status),
			attribute.String("send_type", "single"),
		)
		sendTotal.Add(ctx, 1, attrs)
		sendDuration.Record(ctx, duration, attrs)
	}()

	if err := p.rateLimiter.Wait(ctx); err != nil {
		status = statusError
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	message, err := p.buildMessage(params)
	if err != nil {
		status = statusError
		return fmt.Errorf("failed to build mail message: %w", err)
	}

	client, err := p.getClient(ctx)
	if err != nil {
		status = statusError
		return fmt.Errorf("failed to get or create SMTP client: %w", err)
	}

	if err := client.Send(message); err != nil {
		status = statusError
		return fmt.Errorf("failed to send email over persistent connection: %w", err)
	}

	contextLog.Trace("Email sent successfully via persistent SMTP connection",
		logger.Strings("to", params.To),
		logger.String("subject", params.Subject),
	)
	return nil
}

// Close ends the persistent SMTP connection.
// This should be called during application shutdown.
//
// Returns error when the connection fails to close cleanly.
//
// Safe for concurrent use.
func (p *SMTPProvider) Close(ctx context.Context) error {
	_, l := logger.From(ctx, log)
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		l.Internal("Closing persistent SMTP connection", logger.String("host", p.host))
		err := p.client.Close()
		p.client = nil
		if err != nil {
			return fmt.Errorf("closing SMTP connection to %s: %w", p.host, err)
		}
	}

	return nil
}

// SupportsBulkSending indicates that the generic SMTP protocol does not have
// a native bulk sending API. The provider will send emails one by one over its
// persistent connection.
//
// Returns bool which is always false for this provider.
func (*SMTPProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by calling Send individually.
//
// Because this provider uses a persistent connection, this is still highly
// efficient. Errors are collected into a MultiError.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when one or more emails fail to send.
func (p *SMTPProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	startTime := time.Now()

	if len(emails) == 0 {
		return nil
	}

	ctx, l := logger.From(ctx, log)
	contextLog := l.With(logger.String("method", "SMTP.SendBulk"))
	contextLog.Trace("Sending emails via SMTP (individual sends over persistent connection)", logger.Int("count", len(emails)))

	status := statusSuccess
	defer func() {
		duration := float64(time.Since(startTime).Milliseconds())
		attrs := metric.WithAttributes(
			attribute.String("status", status),
			attribute.String("send_type", "bulk"),
		)
		sendTotal.Add(ctx, int64(len(emails)), attrs)
		sendDuration.Record(ctx, duration, attrs)
	}()

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

	if multiError != nil && multiError.HasErrors() {
		status = statusError
		return multiError
	}

	return nil
}

// Name returns the display name of this provider.
//
// Returns string which is the human-readable name for this provider.
func (*SMTPProvider) Name() string {
	return "EmailProvider (SMTP)"
}

// Check implements the healthprobe_domain.Probe interface and performs health
// checks on the SMTP provider. For liveness it verifies configuration, and
// for readiness it checks the client connection.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which contains the health state and details.
//
// Safe for concurrent use; uses mutex protection when checking connection
// state.
func (p *SMTPProvider) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if p.host == "" || p.port <= 0 {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "SMTP host or port not configured",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   fmt.Sprintf("SMTP provider configured for %s:%d", p.host, p.port),
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	p.mu.Lock()
	clientConnected := p.client != nil
	p.mu.Unlock()

	if !clientConnected {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateDegraded,
			Message:   "SMTP client not connected (will connect on first send)",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   fmt.Sprintf("SMTP client connected to %s:%d", p.host, p.port),
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// getClient provides a thread-safe way to get a connected mail.Client.
// If the client is not connected or the connection has been lost, it will
// attempt to establish a new persistent connection.
//
// Returns *mail.Client which is a connected and healthy SMTP client.
// Returns error when the client cannot be created or the connection fails.
func (p *SMTPProvider) getClient(ctx context.Context) (*mail.Client, error) {
	ctx, l := logger.From(ctx, log)
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		resetErr := p.client.Reset()
		if resetErr == nil {
			return p.client, nil
		}
		l.Trace("SMTP connection reset failed, reconnecting",
			logger.String("host", p.host),
			logger.Error(resetErr))
		_ = p.client.Close()
		p.client = nil
	}

	client, err := mail.NewClient(p.host,
		mail.WithPort(p.port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(p.username),
		mail.WithPassword(p.password),
		mail.WithTLSPortPolicy(mail.TLSMandatory),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new mail client config: %w", err)
	}

	if err := client.DialWithContext(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to dial SMTP server %s:%d: %w", p.host, p.port, err)
	}

	p.client = client
	return p.client, nil
}

// buildMessage constructs a mail message from the given send
// parameters.
//
// This helper keeps the logic for creating the MIME message in one
// place.
//
// Takes params (*email_dto.SendParams) which contains the email
// details.
//
// Returns *mail.Msg which is the constructed message ready for
// sending.
// Returns error when validation fails or attachments cannot be added.
func (p *SMTPProvider) buildMessage(params *email_dto.SendParams) (*mail.Msg, error) {
	if err := p.validateMessageParams(params); err != nil {
		return nil, fmt.Errorf("validating SMTP message parameters: %w", err)
	}

	message := mail.NewMsg()

	if err := p.setMessageAddresses(message, params); err != nil {
		return nil, fmt.Errorf("setting SMTP message addresses: %w", err)
	}

	message.Subject(params.Subject)

	p.setMessageBody(message, params)
	if err := p.addMessageAttachments(message, params); err != nil {
		return nil, fmt.Errorf("adding SMTP message attachments: %w", err)
	}

	return message, nil
}

// validateMessageParams checks that the required fields are present in
// SendParams.
//
// Takes params (*email_dto.SendParams) which contains the email parameters to
// validate.
//
// Returns error when no recipients are specified or when both body fields are
// empty.
func (*SMTPProvider) validateMessageParams(params *email_dto.SendParams) error {
	if len(params.To) == 0 {
		return email_domain.ErrRecipientRequired
	}
	if params.BodyHTML == "" && params.BodyPlain == "" {
		return email_domain.ErrBodyRequired
	}
	return nil
}

// setMessageAddresses configures the From, To, Cc, and Bcc addresses for the
// message.
//
// Takes message (*mail.Message) which is the message to configure.
// Takes params (*email_dto.SendParams) which provides the address values.
//
// Returns error when any address is invalid.
func (p *SMTPProvider) setMessageAddresses(message *mail.Msg, params *email_dto.SendParams) error {
	from := p.fromEmail
	if params.From != nil && *params.From != "" {
		from = *params.From
	}

	if err := message.From(from); err != nil {
		return fmt.Errorf("set From address: %w", err)
	}
	if err := message.To(params.To...); err != nil {
		return fmt.Errorf("set To addresses: %w", err)
	}
	if len(params.Cc) > 0 {
		if err := message.Cc(params.Cc...); err != nil {
			return fmt.Errorf("set Cc addresses: %w", err)
		}
	}
	if len(params.Bcc) > 0 {
		if err := message.Bcc(params.Bcc...); err != nil {
			return fmt.Errorf("set Bcc addresses: %w", err)
		}
	}

	return nil
}

// setMessageBody sets the message body content on the given mail message.
// It handles plain text, HTML, or both formats together.
//
// Takes message (*mail.Message) which is the message to set the body on.
// Takes params (*email_dto.SendParams) which provides the body content.
func (*SMTPProvider) setMessageBody(message *mail.Msg, params *email_dto.SendParams) {
	if params.BodyHTML != "" && params.BodyPlain != "" {
		message.SetBodyString(mail.TypeTextPlain, params.BodyPlain)
		message.AddAlternativeString(mail.TypeTextHTML, params.BodyHTML)
	} else if params.BodyHTML != "" {
		message.SetBodyString(mail.TypeTextHTML, params.BodyHTML)
	} else {
		message.SetBodyString(mail.TypeTextPlain, params.BodyPlain)
	}
}

// addMessageAttachments adds all attachments to the message, handling
// CID-embedded images separately.
//
// Takes message (*mail.Message) which receives the attachments.
// Takes params (*email_dto.SendParams) which provides the attachments to add.
//
// Returns error when an attachment cannot be embedded or attached.
func (*SMTPProvider) addMessageAttachments(message *mail.Msg, params *email_dto.SendParams) error {
	for _, attachment := range params.Attachments {
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
