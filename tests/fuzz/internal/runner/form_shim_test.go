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

package runner

import (
	"errors"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestFormShimSensitivity(t *testing.T) {
	model := generate.New("form-shim-contracts", 7, 0).Expected.FormShim
	good := FormShimReceipt{PageStatus: http.StatusOK, FormCount: len(model.Steps)}
	for _, step := range model.Steps {
		good.Steps = append(good.Steps, FormShimRead{Name: step.Name, Status: step.Status, Code: step.Code, RetryAdvertised: step.Status == http.StatusTooManyRequests, Write: step.Write, After: step.After})
	}
	require.NoError(t, checkFormShim(model, &good))
	data, err := json.Marshal(good)
	require.NoError(t, err)
	tests := []struct {
		mutate func(*FormShimReceipt)
		field  string
	}{
		{field: "order", mutate: func(receipt *FormShimReceipt) {
			receipt.Steps[0], receipt.Steps[1] = receipt.Steps[1], receipt.Steps[0]
		}},
		{field: "status", mutate: func(receipt *FormShimReceipt) { receipt.Steps[1].Status = http.StatusOK }},
		{field: "code", mutate: func(receipt *FormShimReceipt) { receipt.Steps[1].Code = "OTHER" }},
		{field: "write-presence", mutate: func(receipt *FormShimReceipt) { receipt.Steps[0].Write = nil }},
		{field: "write-identity", mutate: func(receipt *FormShimReceipt) { receipt.Steps[0].Write.Revision++ }},
		{field: "score", mutate: func(receipt *FormShimReceipt) { receipt.Steps[0].Write.ScorePercent = 0 }},
		{field: "request-metadata", mutate: func(receipt *FormShimReceipt) { receipt.Steps[0].Write.Method = "OTHER" }},
		{field: "provider-count", mutate: func(receipt *FormShimReceipt) { receipt.Steps[len(receipt.Steps)-1].After.Checks++ }},
		{field: "write-count", mutate: func(receipt *FormShimReceipt) { receipt.Steps[1].After.Writes++ }},
		{field: "history", mutate: func(receipt *FormShimReceipt) {
			events := receipt.Steps[0].After.Events
			events[0], events[1] = events[1], events[0]
		}},
		{field: "retry", mutate: func(receipt *FormShimReceipt) { receipt.Steps[len(receipt.Steps)-1].RetryAdvertised = false }},
	}
	for _, testCase := range tests {
		t.Run(testCase.field, func(t *testing.T) {
			var changed FormShimReceipt
			require.NoError(t, json.Unmarshal(data, &changed))
			testCase.mutate(&changed)
			err := checkFormShim(model, &changed)
			require.Error(t, err, "wanted %s", testCase.field)
			assert.Contains(t, err.Error(), testCase.field)
			assert.Error(t, compareFormShim(&good, &changed), "mode comparator accepted changed receipt")
		})
	}
}

func newFakeFormShimServer(t *testing.T, model *casefile.FormShimContract) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	submitted := 0
	var page strings.Builder
	page.WriteString(fakeActionPage[:strings.Index(fakeActionPage, "<body>")+len("<body>")])
	for _, step := range model.Steps {
		page.WriteString(`<form data-case="` + html.EscapeString(step.Name) + `" action="/_piko/actions/intake.Submit" method="post">`)
		page.WriteString(`<input name="label" value="` + html.EscapeString(step.Label) + `">`)
		if step.Ticket != "" {
			page.WriteString(`<input name="_captcha_token" value="` + html.EscapeString(step.Ticket) + `">`)
		}
		page.WriteString(`</form>`)
	}
	page.WriteString(`</body></html>`)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /intake", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, page.String())
	})
	mux.HandleFunc("POST /_piko/actions/intake.Read", func(writer http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		state := casefile.FormShimState{Events: []string{}}
		if submitted > 0 {
			state = model.Steps[submitted-1].After
		}
		mu.Unlock()
		writeFakeJSON(writer, http.StatusOK, shimStateWire{Checks: &state.Checks, Writes: &state.Writes, Events: &state.Events})
	})
	mux.HandleFunc("POST /_piko/actions/intake.Submit", func(writer http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		step := model.Steps[submitted]
		submitted++
		mu.Unlock()
		switch step.Status {
		case http.StatusOK:
			writeFakeJSON(writer, http.StatusOK, step.Write)
		case http.StatusTooManyRequests:
			writer.Header().Set("Retry-After", "30")
			writer.Header().Set(headerContentType, "text/plain; charset=utf-8")
			writer.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(writer, "Rate limit exceeded")
		default:
			writeFakeJSON(writer, step.Status, shimRefusal{Code: step.Code, Status: step.Status})
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestObserveFormShimRecordsEachSubmittedForm(t *testing.T) {
	model := generate.New("form-shim-contracts", 7, 0).Expected.FormShim
	server := newFakeFormShimServer(t, model)

	receipt, err := observeFormShim(t.Context(), server.Client(), server.URL, fakeActionPage, model)

	require.NoError(t, err)
	assert.NoError(t, checkFormShim(model, receipt))
}

func TestObserveFormShimRejectsMismatchedPages(t *testing.T) {
	model := generate.New("form-shim-contracts", 7, 0).Expected.FormShim
	server := newFakeFormShimServer(t, model)
	fewer := *model
	fewer.Steps = model.Steps[:1]
	wrongRoute := *model
	wrongRoute.Page = "/other"
	renamed := *model
	renamed.Steps = append([]casefile.FormShimStep{{Name: "renamed"}}, model.Steps[1:]...)

	for name, testCase := range map[string]struct {
		model *casefile.FormShimContract
		page  string
		want  string
	}{
		"route":         {model: &wrongRoute, page: fakeActionPage, want: "form-shim/model"},
		"form count":    {model: &fewer, page: fakeActionPage, want: "form-shim/forms"},
		"form identity": {model: &renamed, page: fakeActionPage, want: "form-shim/renamed/form"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := observeFormShim(t.Context(), server.Client(), server.URL, testCase.page, testCase.model)
			mismatch, ok := errors.AsType[*responseMismatch](err)
			require.True(t, ok, "%v", err)
			assert.Equal(t, testCase.want, mismatch.FailureSubject())
		})
	}
	_, err := observeFormShim(t.Context(), server.Client(), server.URL, "<html></html>", model)
	assert.ErrorContains(t, err, "action/csrf")
}

func TestReadShimPageRejectsFailedAndOversizedPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/large" {
			_, _ = io.WriteString(writer, strings.Repeat("a", shimPageLimit+1))
			return
		}
		http.NotFound(writer, request)
	}))
	t.Cleanup(server.Close)

	receipt := &FormShimReceipt{}
	_, err := readShimPage(t.Context(), server.Client(), server.URL, "/missing", receipt)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, receipt.PageStatus)
	_, err = readShimPage(t.Context(), server.Client(), server.URL, "/large", receipt)
	assert.ErrorContains(t, err, "256 KiB")
}

func TestReadQuotaResponseRequiresThePlainQuotaBody(t *testing.T) {
	respond := func(status int, media, body string) *http.Response {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{headerContentType: {media}, "Retry-After": {"5"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}
	}

	advertised, err := readQuotaResponse(respond(http.StatusTooManyRequests, "text/plain", "Rate limit exceeded"), "step")
	require.NoError(t, err)
	assert.True(t, advertised)
	advertised, err = readQuotaResponse(respond(http.StatusServiceUnavailable, "text/html", "down"), "step")
	require.NoError(t, err)
	assert.True(t, advertised)
	_, err = readQuotaResponse(respond(http.StatusTooManyRequests, "text/html", "Rate limit exceeded"), "step")
	mismatch, ok := errors.AsType[*responseMismatch](err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, "form-shim/step/quota-response", mismatch.FailureSubject())
}
