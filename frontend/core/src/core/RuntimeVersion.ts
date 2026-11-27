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
 * Response header in which the server names the runtime version it hands out.
 */
export const RUNTIME_VERSION_HEADER = 'X-PP-Runtime';

/** Shape of a runtime version, which is sixteen lowercase hex characters. */
const RUNTIME_VERSION_PATTERN = /^[0-9a-f]{16}$/;

/**
 * Session storage key that records when this tab last reloaded for a runtime change. The
 * runtime recovery script uses the same key, so the two share one limit.
 */
export const RELOAD_STORAGE_KEY = 'piko:runtime-reload-at';

/** Shortest time between two reloads forced by a runtime change, in milliseconds. */
export const RELOAD_INTERVAL_MS = 60_000;

/** Number of path segments from the end of a bundle URL to its version segment. */
const VERSION_SEGMENT_FROM_END = 2;

/**
 * Minimal session storage used to rate-limit forced reloads.
 */
export type ReloadStore = Pick<Storage, 'getItem' | 'setItem'>;

/**
 * Tracks whether the server still runs the same runtime build as this tab.
 *
 * A tab keeps the runtime it first loaded. Pages, partials and actions fetched later come
 * from whichever build the server runs now, and their modules may need exports this
 * runtime does not have. The gate notices the change so callers can load a full page
 * instead of mixing two builds.
 */
export interface RuntimeVersionGate {
    /** Version of the runtime running in this tab, or null when it is not known. */
    readonly ownVersion: string | null;

    /**
     * Records the runtime version named by a response.
     *
     * @param headers - The response headers, or nothing when the response has none.
     * @returns True when the response came from a different runtime build than this tab's.
     */
    observe(headers: Pick<Headers, 'get'> | null | undefined): boolean;

    /**
     * Reports whether any response so far came from a different runtime build.
     *
     * @returns True once a different build has been seen.
     */
    isStale(): boolean;

    /**
     * Claims the right to force a page reload because of a runtime change.
     *
     * A tab may be reloaded for this reason at most once a minute, so a deploy that keeps
     * two builds behind one address cannot trap a page in a reload loop. Without session
     * storage the limit cannot be kept across reloads, so no forced reload is allowed.
     *
     * @returns True when the caller may reload now.
     */
    claimReload(): boolean;
}

/**
 * Options for creating a runtime version gate.
 */
export interface RuntimeVersionGateOptions {
    /** Version of the runtime running in this tab, or null when it is not known. */
    ownVersion: string | null;
    /** Session storage used to rate-limit forced reloads, or null when unavailable. */
    reloadStore?: ReloadStore | null;
    /** Returns the current time in milliseconds. Defaults to Date.now. */
    now?: () => number;
}

/**
 * Reports whether a value has the exact shape of a runtime version.
 *
 * @param value - The value to check.
 * @returns True for exactly sixteen lowercase hex characters.
 */
export function isRuntimeVersion(value: string | null | undefined): value is string {
    return typeof value === 'string' && RUNTIME_VERSION_PATTERN.test(value);
}

/**
 * Reads the runtime version from the URL the runtime module was loaded from.
 *
 * Versioned bundle URLs have the form `<prefix>/<version>/<file>`.
 *
 * @param moduleURL - The URL of the runtime module, usually `import.meta.url`.
 * @returns The version, or null when the URL carries none.
 */
export function runtimeVersionFromModuleURL(moduleURL: string): string | null {
    let pathname: string;
    try {
        pathname = new URL(moduleURL).pathname;
    } catch {
        return null;
    }
    const segments = pathname.split('/');
    if (segments.length < VERSION_SEGMENT_FROM_END || segments[segments.length - 1] === '') {
        return null;
    }
    const candidate = segments[segments.length - VERSION_SEGMENT_FROM_END];
    return isRuntimeVersion(candidate) ? candidate : null;
}

/**
 * Reports whether a recorded forced reload is recent enough to block another one. A
 * record from the future, which a clock change can leave, never blocks a reload.
 *
 * @param lastReloadAt - When the last forced reload happened, or zero or NaN when none is
 *     recorded.
 * @param now - The current time in milliseconds.
 * @returns True when the last reload is within the interval.
 */
function isWithinReloadInterval(lastReloadAt: number, now: number): boolean {
    if (!Number.isFinite(lastReloadAt) || lastReloadAt <= 0) {
        return false;
    }
    const elapsed = now - lastReloadAt;
    return elapsed >= 0 && elapsed < RELOAD_INTERVAL_MS;
}

/**
 * Claims a forced reload against the rate limit kept in session storage.
 *
 * The record must be written before the reload, or the next page could not see it. When
 * session storage is missing or fails, the claim is refused, because a reload that
 * cannot be recorded could repeat without end.
 *
 * @param reloadStore - The session storage, or null when unavailable.
 * @param now - The current time in milliseconds.
 * @returns True when no forced reload happened within the interval and this one was
 *     recorded.
 */
function claimReloadSlot(reloadStore: ReloadStore | null, now: number): boolean {
    if (!reloadStore) {
        return false;
    }
    try {
        if (isWithinReloadInterval(Number(reloadStore.getItem(RELOAD_STORAGE_KEY)), now)) {
            return false;
        }
        reloadStore.setItem(RELOAD_STORAGE_KEY, String(now));
        return true;
    } catch {
        return false;
    }
}

/**
 * Creates a runtime version gate.
 *
 * When either side's version is unknown the gate never reports a change, so a server or
 * page without versioning behaves as before.
 *
 * @param options - The tab's own version and the reload rate-limit dependencies.
 * @returns A new gate.
 */
export function createRuntimeVersionGate(options: RuntimeVersionGateOptions): RuntimeVersionGate {
    const ownVersion = isRuntimeVersion(options.ownVersion) ? options.ownVersion : null;
    const reloadStore = options.reloadStore ?? null;
    const now = options.now ?? Date.now;
    let stale = false;

    return {
        ownVersion,

        observe(headers: Pick<Headers, 'get'> | null | undefined): boolean {
            const serverVersion = headers?.get(RUNTIME_VERSION_HEADER);
            if (ownVersion === null || !isRuntimeVersion(serverVersion) || serverVersion === ownVersion) {
                return false;
            }
            stale = true;
            return true;
        },

        isStale(): boolean {
            return stale;
        },

        claimReload(): boolean {
            return claimReloadSlot(reloadStore, now());
        }
    };
}

/**
 * Returns the session storage of this tab, or null when it cannot be used.
 *
 * @returns The session storage, or null.
 */
function browserReloadStore(): ReloadStore | null {
    try {
        return typeof sessionStorage === 'undefined' ? null : sessionStorage;
    } catch {
        return null;
    }
}

/** The gate for the runtime running in this tab. */
export const runtimeVersionGate: RuntimeVersionGate = createRuntimeVersionGate({
    ownVersion: runtimeVersionFromModuleURL(import.meta.url),
    reloadStore: browserReloadStore()
});
