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

package provider_disk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultFilePermissions is the file mode used when writing .eml files.
	// Owner read-write only; email content may contain sensitive data.
	defaultFilePermissions = 0o600

	// filenameTimestampFormat defines the timestamp format for .eml filenames.
	filenameTimestampFormat = "20060102_150405.000000"

	// metricStatusSuccess is the metric attribute value for a successful
	// operation.
	metricStatusSuccess = "success"

	// metricStatusError is the metric status value for failed operations.
	metricStatusError = "error"

	// metricSendTypeSingle is the metric attribute value for a single send.
	metricSendTypeSingle = "single"

	// metricSendTypeBulk is the metric value for bulk email sends.
	metricSendTypeBulk = "bulk"

	// metricKeyStatus is the metric attribute key for operation status.
	metricKeyStatus = "status"

	// metricKeySendType is the metric attribute key for send type (single or
	// bulk).
	metricKeySendType = "send_type"
)

// DiskProviderArgs holds settings for the disk-based email provider.
type DiskProviderArgs struct {
	// SandboxFactory creates sandboxes when no sandbox is directly injected.
	// When non-nil, this factory is used instead of safedisk.NewSandbox.
	SandboxFactory safedisk.Factory

	// OutboxPath is the directory where email messages are saved.
	OutboxPath string
}

// DiskProvider implements the EmailProviderPort interface by writing emails to
// disk.
type DiskProvider struct {
	// rateLimiter controls how often email sending operations may run.
	rateLimiter *email_domain.ProviderRateLimiter

	// sandbox provides sandboxed filesystem access to the outbox directory.
	sandbox safedisk.Sandbox

	// outboxPath is the directory where email files are saved (for metadata only).
	outboxPath string
}

var _ email_domain.EmailProviderPort = (*DiskProvider)(nil)
var _ provider_domain.ProviderMetadata = (*DiskProvider)(nil)

// NewDiskProvider creates a DiskProvider with a specified outbox directory.
// If the directory does not exist, it is created.
//
// Takes arguments (DiskProviderArgs) which specifies the outbox path.
// Takes opts (...email_domain.ProviderOption) which provides optional rate
// limit settings.
//
// Returns *DiskProvider which is the configured provider ready for use.
// Returns error when the outbox path is empty or cannot be created.
func NewDiskProvider(_ context.Context, arguments DiskProviderArgs, opts ...email_domain.ProviderOption) (*DiskProvider, error) {
	if arguments.OutboxPath == "" {
		return nil, errors.New("outbox path cannot be empty for disk email provider")
	}

	var sandbox safedisk.Sandbox
	var err error
	if arguments.SandboxFactory != nil {
		sandbox, err = arguments.SandboxFactory.Create("email-outbox", arguments.OutboxPath, safedisk.ModeReadWrite)
	} else {
		sandbox, err = safedisk.NewSandbox(arguments.OutboxPath, safedisk.ModeReadWrite)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox sandbox at %q: %w", arguments.OutboxPath, err)
	}

	defaultConfig := email_domain.ProviderRateLimitConfig{
		CallsPerSecond: 0,
		Burst:          0,
		Clock:          nil,
	}
	rateLimiter := email_domain.ApplyProviderOptions(defaultConfig, opts...)

	return &DiskProvider{
		sandbox:     sandbox,
		outboxPath:  arguments.OutboxPath,
		rateLimiter: rateLimiter,
	}, nil
}

// GetProviderType returns the type identifier for this email provider.
//
// Returns string which is "disk".
func (*DiskProvider) GetProviderType() string {
	return "disk"
}

// GetProviderMetadata returns metadata about this disk email provider.
//
// Returns map[string]any which describes the provider configuration.
func (p *DiskProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"outbox_path": p.outboxPath,
		"description": "Email provider that saves .eml files to disk",
	}
}

// Send writes the email details to a standard .eml file in the outboxPath.
//
// Takes params (*email_dto.SendParams) which contains the email details to
// write.
//
// Returns error when validation fails or the file cannot be written.
func (p *DiskProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	contextLog := l.With(logger_domain.String("method", "Disk.Send"))
	contextLog.Trace("Simulating send email, writing to .eml file",
		logger_domain.Strings("to", params.To),
		logger_domain.String("subject", params.Subject),
	)

	if err := p.validateSendParams(ctx, params); err != nil {
		return fmt.Errorf("validating disk send parameters: %w", err)
	}

	filename := p.generateEmailFilename(params.To[0])

	sendErr := p.writeEmailFile(ctx, filename, params)
	recordDiskSendMetrics(ctx, time.Since(startTime), sendErr)
	return sendErr
}

// Close releases the sandboxed filesystem resources held by the provider.
//
// Returns error when closing the sandbox fails.
func (p *DiskProvider) Close(_ context.Context) error {
	return p.sandbox.Close()
}

// SupportsBulkSending indicates that the disk provider does not have a native
// bulk sending mechanism.
//
// Returns bool which is always false for disk providers.
func (*DiskProvider) SupportsBulkSending() bool {
	return false
}

