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
 * Pure utility functions for the Piko VSCode extension.
 */

/**
 * Holds settings for the LSP server command-line arguments.
 */
export interface LspConfig {
    /** Whether to enable pprof profiling. */
    enablePprof: boolean;
    /** The port for the pprof server. */
    pprofPort: number;
    /** Whether to enable document formatting. */
    enableFormatting: boolean;
    /** Whether to enable file-based logging. */
    enableFileLogging: boolean;
}

/**
 * Checks if a file name is valid for new Piko files.
 *
 * A valid name contains only letters, numbers, hyphens, and underscores.
 *
 * @param value - The file name to check.
 * @returns An error message if the name is not valid, or null when valid.
 */
export function validateFileName(value: string): string | null {
    if (!value || value.trim() === '') {
        return 'File name cannot be empty';
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(value)) {
        return 'File name can only contain letters, numbers, hyphens, and underscores';
    }
    return null;
}

/**
 * Builds the path to the LSP binary for the current platform.
 *
 * @param platform - The OS name (linux, darwin, or win32).
 * @param arch - The CPU type (x64 or arm64).
 * @returns The relative path to the binary file.
 */
export function resolvePlatformBinaryPath(platform: string, arch: string): string {
    const binaryName = platform === 'win32' ? 'piko-lsp.exe' : 'piko-lsp';
    const goArch = arch === 'x64' ? 'amd64' : arch;
    return `bin/${platform}-${goArch}/${binaryName}`;
}

/**
 * Creates command-line arguments for starting the LSP server.
 *
 * @param port - The TCP port for the server to listen on.
 * @param config - The server settings.
 * @returns The list of command-line arguments.
 */
export function buildLspArgsFromConfig(port: number, config: LspConfig): string[] {
    const args = ['--tcp', `--port=${port}`];

    if (config.enablePprof) {
        args.push('--pprof', `--pprof-port=${config.pprofPort}`);
    }

    if (config.enableFormatting) {
        args.push('--formatting');
    }

    if (config.enableFileLogging) {
        args.push('--file-logging');
    }

    return args;
}

/**
 * Checks if a file name contains only valid characters.
 *
 * @param value - The file name to check.
 * @returns True if the name is valid, false if not.
 */
export function isValidFileName(value: string): boolean {
    return validateFileName(value) === null;
}

/**
 * Gets the LSP binary file name for a platform.
 *
 * @param platform - The OS name.
 * @returns The binary name with extension if needed.
 */
export function getBinaryName(platform: string): string {
    return platform === 'win32' ? 'piko-lsp.exe' : 'piko-lsp';
}

/**
 * Converts a Node.js architecture name to a Go architecture name.
 *
 * @param nodeArch - The Node.js arch (x64 or arm64).
 * @returns The Go arch name (amd64 or arm64).
 */
export function nodeArchToGoArch(nodeArch: string): string {
    return nodeArch === 'x64' ? 'amd64' : nodeArch;
}
