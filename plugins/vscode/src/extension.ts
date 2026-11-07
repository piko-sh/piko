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
import * as fs from 'fs';
import * as os from 'os';
import * as net from 'net';
import {setTimeout as delay} from 'timers/promises';
import type {ChildProcessWithoutNullStreams} from 'child_process';
import * as vscode from 'vscode';
import {LanguageClient, LanguageClientOptions, ServerOptions, TransportKind} from 'vscode-languageclient/node';
import {createPikoLspMiddleware} from './middleware';
import {parseSfcBlocks} from './sfcBlocks';
import {registerCommentCommands} from './commentHandler';
import {PikoDocumentSymbolProvider} from './documentSymbolProvider';
import {PikoFoldingRangeProvider} from './foldingRangeProvider';
import {registerSelectionRangeProvider} from './selectionRangeProvider';
import {PIKO_FILE_TEMPLATES} from './templates';
import {
    buildServerArgs,
    LspConfig,
    LspTransport,
    MAX_GOPLS_CHILDREN,
    MAX_GOPLS_OVERLAYS,
    parseLimit,
    parsePort,
    resolvePlatformBinaryPath,
    validateFileName,
} from './extensionUtils';
import {buildEnvWithGo, isExecutableFile, resolveGoBinPath} from './goPathResolver';
import {extractPRefNames, generatePKCDeclaration, generatePKDeclaration} from './pRefExtractor';
import {ensureFallbackTypes} from './fallbackTypes';
import {isPathWithin, resolveCommandPath} from './pathUtils';
import {debounce, stopProcess} from './processUtils';
import {
    buildTsContent,
    createVirtualTsUri,
    forgetHeaderLineCount,
    recordHeaderLineCount,
    sourcePathFromVirtualTsUri,
    TsContent,
    VIRTUAL_TS_SCHEME,
} from './virtualTs';

/** Default TCP port for an external LSP server. */
const DEFAULT_EXTERNAL_LSP_PORT = 4389;

/** File permission mode for executable binaries. */
const EXECUTABLE_MODE = 0o755;

/** Maximum attempts to connect to a TCP server after starting it. */
const MAX_CONNECTION_ATTEMPTS = 50;

/** Delay between TCP connection attempts in milliseconds. */
const CONNECTION_RETRY_DELAY_MS = 100;

/** How long to wait for an external server to accept a TCP connection. */
const EXTERNAL_CONNECT_TIMEOUT_MS = 10_000;

/** The largest .pk or .pkc file (16 MiB) read from disk to build a virtual TypeScript document. */
const MAX_SOURCE_FILE_BYTES = 16_777_216;

/** How long to wait for the server process to exit at each shutdown stage. */
const PROCESS_EXIT_GRACE_MS = 2000;

/** Default pprof port when enabled. */
const DEFAULT_PPROF_PORT = 6060;

/** Glob (relative to each workspace folder) for Piko type definition files. */
const PIKO_TYPES_GLOB = '**/dist/ts/piko{-ide,-actions}.d.ts';

/** Debounce for reacting to type definition changes. */
const TYPES_CHANGE_DEBOUNCE_MS = 500;

/** Files whose changes the server needs to hear about. */
const SERVER_WATCH_GLOB = '**/{*.pk,*.pkc,*.go,go.mod,go.work}';

/** How long the restart confirmation stays in the status bar. */
const RESTART_STATUS_MESSAGE_MS = 3000;

/** The log file name inside the extension's log directory. */
const LOG_FILE_NAME = 'pikopls.log';

/** Settings that require an LSP restart when changed. */
const SETTINGS_REQUIRING_RESTART = [
    'piko.lspPath',
    'piko.transport',
    'piko.useExternalServer',
    'piko.externalServerPort',
    'piko.enablePprof',
    'piko.pprofPort',
    'piko.enableFormatting',
    'piko.enableFileLogging',
    'piko.enableTemplateSupport',
    'piko.goBinPath',
    'piko.detectGoFromExtension',
    'piko.searchGlobalGoLocations',
    'piko.goplsPath',
    'piko.goplsDisable',
    'piko.goplsMaxChildren',
    'piko.goplsMaxOverlays',
    'piko.trace.server',
] as const;

/**
 * Watches a server process for a failed start or an early exit.
 */
interface EarlyExitWatch {
    /** Rejects when the process fails to start or exits. */
    failure: Promise<never>;
    /** Reports whether the process has failed or exited. */
    hasFailed: () => boolean;
    /** Stops watching the process. */
    dispose: () => void;
}

/** The running language client, if any. */
let client: LanguageClient | undefined;

