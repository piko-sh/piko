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
	"net/http"
	"time"
)

const (
	// httpStatusOK is the lower bound for successful HTTP responses.
	httpStatusOK = 200

	// httpStatusMultiStatus is the upper bound for successful HTTP status codes.
	httpStatusMultiStatus = 300

	// defaultHTTPTimeout is the timeout for webhook HTTP requests when no
	// custom client is provided. This prevents goroutine leaks when a
	// webhook endpoint is unresponsive.
	defaultHTTPTimeout = 30 * time.Second
)

// defaultHTTPClient is a shared HTTP client with a timeout, used by all
// notification providers when no custom client is supplied.
var defaultHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}

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
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < httpStatusOK || response.StatusCode >= httpStatusMultiStatus {
		return fmt.Errorf("%s returned status %d: %s", providerName, response.StatusCode, response.Status)
	}

	return nil
}
