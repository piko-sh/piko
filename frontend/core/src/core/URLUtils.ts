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

/**
 * Resolves a possibly-relative URL the way the browser resolves a hyperlink on the
 * current page.
 *
 * @param urlValue - The URL to resolve.
 * @returns The resolved absolute URL.
 */
function resolveAgainstDocument(urlValue: string): URL {
    return new URL(urlValue, globalThis.location?.href ?? '');
}

/**
 * Appends a query parameter to a URL string without parsing it.
 *
 * @param urlValue - The URL to append to.
 * @param name - The parameter name.
 * @param value - The parameter value.
 * @returns The URL with the parameter appended.
 */
function appendRawQuery(urlValue: string, name: string, value: string): string {
    const separator = urlValue.includes('?') ? '&' : '?';
    return `${urlValue}${separator}${encodeURIComponent(name)}=${encodeURIComponent(value)}`;
}

/**
 * Adds the fragment query parameter (_f=1) to a URL.
 *
 * This parameter signals to the server that a fragment response is expected.
 *
 * @param urlValue - The URL to modify.
 * @returns The URL with the fragment query parameter added.
 */
export function addFragmentQuery(urlValue: string): string {
    try {
        const parsedUrl = resolveAgainstDocument(urlValue);
        parsedUrl.searchParams.set('_f', '1');
        return parsedUrl.toString();
    } catch {
        return appendRawQuery(urlValue, '_f', '1');
    }
}

/**
 * Builds a URL with query parameters for remote rendering.
 *
 * Automatically adds the fragment query parameter and any provided arguments.
 *
 * @param base - The base URL path.
 * @param args - Query parameters to add to the URL.
 * @returns The complete URL with all query parameters.
 */
export function buildRemoteUrl(base: string, args: Record<string, string | number>): string {
    let urlObj: URL;
    try {
        urlObj = resolveAgainstDocument(base);
    } catch {
        let raw = appendRawQuery(base, '_f', '1');
        for (const [paramName, paramValue] of Object.entries(args)) {
            raw = appendRawQuery(raw, paramName, String(paramValue));
        }
        return raw;
    }

    urlObj.searchParams.set('_f', '1');
    for (const [paramName, paramValue] of Object.entries(args)) {
        urlObj.searchParams.set(paramName, String(paramValue));
    }
    return urlObj.toString();
}

/**
 * Checks if a location or anchor element is on the same domain.
 *
 * Used to determine if SPA navigation should be used or a full page load.
 *
 * @param loc - The Location or HTMLAnchorElement to check.
 * @returns True if on the same domain, false otherwise.
 */
export function isSameDomain(loc: Location | HTMLAnchorElement): boolean {
    return loc.hostname === window.location.hostname;
}

/**
 * Checks whether an absolute URL is on the same origin as the current page.
 *
 * @param urlValue - The absolute URL to check.
 * @returns True if the URL is on the page's own origin, false otherwise.
 */
export function isSameOriginUrl(urlValue: string): boolean {
    const pageOrigin = globalThis.location?.origin;
    if (!pageOrigin) {
        return false;
    }

    try {
        return new URL(urlValue).origin === pageOrigin;
    } catch {
        return false;
    }
}