/** The pikopls process spawned for TCP mode (stdio processes are owned by the client). */
let tcpServerProcess: ChildProcessWithoutNullStreams | undefined;

/** Disposables tied to the current client (file watchers); disposed on restart. */
let clientDisposables: vscode.Disposable[] = [];

/** Client-side symbol and folding providers, registered only while pikopls is not running. */
let fallbackProviders: vscode.Disposable | undefined;

/** Serialises start/stop/restart so overlapping requests cannot interleave. */
let lifecycle: Promise<void> = Promise.resolve();

/** Signals virtual TypeScript document changes. */
let virtualTsChangeEmitter: vscode.EventEmitter<vscode.Uri>;

/** Output channel for extension logging. */
let outputChannel: vscode.LogOutputChannel;

/** The extension context, captured at activation. */
let extensionContext: vscode.ExtensionContext;

/**
 * Extension activation entry point.
 *
 * Commands, providers and watchers are registered before the language
 * client starts, so they work even when the server fails to start.
 *
 * @param context - The extension context.
 */
export async function activate(context: vscode.ExtensionContext): Promise<void> {
    extensionContext = context;
    outputChannel = vscode.window.createOutputChannel('Piko Language Server', {log: true});
    context.subscriptions.push(outputChannel);
    outputChannel.appendLine('Piko extension activating...');

    virtualTsChangeEmitter = new vscode.EventEmitter<vscode.Uri>();
    context.subscriptions.push(
        virtualTsChangeEmitter,
        vscode.workspace.registerTextDocumentContentProvider(VIRTUAL_TS_SCHEME, createVirtualTsProvider()),
        {dispose: () => setFallbackProvidersActive(false)}
    );

    registerCommands(context);
    registerCommentCommands(context);
    context.subscriptions.push(registerSelectionRangeProvider(context));
    registerConfigWatcher(context);
    registerPikoTypesWatcher(context);
    registerVirtualTsInvalidation(context);
    setFallbackProvidersActive(true);

    await enqueueLifecycle(startLspClient);
}

/**
 * Extension deactivation.
 */
export async function deactivate(): Promise<void> {
    await enqueueLifecycle(stopLspClient);
}

/**
 * Runs a lifecycle step after any in-flight one finishes.
 *
 * @param step - The step to run.
 * @returns A promise that settles when the step has run.
 */
function enqueueLifecycle(step: () => Promise<void>): Promise<void> {
    lifecycle = lifecycle.then(step, step);
    return lifecycle;
}

/**
 * Registers or disposes the client-side symbol and folding providers.
 *
 * pikopls provides both, so the client-side versions are only active while
 * the server is not running (disabled, failed to start or stopped).
 *
 * @param active - Whether the fallback providers should be registered.
 */
function setFallbackProvidersActive(active: boolean): void {
    if (active && fallbackProviders === undefined) {
        fallbackProviders = vscode.Disposable.from(
            vscode.languages.registerDocumentSymbolProvider({language: 'piko'}, new PikoDocumentSymbolProvider(), {label: 'Piko'}),
            vscode.languages.registerFoldingRangeProvider({language: 'piko'}, new PikoFoldingRangeProvider())
        );
    } else if (!active && fallbackProviders !== undefined) {
        fallbackProviders.dispose();
        fallbackProviders = undefined;
    }
}

/**
 * Starts the language client if template support is enabled.
 *
 * Failures are reported and logged but never rethrown, so the rest of the
 * extension keeps working and the user can fix settings and restart.
 */
async function startLspClient(): Promise<void> {
    const config = vscode.workspace.getConfiguration('piko');
    if (!config.get<boolean>('enableTemplateSupport', true)) {
        outputChannel.appendLine('Piko LSP is disabled in settings (piko.enableTemplateSupport).');
        setFallbackProvidersActive(true);
        return;
    }

    try {
        const serverOptions = createServerOptions(extensionContext);
        const clientOptions = createClientOptions();
        client = new LanguageClient('piko', 'Piko Language Server', serverOptions, clientOptions);

        outputChannel.appendLine('Starting Piko LSP client...');
        await client.start();
        setFallbackProvidersActive(false);
        outputChannel.appendLine('Piko LSP client started');
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        outputChannel.appendLine(`Failed to start Piko LSP: ${errorMessage}`);
        setFallbackProvidersActive(true);
        void vscode.window.showErrorMessage(
            'Piko: Failed to start the language server. Check the "Piko Language Server" output for details.',
            'Show Output'
        ).then(choice => {
            if (choice === 'Show Output') {
                outputChannel.show();
            }
        });
    }
}

