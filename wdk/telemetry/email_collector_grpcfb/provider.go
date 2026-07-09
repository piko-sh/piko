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

package email_collector_grpcfb

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/email"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// maxSubjectLen bounds the user-influenced subject line so one pathological message
	// cannot bloat a telemetry frame.
	maxSubjectLen = 256

	// maxErrorLen bounds the send error string so one pathological error cannot bloat a
	// telemetry frame.
	maxErrorLen = 512
)

// Provider decorates an email.ProviderPort: it delegates Send/SendBulk to the inner
// provider and forwards one telemetry_grpcfb.EmailEvent per message (non-blocking).
type Provider struct {
	// inner is the wrapped provider that performs the real sending.
	inner email.ProviderPort

	// client is the shared telemetry transport that receives EmailEvents.
	client *telemetry_grpcfb.Client

	// name labels the EmailEvent Provider field (e.g. "smtp", "ses").
	name string
}

var (
	_ email.ProviderPort = (*Provider)(nil)
)

// Wrap decorates inner so that every send also emits an EmailEvent on the shared client.
//
// Takes name (string) which labels the EmailEvent Provider field (e.g. "smtp", "ses").
// Takes inner (email.ProviderPort) which is the wrapped provider doing the real sending.
// Takes client (*telemetry_grpcfb.Client) which is the shared transport for EmailEvents.
//
// Returns *Provider which delegates to inner and forwards an EmailEvent per message.
func Wrap(name string, inner email.ProviderPort, client *telemetry_grpcfb.Client) *Provider {
	return &Provider{name: name, inner: inner, client: client}
}

// Send delegates to the inner provider and emits an EmailEvent for the message.
//
// Takes params (*email.SendParams) which describes the message to send.
//
// Returns error which is the inner provider's send result.
func (p *Provider) Send(ctx context.Context, params *email.SendParams) error {
	err := p.inner.Send(ctx, params)
	p.emit(ctx, params, err)
	return err
}

// SendBulk delegates to the inner provider and emits an EmailEvent per message.
//
// Takes emails ([]*email.SendParams) which are the messages to send.
//
// Returns error which is the inner provider's bulk send result.
func (p *Provider) SendBulk(ctx context.Context, emails []*email.SendParams) error {
	err := p.inner.SendBulk(ctx, emails)

	var multi *email.MultiError
	switch {
	case err == nil:
		for _, params := range emails {
			p.emit(ctx, params, nil)
		}
	case errors.As(err, &multi):
		failed := failedRecipients(multi)
		for _, params := range emails {
			p.emit(ctx, params, perMessageError(params, failed))
		}
	default:
		for _, params := range emails {
			p.emitUnknown(ctx, params)
		}
	}

	return err
}

// SupportsBulkSending reports the inner provider's bulk capability.
//
// Returns bool which is true when the inner provider supports bulk sending.
func (p *Provider) SupportsBulkSending() bool { return p.inner.SupportsBulkSending() }

// Close releases the inner provider.
//
// Returns error which is the inner provider's close result.
func (p *Provider) Close(ctx context.Context) error { return p.inner.Close(ctx) }

// emit forwards one EmailEvent (sent or failed) for a message (non-blocking, lossy).
//
// Takes params (*email.SendParams) which describes the message that was sent.
// Takes sendErr (error) which is the inner send result; non-nil marks the event failed.
func (p *Provider) emit(ctx context.Context, params *email.SendParams, sendErr error) {
	event, status, errMsg := "sent", "ok", ""
	if sendErr != nil {
		event, status = "failed", "error"
		errMsg, _ = telemetry_grpcfb.TruncateUTF8(sendErr.Error(), maxErrorLen)
	}
	p.emitEvent(ctx, params, event, status, errMsg)
}

// emitUnknown forwards an EmailEvent whose delivery outcome could not be attributed to a
// message because the inner provider returned an opaque bulk error (no per-message
// detail).
//
// Takes params (*email.SendParams) which describes the message whose outcome is unknown.
func (p *Provider) emitUnknown(ctx context.Context, params *email.SendParams) {
	p.emitEvent(ctx, params, "bulk", "unknown", "")
}

