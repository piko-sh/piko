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
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestNewRequestDataBuilder_Defaults(t *testing.T) {
	t.Parallel()

	b := templater_dto.NewRequestDataBuilder()
	rd := b.Build()
	defer rd.Release()

	assert.NotNil(t, rd.Context(), "default context should not be nil")
	assert.Empty(t, rd.Method())
	assert.Empty(t, rd.Host())
	assert.Empty(t, rd.Locale())
	assert.Equal(t, "en_GB", rd.DefaultLocale())
	assert.Nil(t, rd.CollectionData())
	assert.Nil(t, rd.URL())
}

func TestRequestDataBuilder_WithContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	rd := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		Build()
	defer rd.Release()

	assert.Equal(t, ctx, rd.Context())
}

func TestRequestDataBuilder_WithMethod(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithMethod("POST").
		Build()
	defer rd.Release()

	assert.Equal(t, "POST", rd.Method())
}

func TestRequestDataBuilder_WithHost(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithHost("example.com").
		Build()
	defer rd.Release()

	assert.Equal(t, "example.com", rd.Host())
}

func TestRequestDataBuilder_WithURL(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("https://example.com/path?q=1")
	require.NoError(t, err)

	rd := templater_dto.NewRequestDataBuilder().
		WithURL(u).
		Build()
	defer rd.Release()

	got := rd.URL()
	require.NotNil(t, got)
	assert.Equal(t, "https://example.com/path?q=1", got.String())
}

func TestRequestDataBuilder_WithLocale(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("fr-FR").
		Build()
	defer rd.Release()

	assert.Equal(t, "fr-FR", rd.Locale())
}

func TestRequestDataBuilder_WithDefaultLocale(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithDefaultLocale("de-DE").
		Build()
	defer rd.Release()

	assert.Equal(t, "de-DE", rd.DefaultLocale())
}

func TestRequestDataBuilder_WithCollectionData(t *testing.T) {
	t.Parallel()

	data := map[string]int{"count": 42}
	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(data).
		Build()
	defer rd.Release()

	assert.Equal(t, data, rd.CollectionData())
}

func TestRequestDataBuilder_AddPathParam(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddPathParam("id", "123").
		AddPathParam("slug", "hello").
		Build()
	defer rd.Release()

	assert.Equal(t, "123", rd.PathParam("id"))
	assert.Equal(t, "hello", rd.PathParam("slug"))
}

func TestRequestDataBuilder_AddQueryParam(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddQueryParam("tags", []string{"a", "b"}).
		Build()
	defer rd.Release()

	assert.Equal(t, []string{"a", "b"}, rd.QueryParamValues("tags"))
}

func TestRequestDataBuilder_AddQueryParamValue(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddQueryParamValue("page", "1").
		AddQueryParamValue("page", "2").
		Build()
	defer rd.Release()

	assert.Equal(t, "1", rd.QueryParam("page"))
	assert.Equal(t, []string{"1", "2"}, rd.QueryParamValues("page"))
}

func TestRequestDataBuilder_AddFormData(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddFormData("colours", []string{"red", "blue"}).
		Build()
	defer rd.Release()

	assert.Equal(t, []string{"red", "blue"}, rd.FormValues("colours"))
}

func TestRequestDataBuilder_AddFormDataValue(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddFormDataValue("name", "Alice").
		AddFormDataValue("name", "Bob").
		Build()
	defer rd.Release()

	assert.Equal(t, "Alice", rd.FormValue("name"))
	assert.Equal(t, []string{"Alice", "Bob"}, rd.FormValues("name"))
}

func TestRequestDataBuilder_Chaining(t *testing.T) {
	t.Parallel()

	b := templater_dto.NewRequestDataBuilder()
	same := b.
		WithContext(context.Background()).
		WithMethod("GET").
		WithHost("localhost").
		WithLocale("en").
		WithDefaultLocale("en").
		WithCollectionData(nil).
		AddPathParam("k", "v").
		AddQueryParam("k", []string{"v"}).
		AddQueryParamValue("k2", "v2").
		AddFormData("k", []string{"v"}).
		AddFormDataValue("k2", "v2")

	assert.Same(t, b, same, "all builder methods should return the same pointer")
	b.Build().Release()
}

