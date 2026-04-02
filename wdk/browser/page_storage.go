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

package browser

import (
	"fmt"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// GetLocalStorageItem retrieves a value from localStorage by key.
//
// Takes key (string) which specifies the localStorage key to retrieve.
//
// Returns string which is the value if found, or an empty string if not found.
func (p *Page) GetLocalStorageItem(key string) string {
	value, _, err := browser_provider_chromedp.GetLocalStorageItem(p.actionCtx(), key)
	if err != nil {
		p.t.Fatalf("GetLocalStorageItem(%q) failed: %v", key, err)
	}
	return value
}

// HasLocalStorageItem checks if a key exists in localStorage.
//
// Takes key (string) which specifies the localStorage key to check.
//
// Returns bool which is true if the key exists.
func (p *Page) HasLocalStorageItem(key string) bool {
	_, exists, err := browser_provider_chromedp.GetLocalStorageItem(p.actionCtx(), key)
	if err != nil {
		p.t.Fatalf("HasLocalStorageItem(%q) failed: %v", key, err)
	}
	return exists
}

// SetLocalStorageItem sets a value in localStorage.
//
// Takes key (string) which specifies the storage key name.
// Takes value (string) which specifies the value to store.
//
// Returns *Page which allows method chaining.
func (p *Page) SetLocalStorageItem(key, value string) *Page {
	detail := fmt.Sprintf(fmtKeyValue, key, value)
	p.beforeAction("SetLocalStorageItem", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetLocalStorageItem(p.actionCtx(), key, value)
	p.afterAction("SetLocalStorageItem", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetLocalStorageItem(%q, %q) failed: %v", key, value, err)
	}
	return p
}

// RemoveLocalStorageItem removes a value from local storage.
//
// Takes key (string) which specifies the local storage key to remove.
//
// Returns *Page which allows method chaining.
func (p *Page) RemoveLocalStorageItem(key string) *Page {
	p.beforeAction("RemoveLocalStorageItem", key)
	start := time.Now()
	err := browser_provider_chromedp.RemoveLocalStorageItem(p.actionCtx(), key)
	p.afterAction("RemoveLocalStorageItem", key, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("RemoveLocalStorageItem(%q) failed: %v", key, err)
	}
	return p
}

// ClearLocalStorage clears all items from localStorage.
//
// Returns *Page which enables method chaining.
func (p *Page) ClearLocalStorage() *Page {
	p.beforeAction("ClearLocalStorage", "")
	start := time.Now()
	err := browser_provider_chromedp.ClearLocalStorage(p.actionCtx())
	p.afterAction("ClearLocalStorage", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ClearLocalStorage() failed: %v", err)
	}
	return p
}

// GetAllLocalStorage returns all key-value pairs from localStorage.
//
// Returns map[string]string which contains all localStorage entries.
func (p *Page) GetAllLocalStorage() map[string]string {
	items, err := browser_provider_chromedp.GetAllLocalStorage(p.actionCtx())
	if err != nil {
		p.t.Fatalf("GetAllLocalStorage() failed: %v", err)
	}
	return items
}

// GetSessionStorageItem retrieves a value from session storage by key.
//
// Takes key (string) which identifies the item to retrieve.
//
// Returns string which is the value if found, or an empty string if not.
func (p *Page) GetSessionStorageItem(key string) string {
	value, _, err := browser_provider_chromedp.GetSessionStorageItem(p.actionCtx(), key)
	if err != nil {
		p.t.Fatalf("GetSessionStorageItem(%q) failed: %v", key, err)
	}
	return value
}

// HasSessionStorageItem checks if a key exists in sessionStorage.
//
// Takes key (string) which specifies the storage key to check.
//
// Returns bool which indicates whether the key exists in session storage.
func (p *Page) HasSessionStorageItem(key string) bool {
	_, exists, err := browser_provider_chromedp.GetSessionStorageItem(p.actionCtx(), key)
	if err != nil {
		p.t.Fatalf("HasSessionStorageItem(%q) failed: %v", key, err)
	}
	return exists
}

// SetSessionStorageItem sets a value in sessionStorage.
//
// Takes key (string) which specifies the storage key.
// Takes value (string) which specifies the value to store.
//
// Returns *Page which allows method chaining.
func (p *Page) SetSessionStorageItem(key, value string) *Page {
	detail := fmt.Sprintf(fmtKeyValue, key, value)
	p.beforeAction("SetSessionStorageItem", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetSessionStorageItem(p.actionCtx(), key, value)
	p.afterAction("SetSessionStorageItem", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetSessionStorageItem(%q, %q) failed: %v", key, value, err)
	}
	return p
}

// RemoveSessionStorageItem removes a value from session storage.
//
// Takes key (string) which specifies the storage key to remove.
//
// Returns *Page which allows method chaining.
func (p *Page) RemoveSessionStorageItem(key string) *Page {
	p.beforeAction("RemoveSessionStorageItem", key)
	start := time.Now()
	err := browser_provider_chromedp.RemoveSessionStorageItem(p.actionCtx(), key)
	p.afterAction("RemoveSessionStorageItem", key, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("RemoveSessionStorageItem(%q) failed: %v", key, err)
	}
	return p
}

// ClearSessionStorage clears all items from sessionStorage.
//
// Returns *Page which allows method chaining.
func (p *Page) ClearSessionStorage() *Page {
	p.beforeAction("ClearSessionStorage", "")
	start := time.Now()
	err := browser_provider_chromedp.ClearSessionStorage(p.actionCtx())
	p.afterAction("ClearSessionStorage", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ClearSessionStorage() failed: %v", err)
	}
	return p
}

// GetAllSessionStorage returns all key-value pairs stored in sessionStorage.
//
// Returns map[string]string which contains all sessionStorage entries.
func (p *Page) GetAllSessionStorage() map[string]string {
	items, err := browser_provider_chromedp.GetAllSessionStorage(p.actionCtx())
	if err != nil {
		p.t.Fatalf("GetAllSessionStorage() failed: %v", err)
	}
	return items
}

// GetCookie retrieves a cookie by name.
//
// Takes name (string) which specifies the cookie name to retrieve.
//
// Returns *browser_provider_chromedp.Cookie which is the cookie if found, or
// nil if the cookie
// does not exist.
func (p *Page) GetCookie(name string) *browser_provider_chromedp.Cookie {
	cookie, found, err := browser_provider_chromedp.GetCookie(p.actionCtx(), name)
	if err != nil {
		p.t.Fatalf("GetCookie(%q) failed: %v", name, err)
	}
	if !found {
		return nil
	}
	return cookie
}

// GetCookieValue retrieves the value of a cookie by name.
//
// Takes name (string) which specifies the cookie name to look up.
//
// Returns string which is the cookie value.
func (p *Page) GetCookieValue(name string) string {
	value, err := browser_provider_chromedp.GetCookieValue(p.actionCtx(), name)
	if err != nil {
		p.t.Fatalf("GetCookieValue(%q) failed: %v", name, err)
	}
	return value
}

// HasCookie checks whether a cookie with the given name exists.
//
// Takes name (string) which specifies the cookie name to look for.
//
// Returns bool which is true if the cookie exists, false otherwise.
func (p *Page) HasCookie(name string) bool {
	exists, err := browser_provider_chromedp.HasCookie(p.actionCtx(), name)
	if err != nil {
		p.t.Fatalf("HasCookie(%q) failed: %v", name, err)
	}
	return exists
}

// GetAllCookies returns all cookies for the current page.
//
// Returns []*browser_provider_chromedp.Cookie which contains all cookies set in
// the browser.
func (p *Page) GetAllCookies() []*browser_provider_chromedp.Cookie {
	cookies, err := browser_provider_chromedp.GetAllCookies(p.actionCtx())
	if err != nil {
		p.t.Fatalf("GetAllCookies() failed: %v", err)
	}
	return cookies
}

// SetCookie sets a cookie with the given name and value.
//
// Takes name (string) which specifies the cookie name.
// Takes value (string) which specifies the cookie value.
// Takes opts (*browser_provider_chromedp.CookieOptions) which provides optional
// cookie settings.
//
// Returns *Page which allows method chaining.
func (p *Page) SetCookie(name, value string, opts *browser_provider_chromedp.CookieOptions) *Page {
	detail := fmt.Sprintf(fmtKeyValue, name, value)
	p.beforeAction("SetCookie", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetCookie(p.actionCtx(), name, value, opts)
	p.afterAction("SetCookie", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetCookie(%q, %q) failed: %v", name, value, err)
	}
	return p
}

// DeleteCookie removes a browser cookie with the given name.
//
// Takes name (string) which specifies the cookie to remove.
//
// Returns *Page which allows method chaining.
func (p *Page) DeleteCookie(name string) *Page {
	p.beforeAction("DeleteCookie", name)
	start := time.Now()
	err := browser_provider_chromedp.DeleteCookie(p.actionCtx(), name)
	p.afterAction("DeleteCookie", name, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DeleteCookie(%q) failed: %v", name, err)
	}
	return p
}

// ClearCookies deletes all cookies for the current page.
//
// Returns *Page which allows method chaining.
func (p *Page) ClearCookies() *Page {
	p.beforeAction("ClearCookies", "")
	start := time.Now()
	err := browser_provider_chromedp.ClearCookies(p.actionCtx())
	p.afterAction("ClearCookies", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ClearCookies() failed: %v", err)
	}
	return p
}
