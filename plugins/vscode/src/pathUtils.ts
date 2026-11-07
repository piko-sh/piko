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

import * as path from 'path';

/**
 * Pure path and environment helpers for locating executables.
 *
 * Kept free of VS Code and file system access (callers pass predicates) so
 * the logic can be unit tested on any platform.
 */

/**
 * Holds the values used to expand variables in user-supplied paths.
 */
export interface PathExpansionContext {
    /** The user's home directory. */
    homeDir: string;
    /** The first workspace folder, when one is open. */
    workspaceFolder?: string;
}

/** The environment key used for PATH when none exists yet. */
const DEFAULT_PATH_KEY = 'PATH';

/** Sort rank of a final Go release (above release candidates and betas). */
const PRE_RANK_RELEASE = 2;

/** Sort rank of a Go release candidate. */
const PRE_RANK_RC = 1;

/** Sort rank of a Go beta. */
const PRE_RANK_BETA = 0;

/** Executable extensions tried on Windows when PATHEXT is not set. */
const DEFAULT_WINDOWS_PATHEXT = '.COM;.EXE;.BAT;.CMD';

/**
 * Expands `~`, `${userHome}` and `${workspaceFolder}` in a user-supplied path.
 *
 * @param raw - The path as written in settings.
 * @param context - The expansion values.
 * @returns The expanded path.
 */
export function expandUserPath(raw: string, context: PathExpansionContext): string {
    let value = raw.trim();
    if (value === '~') {
        value = context.homeDir;
    } else if (value.startsWith('~/') || value.startsWith('~\\')) {
        value = path.join(context.homeDir, value.slice(2));
    }
    value = value.split('${userHome}').join(context.homeDir);
    if (context.workspaceFolder !== undefined) {
        value = value.split('${workspaceFolder}').join(context.workspaceFolder);
    }
    return value;
}

/**
 * Reports whether a value is a bare command name (no directory part), which
 * should be looked up on PATH.
 *
 * @param value - The command or path.
 * @returns True when the value contains no path separator.
 */
export function isBareCommand(value: string): boolean {
    return value !== '' && !value.includes('/') && !value.includes('\\');
}

/**
 * Finds the environment key that holds PATH.
 *
 * On Windows environment keys are case-insensitive and the variable is often
 * spelt `Path`; elsewhere only `PATH` is used.
 *
 * @param env - The environment.
 * @param platform - The Node.js platform.
 * @returns The key to read and write PATH through.
 */
export function findPathKey(env: Readonly<Record<string, string | undefined>>, platform: string): string {
    if (platform !== 'win32') {
        return DEFAULT_PATH_KEY;
    }
    return Object.keys(env).find(key => key.toUpperCase() === DEFAULT_PATH_KEY) ?? DEFAULT_PATH_KEY;
}

/**
 * Reads PATH from an environment.
 *
 * @param env - The environment.
 * @param platform - The Node.js platform.
 * @returns The PATH value, or an empty string.
 */
export function getPathValue(env: Readonly<Record<string, string | undefined>>, platform: string): string {
    return env[findPathKey(env, platform)] ?? '';
}

/**
 * Returns a copy of an environment with a directory prepended to PATH.
 *
 * On Windows every case variant of PATH is collapsed into a single key, so the
 * child process never receives both `Path` and `PATH` with different values.
 *
 * @param env - The environment to copy.
 * @param directory - The directory to prepend, or undefined to leave PATH alone.
 * @param platform - The Node.js platform.
 * @returns The new environment.
 */
export function withPrependedPath(
    env: Readonly<Record<string, string | undefined>>,
    directory: string | undefined,
    platform: string
): Record<string, string | undefined> {
    const key = findPathKey(env, platform);
    const current = getPathValue(env, platform);
    const result: Record<string, string | undefined> = {};

    for (const [name, value] of Object.entries(env)) {
        if (platform === 'win32' && name.toUpperCase() === DEFAULT_PATH_KEY) {
            continue;
        }
        result[name] = value;
    }

    const separator = platform === 'win32' ? ';' : ':';
    if (!directory) {
        result[key] = current;
    } else {
        result[key] = current === '' ? directory : `${directory}${separator}${current}`;
    }
    return result;
}

/**
 * Searches PATH for an executable.
 *
 * @param command - The bare command name (for example `go` or `pikopls`).
 * @param env - The environment holding PATH (and PATHEXT on Windows).
 * @param platform - The Node.js platform.
 * @param isExecutable - Reports whether a candidate path is an executable file.
 * @returns The full path of the first match, or undefined.
 */