// SendBulk sends multiple emails by calling Send individually and collecting
// any errors.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when one or more emails fail to send, wrapped as a MultiError.
func (p *DiskProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	startTime := time.Now()

	if len(emails) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)
	contextLog := l.With(logger_domain.String("method", "Disk.SendBulk"))
	contextLog.Trace("Sending emails via disk (individual writes)", logger_domain.Int("count", len(emails)))

	var multiError *email_domain.MultiError
	for i, email := range emails {
		if err := p.Send(ctx, email); err != nil {
			contextLog.ReportError(nil, err, "Failed to send email in bulk operation",
				logger_domain.Int("email_index", i),
				logger_domain.String("subject", email.Subject))

			now := time.Now()
			emailError := &email_domain.EmailError{
				Email:        *email,
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

	duration := float64(time.Since(startTime).Milliseconds())
	emailCount := int64(len(emails))

	if multiError != nil && multiError.HasErrors() {
		sendTotal.Add(ctx, emailCount, metric.WithAttributes(
			attribute.String(metricKeyStatus, metricStatusError),
			attribute.String(metricKeySendType, metricSendTypeBulk),
		))
		sendDuration.Record(ctx, duration, metric.WithAttributes(
			attribute.String(metricKeyStatus, metricStatusError),
			attribute.String(metricKeySendType, metricSendTypeBulk),
		))
		return multiError
	}

	sendTotal.Add(ctx, emailCount, metric.WithAttributes(
		attribute.String(metricKeyStatus, metricStatusSuccess),
		attribute.String(metricKeySendType, metricSendTypeBulk),
	))
	sendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String(metricKeyStatus, metricStatusSuccess),
		attribute.String(metricKeySendType, metricSendTypeBulk),
	))

	return nil
}

// Name returns the display name of this provider.
//
// Returns string which is the human-readable name for this provider.
func (*DiskProvider) Name() string {
	return "EmailProvider (Disk)"
}

// Check implements the healthprobe_domain.Probe interface by verifying disk
// provider health.
//
// When checkType is liveness, it verifies the outbox path is configured.
// When checkType is readiness, it checks the outbox directory exists and is
// accessible.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which contains the health state and details.
func (p *DiskProvider) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if p.outboxPath == "" {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Outbox path not configured",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   "Disk email provider configured",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	info, err := p.sandbox.Stat(".")
	if err != nil {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   fmt.Sprintf("Outbox directory not accessible: %v", err),
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	if !info.IsDir() {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   fmt.Sprintf("Outbox path is not a directory: %s", p.outboxPath),
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Outbox directory accessible",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// validateSendParams validates the send parameters and applies rate limiting.
//
// Takes params (*email_dto.SendParams) which contains the email parameters to
// validate.
//
// Returns error when rate limiting fails, no recipients are provided, or both
// body fields are empty.
func (p *DiskProvider) validateSendParams(ctx context.Context, params *email_dto.SendParams) error {
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

// generateEmailFilename generates a unique, sanitised filename for the email.
//
// Takes recipient (string) which is the email recipient address.
//
// Returns string which is the sanitised filename (relative to the sandbox
// root).
func (*DiskProvider) generateEmailFilename(recipient string) string {
	timestamp := time.Now().Format(filenameTimestampFormat)
	safeRecipient := strings.NewReplacer(
		"@", "_at_", ".", "_", "/", "_", "\\", "_", "..", "_", ":", "_",
	).Replace(recipient)

	return fmt.Sprintf("%s_%s.eml", timestamp, safeRecipient)
}

// writeEmailFile writes the email content to a file atomically via the
// sandboxed filesystem. The sandbox handles temporary file creation, writing,
// and atomic rename internally.
//
// Takes ctx (context.Context) which carries the logger and cancellation.
// Takes filename (string) which specifies the destination filename relative to
// the sandbox root.
// Takes params (*email_dto.SendParams) which contains the email content to
// write.
//
// Returns error when building MIME content fails or the atomic write fails.
func (p *DiskProvider) writeEmailFile(ctx context.Context, filename string, params *email_dto.SendParams) error {
	_, l := logger_domain.From(ctx, log)

	content, err := email_domain.BuildMIMEMessage(params)
	if err != nil {
		return fmt.Errorf("failed to build MIME email content: %w", err)
	}

	if err := p.sandbox.WriteFileAtomic(filename, content, defaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write email file %q: %w", filename, err)
	}

	l.Trace("Email successfully written to .eml file", logger_domain.String("filename", filename))
	return nil
}

// recordDiskSendMetrics records metrics for a single email send.
//
// Takes duration (time.Duration) which is the time taken to send the email.
// Takes sendErr (error) which shows if the send failed; nil means success.
func recordDiskSendMetrics(ctx context.Context, duration time.Duration, sendErr error) {
	durationMs := float64(duration.Milliseconds())
	status := metricStatusSuccess
	if sendErr != nil {
		status = metricStatusError
	}
	sendTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String(metricKeyStatus, status),
		attribute.String(metricKeySendType, metricSendTypeSingle),
	))
	sendDuration.Record(ctx, durationMs, metric.WithAttributes(
		attribute.String(metricKeyStatus, status),
		attribute.String(metricKeySendType, metricSendTypeSingle),
	))
}
