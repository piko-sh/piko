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

package email_provider_ses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/logger"
)

var _ email.ProviderPort = (*SESProvider)(nil)

const (
	// defaultCallsPerSecond is the default AWS SES rate limit in calls per second.
	defaultCallsPerSecond = 50.0

	// defaultBurst is the default burst size for rate limiting.
	defaultBurst = 100

	// charsetUTF8 is the UTF-8 character set used for email content encoding.
	charsetUTF8 = "UTF-8"

	// metricStatusError is the metric attribute value for failed operations.
	metricStatusError = "error"

	// metricStatusSuccess is the metric status value for operations that succeed.
	metricStatusSuccess = "success"

	// metricSendTypeSingle is the metric value for single email sends.
	metricSendTypeSingle = "single"

	// metricSendTypeBulk is the metric attribute value for bulk email sends.
	metricSendTypeBulk = "bulk"
)

// SESProvider wraps an AWS SES client for sending emails. It implements
// the EmailProviderPort interface.
type SESProvider struct {
	// client is the AWS SES API client for sending emails.
	client *ses.Client

	// rateLimiter limits how fast email requests can be sent.
	rateLimiter *email_domain.ProviderRateLimiter

	// fromEmail is the default sender address used when none is given.
	fromEmail string
}

// SESProviderArgs holds settings for setting up the SES email provider.
type SESProviderArgs struct {
	// Region is the AWS region for the SES service; empty uses the default.
	Region string

	// FromEmail is the sender email address; must not be empty.
	FromEmail string

	// AWSKey is the AWS access key ID for static credentials.
	AWSKey string

	// AWSSecret is the AWS secret access key for static credentials.
	AWSSecret string

	// AWSLocalEndpoint is an optional custom endpoint URL for local testing.
	AWSLocalEndpoint string
}

// ProviderOption is a functional option for setting up the SES provider.
type ProviderOption = email_domain.ProviderOption

// Send delivers an email through AWS SES, selecting the correct API based on
// whether the email has attachments.
//
// Takes params (*email_dto.SendParams) which contains the email details
// including recipients, subject, body, and optional attachments.
//
// Returns error when validation fails or the email cannot be sent.
func (p *SESProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	ctx, l := logger.From(ctx, log)
	startTime := time.Now()

	contextLog := l.With(logger.String("method", "SES.Send"))
	contextLog.Trace("Sending email via SES",
		logger.Strings("to", params.To),
		logger.String("subject", params.Subject),
	)

	if err := p.validateSendParams(ctx, params); err != nil {
		return err
	}

	from := p.determineSenderAddress(params)

	sendErr := p.executeSend(ctx, params, from)

	p.recordSingleSendMetrics(ctx, startTime, sendErr)

	return sendErr
}

// Close releases no resources as the SES client is managed by the SDK.
//
// Returns error which is always nil.
func (*SESProvider) Close(context.Context) error {
	return nil
}

// SupportsBulkSending reports whether SES supports bulk sending.
//
// AWS SES does not have a native bulk sending API. While SES has templated
// bulk sending, it uses a different workflow than what this interface's
// SendBulk method implies (sending discrete, different emails).
//
// Returns bool which is always false for this provider.
func (*SESProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by falling back to individual sends with
// MultiError tracking.
//
// Takes emails ([]*email_dto.SendParams) which contains the email messages to
// send.
//
// Returns error when one or more emails fail to send.
func (p *SESProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	ctx, l := logger.From(ctx, log)
	startTime := time.Now()

	if len(emails) == 0 {
		return nil
	}

	contextLog := l.With(logger.String("method", "SES.SendBulk"))
	contextLog.Trace("Sending emails via AWS SES (individual sends)", logger.Int("count", len(emails)))

	multiError := p.sendBulkEmails(ctx, emails)

	p.recordBulkSendMetrics(ctx, startTime, len(emails), multiError)

	if multiError != nil && multiError.HasErrors() {
		return multiError
	}

	return nil
}

// validateSendParams performs all validation checks for send parameters.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// validate.
//
// Returns error when rate limiting fails, no recipients are provided, or both
// body fields are empty.
func (p *SESProvider) validateSendParams(ctx context.Context, params *email_dto.SendParams) error {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}
	if len(params.To) == 0 {
		return email_domain.ErrRecipientRequired
	}
	if params.BodyHTML == "" && params.BodyPlain == "" {
		return email_domain.ErrBodyRequired
	}
	return nil
}