export function findExecutableOnPath(
    command: string,
    env: Readonly<Record<string, string | undefined>>,
    platform: string,
    isExecutable: (candidate: string) => boolean
): string | undefined {
    const separator = platform === 'win32' ? ';' : ':';
    const pathModule = platform === 'win32' ? path.win32 : path.posix;
    const directories = getPathValue(env, platform).split(separator).filter(dir => dir !== '');
    const extensions = executableExtensions(command, env, platform);

    for (const directory of directories) {
        for (const extension of extensions) {
            const candidate = pathModule.join(directory, `${command}${extension}`);
            if (isExecutable(candidate)) {
                return candidate;
            }
        }
    }
    return undefined;
}

/**
 * Lists the file extensions to try for a command.
 *
 * @param command - The command name.
 * @param env - The environment (for PATHEXT).
 * @param platform - The Node.js platform.
 * @returns The extensions, including the empty string.
 */
function executableExtensions(
    command: string,
    env: Readonly<Record<string, string | undefined>>,
    platform: string
): string[] {
    if (platform !== 'win32' || path.win32.extname(command) !== '') {
        return [''];
    }
    const pathExtKey = Object.keys(env).find(key => key.toUpperCase() === 'PATHEXT');
    const pathExt = (pathExtKey ? env[pathExtKey] : undefined) ?? DEFAULT_WINDOWS_PATHEXT;
    return ['', ...pathExt.split(';').filter(ext => ext !== '').map(ext => ext.toLowerCase())];
}

/**
 * Parses a Go SDK directory name such as `go1.22.10` or `go1.23rc1`.
 *
 * @param name - The directory name.
 * @returns The numeric version parts (pre-releases sort below releases), or undefined.
 */
export function parseGoVersion(name: string): number[] | undefined {
    const match = /^go(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:(rc|beta)(\d+))?$/.exec(name);
    if (!match) {
        return undefined;
    }
    const groups: Array<string | undefined> = match.slice(1);
    const [major, minor, patch, preKind, preNumber] = groups;
    let preRank = PRE_RANK_RELEASE;
    if (preKind === 'rc') {
        preRank = PRE_RANK_RC;
    } else if (preKind === 'beta') {
        preRank = PRE_RANK_BETA;
    }
    return [Number(major), Number(minor ?? 0), Number(patch ?? 0), preRank, Number(preNumber ?? 0)];
}

/**
 * Compares two Go SDK directory names so the newest sorts first.
 *
 * Unparseable names sort after every version, in reverse lexical order.
 *
 * @param a - The first directory name.
 * @param b - The second directory name.
 * @returns A negative number when `a` is newer than `b`.
 */
export function compareGoVersionsDescending(a: string, b: string): number {
    const versionA = parseGoVersion(a);
    const versionB = parseGoVersion(b);
    if (versionA === undefined || versionB === undefined) {
        if (versionA !== versionB) {
            return versionA === undefined ? 1 : -1;
        }
        return b.localeCompare(a);
    }
    for (let i = 0; i < versionA.length; i++) {
        if (versionA[i] !== versionB[i]) {
            return versionB[i] - versionA[i];
        }
    }
    return 0;
}

/**
 * Reports whether a path is inside (or equal to) a directory.
 *
 * @param candidate - The path to check.
 * @param directory - The containing directory.
 * @returns True when `candidate` is within `directory`.
 */
export function isPathWithin(candidate: string, directory: string): boolean {
    const relative = path.relative(path.resolve(directory), path.resolve(candidate));
    return relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative));
}

/**
 * Resolves a user-configured command (such as `piko.lspPath`) to an
 * executable path.
 *
 * Expands `~`, `${userHome}` and `${workspaceFolder}`; bare command names are
 * looked up on PATH, and relative paths resolve against the workspace folder.
 *
 * @param raw - The configured value.
 * @param context - The expansion values.
 * @param env - The environment holding PATH.
 * @param platform - The Node.js platform.
 * @param isExecutable - Reports whether a candidate path is an executable file.
 * @returns The expanded value and the resolved executable path, if found.
 */
export function resolveCommandPath(
    raw: string,
    context: PathExpansionContext,
    env: Readonly<Record<string, string | undefined>>,
    platform: string,
    isExecutable: (candidate: string) => boolean
): { expanded: string; resolved?: string } {
    const expanded = expandUserPath(raw, context);
    if (isBareCommand(expanded)) {
        return {expanded, resolved: findExecutableOnPath(expanded, env, platform, isExecutable)};
    }

    const pathModule = platform === 'win32' ? path.win32 : path.posix;
    const candidate = pathModule.isAbsolute(expanded) || context.workspaceFolder === undefined
        ? expanded
        : pathModule.resolve(context.workspaceFolder, expanded);
    return {expanded, resolved: isExecutable(candidate) ? candidate : undefined};
}
