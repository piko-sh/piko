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
 * Window property under which the running runtime records itself.
 */
export const ACTIVE_RUNTIME_PROPERTY = '__pikoRuntime';

/**
 * Window property that every runtime build sets while it loads, including builds from
 * before runtimes recorded themselves. Its presence means an older runtime already runs.
 */
const EARLIER_RUNTIME_PROPERTY = '__pikoPageContext';

/**
 * Describes a runtime instance running in a window.
 */
export interface ActiveRuntime {
    /** The runtime version, or null when it is not known. */
    readonly version: string | null;
    /** The URL the runtime module was loaded from. */
    readonly moduleURL: string;
}

/**
 * Thrown while a second runtime loads into a window that already runs one.
 *
 * Throwing during module evaluation stops the rest of the second runtime, and every module
 * that imports it, from running, so the first runtime keeps sole control of the page.
 */
export class RuntimeConflictError extends Error {
    /**
     * Creates the error.
     *
     * @param message - Explains which runtimes met.
     */
    constructor(message: string) {
        super(message);
        this.name = 'RuntimeConflictError';
    }
}

/**
 * Dependencies for claiming the window.
 */
export interface RuntimeClaimDependencies {
    /** The window to claim. */
    win: object;
    /** Reloads the page. */
    reload: () => void;
    /** Returns true when a forced reload is allowed now. */
    claimReload: () => boolean;
}

/**
 * Finds the runtime already running in a window.
 *
 * The window properties are read through Reflect, which keeps their names as plain
 * strings. The minifier's property renaming cannot change them, so every build checks
 * the same names.
 *
 * @param win - The window to inspect.
 * @returns The running runtime, or null when none runs.
 */
function findActiveRuntime(win: object): ActiveRuntime | null {
    const recorded: unknown = Reflect.get(win, ACTIVE_RUNTIME_PROPERTY);
    if (isActiveRuntime(recorded)) {
        return recorded;
    }
    if (isPageContext(Reflect.get(win, EARLIER_RUNTIME_PROPERTY))) {
        return {version: null, moduleURL: 'an earlier runtime'};
    }
    return null;
}

/**
 * Reports whether a window property holds a runtime record. Anything else under that
 * name, such as an element whose id the browser exposes on the window, is ignored.
 *
 * @param value - The property's value.
 * @returns True for an object with a string module URL and a string or null version.
 */
function isActiveRuntime(value: unknown): value is ActiveRuntime {
    if (typeof value !== 'object' || value === null) {
        return false;
    }
    const version: unknown = Reflect.get(value, 'version');
    const moduleURL: unknown = Reflect.get(value, 'moduleURL');
    return typeof moduleURL === 'string' && (version === null || typeof version === 'string');
}

/**
 * Reports whether a window property holds the page context an earlier runtime sets.
 * Anything else under that name, such as an element whose id the browser exposes on the
 * window, is ignored.
 *
 * @param value - The property's value.
 * @returns True for an object with a loadModule function.
 */
function isPageContext(value: unknown): boolean {
    return typeof value === 'object' && value !== null && typeof Reflect.get(value, 'loadModule') === 'function';
}

/**
 * Describes a runtime for an error message.
 *
 * @param runtime - The runtime to describe.
 * @returns A short description.
 */
function describeRuntime(runtime: ActiveRuntime): string {
    return runtime.version ? `version ${runtime.version} from ${runtime.moduleURL}` : runtime.moduleURL;
}

/**
 * Records this runtime as the one running in the window, or refuses to start when one
 * already runs there.
 *
 * Two runtimes meet in one tab when a page module from another build imports its own
 * runtime URL, for example after the server was updated while the tab stayed open, or
 * when the same runtime is reached through two different URLs. Letting the second one
 * start would bind every event twice and split page state between two copies. Instead the
 * page reloads, at most once a minute, so it comes back with a single consistent build.
 *
 * @param deps - The window and reload dependencies.
 * @param own - This runtime.
 * @throws RuntimeConflictError when another runtime already runs in the window.
 */
export function claimRuntime(deps: RuntimeClaimDependencies, own: ActiveRuntime): void {
    const existing = findActiveRuntime(deps.win);
    if (!existing) {
        Reflect.defineProperty(deps.win, ACTIVE_RUNTIME_PROPERTY, {
            value: Object.freeze({...own}),
            configurable: true,
            enumerable: false,
            writable: false
        });
        return;
    }

    const reloading = deps.claimReload();
    if (reloading) {
        deps.reload();
    }
    throw new RuntimeConflictError(
        `[piko] A runtime (${describeRuntime(existing)}) already runs in this page, so this copy ` +
        `(${describeRuntime(own)}) will not start. ${reloading ? 'Reloading the page.' : 'Reload the page to update it.'}`
    );
}
