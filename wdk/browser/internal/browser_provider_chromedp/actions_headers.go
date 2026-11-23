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

package browser_provider_chromedp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

const (
	// maxRecordedDocuments caps the number of document loads whose headers are kept.
	maxRecordedDocuments = 32
)

var (
	// errNoMainFrame is returned when the page reports no main frame.
	errNoMainFrame = errors.New("page has no main frame")
)

// documentHeaderStore keeps response headers available after history navigation.
type documentHeaderStore struct {
	// byLoader maps a document's loader to its response headers with lower-case names.
	byLoader map[cdp.LoaderID]map[string]string

	// order lists recorded loaders oldest first, for eviction.
	order []cdp.LoaderID

	// mu guards byLoader and order.
	mu sync.Mutex
}

// record stores the headers of a document response event, ignoring every other event.
//
// Takes ev (any) which is the raw target event.
//
// Concurrency: Safe for concurrent use.
func (s *documentHeaderStore) record(ev any) {
	response, ok := ev.(*network.EventResponseReceived)
	if !ok || response.Type != network.ResourceTypeDocument || response.Response == nil || response.LoaderID == "" {
		return
	}

	headers := make(map[string]string, len(response.Response.Headers))
	for name, value := range response.Response.Headers {
		text, isString := value.(string)
		if !isString {
			text = fmt.Sprint(value)
		}
		headers[strings.ToLower(name)] = text
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byLoader[response.LoaderID]; !exists {
		s.order = append(s.order, response.LoaderID)
		if len(s.order) > maxRecordedDocuments {
			delete(s.byLoader, s.order[0])
			s.order = s.order[1:]
		}
	}
	s.byLoader[response.LoaderID] = headers
}

// lookup returns a copy of the headers recorded for a loader.
//
// Takes loaderID (cdp.LoaderID) which identifies the document load.
//
// Returns map[string]string which is a copy of the headers, or nil when none were
// recorded.
//
// Concurrency: Safe for concurrent use.
func (s *documentHeaderStore) lookup(loaderID cdp.LoaderID) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return maps.Clone(s.byLoader[loaderID])
}

// SetExtraHTTPHeaders sets additional HTTP headers that will be sent with all requests.
// These headers are added to all subsequent requests until cleared or changed.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes headers (map[string]string) which contains header name to value pairs.
//
// Returns error when the network domain cannot be enabled or headers cannot be set.
func SetExtraHTTPHeaders(ctx *ActionContext, headers map[string]string) error {
	networkHeaders := make(network.Headers, len(headers))
	for k, v := range headers {
		networkHeaders[k] = v
	}

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		if err := network.Enable().Do(ctx2); err != nil {
			return fmt.Errorf("enabling network: %w", err)
		}
		return network.SetExtraHTTPHeaders(networkHeaders).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting extra HTTP headers: %w", err)
	}
	return nil
}

// SetAuthorizationHeader sets an Authorization header for all requests. This is a
// convenience function for Bearer token authentication.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes token (string) which is the Bearer token value.
//
// Returns error when the header cannot be set.
func SetAuthorizationHeader(ctx *ActionContext, token string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// SetBasicAuthHeader sets a Basic Authorization header for all requests. The credentials
// should be in "username:password" format.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes credentials (string) which specifies the username:password value.
//
// Returns error when the header cannot be set.
func SetBasicAuthHeader(ctx *ActionContext, credentials string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		"Authorization": "Basic " + credentials,
	})
}

// SetUserAgent sets the User-Agent header for all requests. For full device emulation,
// use the EmulateDevice functions instead.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes userAgent (string) which specifies the User-Agent header value.
//
// Returns error when the user agent cannot be set.
func SetUserAgent(ctx *ActionContext, userAgent string) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return emulation.SetUserAgentOverride(userAgent).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting user agent: %w", err)
	}
	return nil
}

// ClearExtraHTTPHeaders clears all extra HTTP headers by setting an empty map.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the headers cannot be cleared.
func ClearExtraHTTPHeaders(ctx *ActionContext) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{})
}

// SetAcceptLanguageHeader sets the Accept-Language header for all requests. Use it to
// test localisation.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes language (string) which specifies the language code to use.
//
// Returns error when setting the HTTP header fails.
func SetAcceptLanguageHeader(ctx *ActionContext, language string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		"Accept-Language": language,
	})
}

// SetCustomHeader sets a single custom header. Use it when only one header is needed.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes name (string) which specifies the header name.
// Takes value (string) which specifies the header value.
//
// Returns error when the header cannot be set.
func SetCustomHeader(ctx *ActionContext, name, value string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		name: value,
	})
}

// GetResponseHeaders returns the response headers of the document currently shown.
//
// The result always includes contentType and lastModified from the document. An action
// context with a PageHelper also supplies the recorded HTTP headers in lower case.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns map[string]string which contains the response headers as key-value pairs.
// Returns error when the frame tree or the document cannot be read.
func GetResponseHeaders(ctx *ActionContext) (map[string]string, error) {
	headers := make(map[string]string)
	if ctx.PageHelper != nil {
		recorded, err := ctx.PageHelper.currentDocumentHeaders(ctx.Ctx)
		if err != nil {
			return nil, fmt.Errorf("getting response headers: %w", err)
		}
		maps.Copy(headers, recorded)
	}

	js := scripts.MustGet("get_response_headers.js")

	var result map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting response headers: %w", err)
	}

	for k, v := range result {
		if str, ok := v.(string); ok {
			headers[k] = str
		}
	}
	return headers, nil
}

// newDocumentHeaderStore creates an empty store.
//
// Returns *documentHeaderStore which is ready to record responses.
func newDocumentHeaderStore() *documentHeaderStore {
	return &documentHeaderStore{
		byLoader: make(map[cdp.LoaderID]map[string]string),
		order:    nil,
		mu:       sync.Mutex{},
	}
}

// mainFrameLoaderID returns the loader of the document shown in the main frame.
//
// Returns cdp.LoaderID which identifies the current main frame document.
// Returns error when the frame tree cannot be read or has no main frame.
func mainFrameLoaderID(ctx context.Context) (cdp.LoaderID, error) {
	var loaderID cdp.LoaderID
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		tree, err := page.GetFrameTree().Do(runCtx)
		if err != nil {
			return err
		}
		if tree == nil || tree.Frame == nil {
			return errNoMainFrame
		}
		loaderID = tree.Frame.LoaderID
		return nil
	}))
	if err != nil {
		return "", fmt.Errorf("reading frame tree: %w", err)
	}
	return loaderID, nil
}
