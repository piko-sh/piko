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

package pikotest_domain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pikotest/pikotest_domain"
)

func TestNewRequest_Defaults(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/test").Build(context.Background())

	assert.Equal(t, "GET", request.Method())
	assert.Equal(t, "localhost", request.Host())
	assert.Equal(t, "en", request.Locale())
	assert.Equal(t, "en", request.DefaultLocale())
	require.NotNil(t, request.URL())
	assert.Equal(t, "/test", request.URL().Path)
}

func TestRequestBuilder_WithQueryParam(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/search").
		WithQueryParam("q", "hello").
		WithQueryParam("page", "1").
		Build(context.Background())

	assert.Equal(t, "hello", request.QueryParam("q"))
	assert.Equal(t, "1", request.QueryParam("page"))
}

func TestRequestBuilder_WithQueryParams(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/search").
		WithQueryParams(map[string][]string{
			"q":    {"hello"},
			"tags": {"go", "piko"},
		}).
		Build(context.Background())

	assert.Equal(t, "hello", request.QueryParam("q"))
	assert.Equal(t, []string{"go", "piko"}, request.QueryParamValues("tags"))
}

func TestRequestBuilder_WithFormData(t *testing.T) {
	request := pikotest_domain.NewRequest("POST", "/submit").
		WithFormData("name", "Alice").
		WithFormData("age", "30").
		Build(context.Background())

	assert.Equal(t, "Alice", request.FormValue("name"))
	assert.Equal(t, "30", request.FormValue("age"))
}

func TestRequestBuilder_WithFormDataMap(t *testing.T) {
	request := pikotest_domain.NewRequest("POST", "/submit").
		WithFormDataMap(map[string][]string{
			"name":    {"Alice"},
			"hobbies": {"reading", "coding"},
		}).
		Build(context.Background())

	assert.Equal(t, "Alice", request.FormValue("name"))
	assert.Equal(t, []string{"reading", "coding"}, request.FormValues("hobbies"))
}

func TestRequestBuilder_WithCollectionData(t *testing.T) {
	type Item struct{ Name string }
	data := []Item{{Name: "first"}, {Name: "second"}}

	request := pikotest_domain.NewRequest("GET", "/").
		WithCollectionData(data).
		Build(context.Background())

	require.NotNil(t, request)
}

func TestRequestBuilder_WithPathParam(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/users/:id").
		WithPathParam("id", "42").
		Build(context.Background())

	assert.Equal(t, "42", request.PathParam("id"))
}

func TestRequestBuilder_WithPathParams(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/orgs/:orgID/users/:userID").
		WithPathParams(map[string]string{
			"orgID":  "abc",
			"userID": "123",
		}).
		Build(context.Background())

	assert.Equal(t, "abc", request.PathParam("orgID"))
	assert.Equal(t, "123", request.PathParam("userID"))
}

func TestRequestBuilder_WithLocale(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/").
		WithLocale("es").
		Build(context.Background())

	assert.Equal(t, "es", request.Locale())
}

func TestRequestBuilder_WithDefaultLocale(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/").
		WithDefaultLocale("fr").
		Build(context.Background())

	assert.Equal(t, "fr", request.DefaultLocale())
}

func TestRequestBuilder_WithHost(t *testing.T) {
	request := pikotest_domain.NewRequest("GET", "/").
		WithHost("example.com").
		Build(context.Background())

	assert.Equal(t, "example.com", request.Host())
}

func TestRequestBuilder_Build_WithContext(t *testing.T) {
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("db"), "mock-db")

	request := pikotest_domain.NewRequest("GET", "/").
		Build(ctx)

	assert.Equal(t, "mock-db", request.Context().Value(ctxKey("db")))
}

func TestRequestBuilder_WithGlobalTranslations(t *testing.T) {
	translations := map[string]map[string]string{
		"en": {"greeting": "Hello"},
		"es": {"greeting": "Hola"},
	}

	request := pikotest_domain.NewRequest("GET", "/").
		WithGlobalTranslations(translations).
		Build(context.Background())

	result := request.T("greeting")
	require.NotNil(t, result)
	assert.Equal(t, "Hello", result.String())
}

func TestRequestBuilder_WithLocalTranslations(t *testing.T) {
	localTranslations := map[string]map[string]string{
		"en": {"label": "Submit"},
	}

	request := pikotest_domain.NewRequest("GET", "/").
		WithLocalTranslations(localTranslations).
		Build(context.Background())

	result := request.LT("label")
	require.NotNil(t, result)
	assert.Equal(t, "Submit", result.String())
}

func TestRequestBuilder_BuildHTTPRequest(t *testing.T) {
	httpReq, reqData := pikotest_domain.NewRequest("POST", "/submit").
		WithQueryParam("q", "test").
		WithHost("example.com").
		WithHeader("X-Custom", "value").
		BuildHTTPRequest(context.Background())

	require.NotNil(t, httpReq)
	require.NotNil(t, reqData)

	assert.Equal(t, "POST", httpReq.Method)
	assert.Equal(t, "example.com", httpReq.Host)
	assert.Equal(t, "value", httpReq.Header.Get("X-Custom"))
	assert.Equal(t, "test", reqData.QueryParam("q"))
}

func TestRequestBuilder_Chaining(t *testing.T) {

	builder := pikotest_domain.NewRequest("GET", "/").
		WithQueryParam("a", "1").
		WithQueryParams(map[string][]string{"b": {"2"}}).
		WithFormData("c", "3").
		WithFormDataMap(map[string][]string{"d": {"4"}}).
		WithPathParam("e", "5").
		WithPathParams(map[string]string{"f": "6"}).
		WithLocale("en").
		WithDefaultLocale("en").
		WithHost("localhost").
		WithHeader("X-Test", "test").
		WithGlobalTranslations(nil).
		WithLocalTranslations(nil).
		WithCollectionData(nil)

	request := builder.Build(context.Background())
	require.NotNil(t, request)
}
