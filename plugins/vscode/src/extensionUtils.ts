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
 * Pure utility functions for the Piko VS Code extension.
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
    /** The log file path used when file logging is enabled. */
    logFilePath?: string;
    /** An explicit gopls executable path, or empty to let pikopls discover it. */
    goplsPath?: string;
    /** Whether to hard-disable the gopls bridge for Go script blocks. */
    goplsDisable?: boolean;
    /** The maximum number of gopls child processes (0 uses the server default). */
    goplsMaxChildren?: number;
    /** The maximum number of Go-block overlays per gopls child (0 uses the server default). */
    goplsMaxOverlays?: number;
}

/**
 * Describes how the extension talks to pikopls.
 *
 * `stdio` is the default. `tcp` is a debugging opt-in and is limited to a
 * single connection so a stray client cannot attach to the editor's server.
 */
export type LspTransport = { kind: 'stdio' } | { kind: 'tcp'; port: number };

/** The single connection allowed when the extension starts pikopls over TCP. */
export const TCP_MAX_CONNECTIONS = 1;

/** The lowest valid TCP port. */
export const MIN_TCP_PORT = 1;

/** The highest valid TCP port. */
export const MAX_TCP_PORT = 65_535;

/** The longest file name accepted for new Piko files, before the extension. */
export const MAX_FILE_NAME_LENGTH = 128;

/** The highest accepted `piko.goplsMaxChildren` value. */
export const MAX_GOPLS_CHILDREN = 256;

/** The highest accepted `piko.goplsMaxOverlays` value. */
export const MAX_GOPLS_OVERLAYS = 65_536;

/**
 * Maps Node.js `process.platform` values to Go GOOS values, which name the
 * bundled binary directories.
 */
const NODE_PLATFORM_TO_GO_OS: Readonly<Record<string, string>> = {
    win32: 'windows',
    darwin: 'darwin',
    linux: 'linux',
};

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
    if (value.length > MAX_FILE_NAME_LENGTH) {
        return `File name cannot be longer than ${MAX_FILE_NAME_LENGTH} characters`;
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(value)) {
        return 'File name can only contain letters, numbers, hyphens, and underscores';
    }
    return null;
}

/**
 * Checks a TCP port setting.
 *
 * @param value - The configured value.
 * @returns The port, or undefined when it is not a whole number from 1 to 65535.
 */
export function parsePort(value: unknown): number | undefined {
    if (typeof value !== 'number' || !Number.isInteger(value) || value < MIN_TCP_PORT || value > MAX_TCP_PORT) {
        return undefined;
    }
    return value;
}

/**
 * Checks a numeric limit setting where 0 means "use the server default".
 *
 * @param value - The configured value.
 * @param max - The highest accepted value.
 * @returns The limit, or undefined when it is not a whole number from 0 to `max`.
 */
export function parseLimit(value: unknown, max: number): number | undefined {
    if (typeof value !== 'number' || !Number.isInteger(value) || value < 0 || value > max) {
        return undefined;
    }
    return value;
}

/**
 * Converts a Node.js platform name to a Go GOOS name.
 *
 * @param platform - The Node.js platform (linux, darwin, or win32).
 * @returns The Go OS name (linux, darwin, or windows); unknown values pass through.
 */
export function nodePlatformToGoOs(platform: string): string {
    return NODE_PLATFORM_TO_GO_OS[platform] ?? platform;
}

/**
 * Builds the path to the bundled LSP binary for the current platform.
 *
 * The bundled directories use Go naming (`windows-amd64`, `darwin-arm64`), as
 * produced by hack/lib/lsp.sh.
 *
 * @param platform - The Node.js platform (linux, darwin, or win32).
 * @param arch - The Node.js CPU type (x64 or arm64).
 * @returns The relative path to the binary file.
 */
export function resolvePlatformBinaryPath(platform: string, arch: string): string {
    return `bin/${nodePlatformToGoOs(platform)}-${nodeArchToGoArch(arch)}/${getBinaryName(platform)}`;
}

/**
 * Creates the command-line arguments for starting pikopls.
 *
 * One builder serves both transports so the flags never drift between them.
 *
 * @param transport - How the extension connects to the server.
 * @param config - The server settings.
 * @returns The list of command-line arguments.
 */
export function buildServerArgs(transport: LspTransport, config: LspConfig): string[] {
    const args: string[] = [];

    if (transport.kind === 'tcp') {
        args.push('--tcp', `--port=${transport.port}`, `--max-connections=${TCP_MAX_CONNECTIONS}`);
    }

    if (config.enablePprof) {
        args.push('--pprof', `--pprof-port=${config.pprofPort}`);
    }

    if (config.enableFormatting) {
        args.push('--formatting');
    }

    if (config.enableFileLogging) {
        args.push('--file-logging');
        if (config.logFilePath) {
            args.push(`--log-file=${config.logFilePath}`);
        }
    }

    const goplsPath = config.goplsPath?.trim();
    if (goplsPath) {
        args.push(`--gopls-path=${goplsPath}`);
    }

    if (config.goplsDisable) {
        args.push('--gopls-disable');
    }

    if (config.goplsMaxChildren !== undefined && config.goplsMaxChildren > 0) {
        args.push(`--gopls-max-children=${config.goplsMaxChildren}`);
    }

    if (config.goplsMaxOverlays !== undefined && config.goplsMaxOverlays > 0) {
        args.push(`--gopls-max-overlays=${config.goplsMaxOverlays}`);
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
    return platform === 'win32' ? 'pikopls.exe' : 'pikopls';
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
