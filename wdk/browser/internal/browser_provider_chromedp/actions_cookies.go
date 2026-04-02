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
	"fmt"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Cookie represents a browser cookie with its standard attributes.
type Cookie struct {
	// Expires is when the cookie expires; zero value means session cookie.
	Expires time.Time

	// Name is the cookie name used for identification.
	Name string

	// Value is the cookie content; empty string indicates no value was set.
	Value string

	// Domain specifies the cookie's domain scope.
	Domain string

	// Path specifies the URL path scope for the cookie.
	Path string

	// SameSite specifies the cookie's SameSite attribute; valid values are
	// "Strict", "Lax", or "None".
	SameSite string

	// HTTPOnly indicates whether the cookie is inaccessible to JavaScript.
	HTTPOnly bool

	// Secure indicates whether the cookie should only be sent over HTTPS.
	Secure bool
}

// CookieOptions contains optional parameters for setting a cookie.
type CookieOptions struct {
	// Expires specifies when the cookie should expire; zero value means no expiry.
	Expires time.Time

	// Domain specifies the cookie domain scope; empty means current domain only.
	Domain string

	// Path is the cookie path; empty defaults to "/".
	Path string

	// SameSite controls cross-site request cookie behaviour; valid values are
	// "Strict", "Lax", or "None"; empty string uses browser default.
	SameSite string

	// HTTPOnly indicates whether the cookie is inaccessible to JavaScript.
	HTTPOnly bool

	// Secure indicates whether the cookie should only be sent over HTTPS.
	Secure bool
}

// GetCookie retrieves a cookie by name from the browser.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes name (string) which specifies the cookie name to find.
//
// Returns *Cookie which contains the cookie data if found, or nil if not.
// Returns bool which is true if the cookie was found, false otherwise.
// Returns error when the browser fails to retrieve cookies.
func GetCookie(ctx *ActionContext, name string) (*Cookie, bool, error) {
	var cookies []*network.Cookie
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx2)
		return err
	}))
	if err != nil {
		return nil, false, fmt.Errorf("getting cookies: %w", err)
	}

	for _, c := range cookies {
		if c.Name == name {
			return convertCookie(c), true, nil
		}
	}
	return nil, false, nil
}

// GetAllCookies returns all cookies for the current page.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns []*Cookie which contains all cookies from the current page.
// Returns error when the cookies cannot be retrieved.
func GetAllCookies(ctx *ActionContext) ([]*Cookie, error) {
	var cookies []*network.Cookie
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx2)
		return err
	}))
	if err != nil {
		return nil, fmt.Errorf("getting cookies: %w", err)
	}

	result := make([]*Cookie, len(cookies))
	for i, c := range cookies {
		result[i] = convertCookie(c)
	}
	return result, nil
}

// SetCookie sets a cookie with the given name and value.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes name (string) which specifies the cookie name.
// Takes value (string) which specifies the cookie value.
// Takes opts (*CookieOptions) which specifies additional cookie properties
// such as domain, path, expiry, and security settings. May be nil for
// defaults.
//
// Returns error when the current URL cannot be retrieved or when the cookie
// cannot be set.
func SetCookie(ctx *ActionContext, name, value string, opts *CookieOptions) error {
	var currentURL string
	err := chromedp.Run(ctx.Ctx, chromedp.Location(&currentURL))
	if err != nil {
		return fmt.Errorf("getting current URL for cookie: %w", err)
	}

	params := network.SetCookie(name, value)

	if opts != nil {
		if opts.Domain != "" {
			params = params.WithDomain(opts.Domain)
		}
		if opts.Path != "" {
			params = params.WithPath(opts.Path)
		} else {
			params = params.WithPath("/")
		}
		if !opts.Expires.IsZero() {
			params = params.WithExpires(new(cdp.TimeSinceEpoch(opts.Expires)))
		}
		params = params.WithHTTPOnly(opts.HTTPOnly)
		params = params.WithSecure(opts.Secure)
		if opts.SameSite != "" {
			sameSite := parseSameSite(opts.SameSite)
			params = params.WithSameSite(sameSite)
		}
	} else {
		params = params.WithPath("/")
	}

	params = params.WithURL(currentURL)

	err = chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return params.Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting cookie %q: %w", name, err)
	}
	return nil
}

// DeleteCookie deletes a cookie by name.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes name (string) which specifies the cookie name to delete.
//
// Returns error when the current URL cannot be retrieved or the cookie cannot
// be deleted.
func DeleteCookie(ctx *ActionContext, name string) error {
	var currentURL string
	err := chromedp.Run(ctx.Ctx, chromedp.Location(&currentURL))
	if err != nil {
		return fmt.Errorf("getting current URL for cookie deletion: %w", err)
	}

	err = chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return network.DeleteCookies(name).WithURL(currentURL).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("deleting cookie %q: %w", name, err)
	}
	return nil
}

// ClearCookies deletes all cookies for the current page.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
//
// Returns error when the cookies cannot be cleared.
func ClearCookies(ctx *ActionContext) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return network.ClearBrowserCookies().Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("clearing cookies: %w", err)
	}
	return nil
}

// GetCookieValue returns just the cookie value as a convenience function.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes name (string) which specifies the cookie name to retrieve.
//
// Returns string which is the cookie value, or empty string if not found.
// Returns error when the underlying cookie retrieval fails.
func GetCookieValue(ctx *ActionContext, name string) (string, error) {
	cookie, found, err := GetCookie(ctx, name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}
	return cookie.Value, nil
}

// HasCookie checks if a cookie with the given name exists.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes name (string) which specifies the cookie name to check.
//
// Returns bool which indicates whether the cookie exists.
// Returns error when the cookie lookup fails.
func HasCookie(ctx *ActionContext, name string) (bool, error) {
	_, found, err := GetCookie(ctx, name)
	return found, err
}

// convertCookie converts a CDP cookie to our Cookie type.
//
// Takes c (*network.Cookie) which is the CDP cookie to convert.
//
// Returns *Cookie which is the converted cookie with mapped fields.
func convertCookie(c *network.Cookie) *Cookie {
	var expires time.Time
	if c.Expires > 0 {
		expires = time.Unix(int64(c.Expires), 0)
	}

	sameSite := ""
	switch c.SameSite {
	case network.CookieSameSiteStrict:
		sameSite = "Strict"
	case network.CookieSameSiteLax:
		sameSite = "Lax"
	case network.CookieSameSiteNone:
		sameSite = "None"
	}

	return &Cookie{
		Expires:  expires,
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		SameSite: sameSite,
		HTTPOnly: c.HTTPOnly,
		Secure:   c.Secure,
	}
}

// parseSameSite converts a string to the CDP SameSite enum.
// Defaults to Lax if the value is not recognised.
//
// Takes s (string) which specifies the SameSite value to parse.
//
// Returns network.CookieSameSite which is the corresponding enum value.
func parseSameSite(s string) network.CookieSameSite {
	switch s {
	case "Strict":
		return network.CookieSameSiteStrict
	case "None":
		return network.CookieSameSiteNone
	default:
		return network.CookieSameSiteLax
	}
}