/**
 * Stops the language client and any process it was using.
 */
async function stopLspClient(): Promise<void> {
    const stoppingClient = client;
    client = undefined;

    if (stoppingClient) {
        outputChannel.appendLine('Stopping Piko LSP client...');
        try {
            await stoppingClient.dispose(PROCESS_EXIT_GRACE_MS);
        } catch (error) {
            outputChannel.appendLine(`Error stopping client: ${error}`);
        }
    }

    for (const disposable of clientDisposables) {
        disposable.dispose();
    }
    clientDisposables = [];

    await stopTcpServerProcess();
    setFallbackProvidersActive(true);
}

/**
 * Stops the TCP-mode server process, waiting for it to exit before forcing it.
 */
async function stopTcpServerProcess(): Promise<void> {
    const child = tcpServerProcess;
    tcpServerProcess = undefined;
    if (!child) {
        return;
    }
    const outcome = await stopProcess(child, PROCESS_EXIT_GRACE_MS);
    outputChannel.appendLine(`TCP LSP process ${outcome}`);
}

/**
 * Restarts the language client with fresh configuration.
 *
 * Works whether or not a client is currently running.
 */
async function restartLspClient(): Promise<void> {
    await enqueueLifecycle(async () => {
        await stopLspClient();
        await startLspClient();
    });
}

/**
 * Reads the server settings from configuration.
 *
 * @param config - The piko configuration section.
 * @returns The server settings.
 */
function readLspConfig(config: vscode.WorkspaceConfiguration): LspConfig {
    const enableFileLogging = config.get<boolean>('enableFileLogging', false);
    return {
        enablePprof: config.get<boolean>('enablePprof', false),
        pprofPort: readPortSetting(config, 'pprofPort', DEFAULT_PPROF_PORT),
        enableFormatting: config.get<boolean>('enableFormatting', false),
        enableFileLogging,
        logFilePath: enableFileLogging ? prepareLogFilePath() : undefined,
        goplsPath: expandConfiguredPath(config.get<string>('goplsPath', '')),
        goplsDisable: config.get<boolean>('goplsDisable', false),
        goplsMaxChildren: readLimitSetting(config, 'goplsMaxChildren', MAX_GOPLS_CHILDREN),
        goplsMaxOverlays: readLimitSetting(config, 'goplsMaxOverlays', MAX_GOPLS_OVERLAYS),
    };
}

/**
 * Reads a port setting, falling back to the default when the value is not a
 * valid port.
 *
 * @param config - The piko configuration section.
 * @param key - The setting key inside the piko section.
 * @param fallback - The port used when the setting is not valid.
 * @returns The port to use.
 */
function readPortSetting(config: vscode.WorkspaceConfiguration, key: string, fallback: number): number {
    const value = config.get<unknown>(key, fallback);
    const port = parsePort(value);
    if (port === undefined) {
        outputChannel.appendLine(`Ignoring piko.${key}: ${String(value)} is not a whole number from 1 to 65535. Using ${fallback}.`);
        return fallback;
    }
    return port;
}

/**
 * Reads a numeric limit setting where 0 means the server default, falling
 * back to 0 when the value is not valid.
 *
 * @param config - The piko configuration section.
 * @param key - The setting key inside the piko section.
 * @param max - The highest accepted value.
 * @returns The limit to use.
 */
function readLimitSetting(config: vscode.WorkspaceConfiguration, key: string, max: number): number {
    const value = config.get<unknown>(key, 0);
    const limit = parseLimit(value, max);
    if (limit === undefined) {
        outputChannel.appendLine(`Ignoring piko.${key}: ${String(value)} is not a whole number from 0 to ${max}. Using the server default.`);
        return 0;
    }
    return limit;
}

/**
 * Returns the server log file path inside the extension's log directory.
 *
 * @returns The log file path.
 */
function getLogFilePath(): string {
    return path.join(extensionContext.logUri.fsPath, LOG_FILE_NAME);
}

/**
 * Ensures the log directory exists and returns the log file path.
 *
 * @returns The log file path.
 */
function prepareLogFilePath(): string {
    const logPath = getLogFilePath();
    try {
        fs.mkdirSync(path.dirname(logPath), {recursive: true});
    } catch (error) {
        outputChannel.appendLine(`Could not create log directory: ${error}`);
    }
    return logPath;
}

/**
 * Returns the path expansion context for user-configured paths.
 *
 * @returns The home directory and first workspace folder.
 */
function pathExpansionContext(): { homeDir: string; workspaceFolder?: string } {
    const folders = vscode.workspace.workspaceFolders;
    return {homeDir: os.homedir(), workspaceFolder: folders && folders.length > 0 ? folders[0].uri.fsPath : undefined};
}

