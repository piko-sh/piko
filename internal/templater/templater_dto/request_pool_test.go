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

package templater_dto_test

import (
	"net/http/httptest"
	"testing"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkRequestDataBuilder_Build(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rd := templater_dto.NewRequestDataBuilder().
			WithMethod("GET").
			WithHost("localhost").
			WithLocale("en-GB").
			WithDefaultLocale("en-GB").
			AddPathParam("id", "123").
			AddQueryParamValue("page", "1").
			Build()

		_ = rd.Method()
		_ = rd.PathParam("id")
		_ = rd.QueryParam("page")

		rd.Release()
	}
}

func BenchmarkRequestDataBuilder_BuildNoRelease(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rd := templater_dto.NewRequestDataBuilder().
			WithMethod("GET").
			WithHost("localhost").
			WithLocale("en-GB").
			WithDefaultLocale("en-GB").
			AddPathParam("id", "123").
			AddQueryParamValue("page", "1").
			Build()

		_ = rd.Method()
		_ = rd.PathParam("id")
		_ = rd.QueryParam("page")
	}
}

func BenchmarkLocaleFallbackOrder(b *testing.B) {
	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en-GB").
		WithDefaultLocale("fr-FR").
		Build()
	defer rd.Release()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = rd.T("test.key")
	}
}

func BenchmarkRequestDataWithTranslations(b *testing.B) {
	translations := map[string]map[string]string{
		"en-GB": {
			"greeting": "Hello",
			"farewell": "Goodbye",
		},
		"en": {
			"greeting": "Hi",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rd := templater_dto.NewRequestDataBuilder().
			WithLocale("en-GB").
			WithDefaultLocale("en").
			Build()

		rd.SetGlobalStore(i18n_domain.NewStoreFromTranslations(translations, "en"))

		_ = rd.T("greeting")
		_ = rd.T("farewell")
		_ = rd.T("missing", "fallback")

		rd.Release()
	}
}

func BenchmarkParseRequestDataFromHTTP(b *testing.B) {
	request := httptest.NewRequest("GET", "/test?foo=bar&baz=qux&page=1", nil)
	request.Header.Set("Accept-Language", "en-GB,en;q=0.9")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rd := templater_dto.NewRequestDataBuilder().
			WithMethod(request.Method).
			WithHost(request.Host).
			WithLocale("en-GB").
			WithDefaultLocale("en-GB").
			Build()

		_ = rd.QueryParam("foo")
		_ = rd.QueryParam("baz")
		_ = rd.QueryParam("page")

		rd.Release()
	}
}

func BenchmarkPoolContention(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rd := templater_dto.NewRequestDataBuilder().
				WithMethod("GET").
				WithHost("localhost").
				WithLocale("en-GB").
				WithDefaultLocale("en-GB").
				AddPathParam("id", "123").
				Build()

			_ = rd.Method()
			_ = rd.PathParam("id")
			_ = rd.T("test")

			rd.Release()
		}
	})
}
