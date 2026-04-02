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

/** Meta tag name for the signed action token. */
const CSRF_TOKEN_META_NAME = 'csrf-token';

/** Meta tag name for the ephemeral token. */
const CSRF_EPHEMERAL_META_NAME = 'csrf-ephemeral';

/**
 * Reads the CSRF action token from the page meta tag.
 *
 * This is the signed token that includes the cookie binding,
 * ephemeral token, timestamp, and HMAC signature.
 *
 * @returns The CSRF action token, or null if the meta tag is not present.
 */
export function getCSRFTokenFromMeta(): string | null {
    return document.querySelector<HTMLMetaElement>(`meta[name="${CSRF_TOKEN_META_NAME}"]`)?.content ?? null;
}

/**
 * Reads the CSRF ephemeral token from the page meta tag.
 *
 * This is the raw ephemeral token that needs to be sent in the
 * request body or query parameters.
 *
 * @returns The CSRF ephemeral token, or null if the meta tag is not present.
 */
export function getCSRFEphemeralFromMeta(): string | null {
    return document.querySelector<HTMLMetaElement>(`meta[name="${CSRF_EPHEMERAL_META_NAME}"]`)?.content ?? null;
}

/**
 * Reads both CSRF tokens from the page meta tags.
 *
 * @returns An object with actionToken and ephemeralToken, both of which may be null.
 */
export function getCSRFTokensFromMeta(): {
    actionToken: string | null;
    ephemeralToken: string | null;
} {
    return {
        actionToken: getCSRFTokenFromMeta(),
        ephemeralToken: getCSRFEphemeralFromMeta()
    };
}