// determineSenderAddress returns the appropriate sender address from params
// or the default.
//
// Takes params (*email_dto.SendParams) which contains the email send options.
//
// Returns string which is the sender address to use.
func (p *SESProvider) determineSenderAddress(params *email_dto.SendParams) string {
	if params.From != nil {
		return *params.From
	}
	return p.fromEmail
}

// executeSend chooses the send method based on whether there are attachments.
//
// Takes params (*email_dto.SendParams) which contains the email details.
// Takes from (string) which specifies the sender address.
//
// Returns error when the send operation fails.
func (p *SESProvider) executeSend(ctx context.Context, params *email_dto.SendParams, from string) error {
	if len(params.Attachments) > 0 {
		return p.sendRawEmail(ctx, params, from)
	}
	return p.sendSimpleEmail(ctx, params, from)
}

// recordSingleSendMetrics records metrics for a single email send.
//
// Takes startTime (time.Time) which marks when the send started.
// Takes err (error) which shows whether the send worked or failed.
func (*SESProvider) recordSingleSendMetrics(ctx context.Context, startTime time.Time, err error) {
	duration := float64(time.Since(startTime).Milliseconds())
	status := metricStatusSuccess
	if err != nil {
		status = metricStatusError
	}

	attrs := metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("send_type", metricSendTypeSingle),
	)

	SendTotal.Add(ctx, 1, attrs)
	SendDuration.Record(ctx, duration, attrs)
}

// sendSimpleEmail uses the ses.SendEmail API for messages without attachments.
//
// Takes params (*email_dto.SendParams) which contains the email recipients,
// subject, and body content.
// Takes from (string) which specifies the sender email address.
//
// Returns error when the SES API call fails.
func (p *SESProvider) sendSimpleEmail(ctx context.Context, params *email_dto.SendParams, from string) error {
	input := &ses.SendEmailInput{
		Source: aws.String(from),
		Destination: &types.Destination{
			ToAddresses:  params.To,
			CcAddresses:  params.Cc,
			BccAddresses: params.Bcc,
		},
		Message: &types.Message{
			Subject: &types.Content{
				Charset: aws.String(charsetUTF8),
				Data:    aws.String(params.Subject),
			},
			Body: buildSESMessageBody(params),
		},
	}

	_, err := p.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}
	return nil
}

// sendRawEmail constructs and sends a standard RFC 5322 MIME message for emails
// with attachments. It uses a dedicated mail library to handle the complexities
// of MIME encoding.
//
// Takes params (*email_dto.SendParams) which contains the email content and
// recipient details.
// Takes from (string) which specifies the sender address.
//
// Returns error when building the MIME message fails or sending via SES fails.
func (p *SESProvider) sendRawEmail(ctx context.Context, params *email_dto.SendParams, from string) error {
	params.From = &from

	rawMessageBytes, err := email_domain.BuildMIMEMessage(params)
	if err != nil {
		return fmt.Errorf("failed to build raw email message: %w", err)
	}

	allRecipients := make([]string, 0, len(params.To)+len(params.Cc)+len(params.Bcc))
	allRecipients = append(allRecipients, params.To...)
	allRecipients = append(allRecipients, params.Cc...)
	allRecipients = append(allRecipients, params.Bcc...)

	input := &ses.SendRawEmailInput{
		Source:       aws.String(from),
		Destinations: allRecipients,
		RawMessage:   &types.RawMessage{Data: rawMessageBytes},
	}

	_, err = p.client.SendRawEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send raw email via SES: %w", err)
	}
	return nil
}