/**
 * Expands `~`, `${userHome}` and `${workspaceFolder}` in a configured path.
 *
 * @param raw - The configured value.
 * @returns The expanded value, or undefined when blank.
 */
function expandConfiguredPath(raw: string): string | undefined {
    if (raw.trim() === '') {
        return undefined;
    }
    const {expanded, resolved} = resolveCommandPath(raw, pathExpansionContext(), process.env, process.platform, isExecutableFile);
    return resolved ?? expanded;
}

/**
 * Creates server options for connecting to an external LSP server.
 *
 * @param port - The port to connect to.
 * @returns The server options.
 */
function createExternalServerOptions(port: number): ServerOptions {
    outputChannel.appendLine(`Connecting to external LSP server on port ${port}...`);
    return () => new Promise((resolve, reject) => {
        const socket = new net.Socket();
        const onError = (err: Error): void => {
            socket.destroy();
            outputChannel.appendLine(`External LSP connection error: ${err.message}`);
            reject(err);
        };
        socket.setTimeout(EXTERNAL_CONNECT_TIMEOUT_MS, () => {
            onError(new Error(`Timed out after ${EXTERNAL_CONNECT_TIMEOUT_MS} ms connecting to port ${port}`));
        });
        socket.once('error', onError);
        socket.connect(port, '127.0.0.1', () => {
            socket.setTimeout(0);
            socket.removeListener('error', onError);
            socket.on('error', err => outputChannel.appendLine(`External LSP connection error: ${err.message}`));
            resolve({reader: socket, writer: socket});
        });
    });
}

/**
 * Resolves the LSP binary path from configuration or the bundled binary.
 *
 * `piko.lspPath` may use `~`, `${userHome}` and `${workspaceFolder}`, may be
 * relative to the workspace folder, or may be a bare command found on PATH.
 *
 * @param context - The extension context.
 * @returns The path to the LSP binary.
 */
function resolveLspBinaryPath(context: vscode.ExtensionContext): string {
    const config = vscode.workspace.getConfiguration('piko');
    const customPath = config.get<string>('lspPath', '');

    if (customPath.trim() !== '') {
        const {expanded, resolved} = resolveCommandPath(
            customPath, pathExpansionContext(), process.env, process.platform, isExecutableFile
        );
        if (!resolved) {
            throw new Error(`Custom pikopls binary not found or not executable: ${expanded}\nPlease check your "piko.lspPath" setting.`);
        }
        outputChannel.appendLine(`Using custom LSP binary: ${resolved}`);
        return resolved;
    }

    const relativeBinaryPath = resolvePlatformBinaryPath(process.platform, process.arch);
    const binaryPath = context.asAbsolutePath(relativeBinaryPath);
    outputChannel.appendLine(`Using bundled LSP binary: ${binaryPath}`);

    if (!fs.existsSync(binaryPath)) {
        throw new Error(
            `Bundled pikopls binary not found at: ${binaryPath}\n` +
            `Expected path: ${relativeBinaryPath} (Node.js reports: ${process.platform}-${process.arch})\n` +
            'The extension may not be installed correctly.'
        );
    }

    if (process.platform !== 'win32') {
        try {
            fs.chmodSync(binaryPath, EXECUTABLE_MODE);
        } catch {
            outputChannel.appendLine('Warning: Could not make binary executable');
        }
    }

    return binaryPath;
}

/**
 * Finds a free TCP port by briefly opening a server socket.
 *
 * @returns The available port number.
 */
function findFreePort(): Promise<number> {
    return new Promise((resolve, reject) => {
        const server = net.createServer();
        server.listen(0, '127.0.0.1', () => {
            const address = server.address();
            if (address && typeof address === 'object') {
                const port = address.port;
                server.close(() => resolve(port));
            } else {
                server.close(() => reject(new Error('Could not determine port')));
            }
        });
        server.on('error', reject);
    });
}

/**
 * Waits for a TCP connection to become available.
 *
 * Gives up early when the server process fails to start or exits before
 * accepting a connection.
 *
 * @param port - The port to connect to.
 * @param child - The server process that should be listening on the port.
 * @returns The connected socket.
 */
