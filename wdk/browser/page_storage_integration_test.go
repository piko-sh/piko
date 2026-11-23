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

//go:build integration

package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

func TestPageStorageAndCookies(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")

	page.SetLocalStorageItem("theme", "dark").SetLocalStorageItem("lang", "en")
	assert.Equal(t, "dark", page.GetLocalStorageItem("theme"))
	assert.True(t, page.HasLocalStorageItem("lang"))
	assert.Equal(t, map[string]string{"theme": "dark", "lang": "en"}, page.GetAllLocalStorage())
	page.RemoveLocalStorageItem("lang")
	assert.False(t, page.HasLocalStorageItem("lang"))
	page.ClearLocalStorage()
	assert.Empty(t, page.GetAllLocalStorage())

	page.SetSessionStorageItem("step", "2")
	assert.Equal(t, "2", page.GetSessionStorageItem("step"))
	assert.True(t, page.HasSessionStorageItem("step"))
	assert.Equal(t, map[string]string{"step": "2"}, page.GetAllSessionStorage())
	page.RemoveSessionStorageItem("step")
	assert.False(t, page.HasSessionStorageItem("step"))
	page.SetSessionStorageItem("other", "x").ClearSessionStorage()
	assert.Empty(t, page.GetAllSessionStorage())

	page.SetCookie("flavour", "oat", nil)
	page.SetCookie("scoped", "yes", &browser_provider_chromedp.CookieOptions{Path: "/", SameSite: "Lax"})
	cookie := page.GetCookie("flavour")
	require.NotNil(t, cookie)
	assert.Equal(t, "oat", cookie.Value)
	assert.Nil(t, page.GetCookie("absent"))
	assert.Equal(t, "yes", page.GetCookieValue("scoped"))
	assert.True(t, page.HasCookie("flavour"))
	assert.Len(t, page.GetAllCookies(), 2)
	page.DeleteCookie("flavour")
	assert.False(t, page.HasCookie("flavour"))
	page.ClearCookies()
	assert.Empty(t, page.GetAllCookies())
	require.Empty(t, recorder.take())
}

func TestPageStorageReportsFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "GetLocalStorageItem", run: func() { page.GetLocalStorageItem("k") }},
		{name: "HasLocalStorageItem", run: func() { page.HasLocalStorageItem("k") }},
		{name: "SetLocalStorageItem", run: func() { page.SetLocalStorageItem("k", "v") }},
		{name: "RemoveLocalStorageItem", run: func() { page.RemoveLocalStorageItem("k") }},
		{name: "ClearLocalStorage", run: func() { page.ClearLocalStorage() }},
		{name: "GetAllLocalStorage", run: func() { page.GetAllLocalStorage() }},
		{name: "GetSessionStorageItem", run: func() { page.GetSessionStorageItem("k") }},
		{name: "HasSessionStorageItem", run: func() { page.HasSessionStorageItem("k") }},
		{name: "SetSessionStorageItem", run: func() { page.SetSessionStorageItem("k", "v") }},
		{name: "RemoveSessionStorageItem", run: func() { page.RemoveSessionStorageItem("k") }},
		{name: "ClearSessionStorage", run: func() { page.ClearSessionStorage() }},
		{name: "GetAllSessionStorage", run: func() { page.GetAllSessionStorage() }},
		{name: "GetCookieValue", run: func() { page.GetCookieValue("k") }},
		{name: "GetCookie", run: func() { page.GetCookie("k") }},
		{name: "HasCookie", run: func() { page.HasCookie("k") }},
		{name: "GetAllCookies", run: func() { page.GetAllCookies() }},
		{name: "SetCookie", run: func() { page.SetCookie("k", "v", nil) }},
		{name: "DeleteCookie", run: func() { page.DeleteCookie("k") }},
		{name: "ClearCookies", run: func() { page.ClearCookies() }},
	})
}
