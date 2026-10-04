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

package siteverify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/json"
)

const (
	// DefaultTimeout bounds one verification call when no timeout is configured.
	DefaultTimeout = 10 * time.Second

	// DefaultMaxResponseBytes caps the verification response body when no limit is
	// configured. Real responses are a few hundred bytes.
	DefaultMaxResponseBytes int64 = 64 * 1024
)

// Client posts verification requests to one provider's siteverify endpoint.
type Client struct {
	// httpClient sends the requests and enforces the timeout.
	httpClient *http.Client

	// provider names the captcha provider in error messages.
	provider string

	// endpoint is the siteverify URL.
	endpoint string

	// maxResponseBytes is the largest response body accepted.
	maxResponseBytes int64
}

// Option configures a Client.
type Option func(*settings)

// settings collects the values the options set.
type settings struct {
	// timeout bounds one verification call.
	timeout time.Duration

	// maxResponseBytes is the largest response body accepted.
	maxResponseBytes int64
}

// NewClient creates a client for one provider's siteverify endpoint.
//
// Takes provider (string) which names the provider in error messages.
// Takes endpoint (string) which is the siteverify URL.
// Takes options (...Option) which override the timeout and response size limit.
//
// Returns *Client which is ready for use.
func NewClient(provider, endpoint string, options ...Option) *Client {
	configured := settings{timeout: DefaultTimeout, maxResponseBytes: DefaultMaxResponseBytes}
	for _, option := range options {
		option(&configured)
	}

	return &Client{
		httpClient:       &http.Client{Timeout: configured.timeout},
		provider:         provider,
		endpoint:         endpoint,
		maxResponseBytes: configured.maxResponseBytes,
	}
}

// readBody checks the response status and content type and reads the body within the size
// limit.
//
// Takes response (*http.Response) which is the siteverify response; the caller drains and
// closes its body.
//
// Returns []byte which is the whole body.
// Returns error when the status or content type is unexpected, the body cannot be read,
// or it is larger than the limit.
func (c *Client) readBody(response *http.Response) ([]byte, error) {
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s verification returned HTTP %d: %w",
			c.provider, response.StatusCode, captcha_dto.ErrProviderUnavailable)
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("%s returned unexpected content type %q: %w",
			c.provider, contentType, captcha_dto.ErrProviderUnavailable)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s response body: %w", c.provider, err)
	}

	if int64(len(body)) > c.maxResponseBytes {
		return nil, fmt.Errorf("%s response body exceeded %d byte limit: %w",
			c.provider, c.maxResponseBytes, captcha_dto.ErrProviderUnavailable)
	}

	return body, nil
}

// WithTimeout bounds each verification call, including reading the response.
//
// Takes timeout (time.Duration) which is the limit; zero or negative keeps
// DefaultTimeout.
//
// Returns Option which applies the timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(s *settings) {
		if timeout > 0 {
			s.timeout = timeout
		}
	}
}

// WithMaxResponseBytes caps the size of a verification response body; a larger body is
// reported as an error rather than truncated.
//
// Takes limit (int64) which is the largest body accepted; zero or negative keeps
// DefaultMaxResponseBytes.
//
// Returns Option which applies the limit.
func WithMaxResponseBytes(limit int64) Option {
	return func(s *settings) {
		if limit > 0 {
			s.maxResponseBytes = limit
		}
	}
}

// Verify posts the form to the client's endpoint and decodes the JSON verdict into T.
//
// Takes client (*Client) which identifies the endpoint and limits.
// Takes form (url.Values) which is the form-encoded verification request.
//
// Returns T which is the decoded verdict.
// Returns error when the request fails, the JSON cannot be parsed, or the endpoint
// answers with a status other than 200, a content type other than JSON, or a body larger
// than the limit; the last three wrap captcha_dto.ErrProviderUnavailable.
func Verify[T any](ctx context.Context, client *Client, form url.Values) (T, error) {
	var verdict T

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return verdict, fmt.Errorf("creating %s request: %w", client.provider, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return verdict, fmt.Errorf("sending %s verification request: %w", client.provider, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, client.maxResponseBytes))
		_ = response.Body.Close()
	}()

	body, err := client.readBody(response)
	if err != nil {
		return verdict, err
	}

	if err := json.Unmarshal(body, &verdict); err != nil {
		return verdict, fmt.Errorf("parsing %s response: %w", client.provider, err)
	}

	return verdict, nil
}
