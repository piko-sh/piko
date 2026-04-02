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

package provider_stdout

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

const (
	// metricAttrStatus is the attribute key for recording operation status in metrics.
	metricAttrStatus = "status"

	// metricAttrSendType is the metric attribute key for the type of send operation.
	metricAttrSendType = "send_type"

	// statusSuccess indicates the email was sent without error.
	statusSuccess = "success"

	// statusError is the metrics status value for failed operations.
	statusError = "error"

	// sendTypeSingle is the metric label for single email sends.
	sendTypeSingle = "single"

	// sendTypeBulk is the metric label for bulk email sending.
	sendTypeBulk = "bulk"

	// recipientSeparator is the delimiter used to join email addresses in output.
	recipientSeparator = ", "
)

// stdoutProvider writes email content to stdout as structured JSON.
// It implements EmailProviderPort for testing and debugging.
type stdoutProvider struct {
	// rateLimiter limits how often emails can be sent.
	rateLimiter *email_domain.ProviderRateLimiter
}

var _ email_domain.EmailProviderPort = (*stdoutProvider)(nil)
var _ provider_domain.ProviderMetadata = (*stdoutProvider)(nil)

// Send formats the email details into a readable block and prints it to
// stdout for debugging.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// format and display.
//
// Returns error when the rate limiter context is cancelled or times out.
func (p *stdoutProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	if err := p.rateLimiter.Wait(ctx); err != nil {
		p.recordMetrics(ctx, startTime, statusError, sendTypeSingle, 1)
		return fmt.Errorf("waiting for stdout rate limiter: %w", err)
	}

	var builder strings.Builder
	p.formatEmailHeader(&builder)
	p.formatEmailMetadata(&builder, params)
	p.formatEmailBody(&builder, params)
	p.formatAttachments(&builder, params)
	builder.WriteString("======================================================================\n\n")

	_, _ = fmt.Fprint(os.Stdout, builder.String())

	l.Trace("Email sent via stdout provider (development default)", logger_domain.String("provider", "stdout"))
	l.Internal("Configure a real email provider for production via piko.WithEmailService or bootstrap.WithEmailService")

	p.recordMetrics(ctx, startTime, statusSuccess, sendTypeSingle, 1)
	return nil
}

// SendBulk sends multiple emails by calling Send for each email in the batch.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when the rate limiter fails or sending an individual email
// fails.
func (p *stdoutProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	startTime := time.Now()

	if len(emails) == 0 {
		return nil
	}
	if err := p.rateLimiter.Wait(ctx); err != nil {
		p.recordMetrics(ctx, startTime, statusError, sendTypeBulk, len(emails))
		return fmt.Errorf("waiting for stdout bulk rate limiter: %w", err)
	}
	for _, e := range emails {
		if err := p.Send(ctx, e); err != nil {
			p.recordMetrics(ctx, startTime, statusError, sendTypeBulk, len(emails))
			return fmt.Errorf("sending bulk email via stdout: %w", err)
		}
	}

	p.recordMetrics(ctx, startTime, statusSuccess, sendTypeBulk, len(emails))
	return nil
}

// SupportsBulkSending indicates whether this provider can send multiple emails
// in one operation.
//
// Returns bool which is true when bulk sending is supported.
func (*stdoutProvider) SupportsBulkSending() bool { return true }

// Close releases any resources held by the provider.
//
// Returns error when the provider cannot be closed; always returns nil.
func (*stdoutProvider) Close(_ context.Context) error { return nil }

// GetProviderType returns the provider implementation type.
//
// Returns string which identifies the provider type for monitoring.
func (*stdoutProvider) GetProviderType() string {
	return "stdout"
}

// GetProviderMetadata returns metadata about the provider.
//
// Returns map[string]any which contains provider capabilities and configuration.
func (*stdoutProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"version":     "1.0.0",
		"environment": "development",
		"description": "Development email provider that writes to stdout",
	}
}

// Name returns the display name of this provider.
//
// Returns string which is the human-readable name for this provider.
func (*stdoutProvider) Name() string {
	return "EmailProvider (Stdout)"
}

