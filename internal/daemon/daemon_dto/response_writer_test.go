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

package daemon_dto

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResponseWriter(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()

	require.NotNil(t, w)
	assert.Empty(t, w.GetCookies())
	assert.Empty(t, w.GetHeaders())
	assert.Empty(t, w.GetHelpers())
}

func TestResponseWriter_SetCookie(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()

	w.SetCookie(&http.Cookie{Name: "session", Value: "abc"})
	w.SetCookie(&http.Cookie{Name: "theme", Value: "dark"})

	cookies := w.GetCookies()
	assert.Len(t, cookies, 2)
	assert.Equal(t, "session", cookies[0].Name)
	assert.Equal(t, "dark", cookies[1].Value)
}

func TestResponseWriter_GetCookies_ReturnsCopy(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()
	w.SetCookie(&http.Cookie{Name: "a", Value: "1"})

	cookies1 := w.GetCookies()
	cookies2 := w.GetCookies()

	cookies1[0] = &http.Cookie{Name: "modified"}
	assert.Equal(t, "a", cookies2[0].Name)
}

func TestResponseWriter_AddHeader(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()

	w.AddHeader("X-Custom", "value1")
	w.AddHeader("X-Custom", "value2")

	headers := w.GetHeaders()
	assert.Equal(t, []string{"value1", "value2"}, headers["X-Custom"])
}

func TestResponseWriter_SetHeader(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()

	w.AddHeader("X-Custom", "old")
	w.SetHeader("X-Custom", "new")

	headers := w.GetHeaders()
	assert.Equal(t, []string{"new"}, headers["X-Custom"])
}

func TestResponseWriter_GetHeaders_ReturnsCopy(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()
	w.AddHeader("X-Test", "value")

	headers1 := w.GetHeaders()
	headers2 := w.GetHeaders()

	headers1.Set("X-Test", "modified")
	assert.Equal(t, "value", headers2.Get("X-Test"))
}

func TestResponseWriter_AddHelper(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()

	w.AddHelper("showToast", "Item deleted", "success")
	w.AddHelper("redirect", "/dashboard")

	helpers := w.GetHelpers()
	require.Len(t, helpers, 2)
	assert.Equal(t, "showToast", helpers[0].Name)
	assert.Equal(t, []any{"Item deleted", "success"}, helpers[0].Args)
	assert.Equal(t, "redirect", helpers[1].Name)
}

func TestResponseWriter_GetHelpers_ReturnsCopy(t *testing.T) {
	t.Parallel()

	w := NewResponseWriter()
	w.AddHelper("test", "argument")

	helpers1 := w.GetHelpers()
	helpers2 := w.GetHelpers()

	helpers1[0].Name = "modified"
	assert.Equal(t, "test", helpers2[0].Name)
}