// emitEvent builds and forwards one EmailEvent for a message. A deferred recover backstop
// guarantees a panic in the telemetry path can never break a real send; it never wraps
// the inner provider's send call.
//
// Takes params (*email.SendParams) which describes the message that was sent.
// Takes event (string) which names the lifecycle transition.
// Takes status (string) which is the lifecycle outcome status.
// Takes errMsg (string) which is the truncated failure message, empty on success.
func (p *Provider) emitEvent(ctx context.Context, params *email.SendParams, event, status, errMsg string) {
	if p == nil || p.client == nil || params == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			_, l := logger_domain.From(ctx, log)
			l.Error("recovered panic while emitting email telemetry event",
				logger_domain.String("panic", fmt.Sprint(r)))
		}
	}()

	recipient := ""
	if len(params.To) > 0 {
		recipient = params.To[0]
	}
	template := ""
	if params.ProviderOptions != nil {
		if v, ok := params.ProviderOptions["template"].(string); ok {
			template = v
		}
	}
	subject, _ := telemetry_grpcfb.TruncateUTF8(params.Subject, maxSubjectLen)
	nowMs := time.Now().UnixMilli()
	p.client.AddEmailEvent(ctx, telemetry_grpcfb.EmailEvent{
		MessageID: messageID(recipient, params.Subject, nowMs),
		Provider:  p.name,
		Template:  template,
		Recipient: maskRecipient(recipient),
		Subject:   subject,
		Event:     event,
		Status:    status,
		Error:     errMsg,
		TsMs:      nowMs,
	})
}

// maskRecipient redacts the local part of an email address, keeping the first character
// and the full domain, so recipient PII is not streamed verbatim to the sink.
//
// "alice@example.com" becomes "a***@example.com". A local part of two runes or fewer is
// fully masked ("ab@example.com" becomes "***@example.com") so a short local part is not
// effectively revealed. A value with no "@" is fully masked.
//
// Takes addr (string) which is the raw recipient address.
//
// Returns string which is the masked address.
func maskRecipient(addr string) string {
	if addr == "" {
		return ""
	}
	at := strings.LastIndexByte(addr, '@')
	if at <= 0 {
		return "***"
	}
	if utf8.RuneCountInString(addr[:at]) <= 2 {
		return "***" + addr[at:]
	}

	first, _ := utf8.DecodeRuneInString(addr[:at])
	return string(first) + "***" + addr[at:]
}

// messageID derives a stable-per-send hex id correlating a message's lifecycle records.
//
// Takes recipient (string) which is the message recipient address.
// Takes subject (string) which is the message subject.
// Takes ts (int64) which is the send timestamp in milliseconds.
//
// Returns string which is the stable per-send message id.
func messageID(recipient, subject string, ts int64) string {
	h := fnv.New64a()
	_, _ = io.WriteString(h, recipient+"|"+subject+"|"+strconv.FormatInt(ts, 10))
	return "msg_" + strconv.FormatUint(h.Sum64(), 16)
}

// failedRecipients indexes the failed messages of a bulk send by their first recipient,
// mapping each to the error that caused that message to fail.
//
// Takes multi (*email.MultiError) which holds the per-message bulk failures.
//
// Returns map[string]error keyed by first recipient address.
func failedRecipients(multi *email.MultiError) map[string]error {
	if multi == nil {
		return nil
	}
	out := make(map[string]error, len(multi.Errors))
	for i := range multi.Errors {
		entry := &multi.Errors[i]
		recipient := ""
		if len(entry.Email.To) > 0 {
			recipient = entry.Email.To[0]
		}
		out[recipient] = entry.Error
	}
	return out
}

// perMessageError returns the failure attributed to a single bulk message, or nil when
// that message is not present in the failed set and so was delivered.
//
// Takes params (*email.SendParams) which describes one bulk message.
// Takes failed (map[string]error) which maps failed first recipients to their errors.
//
// Returns error which is the message's failure, or nil when it was delivered.
func perMessageError(params *email.SendParams, failed map[string]error) error {
	if params == nil || len(failed) == 0 {
		return nil
	}
	recipient := ""
	if len(params.To) > 0 {
		recipient = params.To[0]
	}
	return failed[recipient]
}