// Check implements the healthprobe_domain.Probe interface.
// The stdout provider is always healthy as it has no external dependencies.
//
// Returns healthprobe_dto.Status which always reports healthy.
func (p *stdoutProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Stdout email provider operational",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// formatEmailHeader writes the email header section to the string builder.
//
// Takes builder (*strings.Builder) which receives the formatted header output.
func (*stdoutProvider) formatEmailHeader(builder *strings.Builder) {
	builder.WriteString("======================================================================\n")
	_, _ = fmt.Fprintf(builder, " Piko Stdout Email | %s\n", time.Now().Format(time.RFC3339))
	builder.WriteString("======================================================================\n")
}

// formatEmailMetadata writes the email metadata to the string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes params (*email_dto.SendParams) which provides the email fields to
// format, including from, to, cc, bcc, and subject.
func (*stdoutProvider) formatEmailMetadata(builder *strings.Builder, params *email_dto.SendParams) {
	from := "[not specified]"
	if params.From != nil {
		from = *params.From
	}
	_, _ = fmt.Fprintf(builder, "From:    %s\n", from)

	if len(params.To) > 0 {
		_, _ = fmt.Fprintf(builder, "To:      %s\n", strings.Join(params.To, recipientSeparator))
	}
	if len(params.Cc) > 0 {
		_, _ = fmt.Fprintf(builder, "Cc:      %s\n", strings.Join(params.Cc, recipientSeparator))
	}
	if len(params.Bcc) > 0 {
		_, _ = fmt.Fprintf(builder, "Bcc:     %s\n", strings.Join(params.Bcc, recipientSeparator))
	}
	_, _ = fmt.Fprintf(builder, "Subject: %s\n", params.Subject)
	builder.WriteString("----------------------------------------------------------------------\n\n")
}

// formatEmailBody writes the email body content (plain text and HTML) to the
// string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes params (*email_dto.SendParams) which provides the body content.
func (*stdoutProvider) formatEmailBody(builder *strings.Builder, params *email_dto.SendParams) {
	if params.BodyPlain != "" {
		builder.WriteString("--- BODY (PLAIN TEXT) ---\n")
		builder.WriteString(params.BodyPlain)
		builder.WriteString("\n\n")
	}

	if params.BodyHTML != "" {
		builder.WriteString("--- BODY (HTML) ---\n")
		builder.WriteString(params.BodyHTML)
		builder.WriteString("\n\n")
	}
}

// formatAttachments writes attachment details to the string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes params (*email_dto.SendParams) which provides the attachments to format.
func (*stdoutProvider) formatAttachments(builder *strings.Builder, params *email_dto.SendParams) {
	if len(params.Attachments) == 0 {
		return
	}

	_, _ = fmt.Fprintf(builder, "--- ATTACHMENTS (%d) ---\n", len(params.Attachments))
	for _, a := range params.Attachments {
		mime := a.MIMEType
		if mime == "" {
			mime = "application/octet-stream"
		}
		infoParts := make([]string, 0, 2)
		infoParts = append(infoParts, fmt.Sprintf("mime=%s", mime), fmt.Sprintf("size=%d bytes", len(a.Content)))
		if a.ContentID != "" {
			infoParts = append(infoParts, fmt.Sprintf("cid=%s", a.ContentID))
		}

		_, _ = fmt.Fprintf(builder, "- %s (%s)\n", a.Filename, strings.Join(infoParts, recipientSeparator))
	}
	builder.WriteString("\n")
}

// recordMetrics records duration and total metrics for send operations.
//
// Takes startTime (time.Time) which marks when the operation began.
// Takes status (string) which shows the result of the operation.
// Takes sendType (string) which names the type of send operation.
// Takes count (int) which is the number of items sent.
func (*stdoutProvider) recordMetrics(ctx context.Context, startTime time.Time, status, sendType string, count int) {
	duration := float64(time.Since(startTime).Milliseconds())
	sendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendType),
	))
	sendTotal.Add(ctx, int64(count), metric.WithAttributes(
		attribute.String(metricAttrStatus, status),
		attribute.String(metricAttrSendType, sendType),
	))
}

// New creates a new stdout email provider with optional rate limiting.
//
// Takes opts (...email_domain.ProviderOption) which configures rate limiting.
//
// Returns email_domain.EmailProviderPort which is ready to use for email output.
// Returns error when the provider cannot be created.
func New(_ context.Context, opts ...email_domain.ProviderOption) (email_domain.EmailProviderPort, error) {
	defaultConfig := email_domain.ProviderRateLimitConfig{CallsPerSecond: 0, Burst: 0, Clock: nil}
	rl := email_domain.ApplyProviderOptions(defaultConfig, opts...)
	return &stdoutProvider{rateLimiter: rl}, nil
}