// sendBulkEmails sends each email individually and collects errors.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns *email_domain.MultiError which contains all failed email attempts,
// or nil if all emails were sent successfully.
func (p *SESProvider) sendBulkEmails(ctx context.Context, emails []*email_dto.SendParams) *email_domain.MultiError {
	ctx, l := logger.From(ctx, log)
	var multiError *email_domain.MultiError

	for i, emailMessage := range emails {
		if err := p.Send(ctx, emailMessage); err != nil {
			l.ReportError(nil, err, "Failed to send email in bulk operation",
				logger.Int("email_index", i),
				logger.String("subject", emailMessage.Subject))

			emailError := &email_domain.EmailError{
				Email:       *emailMessage,
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

	return multiError
}

// recordBulkSendMetrics records metrics for a bulk email send operation.
//
// Takes startTime (time.Time) which marks when the send operation began.
// Takes emailCount (int) which is the number of emails in the bulk send.
// Takes multiError (*email_domain.MultiError) which holds any errors from the
// send operation.
func (*SESProvider) recordBulkSendMetrics(ctx context.Context, startTime time.Time, emailCount int, multiError *email_domain.MultiError) {
	duration := float64(time.Since(startTime).Milliseconds())
	count := int64(emailCount)

	status := metricStatusSuccess
	if multiError != nil && multiError.HasErrors() {
		status = metricStatusError
	}

	attrs := metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("send_type", metricSendTypeBulk),
	)

	SendTotal.Add(ctx, count, attrs)
	SendDuration.Record(ctx, duration, attrs)
}

// NewSESProvider creates an SESProvider with a new AWS config and SES client.
//
// Takes arguments (SESProviderArgs) which specifies the AWS and email settings.
// Takes opts (...ProviderOption) which provides optional rate
// limiting controls.
//
// Returns email.ProviderPort which is the configured provider ready for use.
// Returns error when FromEmail is empty or AWS config cannot be loaded.
func NewSESProvider(ctx context.Context, arguments SESProviderArgs, opts ...ProviderOption) (email.ProviderPort, error) {
	if arguments.FromEmail == "" {
		return nil, errors.New("fromEmail cannot be empty")
	}

	awsConfig, err := loadAWSConfig(ctx, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for SES: %w", err)
	}

	sesClient := createSESClient(ctx, &awsConfig, arguments.AWSLocalEndpoint)

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &SESProvider{
		client:      sesClient,
		fromEmail:   arguments.FromEmail,
		rateLimiter: rateLimiter,
	}, nil
}

// loadAWSConfig loads the AWS configuration using the given arguments.
//
// Takes arguments (SESProviderArgs) which specifies the region and credentials.
//
// Returns aws.Config which is the loaded AWS configuration.
// Returns error when the configuration cannot be loaded.
func loadAWSConfig(ctx context.Context, arguments SESProviderArgs) (aws.Config, error) {
	var configOptions []func(*config.LoadOptions) error

	if arguments.Region != "" {
		configOptions = append(configOptions, config.WithRegion(arguments.Region))
	}

	if arguments.AWSKey != "" && arguments.AWSSecret != "" {
		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(arguments.AWSKey, arguments.AWSSecret, ""),
		))
	}

	return config.LoadDefaultConfig(ctx, configOptions...)
}

// createSESClient creates an SES client with the given AWS settings.
//
// Takes awsConfig (*aws.Config) which provides the AWS settings.
// Takes localEndpoint (string) which sets a custom endpoint for local testing;
// when empty, the default AWS endpoint is used.
//
// Returns *ses.Client which is the configured SES client ready for use.
func createSESClient(ctx context.Context, awsConfig *aws.Config, localEndpoint string) *ses.Client {
	_, l := logger.From(ctx, log)
	var clientOptions []func(*ses.Options)
	if localEndpoint != "" {
		clientOptions = append(clientOptions, func(o *ses.Options) {
			o.BaseEndpoint = aws.String(localEndpoint)
		})
		l.Internal("Using LocalStack endpoint for SES", logger.String("endpoint", localEndpoint))
	}
	return ses.NewFromConfig(*awsConfig, clientOptions...)
}

// buildSESMessageBody constructs the Body part of a ses.Message for the
// simple send API.
//
// Takes params (*email_dto.SendParams) which contains the email body content.
//
// Returns *types.Body which contains the formatted message body with charset
// encoding.
func buildSESMessageBody(params *email_dto.SendParams) *types.Body {
	body := &types.Body{}
	if params.BodyHTML != "" {
		body.Html = &types.Content{
			Charset: aws.String(charsetUTF8),
			Data:    aws.String(params.BodyHTML),
		}
	}
	if params.BodyPlain != "" {
		body.Text = &types.Content{
			Charset: aws.String(charsetUTF8),
			Data:    aws.String(params.BodyPlain),
		}
	}
	return body
}
