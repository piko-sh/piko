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
 * Manages ES module exports from PK page scripts.
 * Supports scoped function resolution for partials, where functions are stored per-partial
 * using the partial's hashed ID. Friendly partial names map to hashed IDs, and the
 * "@partial-name.fn()" syntax broadcasts to all instances of a partial.
 */
export interface PageContext {
    /**
     * Gets an exported function by name, searching the global scope then all partial scopes.
     * @param name - The function name to look up.
     * @returns The function, or undefined if not found.
     */
    getFunction(name: string): ((...args: unknown[]) => unknown) | undefined;

    /**
     * Checks whether a function is exported in any scope.
     * @param name - The function name to check.
     * @returns True if the function exists.
     */
    hasFunction(name: string): boolean;

    /**
     * Gets all exported function names across all scopes.
     * @returns An array of function names.
     */
    getExportedFunctions(): string[];

    /** Clears all global and scoped exports, typically called on navigation. */
    clear(): void;

    /**
     * Loads exports from an ES module URL, optionally scoped to a partial.
     * @param url - The module URL to import.
     * @param partialName - An optional partial name to scope the exports under.
     */
    loadModule(url: string, partialName?: string): Promise<void>;

    /**
     * Sets exports directly, merging with existing global exports.
     * @param exports - The key-value exports to set.
     */
    setExports(exports: Record<string, unknown>): void;

    /**
     * Gets a function from a specific partial's scope by its hashed ID.
     * @param name - The function name.
     * @param partialId - The partial's hashed identifier.
     * @returns The function, or undefined if not found.
     */
    getScopedFunction(name: string, partialId: string): ((...args: unknown[]) => unknown) | undefined;

    /**
     * Gets all functions matching a name across all instances of a partial by friendly name.
     * @param partialName - The friendly partial name.
     * @param fnName - The function name to look up.
     * @returns An array of matching functions.
     */
    getFunctionsByPartialName(partialName: string, fnName: string): ((...args: unknown[]) => unknown)[];

    /**
     * Registers a partial instance by associating a friendly name with a hashed ID.
     * @param partialName - The friendly partial name.
     * @param partialId - The partial's hashed identifier.
     */
    registerPartialInstance(partialName: string, partialId: string): void;

    /**
     * Gets all registered partial names.
     * @returns An array of partial names.
     */
    getRegisteredPartialNames(): string[];
}

/** Configuration options for PageContext. */
export interface PageContextOptions {
    /**
     * Callback invoked when a module fails to load.
     * @param error - The error that occurred.
     * @param context - A description of the context in which the error occurred.
     */
    onError?: (error: Error, context: string) => void;

    /**
     * Callback invoked when a module is successfully loaded.
     * @param url - The URL of the loaded module.
     * @param exports - The names of exported functions.
     */
    onModuleLoaded?: (url: string, exports: string[]) => void;
}

/** Generic function type for page exports. */
type FunctionType = (...args: unknown[]) => unknown;

/** Map of partial IDs to their exported values. */
type ScopedExports = Record<string, Record<string, unknown>>;

/** Map of partial names to their instance IDs. */
type NameToIdsMap = Record<string, string[]>;

/**
 * Finds a function by name in the global exports.
 * @param name - The function name.
 * @param exports - The global exports record.
 * @returns The function, or undefined if not found or not a function.
 */
function findFunctionInGlobal(name: string, exports: Record<string, unknown>): FunctionType | undefined {
    const exportedFunction = exports[name];
    return typeof exportedFunction === 'function' ? exportedFunction as FunctionType : undefined;
}

/**
 * Finds a function by name across all scoped exports.
 * @param name - The function name.
 * @param scopes - The scoped exports record.
 * @returns The first matching function, or undefined if not found.
 */
function findFunctionInScopes(name: string, scopes: ScopedExports): FunctionType | undefined {
    for (const scope of Object.values(scopes)) {
        const scopedFunction = scope[name];
        if (typeof scopedFunction === 'function') {
            return scopedFunction as FunctionType;
        }
    }
    return undefined;
}

/**
 * Collects all exported function names from global and scoped exports.
 * @param global - The global exports record.
 * @param scopes - The scoped exports record.
 * @returns A deduplicated array of function names.
 */
function collectExportedFunctionNames(global: Record<string, unknown>, scopes: ScopedExports): string[] {
    const fns = new Set<string>();
    for (const key of Object.keys(global)) {
        if (typeof global[key] === 'function') {
            fns.add(key);
        }
    }
    for (const scope of Object.values(scopes)) {
        for (const key of Object.keys(scope)) {
            if (typeof scope[key] === 'function') {
                fns.add(key);
            }
        }
    }
    return Array.from(fns);
}

/**
 * Creates a new PageContext for managing ES module exports from PK page scripts.
 * @param options - The optional configuration for the page context.
 * @returns A new PageContext instance.
 */
