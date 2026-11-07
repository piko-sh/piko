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

import * as fs from 'fs';
import * as path from 'path';
import * as vscode from 'vscode';
import {compareGoVersionsDescending, findExecutableOnPath, withPrependedPath} from './pathUtils';

/** Common Go installation locations on Linux. */
const LINUX_LOCATIONS = [
    '/usr/local/go/bin',
    '/opt/go/bin',
];

/** Common Go installation locations on macOS (including Homebrew). */
const MACOS_LOCATIONS = [
    '/usr/local/go/bin',
    '/opt/go/bin',
    '/opt/homebrew/opt/go/bin',
    '/usr/local/opt/go/bin',
];

/** Common Go installation locations on Windows. */
const WINDOWS_LOCATIONS = [
    'C:\\Go\\bin',
    'C:\\Program Files\\Go\\bin',
];

/** The most Go SDK manager installs (`~/sdk/go*`) considered. */
const MAX_SDK_ENTRIES = 256;

/**
 * Resolves the Go binary directory path using a priority chain.
 *
 * Resolution order:
 * 1. Manual override from piko.goBinPath setting
 * 2. IDE detection from Go extension settings (go.goroot, go.alternateTools)
 * 3. The `go` found on the current PATH
 * 4. Global fallback locations (common installation paths, newest ~/sdk first)
 * 5. Return undefined (leave PATH unchanged)
 *
 * @param outputChannel - Optional output channel for logging.
 * @returns The resolved Go bin directory path, or undefined if not found.
 */
export function resolveGoBinPath(outputChannel?: vscode.OutputChannel): string | undefined {
    const config = vscode.workspace.getConfiguration('piko');

    const manualPath = config.get<string>('goBinPath', '');
    if (manualPath && manualPath.trim() !== '') {
        const trimmedPath = manualPath.trim();
        if (isValidGoBinDirectory(trimmedPath)) {
            outputChannel?.appendLine(`Using manual Go bin path: ${trimmedPath}`);
            return trimmedPath;
        }
        outputChannel?.appendLine(`Warning: Manual Go bin path is invalid: ${trimmedPath}`);
    }

    if (config.get<boolean>('detectGoFromExtension', true)) {
        const extensionPath = detectFromGoExtension(outputChannel);
        if (extensionPath) {
            outputChannel?.appendLine(`Detected Go bin path from Go extension: ${extensionPath}`);
            return extensionPath;
        }
    }

    const pathGo = findGoOnPath();
    if (pathGo) {
        outputChannel?.appendLine(`Found Go on PATH: ${pathGo}`);
        return pathGo;
    }

    if (config.get<boolean>('searchGlobalGoLocations', true)) {
        const globalPath = findInGlobalLocations();
        if (globalPath) {
            outputChannel?.appendLine(`Found Go bin in global location: ${globalPath}`);
            return globalPath;
        }
    }

    outputChannel?.appendLine('No specific Go bin path found, using system PATH');
    return undefined;
}

/**
 * Detects the Go binary path from the VS Code Go extension settings.
 *
 * Checks go.alternateTools.go and go.goroot settings.
 *
 * @param outputChannel - Optional output channel for logging.
 * @returns The Go bin directory path, or undefined if not available.
 */
function detectFromGoExtension(outputChannel?: vscode.OutputChannel): string | undefined {
    try {
        const goConfig = vscode.workspace.getConfiguration('go');

        const alternateTools = goConfig.get<Record<string, string>>('alternateTools', {});
        const goAlternatePath = alternateTools['go'];
        if (goAlternatePath) {
            const binDir = path.dirname(goAlternatePath);
            if (isValidGoBinDirectory(binDir)) {
                return binDir;
            }
            if (isValidGoBinDirectory(goAlternatePath)) {
                return goAlternatePath;
            }
        }

        const goRoot = goConfig.get<string>('goroot', '');
        if (goRoot && goRoot.trim() !== '') {
            const binPath = path.join(goRoot.trim(), 'bin');
            if (isValidGoBinDirectory(binPath)) {
                return binPath;
            }
        }

        const envGoRoot = process.env['GOROOT'];
        if (envGoRoot) {
            const binPath = path.join(envGoRoot, 'bin');
            if (isValidGoBinDirectory(binPath)) {
                return binPath;
            }
        }
    } catch (error) {
        outputChannel?.appendLine(`Error detecting Go from extension: ${error}`);
    }

    return undefined;
}