func TestRequestDataBuilder_Release_Nil(t *testing.T) {
	t.Parallel()

	var b *templater_dto.RequestDataBuilder
	assert.NotPanics(t, func() { b.Release() })
}

func TestRequestData_URL_Nil(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().Build()
	defer rd.Release()

	assert.Nil(t, rd.URL())
}

func TestRequestData_URL_DeepCopy(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("https://example.com/page")
	require.NoError(t, err)

	rd := templater_dto.NewRequestDataBuilder().
		WithURL(u).
		Build()
	defer rd.Release()

	copy1 := rd.URL()
	copy2 := rd.URL()

	require.NotNil(t, copy1)
	require.NotNil(t, copy2)

	copy1.Path = "/changed"
	assert.Equal(t, "/page", copy2.Path)
}

func TestRequestData_URL_WithUserInfo(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("https://user:pass@example.com/path")
	require.NoError(t, err)

	rd := templater_dto.NewRequestDataBuilder().
		WithURL(u).
		Build()
	defer rd.Release()

	got := rd.URL()
	require.NotNil(t, got)
	require.NotNil(t, got.User)
	assert.Equal(t, "user", got.User.Username())
}

func TestRequestData_PathParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "found", key: "id", want: "42"},
		{name: "not found", key: "missing", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rd := templater_dto.NewRequestDataBuilder().
				AddPathParam("id", "42").
				Build()
			defer rd.Release()
			assert.Equal(t, tt.want, rd.PathParam(tt.key))
		})
	}
}

func TestRequestData_QueryParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "found returns first", key: "sort", want: "asc"},
		{name: "not found", key: "missing", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rd := templater_dto.NewRequestDataBuilder().
				AddQueryParam("sort", []string{"asc", "desc"}).
				Build()
			defer rd.Release()
			assert.Equal(t, tt.want, rd.QueryParam(tt.key))
		})
	}
}

func TestRequestData_QueryParamValues(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddQueryParam("tag", []string{"a", "b", "c"}).
		Build()
	defer rd.Release()

	assert.Equal(t, []string{"a", "b", "c"}, rd.QueryParamValues("tag"))
	assert.Nil(t, rd.QueryParamValues("missing"))
}

func TestRequestData_FormValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "found returns first", key: "field", want: "first"},
		{name: "not found", key: "missing", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rd := templater_dto.NewRequestDataBuilder().
				AddFormData("field", []string{"first", "second"}).
				Build()
			defer rd.Release()
			assert.Equal(t, tt.want, rd.FormValue(tt.key))
		})
	}
}

func TestRequestData_FormValues(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddFormData("colour", []string{"red", "green"}).
		Build()
	defer rd.Release()

	assert.Equal(t, []string{"red", "green"}, rd.FormValues("colour"))
	assert.Nil(t, rd.FormValues("missing"))
}

func TestRequestData_PathParams(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddPathParam("a", "1").
		AddPathParam("b", "2").
		Build()
	defer rd.Release()

	params := rd.PathParams()
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, params)

	params["c"] = "3"
	assert.Empty(t, rd.PathParam("c"), "mutation of returned map must not affect original")
}

func TestRequestData_QueryParams(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddQueryParam("x", []string{"1"}).
		AddQueryParam("y", []string{"2", "3"}).
		Build()
	defer rd.Release()

	params := rd.QueryParams()
	assert.Equal(t, []string{"1"}, params["x"])
	assert.Equal(t, []string{"2", "3"}, params["y"])

	params["x"] = append(params["x"], "mutated")
	assert.Equal(t, []string{"1"}, rd.QueryParamValues("x"))
}

func TestRequestData_FormData(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		AddFormData("a", []string{"v1"}).
		Build()
	defer rd.Release()

	data := rd.FormData()
	assert.Equal(t, []string{"v1"}, data["a"])

	data["a"] = append(data["a"], "mutated")
	assert.Equal(t, []string{"v1"}, rd.FormValues("a"))
}

