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
 * The runtime recovery script, served at /_piko/runtime-recovery.js.
 *
 * It is built on its own into a small classic script, so it must not import anything.
 * Pages load it with `defer` before the module scripts, so it runs before them. When a
 * framework bundle or a generated module fails to load, for example because a deploy
 * sent the page to an instance of another build, it reloads the page once.
 */

/**
 * Session storage key that records when this tab last reloaded for a runtime change. It
 * must match RELOAD_STORAGE_KEY in core/RuntimeVersion.ts, so the runtime and this script
 * share one limit.
 */
export const RECOVERY_RELOAD_STORAGE_KEY = 'piko:runtime-reload-at';

/**
 * Shortest time between two forced reloads, in milliseconds. It must match
 * RELOAD_INTERVAL_MS in core/RuntimeVersion.ts.
 */
export const RECOVERY_RELOAD_INTERVAL_MS = 60_000;

/** URL path prefixes of the scripts whose load failure a reload can fix. */
export const RECOVERY_WATCHED_PATH_PREFIXES: readonly string[] = ['/_piko/dist/', '/_piko/assets/pk-js/'];

/** Link relation of a module preload. */
const MODULE_PRELOAD_RELATION = 'modulepreload';

/** Disposition of a policy violation that the browser enforced, not only reported. */
const ENFORCED_DISPOSITION = 'enforce';

/** Start of the names of the policy directives that control which scripts load. */
const SCRIPT_DIRECTIVE_PREFIX = 'script-src';

/** Message for a failure while the policy blocks scripts on this page. */
const BLOCKED_BY_POLICY_MESSAGE = '[piko] The Content Security Policy blocked a script on this page, so the page is not reloaded.';

/** Message for a failure while the browser is offline. */
const OFFLINE_MESSAGE = '[piko] A framework script failed to load while the browser is offline, so the page is not reloaded.';

/** Message for a failure after this page has already started a reload. */
const ALREADY_RELOADING_MESSAGE = '[piko] A framework script failed to load while the page is already reloading.';

/** Message for a failure soon after an earlier forced reload. */
const RECENTLY_RELOADED_MESSAGE = '[piko] A framework script failed to load again soon after a reload, so the page is not reloaded again. Reload the page to try again.';

/**
 * Minimal session storage used to rate-limit forced reloads.
 */
export type RecoveryReloadStore = Pick<Storage, 'getItem' | 'setItem'>;

/**
 * Dependencies of the runtime recovery script.
 */
export interface RuntimeRecoveryDependencies {
    /** The window, which receives load errors in the capture phase. */
    target: Pick<EventTarget, 'addEventListener'>;
    /** The page location, used to resolve URLs and to reload. */
    location: Pick<Location, 'href' | 'origin' | 'reload'>;
    /** Session storage used to rate-limit forced reloads, or null when unavailable. */
    reloadStore: RecoveryReloadStore | null;
    /** Returns the current time in milliseconds. */
    now: () => number;
    /** Returns true when the browser reports that it has no network connection. */
    isOffline: () => boolean;
    /** Runs a task after the events already queued, so a policy violation is seen first. */
    defer: (task: () => void) => void;
    /** Reports a failure the script cannot fix. */
    reportError: (message: string, url: string) => void;
}

/**
 * Finds the watched URL of an element whose load failed.
 *
 * Only script elements and module preload links count, and only when their URL is on
 * this origin and below one of the watched path prefixes.
 *
 * @param element - The target of the error event.
 * @param location - The page location, used to resolve relative URLs.
 * @returns The absolute URL, or null when the element is not watched.
 */
export function watchedFailureURL(element: unknown, location: Pick<Location, 'href' | 'origin'>): string | null {
    const attribute = failedResourceAttribute(element);
    if (attribute === null) {
        return null;
    }
    let url: URL;
    try {
        url = new URL(attribute, location.href);
    } catch {
        return null;
    }
    if (url.origin !== location.origin) {
        return null;
    }
    if (!RECOVERY_WATCHED_PATH_PREFIXES.some(prefix => url.pathname.startsWith(prefix))) {
        return null;
    }
    return url.href;
}

/**
 * Claims a forced reload against the rate limit kept in session storage.
 *
 * The claim works as the runtime's own does. Without session storage, or when it
 * fails, the claim is refused, because a reload that cannot be recorded could repeat
 * without end.
 *
 * @param reloadStore - The session storage, or null when unavailable.
 * @param now - The current time in milliseconds.
 * @returns True when no forced reload happened within the interval and this one was
 *     recorded.
 */
export function claimRecoveryReload(reloadStore: RecoveryReloadStore | null, now: number): boolean {
    if (!reloadStore) {
        return false;
    }
    try {
        if (isWithinReloadInterval(readLastReloadAt(reloadStore), now)) {
            return false;
        }
        reloadStore.setItem(RECOVERY_RELOAD_STORAGE_KEY, String(now));
        return true;
    } catch {
        return false;
    }
}

/**
 * Starts watching the page for framework scripts that fail to load.
 *
 * The first failure reloads the page once, within the shared rate limit. The page is not
 * reloaded after the Content Security Policy blocked a script, because a reload cannot
 * change the policy, nor while the browser is offline, because the reload would fail too.
 *
 * @param deps - The window, location, storage and reporting dependencies.
 */
