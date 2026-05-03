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

package driver_providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/safeerror"
)

const (
	// httpStatusOK is the lower bound for successful HTTP responses.
	httpStatusOK = 200

	// httpStatusMultiStatus is the upper bound for successful HTTP status codes.
	httpStatusMultiStatus = 300

	// httpStatusClientErrorMin is the lower bound (inclusive) for 4xx client
	// errors that surface to users via safeerror wrapping.
	httpStatusClientErrorMin = 400

	// httpStatusClientErrorMax is the upper bound (exclusive) for 4xx client
	// errors.
	httpStatusClientErrorMax = 500

	// defaultHTTPTimeout is the timeout for webhook HTTP requests when no
	// custom client is provided. This prevents goroutine leaks when a
	// webhook endpoint is unresponsive.
	defaultHTTPTimeout = 30 * time.Second

	// maxNotificationResponseBytes caps the bytes read from a third-party
	// webhook response body, guarding against hostile or runaway servers.
	maxNotificationResponseBytes = 64 * 1024

	// safeNotificationDeliveryFailed is the user-facing message returned when a
	// notification provider rejects the delivery with a client-error response.
	safeNotificationDeliveryFailed = "notification delivery failed"

	// truncateRunesEllipsisLength is the rune length of the ellipsis suffix
	// appended when truncateRunes shortens an oversized string.
	truncateRunesEllipsisLength = 3
)

// defaultHTTPClient is a shared HTTP client with a timeout, used by all
// notification providers when no custom client is supplied.
var defaultHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}

// drainAndCloseResponse drains a capped amount of body bytes then closes.
//
// Drains up to maxNotificationResponseBytes from response.Body before
// closing it. The cap prevents a hostile or runaway upstream server from
// forcing the client to read an unbounded amount of bytes purely to
// enable connection reuse. Bytes beyond the cap are left on the wire and
// the connection may not be reused, which is the safe trade-off.
//
// Takes response (*http.Response) which is the upstream response whose body
// should be drained and closed.
func drainAndCloseResponse(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.CopyN(io.Discard, response.Body, maxNotificationResponseBytes)
	_ = response.Body.Close()
}

// sendHTTPJSONPayload is a common helper for sending a JSON payload via POST.
//
// Takes httpClient (*http.Client) which is used to execute the request.
// Takes webhookURL (string) which specifies the destination URL.
// Takes payload ([]byte) which contains the JSON data to send.
// Takes providerName (string) which is used in error messages.
//
// Returns error when request creation fails, the HTTP call fails, or the
// status code is not in the success range [200, 300).
func sendHTTPJSONPayload(
	ctx context.Context,
	httpClient *http.Client,
	webhookURL string,
	payload []byte,
	providerName string,
) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating %s request: %w", providerName, err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending to %s: %w", providerName, err)
	}
	defer drainAndCloseResponse(response)

	if response.StatusCode < httpStatusOK || response.StatusCode >= httpStatusMultiStatus {
		statusErr := buildProviderStatusError(providerName, response)
		if isClientError(response.StatusCode) {
			return safeerror.NewError(safeNotificationDeliveryFailed, statusErr)
		}
		return statusErr
	}

	return nil
}

// buildProviderStatusError wraps a non-success HTTP response in a
// notification_domain.ProviderError carrying the upstream status code and any
// Retry-After hint, so the dispatcher can honour rate-limiting guidance.
//
// Takes providerName (string) which identifies the provider for diagnostics.
// Takes response (*http.Response) which is the non-success upstream response.
//
// Returns error which is a *notification_domain.ProviderError wrapping the
// status detail.
func buildProviderStatusError(providerName string, response *http.Response) error {
	cause := fmt.Errorf("%s returned status %d: %s", providerName, response.StatusCode, response.Status)
	return &notification_domain.ProviderError{
		Err:        cause,
		Provider:   providerName,
		StatusCode: response.StatusCode,
		RetryAfter: notification_domain.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now()),
	}
}

// isClientError reports whether the HTTP status code is in the 4xx range. The
// caller is expected to wrap such errors with safeerror so they reach the HTTP
// edge sanitised.
//
// Takes statusCode (int) which is the HTTP status code from the upstream
// response.
//
// Returns bool which is true when the status code lies in [400, 500).
func isClientError(statusCode int) bool {
	return statusCode >= httpStatusClientErrorMin && statusCode < httpStatusClientErrorMax
}

// truncateRunes shortens s to at most maxRunes runes, appending "..." when the
// original string was longer. The function is rune-aware so it never cuts
// through a multi-byte UTF-8 sequence.
//
// Takes s (string) which is the input to truncate.
// Takes maxRunes (int) which is the maximum number of runes the result may
// contain.
//
// Returns string which is at most maxRunes runes long.
func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if maxRunes <= truncateRunesEllipsisLength {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-truncateRunesEllipsisLength]) + "..."
}
