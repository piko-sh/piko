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
	"testing"
	"time"
)

const testHTMLCookies = `<!DOCTYPE html>
<html>
<head><title>Cookie Test</title></head>
<body>
<div id="content">Cookie Test Page</div>
</body>
</html>`

func TestCookies(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLCookies)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("set and get cookie", func(t *testing.T) {
			if err := SetCookie(ctx, "testCookie", "testValue", nil); err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			cookie, found, err := GetCookie(ctx, "testCookie")
			if err != nil {
				t.Fatalf("GetCookie() error = %v", err)
			}
			if !found {
				t.Fatal("GetCookie() returned found=false")
			}
			if cookie.Value != "testValue" {
				t.Errorf("cookie.Value = %q, want %q", cookie.Value, "testValue")
			}
		})

		t.Run("get cookie value helper", func(t *testing.T) {
			err := SetCookie(ctx, "simpleGet", "simpleValue", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			value, err := GetCookieValue(ctx, "simpleGet")
			if err != nil {
				t.Fatalf("GetCookieValue() error = %v", err)
			}
			if value != "simpleValue" {
				t.Errorf("GetCookieValue() = %q, want %q", value, "simpleValue")
			}
		})

		t.Run("has cookie", func(t *testing.T) {
			err := SetCookie(ctx, "exists", "yes", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			has, err := HasCookie(ctx, "exists")
			if err != nil {
				t.Fatalf("HasCookie() error = %v", err)
			}
			if !has {
				t.Error("HasCookie() = false, want true")
			}

			has, err = HasCookie(ctx, "nonExistent")
			if err != nil {
				t.Fatalf("HasCookie() error = %v", err)
			}
			if has {
				t.Error("HasCookie(nonExistent) = true, want false")
			}
		})

		t.Run("get all cookies", func(t *testing.T) {

			err := ClearCookies(ctx)
			if err != nil {
				t.Fatalf("ClearCookies() error = %v", err)
			}

			err = SetCookie(ctx, "cookie1", "value1", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}
			err = SetCookie(ctx, "cookie2", "value2", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			cookies, err := GetAllCookies(ctx)
			if err != nil {
				t.Fatalf("GetAllCookies() error = %v", err)
			}

			if len(cookies) < 2 {
				t.Errorf("GetAllCookies() returned %d cookies, want at least 2", len(cookies))
			}
		})

		t.Run("delete cookie", func(t *testing.T) {
			err := SetCookie(ctx, "toDelete", "value", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			err = DeleteCookie(ctx, "toDelete")
			if err != nil {
				t.Fatalf("DeleteCookie() error = %v", err)
			}

			has, err := HasCookie(ctx, "toDelete")
			if err != nil {
				t.Fatalf("HasCookie() error = %v", err)
			}
			if has {
				t.Error("cookie still exists after deletion")
			}
		})

		t.Run("set cookie with options", func(t *testing.T) {
			opts := &CookieOptions{
				Path:     "/",
				Expires:  time.Now().Add(time.Hour),
				HTTPOnly: true,
				Secure:   false,
			}

			if err := SetCookie(ctx, "withOpts", "optValue", opts); err != nil {
				t.Fatalf("SetCookie() with options error = %v", err)
			}

			cookie, found, err := GetCookie(ctx, "withOpts")
			if err != nil {
				t.Fatalf("GetCookie() error = %v", err)
			}
			if !found {
				t.Fatal("GetCookie() returned found=false")
			}
			if cookie.Value != "optValue" {
				t.Errorf("cookie.Value = %q, want %q", cookie.Value, "optValue")
			}
			if !cookie.HTTPOnly {
				t.Error("cookie.HTTPOnly = false, want true")
			}
		})

		t.Run("clear all cookies", func(t *testing.T) {
			err := SetCookie(ctx, "toClear1", "v1", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}
			err = SetCookie(ctx, "toClear2", "v2", nil)
			if err != nil {
				t.Fatalf("SetCookie() error = %v", err)
			}

			err = ClearCookies(ctx)
			if err != nil {
				t.Fatalf("ClearCookies() error = %v", err)
			}

			cookies, err := GetAllCookies(ctx)
			if err != nil {
				t.Fatalf("GetAllCookies() error = %v", err)
			}
			if len(cookies) != 0 {
				t.Errorf("GetAllCookies() returned %d cookies after clear, want 0", len(cookies))
			}
		})
	})
}