func TestRequestData_RangePathParams(t *testing.T) {
	t.Parallel()

	t.Run("full iteration", func(t *testing.T) {
		t.Parallel()
		rd := templater_dto.NewRequestDataBuilder().
			AddPathParam("a", "1").
			AddPathParam("b", "2").
			AddPathParam("c", "3").
			Build()
		defer rd.Release()
		var keys []string
		rd.RangePathParams(func(k, _ string) bool {
			keys = append(keys, k)
			return true
		})
		assert.Equal(t, []string{"a", "b", "c"}, keys)
	})

	t.Run("early stop", func(t *testing.T) {
		t.Parallel()
		rd := templater_dto.NewRequestDataBuilder().
			AddPathParam("a", "1").
			AddPathParam("b", "2").
			Build()
		defer rd.Release()
		count := 0
		rd.RangePathParams(func(_, _ string) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})
}

func TestRequestData_RangeQueryParams(t *testing.T) {
	t.Parallel()

	t.Run("full iteration", func(t *testing.T) {
		t.Parallel()
		rd := templater_dto.NewRequestDataBuilder().
			AddQueryParam("x", []string{"1"}).
			Build()
		defer rd.Release()
		visited := 0
		rd.RangeQueryParams(func(_ string, _ []string) bool {
			visited++
			return true
		})
		assert.Equal(t, 1, visited)
	})

	t.Run("early stop", func(t *testing.T) {
		t.Parallel()
		rd := templater_dto.NewRequestDataBuilder().
			AddQueryParam("x", []string{"1"}).
			Build()
		defer rd.Release()
		count := 0
		rd.RangeQueryParams(func(_ string, _ []string) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})
}

func TestRequestData_RangeFormData(t *testing.T) {
	t.Parallel()

	t.Run("full iteration", func(t *testing.T) {
		t.Parallel()
		rd := templater_dto.NewRequestDataBuilder().
			AddFormData("f", []string{"v"}).
			Build()
		defer rd.Release()
		visited := 0
		rd.RangeFormData(func(_ string, _ []string) bool {
			visited++
			return true
		})
		assert.Equal(t, 1, visited)
	})

	t.Run("early stop", func(t *testing.T) {
		t.Parallel()
		rd := templater_dto.NewRequestDataBuilder().
			AddFormData("f", []string{"v"}).
			Build()
		defer rd.Release()
		count := 0
		rd.RangeFormData(func(_ string, _ []string) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})
}

func TestRequestData_WithCollectionData(t *testing.T) {
	t.Parallel()

	original := templater_dto.NewRequestDataBuilder().
		WithMethod("GET").
		WithHost("example.com").
		WithLocale("en-GB").
		Build()
	defer original.Release()

	data := []string{"item1", "item2"}
	copied := original.WithCollectionData(data)

	assert.Equal(t, data, copied.CollectionData())
	assert.Equal(t, "GET", copied.Method())
	assert.Equal(t, "example.com", copied.Host())
	assert.Equal(t, "en-GB", copied.Locale())
	assert.Nil(t, original.CollectionData(), "original should be unchanged")
}

func TestRequestData_WithDefaultLocale(t *testing.T) {
	t.Parallel()

	original := templater_dto.NewRequestDataBuilder().
		WithMethod("POST").
		WithDefaultLocale("en-GB").
		Build()
	defer original.Release()

	copied := original.WithDefaultLocale("fr-FR")

	assert.Equal(t, "fr-FR", copied.DefaultLocale())
	assert.Equal(t, "POST", copied.Method())
	assert.Equal(t, "en-GB", original.DefaultLocale(), "original should be unchanged")
}

func TestRequestData_CSPTokenAttr_NilContext(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithContext(nil).
		Build()
	defer rd.Release()

	assert.Empty(t, string(rd.CSPTokenAttr()))
}

func TestRequestData_CSPTokenAttr_NoToken(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithContext(context.Background()).
		Build()
	defer rd.Release()

	assert.Empty(t, string(rd.CSPTokenAttr()))
}

func TestRequestData_Release_Nil(t *testing.T) {
	t.Parallel()

	var rd *templater_dto.RequestData
	assert.NotPanics(t, func() { rd.Release() })
}

func TestRequestData_SetGlobalStore(t *testing.T) {
	t.Parallel()

	translations := map[string]map[string]string{
		"en": {"greeting": "Hello"},
	}
	store := i18n_domain.NewStoreFromTranslations(translations, "en")

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	rd.SetGlobalStore(store)

	result := rd.T("greeting")
	assert.Equal(t, "Hello", result.String())
}

func TestRequestData_SetLocalStore(t *testing.T) {
	t.Parallel()

	translations := map[string]map[string]string{
		"en": {"label": "Save"},
	}
	store := i18n_domain.NewStoreFromTranslations(translations, "en")

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	rd.SetLocalStore(store)

	result := rd.LT("label")
	assert.Equal(t, "Save", result.String())
}

func TestRequestData_SetLocalStoreFromMap(t *testing.T) {
	t.Parallel()

	t.Run("non-empty map", func(t *testing.T) {
		t.Parallel()

		rd := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithDefaultLocale("en").
			Build()
		defer rd.Release()

		rd.SetLocalStoreFromMap(map[string]map[string]string{
			"en": {"btn": "Click"},
		})

		result := rd.LT("btn")
		assert.Equal(t, "Click", result.String())
	})

	t.Run("empty map is no-op", func(t *testing.T) {
		t.Parallel()

		rd := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithDefaultLocale("en").
			Build()
		defer rd.Release()

		rd.SetLocalStoreFromMap(map[string]map[string]string{})

		result := rd.LT("missing", "fallback")
		assert.Equal(t, "fallback", result.String())
	})
}

func TestRequestData_SetStrBufPool(t *testing.T) {
	t.Parallel()

	pool := i18n_domain.NewStrBufPool(64)
	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	rd.SetStrBufPool(pool)

	result := rd.T("missing", "fallback")
	assert.Equal(t, "fallback", result.String())
}

func TestRequestData_SetI18n(t *testing.T) {
	t.Parallel()

	globalTranslations := map[string]map[string]string{
		"en": {"global_key": "Global"},
	}
	localTranslations := map[string]map[string]string{
		"en": {"local_key": "Local"},
	}
	globalStore := i18n_domain.NewStoreFromTranslations(globalTranslations, "en")
	localStore := i18n_domain.NewStoreFromTranslations(localTranslations, "en")
	pool := i18n_domain.NewStrBufPool(64)

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	rd.SetI18n(globalStore, localStore, pool)

	assert.Equal(t, "Global", rd.T("global_key").String())
	assert.Equal(t, "Local", rd.LT("local_key").String())
}

func TestRequestData_T_EmptyKey(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	result := rd.T()
	assert.Empty(t, result.String())
}

func TestRequestData_T_WithFallback(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	result := rd.T("missing.key", "Fallback Value")
	assert.Equal(t, "Fallback Value", result.String())
}

func TestRequestData_T_KeyWithoutFallback(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	result := rd.T("some.key")
	assert.Equal(t, "some.key", result.String())
}

func TestRequestData_LT_EmptyKey(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	result := rd.LT()
	assert.Empty(t, result.String())
}

func TestRequestData_LT_WithFallback(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	result := rd.LT("missing.key", "Local Fallback")
	assert.Equal(t, "Local Fallback", result.String())
}

func TestRequestData_T_GlobalStoreHit(t *testing.T) {
	t.Parallel()

	translations := map[string]map[string]string{
		"en-GB": {"greeting": "Good day"},
		"en":    {"farewell": "Cheerio"},
	}
	store := i18n_domain.NewStoreFromTranslations(translations, "en")

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en-GB").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	rd.SetGlobalStore(store)

	assert.Equal(t, "Good day", rd.T("greeting").String())
	assert.Equal(t, "Cheerio", rd.T("farewell").String())
}

func TestRequestData_LT_LocalStoreHit(t *testing.T) {
	t.Parallel()

	translations := map[string]map[string]string{
		"en": {"button.save": "Save"},
	}
	store := i18n_domain.NewStoreFromTranslations(translations, "en")

	rd := templater_dto.NewRequestDataBuilder().
		WithLocale("en").
		WithDefaultLocale("en").
		Build()
	defer rd.Release()

	rd.SetLocalStore(store)

	assert.Equal(t, "Save", rd.LT("button.save").String())
}

func TestRequestData_Cookie(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		lookupName string
		wantValue  string
		cookies    []*http.Cookie
		wantErr    bool
	}{
		{
			name: "found by name",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "pref", Value: "dark"},
			},
			lookupName: "pref",
			wantValue:  "dark",
			wantErr:    false,
		},
		{
			name: "not found",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			lookupName: "missing",
			wantErr:    true,
		},
		{
			name:       "empty cookies",
			cookies:    nil,
			lookupName: "any",
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rd := templater_dto.NewRequestDataBuilder().
				WithCookies(tc.cookies).
				Build()
			defer rd.Release()

			cookie, err := rd.Cookie(tc.lookupName)
			if tc.wantErr {
				assert.ErrorIs(t, err, http.ErrNoCookie)
				assert.Nil(t, cookie)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantValue, cookie.Value)
			}
		})
	}
}