/**
 * Searches common global installation locations for Go.
 *
 * @returns The first valid Go bin directory found, or undefined if none found.
 */
function findInGlobalLocations(): string | undefined {
    const homeDir = process.env['HOME'] ?? process.env['USERPROFILE'] ?? '';
    return listGlobalGoLocations(process.platform, homeDir).find(isValidGoBinDirectory);
}

/**
 * Lists the Go bin directories to try when `go` is not on PATH, in order.
 *
 * @param platform - The Node.js platform.
 * @param homeDir - The user's home directory.
 * @returns The common install locations, then `~/go/bin`, then the Go SDK
 * manager installs (newest first).
 */
export function listGlobalGoLocations(platform: string, homeDir: string): string[] {
    let locations: readonly string[];
    switch (platform) {
        case 'win32':
            locations = WINDOWS_LOCATIONS;
            break;
        case 'darwin':
            locations = MACOS_LOCATIONS;
            break;
        default:
            locations = LINUX_LOCATIONS;
            break;
    }
    return [...locations, path.join(homeDir, 'go', 'bin'), ...findGoSdkManagerInstalls(homeDir)];
}

/**
 * Finds Go installations from the Go SDK manager (~/sdk/go*).
 *
 * @param homeDir - The user's home directory.
 * @returns A list of bin directories, sorted semantically by version (newest first).
 */
function findGoSdkManagerInstalls(homeDir: string): string[] {
    const sdkDir = path.join(homeDir, 'sdk');

    try {
        if (!fs.existsSync(sdkDir)) {
            return [];
        }

        const stats = fs.statSync(sdkDir);
        if (!stats.isDirectory()) {
            return [];
        }

        const entries = fs.readdirSync(sdkDir)
            .filter(name => name.startsWith('go'))
            .sort(compareGoVersionsDescending)
            .slice(0, MAX_SDK_ENTRIES);

        return entries.map(name => path.join(sdkDir, name, 'bin'));
    } catch {
        return [];
    }
}

/**
 * Checks if a directory contains a valid Go binary.
 *
 * @param dirPath - The directory path to check.
 * @returns True if the directory contains an executable 'go' binary.
 */
function isValidGoBinDirectory(dirPath: string): boolean {
    try {
        if (!fs.statSync(dirPath).isDirectory()) {
            return false;
        }
    } catch {
        return false;
    }
    const goBinary = process.platform === 'win32' ? 'go.exe' : 'go';
    return isExecutableFile(path.join(dirPath, goBinary));
}

/**
 * Finds the directory of the `go` executable on the current PATH.
 *
 * @returns The directory containing `go`, or undefined when not on PATH.
 */
function findGoOnPath(): string | undefined {
    const goPath = findExecutableOnPath('go', process.env, process.platform, isExecutableFile);
    return goPath ? path.dirname(goPath) : undefined;
}

/**
 * Reports whether a path is an executable regular file.
 *
 * @param candidate - The path to check.
 * @returns True when the file exists and can be executed.
 */
export function isExecutableFile(candidate: string): boolean {
    try {
        if (!fs.statSync(candidate).isFile()) {
            return false;
        }
        if (process.platform !== 'win32') {
            fs.accessSync(candidate, fs.constants.X_OK);
        }
        return true;
    } catch {
        return false;
    }
}

/**
 * Builds the environment for pikopls with the Go bin directory prepended to
 * PATH.
 *
 * On Windows the `Path`/`PATH` keys are normalised into one.
 *
 * @param goBinPath - The Go bin directory to prepend, or undefined.
 * @returns The environment for the child process.
 */
export function buildEnvWithGo(goBinPath: string | undefined): NodeJS.ProcessEnv {
    return {
        ...withPrependedPath(process.env, goBinPath, process.platform),
        PIKO_DISABLE_CONSOLE_LOG: 'true',
    };
}