async function waitForConnection(port: number, child: ChildProcessWithoutNullStreams): Promise<net.Socket> {
    const earlyExit = watchForEarlyExit(child);
    try {
        let lastError: unknown;
        for (let attempt = 1; attempt <= MAX_CONNECTION_ATTEMPTS; attempt++) {
            const outcome = await attemptConnection(port, earlyExit);
            if (outcome.socket !== undefined) {
                outcome.socket.on('error', err => outputChannel.appendLine(`LSP connection error: ${err.message}`));
                outputChannel.appendLine(`Connected to LSP on attempt ${attempt}`);
                return outcome.socket;
            }
            lastError = outcome.error;
            await Promise.race([delay(CONNECTION_RETRY_DELAY_MS), earlyExit.failure]);
        }
        const reason = lastError instanceof Error ? lastError.message : String(lastError);
        throw new Error(`Failed to connect to LSP after ${MAX_CONNECTION_ATTEMPTS} attempts: ${reason}`);
    } finally {
        earlyExit.dispose();
    }
}

/**
 * Makes one connection attempt, giving up when the server process fails.
 *
 * @param port - The port to connect to.
 * @param earlyExit - The watcher for the server process.
 * @returns The socket, or the error from a refused attempt.
 * @throws When the server process failed or exited.
 */
async function attemptConnection(
    port: number,
    earlyExit: EarlyExitWatch
): Promise<{ socket?: net.Socket; error?: unknown }> {
    const connecting = connectOnce(port);
    try {
        return {socket: await Promise.race([connecting, earlyExit.failure])};
    } catch (error) {
        if (!earlyExit.hasFailed()) {
            return {error};
        }
        connecting.then(socket => socket.destroy()).catch(() => undefined);
        throw error;
    }
}

/**
 * Makes one attempt to connect to a local TCP port.
 *
 * @param port - The port to connect to.
 * @returns The connected socket.
 */
function connectOnce(port: number): Promise<net.Socket> {
    return new Promise((resolve, reject) => {
        const socket = new net.Socket();
        const onError = (err: Error): void => {
            socket.destroy();
            reject(err);
        };
        socket.once('error', onError);
        socket.connect(port, '127.0.0.1', () => {
            socket.removeListener('error', onError);
            resolve(socket);
        });
    });
}

/**
 * Watches a server process for a failed start or an exit.
 *
 * @param child - The server process.
 * @returns A promise that rejects when the process fails or exits, a check
 * for whether it has, and a hook that stops watching.
 */
function watchForEarlyExit(child: ChildProcessWithoutNullStreams): EarlyExitWatch {
    let failed = false;
    let onExit: (code: number | null, signal: NodeJS.Signals | null) => void = () => undefined;
    let onError: (err: Error) => void = () => undefined;

    const failure = new Promise<never>((_resolve, reject) => {
        onExit = (code, signal) => {
            failed = true;
            reject(new Error(`pikopls exited before accepting a connection (code ${code}, signal ${signal})`));
        };
        onError = err => {
            failed = true;
            reject(new Error(`pikopls failed to start: ${err.message}`));
        };
        child.once('exit', onExit);
        child.once('error', onError);
    });
    failure.catch(() => undefined);

    return {
        failure,
        hasFailed: () => failed,
        dispose: () => {
            child.removeListener('exit', onExit);
            child.removeListener('error', onError);
        },
    };
}

/**
 * Spawns pikopls for TCP mode and pipes its stderr to the output channel.
 *
 * @param command - The binary to run.
 * @param args - The command arguments.
 * @param env - The process environment.
 * @returns The spawned child process.
 */
async function spawnTcpServerProcess(command: string, args: string[], env: NodeJS.ProcessEnv): Promise<ChildProcessWithoutNullStreams> {
    const {spawn} = await import('child_process');
    const child = spawn(command, args, {env});

    child.stderr.on('data', (data: Buffer) => {
        outputChannel.appendLine(`[LSP stderr] ${data.toString()}`);
    });
    child.on('error', (err) => {
        outputChannel.appendLine(`LSP process error: ${err.message}`);
    });
    child.on('exit', (code, signal) => {
        outputChannel.appendLine(`LSP process exited (code ${code}, signal ${signal})`);
        if (tcpServerProcess === child) {
            tcpServerProcess = undefined;
        }
    });

    return child;
}

/**
 * Creates server options for the LSP client.
 *
 * Stdio is the default transport. TCP is a debugging opt-in limited to one
 * connection. Both use the same argument builder.
 *
 * @param context - The extension context.
 * @returns The server options.
 */