export function installRuntimeRecovery(deps: RuntimeRecoveryDependencies): void {
    const installedAt = deps.now();
    let scriptBlockedByPolicy = false;
    let reloading = false;

    deps.target.addEventListener('securitypolicyviolation', (event: Event) => {
        if (isEnforcedScriptViolation(event as SecurityPolicyViolationEvent)) {
            scriptBlockedByPolicy = true;
        }
    }, true);

    deps.target.addEventListener('error', (event: Event) => {
        const url = watchedFailureURL(event.target, deps.location);
        if (url === null) {
            return;
        }
        deps.defer(() => {
            if (reloading) {
                return;
            }
            const refusal = reloadRefusal(deps, scriptBlockedByPolicy, installedAt);
            if (refusal !== null) {
                deps.reportError(refusal, url);
                return;
            }
            reloading = true;
            deps.location.reload();
        });
    }, true);
}

/**
 * Decides whether a load failure may reload the page, and claims the reload when it may.
 *
 * @param deps - The storage, clock and network dependencies.
 * @param scriptBlockedByPolicy - Whether the policy has blocked a script on this page.
 * @param installedAt - When the recovery script started on this page.
 * @returns Null when the reload was claimed, or the message explaining why it was not.
 */
function reloadRefusal(deps: RuntimeRecoveryDependencies, scriptBlockedByPolicy: boolean, installedAt: number): string | null {
    if (scriptBlockedByPolicy) {
        return BLOCKED_BY_POLICY_MESSAGE;
    }
    if (deps.isOffline()) {
        return OFFLINE_MESSAGE;
    }
    if (claimRecoveryReload(deps.reloadStore, deps.now())) {
        return null;
    }
    if (reloadClaimedSince(deps.reloadStore, installedAt)) {
        return ALREADY_RELOADING_MESSAGE;
    }
    return RECENTLY_RELOADED_MESSAGE;
}

/**
 * Reports whether a policy violation is an enforced block of a script.
 *
 * @param violation - The violation event.
 * @returns True when the browser blocked a script, not only reported it.
 */
function isEnforcedScriptViolation(violation: SecurityPolicyViolationEvent): boolean {
    if (violation.disposition !== ENFORCED_DISPOSITION) {
        return false;
    }
    const directive = violation.effectiveDirective || violation.violatedDirective;
    return typeof directive === 'string' && directive.startsWith(SCRIPT_DIRECTIVE_PREFIX);
}

/**
 * Reads when the tab last made a forced reload.
 *
 * @param reloadStore - The session storage.
 * @returns The time in milliseconds, or NaN when none is recorded.
 */
function readLastReloadAt(reloadStore: RecoveryReloadStore): number {
    const recorded = reloadStore.getItem(RECOVERY_RELOAD_STORAGE_KEY);
    return recorded === null ? Number.NaN : Number(recorded);
}

/**
 * Reports whether a recorded forced reload is recent enough to block another one. A
 * record from the future, which a clock change can leave, never blocks a reload.
 *
 * @param lastReloadAt - When the last forced reload happened, or NaN.
 * @param now - The current time in milliseconds.
 * @returns True when the last reload is within the interval.
 */
function isWithinReloadInterval(lastReloadAt: number, now: number): boolean {
    if (!Number.isFinite(lastReloadAt) || lastReloadAt <= 0) {
        return false;
    }
    const elapsed = now - lastReloadAt;
    return elapsed >= 0 && elapsed < RECOVERY_RELOAD_INTERVAL_MS;
}

/**
 * Reports whether a forced reload was claimed on this page, for example by the runtime
 * when it found another runtime already running.
 *
 * @param reloadStore - The session storage, or null when unavailable.
 * @param since - When the recovery script started on this page.
 * @returns True when the recorded reload is from this page.
 */
function reloadClaimedSince(reloadStore: RecoveryReloadStore | null, since: number): boolean {
    if (!reloadStore) {
        return false;
    }
    try {
        return readLastReloadAt(reloadStore) >= since;
    } catch {
        return false;
    }
}

/**
 * Returns the URL attribute of an element whose load failure the script watches.
 *
 * @param element - The target of the error event.
 * @returns The src of a script or the href of a module preload link, or null.
 */
function failedResourceAttribute(element: unknown): string | null {
    if (typeof element !== 'object' || element === null || !('tagName' in element) || !('getAttribute' in element)) {
        return null;
    }
    const candidate = element as Element;
    if (candidate.tagName === 'SCRIPT') {
        return candidate.getAttribute('src');
    }
    if (candidate.tagName === 'LINK' && isModulePreload(candidate)) {
        return candidate.getAttribute('href');
    }
    return null;
}

/**
 * Reports whether a link element is a module preload.
 *
 * @param link - The link element.
 * @returns True when its rel attribute names modulepreload.
 */
function isModulePreload(link: Element): boolean {
    const relation = link.getAttribute('rel') ?? '';
    return relation.toLowerCase().split(/\s+/).includes(MODULE_PRELOAD_RELATION);
}

/**
 * Returns the session storage of this tab, or null when it cannot be used.
 *
 * @returns The session storage, or null.
 */
function browserRecoveryStore(): RecoveryReloadStore | null {
    try {
        return typeof sessionStorage === 'undefined' ? null : sessionStorage;
    } catch {
        return null;
    }
}

if (typeof document !== 'undefined' && document.currentScript !== null) {
    installRuntimeRecovery({
        target: window,
        location: window.location,
        reloadStore: browserRecoveryStore(),
        now: () => Date.now(),
        isOffline: () => navigator.onLine === false,
        defer: (task) => {
            setTimeout(task, 0);
        },
        reportError: (message, url) => {
            console.error(message, url);
        }
    });
}
