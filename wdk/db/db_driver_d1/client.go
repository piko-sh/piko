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

package db_driver_d1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

const (
	// rawQueryEndpointFormat is the D1 endpoint that returns each result set as an ordered
	// column list plus row arrays, preserving column order and duplicate column names.
	rawQueryEndpointFormat = "/accounts/%s/d1/database/%s/raw"

	// automaticRetries is the number of times the Cloudflare client may resend a request.
	// Statements are not known to be idempotent (an INSERT or a batched COMMIT resent after
	// an ambiguous 5xx would apply twice), so the client never resends on its own.
	automaticRetries = 0
)

var (
	// errCallTimeout is the cancellation cause attached to a D1 API call once the configured
	// request timeout elapses.
	errCallTimeout = errors.New("db_driver_d1: D1 API call exceeded the request timeout")
)

// d1Client performs D1 API calls for every connection of one database handle, so the
// connections share a single HTTP connection pool and a single rate limiter.
type d1Client struct {
	// api is the Cloudflare API client configured without automatic retries.
	api *cloudflare.API

	// httpClient is the bounded HTTP client the API client sends requests through.
	httpClient *http.Client

	// endpoint is the account- and database-specific raw query path.
	endpoint string

	// requestTimeout bounds every call, including the wait for the shared rate limiter.
	requestTimeout time.Duration
}

// rawQueryRequest is the JSON body of a raw query request.
type rawQueryRequest struct {
	// SQL is the statement text, which may hold several semicolon-separated statements.
	SQL string `json:"sql"`

	// Parameters holds the positional parameters bound across all statements.
	Parameters []string `json:"params"`
}

// rawStatementResult is the outcome of one statement in a raw query response.
type rawStatementResult struct {
	// Success reports whether the statement succeeded; nil means the flag was absent.
	Success *bool `json:"success"`

	// Results holds the ordered columns and row arrays the statement produced.
	Results rawResultSet `json:"results"`

	// Meta holds the change counters reported for the statement.
	Meta rawStatementMeta `json:"meta"`
}

// rawResultSet is the ordered tabular payload of a raw query statement.
type rawResultSet struct {
	// Columns lists the result column names in server order, duplicates included.
	Columns []string `json:"columns"`

	// Rows holds one value array per row, positionally aligned with Columns.
	Rows [][]any `json:"rows"`
}

// rawStatementMeta holds the change counters reported for one statement.
type rawStatementMeta struct {
	// LastRowID is the rowid of the last row the statement inserted.
	LastRowID int64 `json:"last_row_id"`

	// Changes is the number of rows the statement modified.
	Changes int64 `json:"changes"`
}

// query sends sql with params to the raw query endpoint and decodes the per-statement
// results.
//
// The call is bounded by the configured request timeout as well as by ctx. The timeout is
// applied through the context rather than http.Client.Timeout, because the Cloudflare
// client abandons error responses without closing them, which would otherwise leave the
// client's timeout watcher running until it fired.
//
// Takes sql (string) which is the statement text to execute.
// Takes params ([]string) which are the positional parameters.
//
// Returns []rawStatementResult which holds one entry per executed statement, each checked
// for success.
// Returns error when the request fails, the API reports failure, the response cannot be
// decoded, or any statement reports failure.
func (c *d1Client) query(ctx context.Context, sql string, params []string) ([]rawStatementResult, error) {
	if params == nil {
		params = []string{}
	}

	ctx, cancel := context.WithTimeoutCause(ctx, c.requestTimeout, errCallTimeout)
	defer cancel()

	response, err := c.api.Raw(ctx, http.MethodPost, c.endpoint, rawQueryRequest{SQL: sql, Parameters: params}, nil)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, describeAPIFailure(response.Errors)
	}

	results, err := decodeStatementResults(response.Result)
	if err != nil {
		return nil, err
	}
	for index := range results {
		if successErr := checkStatementSuccess(results[index]); successErr != nil {
			return nil, fmt.Errorf("statement %d: %w", index, successErr)
		}
	}
	return results, nil
}

// close releases the idle HTTP connections held for this database handle.
func (c *d1Client) close() {
	c.httpClient.CloseIdleConnections()
}

// newD1Client builds the shared client for one D1 database.
//
// Takes config (Config) which supplies the credentials and identifiers.
// Takes options (clientOptions) which supplies the timeout, size, and rate limits.
//
// Returns *d1Client which is ready to issue queries.
// Returns error when the Cloudflare API client cannot be created.
func newD1Client(config Config, options clientOptions) (*d1Client, error) {
	httpClient := &http.Client{
		Transport: &boundedBodyTransport{
			base:             newBaseTransport(),
			maxResponseBytes: options.maxResponseBytes,
		},
	}

	apiOptions := []cloudflare.Option{
		cloudflare.HTTPClient(httpClient),
		cloudflare.UsingRateLimit(options.requestsPerSecond),
		cloudflare.UsingRetryPolicy(automaticRetries, 0, 0),
	}
	if options.baseURL != "" {
		apiOptions = append(apiOptions, cloudflare.BaseURL(options.baseURL))
	}

	api, err := cloudflare.NewWithAPIToken(config.APIToken, apiOptions...)
	if err != nil {
		httpClient.CloseIdleConnections()
		return nil, fmt.Errorf("creating API client: %w", err)
	}

	return &d1Client{
		api:            api,
		httpClient:     httpClient,
		endpoint:       fmt.Sprintf(rawQueryEndpointFormat, url.PathEscape(config.AccountID), url.PathEscape(config.DatabaseID)),
		requestTimeout: options.requestTimeout,
	}, nil
}

// decodeStatementResults decodes the result array of a raw query response, keeping
// numbers as json.Number so integers beyond 2^53 are not rounded through float64.
//
// Takes payload ([]byte) which is the JSON result array.
//
// Returns []rawStatementResult which holds the decoded per-statement results.
// Returns error when the payload is not a valid result array.
func decodeStatementResults(payload []byte) ([]rawStatementResult, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var results []rawStatementResult
	if err := decoder.Decode(&results); err != nil {
		return nil, fmt.Errorf("decoding D1 results: %w", err)
	}
	return results, nil
}

// checkStatementSuccess reports whether a statement result indicates success. The D1 API
// populates a per-statement success flag, so a missing flag is treated as a failure
// rather than silently assumed successful.
//
// Takes result (rawStatementResult) which is the per-statement result to inspect.
//
// Returns error which is non-nil when the result reports failure or omits the success
// flag.
func checkStatementSuccess(result rawStatementResult) error {
	if result.Success == nil {
		return errors.New("D1 result omitted the success flag")
	}
	if !*result.Success {
		return errors.New("D1 query returned failure")
	}
	return nil
}

// describeAPIFailure builds an error from the messages of an unsuccessful API response.
//
// Takes failures ([]cloudflare.ResponseInfo) which are the error entries the API
// returned.
//
// Returns error which lists every reported code and message.
func describeAPIFailure(failures []cloudflare.ResponseInfo) error {
	if len(failures) == 0 {
		return errors.New("D1 API reported failure without an error message")
	}
	messages := make([]string, 0, len(failures))
	for _, failure := range failures {
		messages = append(messages, fmt.Sprintf("%d: %s", failure.Code, failure.Message))
	}
	return fmt.Errorf("D1 API reported failure: %s", strings.Join(messages, "; "))
}