function createServerOptions(context: vscode.ExtensionContext): ServerOptions {
    const config = vscode.workspace.getConfiguration('piko');

    if (config.get<boolean>('useExternalServer', false)) {
        return createExternalServerOptions(readPortSetting(config, 'externalServerPort', DEFAULT_EXTERNAL_LSP_PORT));
    }

    const command = resolveLspBinaryPath(context);
    const lspConfig = readLspConfig(config);
    const env = buildEnvWithGo(resolveGoBinPath(outputChannel));

    if (config.get<string>('transport', 'stdio') !== 'tcp') {
        const args = buildServerArgs({kind: 'stdio'}, lspConfig);
        outputChannel.appendLine(`Starting LSP over stdio: ${command} ${args.join(' ')}`);
        return {command, args, transport: TransportKind.stdio, options: {env}};
    }

    return async () => {
        await stopTcpServerProcess();
        const port = await findFreePort();
        const transport: LspTransport = {kind: 'tcp', port};
        const args = buildServerArgs(transport, lspConfig);
        outputChannel.appendLine(`Starting LSP over TCP: ${command} ${args.join(' ')}`);

        const child = await spawnTcpServerProcess(command, args, env);
        tcpServerProcess = child;
        const socket = await waitForConnection(port, child);
        socket.setNoDelay(true);
        return {reader: socket, writer: socket};
    };
}

/**
 * Creates client options for the LSP client.
 *
 * Uses the client library's default error handler, which restarts a crashed
 * server a bounded number of times.
 *
 * @returns The client options.
 */
function createClientOptions(): LanguageClientOptions {
    const config = vscode.workspace.getConfiguration('piko');
    const traceLevel = config.get<string>('trace.server', 'off');
    const watcher = vscode.workspace.createFileSystemWatcher(SERVER_WATCH_GLOB);
    clientDisposables.push(watcher);

    return {
        documentSelector: [
            {scheme: 'file', language: 'piko'},
            {scheme: 'untitled', language: 'piko'}
        ],
        synchronize: {fileEvents: watcher},
        middleware: createPikoLspMiddleware(outputChannel),
        outputChannel,
        traceOutputChannel: traceLevel !== 'off' ? outputChannel : undefined,
        initializationOptions: {goBridge: !config.get<boolean>('goplsDisable', false)},
    };
}

/**
 * Creates the content provider for virtual TypeScript documents.
 *
 * Resolves piko-virtual-ts URIs to the TypeScript script block content with a
 * header of type references, and records the header length for mapping.
 *
 * @returns The text document content provider.
 */
function createVirtualTsProvider(): vscode.TextDocumentContentProvider {
    return {
        onDidChange: virtualTsChangeEmitter.event,
        provideTextDocumentContent: uri => {
            const sourcePath = sourcePathFromVirtualTsUri(uri);
            const openDocument = vscode.workspace.textDocuments.find(doc => doc.uri.path === sourcePath);
            const fsPath = openDocument?.uri.fsPath ?? vscode.Uri.file(sourcePath).fsPath;

            let text: string;
            try {
                text = openDocument?.getText() ?? readSourceFile(fsPath);
            } catch (error) {
                outputChannel.appendLine(`[VirtualTsProvider] Failed to read ${fsPath}: ${error}`);
                forgetHeaderLineCount(uri);
                return '';
            }

            const built = buildVirtualTsForSource(fsPath, text);
            recordHeaderLineCount(uri, built.headerLineCount);
            return built.content;
        }
    };
}

/**
 * Reads a Piko source file from disk, refusing files that are too large.
 *
 * @param fsPath - The file system path.
 * @returns The file text.
 * @throws When the file cannot be read or is larger than `MAX_SOURCE_FILE_BYTES`.
 */
function readSourceFile(fsPath: string): string {
    const size = fs.statSync(fsPath).size;
    if (size > MAX_SOURCE_FILE_BYTES) {
        throw new Error(`file is ${size} bytes, larger than the ${MAX_SOURCE_FILE_BYTES} byte limit`);
    }
    return fs.readFileSync(fsPath, 'utf-8');
}

/**
 * Builds the virtual TypeScript document for a Piko source file.
 *
 * @param sourcePath - The file system path of the .pk or .pkc file.
 * @param text - The file text.
 * @returns The virtual document content and header length.
 */
function buildVirtualTsForSource(sourcePath: string, text: string): TsContent {
    const blocks = parseSfcBlocks(text);
    const scriptContent = blocks.find(block => block.kind === 'script-ts')?.content ?? '';
    const templateContent = blocks.find(block => block.kind === 'template')?.content ?? '';

    const projectRoot = findProjectRoot(sourcePath);
    if (!projectRoot) {
        return buildTsContent(scriptContent, undefined);
    }

    if (isInsideWorkspace(projectRoot)) {
        ensureFallbackTypes(projectRoot, msg => outputChannel.appendLine(msg));
    }

    const pikoTypesPath = path.join(projectRoot, 'dist', 'ts', 'piko-ide.d.ts');
    const actionsTypesPath = path.join(projectRoot, 'dist', 'ts', 'piko-actions.d.ts');
    const refNames = extractPRefNames(templateContent);

    return buildTsContent(scriptContent, {
        pikoTypesPath: fs.existsSync(pikoTypesPath) ? pikoTypesPath : undefined,
        actionsTypesPath: fs.existsSync(actionsTypesPath) ? actionsTypesPath : undefined,
        contextDeclaration: sourcePath.endsWith('.pkc') ? generatePKCDeclaration(refNames) : generatePKDeclaration(refNames),
    });
}