export function createPageContext(options: PageContextOptions = {}): PageContext {
    let globalExports: Record<string, unknown> = {};
    let scopedExports: ScopedExports = {};
    let nameToIds: NameToIdsMap = {};

    const ctx: PageContext = {
        getFunction(name: string) {
            return findFunctionInGlobal(name, globalExports) ?? findFunctionInScopes(name, scopedExports);
        },

        hasFunction(name: string) {
            return typeof globalExports[name] === 'function' ||
                Object.values(scopedExports).some(scope => typeof scope[name] === 'function');
        },

        getExportedFunctions() {
            return collectExportedFunctionNames(globalExports, scopedExports);
        },

        clear() {
            globalExports = {};
            scopedExports = {};
            nameToIds = {};
        },

        async loadModule(url: string, partialName?: string) {
            try {
                const module = await import(/* @vite-ignore */ url) as Record<string, unknown>;

                if (partialName) {
                    const partialId = (module.__PARTIAL_ID__ as string | undefined) ?? partialName;
                    scopedExports[partialId] = {...(scopedExports[partialId] ?? {}), ...module};
                    ctx.registerPartialInstance(partialName, partialId);
                } else {
                    globalExports = {...globalExports, ...module};
                }

                if (typeof module.__reinit__ === 'function') {
                    (module.__reinit__ as () => void)();
                }

                options.onModuleLoaded?.(url, ctx.getExportedFunctions());
            } catch (error) {
                if (options.onError) {
                    options.onError(error as Error, `loadModule(${url})`);
                } else {
                    console.error('[PageContext] Failed to load module:', {url, error});
                }
            }
        },

        setExports(newExports: Record<string, unknown>) {
            globalExports = {...globalExports, ...newExports};
        },

        getScopedFunction(name: string, partialId: string) {
            const scope = scopedExports[partialId] as Record<string, unknown> | undefined;
            return scope ? findFunctionInGlobal(name, scope) : undefined;
        },

        getFunctionsByPartialName(partialName: string, fnName: string) {
            return (nameToIds[partialName] ?? [])
                .map(id => scopedExports[id] as Record<string, unknown> | undefined)
                .filter((scope): scope is Record<string, unknown> => Boolean(scope))
                .map(scope => scope[fnName])
                .filter((value): value is FunctionType => typeof value === 'function');
        },

        registerPartialInstance(partialName: string, partialId: string) {
            nameToIds[partialName] ??= [];
            if (!nameToIds[partialName].includes(partialId)) {
                nameToIds[partialName].push(partialId);
            }
        },

        getRegisteredPartialNames() {
            return Object.keys(nameToIds);
        }
    };

    return ctx;
}

/** Global singleton instance. */
let globalPageContext: PageContext | null = null;

/**
 * Gets the global PageContext instance, creating one if needed.
 * @returns The global PageContext instance.
 */
export function getGlobalPageContext(): PageContext {
    globalPageContext ??= createPageContext();
    return globalPageContext;
}

/**
 * Resets the global PageContext by clearing and nullifying the singleton.
 */
export function resetGlobalPageContext(): void {
    globalPageContext?.clear();
    globalPageContext = null;
}

/**
 * Finds the closest matching string using Levenshtein distance.
 * @param target - The string to find a match for.
 * @param candidates - The list of candidate strings to search.
 * @param threshold - The maximum distance for a match to be considered.
 * @returns The best match if within threshold, undefined otherwise.
 */
export function findClosestMatch(target: string, candidates: string[], threshold = 3): string | undefined {
    if (candidates.length === 0) {
        return undefined;
    }

    let bestMatch: string | undefined;
    let bestDistance = Infinity;

    for (const candidate of candidates) {
        const distance = levenshteinDistance(target.toLowerCase(), candidate.toLowerCase());
        if (distance < bestDistance && distance <= threshold) {
            bestDistance = distance;
            bestMatch = candidate;
        }
    }

    return bestMatch;
}

/**
 * Calculates the Levenshtein distance between two strings using a dynamic programming matrix.
 * @param a - The first string.
 * @param b - The second string.
 * @returns The minimum number of single-character edits needed to transform a into b.
 */
function levenshteinDistance(a: string, b: string): number {
    if (a.length === 0) {
        return b.length;
    }
    if (b.length === 0) {
        return a.length;
    }

    const matrix: number[][] = [];

    for (let i = 0; i <= b.length; i++) {
        matrix[i] = [i];
    }

    for (let j = 0; j <= a.length; j++) {
        matrix[0][j] = j;
    }

    for (let i = 1; i <= b.length; i++) {
        for (let j = 1; j <= a.length; j++) {
            if (b.charAt(i - 1) === a.charAt(j - 1)) {
                matrix[i][j] = matrix[i - 1][j - 1];
            } else {
                matrix[i][j] = Math.min(
                    matrix[i - 1][j - 1] + 1,
                    matrix[i][j - 1] + 1,
                    matrix[i - 1][j] + 1
                );
            }
        }
    }

    return matrix[b.length][a.length];
}

if (typeof window !== 'undefined') {
    (window as unknown as {__pikoPageContext: PageContext}).__pikoPageContext = getGlobalPageContext();
}