func TestRequestData_Cookies(t *testing.T) {
	t.Parallel()

	cookies := []*http.Cookie{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
	}

	rd := templater_dto.NewRequestDataBuilder().
		WithCookies(cookies).
		Build()
	defer rd.Release()

	result := rd.Cookies()
	require.Len(t, result, 2)
	assert.Equal(t, "a", result[0].Name)
	assert.Equal(t, "b", result[1].Name)

	result[0] = &http.Cookie{Name: "mutated"}
	result2 := rd.Cookies()
	assert.Equal(t, "a", result2[0].Name)
}

func TestRequestDataBuilder_WithCookies(t *testing.T) {
	t.Parallel()

	cookies := []*http.Cookie{
		{Name: "session", Value: "xyz"},
	}

	b := templater_dto.NewRequestDataBuilder()
	returned := b.WithCookies(cookies)

	assert.Same(t, b, returned)

	rd := b.Build()
	defer rd.Release()

	cookie, err := rd.Cookie("session")
	require.NoError(t, err)
	assert.Equal(t, "xyz", cookie.Value)
}

func TestRequestData_SetCookie(t *testing.T) {
	t.Parallel()

	acc := templater_dto.NewCookieAccumulator()
	ctx := templater_dto.WithCookieAccumulator(context.Background(), acc)

	rd := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		Build()
	defer rd.Release()

	rd.SetCookie(&http.Cookie{Name: "pref", Value: "light"})

	cookies := acc.GetCookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "pref", cookies[0].Name)
	assert.Equal(t, "light", cookies[0].Value)
}

func TestRequestData_SetCookie_NoAccumulator(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithContext(context.Background()).
		Build()
	defer rd.Release()

	rd.SetCookie(&http.Cookie{Name: "pref", Value: "dark"})
}

func TestRequestData_WithCollectionData_PreservesCookies(t *testing.T) {
	t.Parallel()

	cookies := []*http.Cookie{
		{Name: "session", Value: "abc"},
	}

	rd := templater_dto.NewRequestDataBuilder().
		WithCookies(cookies).
		Build()
	defer rd.Release()

	rd2 := rd.WithCollectionData("some-data")

	cookie, err := rd2.Cookie("session")
	require.NoError(t, err)
	assert.Equal(t, "abc", cookie.Value)
}

func TestRequestData_WithDefaultLocale_PreservesCookies(t *testing.T) {
	t.Parallel()

	cookies := []*http.Cookie{
		{Name: "theme", Value: "dark"},
	}

	rd := templater_dto.NewRequestDataBuilder().
		WithCookies(cookies).
		Build()
	defer rd.Release()

	rd2 := rd.WithDefaultLocale("fr")

	cookie, err := rd2.Cookie("theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", cookie.Value)
}