/**
 * Reports whether a directory lies inside an open workspace folder.
 *
 * Fallback type files are only ever written inside the workspace.
 *
 * @param directory - The directory to check.
 * @returns True when the directory is inside a workspace folder.
 */
function isInsideWorkspace(directory: string): boolean {
    return (vscode.workspace.workspaceFolders ?? []).some(folder => isPathWithin(directory, folder.uri.fsPath));
}

/**
 * Finds the project root by searching upward for dist/ts/piko-ide.d.ts,
 * falling back to a directory containing go.mod.
 *
 * @param startPath - The path to start searching from.
 * @returns The project root path, or undefined if not found.
 */
function findProjectRoot(startPath: string): string | undefined {
    const currentDir = path.dirname(startPath);
    const root = path.parse(currentDir).root;

    for (const marker of [path.join('dist', 'ts', 'piko-ide.d.ts'), 'go.mod']) {
        let dir = currentDir;
        while (dir !== root) {
            if (fs.existsSync(path.join(dir, marker))) {
                return dir;
            }
            dir = path.dirname(dir);
        }
    }

    return undefined;
}

/**
 * Re-serves virtual TypeScript documents when their .pk source changes, so
 * the TypeScript service never works on stale script content, and forgets a
 * virtual document's header length once it is closed.
 *
 * @param context - The extension context.
 */
function registerVirtualTsInvalidation(context: vscode.ExtensionContext): void {
    context.subscriptions.push(
        vscode.workspace.onDidChangeTextDocument(event => {
            if (event.document.languageId === 'piko' && event.contentChanges.length > 0) {
                virtualTsChangeEmitter.fire(createVirtualTsUri(event.document.uri));
            }
        }),
        vscode.workspace.onDidCloseTextDocument(document => {
            if (document.uri.scheme === VIRTUAL_TS_SCHEME) {
                forgetHeaderLineCount(document.uri);
            }
        })
    );
}

/**
 * Invalidates all open virtual TypeScript documents.
 */
function invalidateAllVirtualTsDocuments(): void {
    for (const doc of vscode.workspace.textDocuments) {
        if (doc.languageId === 'piko') {
            virtualTsChangeEmitter.fire(createVirtualTsUri(doc.uri));
        }
    }
}

/**
 * Registers watchers for the Piko type definition files in each workspace
 * folder's dist/ts directories (never under node_modules).
 *
 * Changes are debounced, then invalidate the virtual TypeScript documents and
 * restart the TypeScript server.
 *
 * @param context - The extension context.
 */
function registerPikoTypesWatcher(context: vscode.ExtensionContext): void {
    const onTypesChanged = debounce(() => {
        outputChannel.appendLine('[PikoTypesWatcher] Type definitions changed');
        invalidateAllVirtualTsDocuments();
        void restartTypeScriptServer();
    }, TYPES_CHANGE_DEBOUNCE_MS);
    context.subscriptions.push({dispose: onTypesChanged.cancel});

    const handleChange = (uri: vscode.Uri): void => {
        if (!uri.path.split('/').includes('node_modules')) {
            onTypesChanged.call();
        }
    };

    for (const folder of vscode.workspace.workspaceFolders ?? []) {
        const watcher = vscode.workspace.createFileSystemWatcher(new vscode.RelativePattern(folder, PIKO_TYPES_GLOB));
        watcher.onDidCreate(handleChange);
        watcher.onDidChange(handleChange);
        watcher.onDidDelete(handleChange);
        context.subscriptions.push(watcher);
    }
}

/**
 * Restarts the TypeScript language server to pick up type changes.
 */
async function restartTypeScriptServer(): Promise<void> {
    try {
        await vscode.commands.executeCommand('typescript.restartTsServer');
        outputChannel.appendLine('[PikoTypesWatcher] TypeScript server restarted');
    } catch {
        outputChannel.appendLine('[PikoTypesWatcher] TypeScript server restart not available');
    }
}

/**
 * Registers a configuration change watcher to restart the LSP when needed.
 *
 * @param context - The extension context.
 */
function registerConfigWatcher(context: vscode.ExtensionContext): void {
    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(async (event) => {
        if (SETTINGS_REQUIRING_RESTART.some(setting => event.affectsConfiguration(setting))) {
            outputChannel.appendLine('Configuration changed, restarting Piko LSP...');
            await restartLspClient();
        }
    }));
}

/**
 * Determines the target folder for a new file.
 *
 * Uses the active file's directory, then the workspace root (also for
 * untitled editors), and otherwise prompts the user.
 *
 * @returns The target folder URI, or undefined if cancelled.
 */
async function determineTargetFolder(): Promise<vscode.Uri | undefined> {
    const activeUri = vscode.window.activeTextEditor?.document.uri;
    if (activeUri?.scheme === 'file') {
        return vscode.Uri.file(path.dirname(activeUri.fsPath));
    }

    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (workspaceFolders && workspaceFolders.length > 0) {
        return workspaceFolders[0].uri;
    }

    const selected = await vscode.window.showOpenDialog({
        canSelectFiles: false,
        canSelectFolders: true,
        canSelectMany: false,
        openLabel: 'Select folder'
    });
    return selected?.[0];
}

/**
 * Reports whether a file already exists.
 *
 * @param uri - The file URI.
 * @returns True when the file exists.
 */
async function fileExists(uri: vscode.Uri): Promise<boolean> {
    try {
        await vscode.workspace.fs.stat(uri);
        return true;
    } catch {
        return false;
    }
}

/**
 * Creates a new Piko file from a template.
 *
 * Prompts for the template, file name and folder, and asks before
 * overwriting an existing file.
 */
async function createNewPikoFile(): Promise<void> {
    const template = await vscode.window.showQuickPick(
        PIKO_FILE_TEMPLATES.map(entry => ({label: entry.label, description: entry.description, entry})),
        {placeHolder: 'Select template type'}
    );
    if (!template) {
        return;
    }

    const fileName = await vscode.window.showInputBox({
        prompt: `Enter file name (without ${template.entry.extension})`,
        placeHolder: 'my-component',
        validateInput: validateFileName
    });
    if (!fileName) {
        return;
    }

    const targetFolder = await determineTargetFolder();
    if (!targetFolder) {
        void vscode.window.showErrorMessage('No folder selected for new file');
        return;
    }

    const filePath = vscode.Uri.joinPath(targetFolder, `${fileName}${template.entry.extension}`);
    if (await fileExists(filePath)) {
        const choice = await vscode.window.showWarningMessage(
            `${path.basename(filePath.fsPath)} already exists. Overwrite it?`,
            {modal: true},
            'Overwrite'
        );
        if (choice !== 'Overwrite') {
            return;
        }
    }

    try {
        await vscode.workspace.fs.writeFile(filePath, Buffer.from(template.entry.render(fileName), 'utf-8'));
        await vscode.window.showTextDocument(await vscode.workspace.openTextDocument(filePath));
        outputChannel.appendLine(`Created new Piko file: ${filePath.fsPath}`);
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        void vscode.window.showErrorMessage(`Failed to create file: ${errorMessage}`);
        outputChannel.appendLine(`Failed to create Piko file: ${errorMessage}`);
    }
}

/**
 * Opens the server log file, when file logging is enabled.
 */
async function openLogs(): Promise<void> {
    const config = vscode.workspace.getConfiguration('piko');
    if (!config.get<boolean>('enableFileLogging', false)) {
        void vscode.window.showInformationMessage(
            'Piko: File logging is off. Enable "piko.enableFileLogging" to write server logs to a file.'
        );
        return;
    }

    const logPath = getLogFilePath();
    if (fs.existsSync(logPath)) {
        await vscode.window.showTextDocument(vscode.Uri.file(logPath));
    } else {
        void vscode.window.showWarningMessage(`Piko LSP log file not found at ${logPath}`);
    }
}

/**
 * Registers the extension's commands.
 *
 * @param context - The extension context.
 */
function registerCommands(context: vscode.ExtensionContext): void {
    context.subscriptions.push(
        vscode.commands.registerCommand('piko.restartServer', async () => {
            outputChannel.appendLine('Manual restart requested...');
            await restartLspClient();
            vscode.window.setStatusBarMessage('Piko: language server restarted', RESTART_STATUS_MESSAGE_MS);
        }),
        vscode.commands.registerCommand('piko.showOutput', () => outputChannel.show()),
        vscode.commands.registerCommand('piko.openLogs', openLogs),
        vscode.commands.registerCommand('piko.newFile', createNewPikoFile)
    );
}
