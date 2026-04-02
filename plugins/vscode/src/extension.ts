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
import * as vscode from 'vscode';
import * as net from 'net';
import {LanguageClient, LanguageClientOptions, ServerOptions} from 'vscode-languageclient/node';
import {createPikoLspMiddleware} from './middleware';
import {parseSfc} from './sfcParserClient';
import {registerCommentCommands} from './commentHandler';
import {PikoDocumentSymbolProvider} from './documentSymbolProvider';
import {PikoFoldingRangeProvider} from './foldingRangeProvider';
import {registerSelectionRangeProvider} from './selectionRangeProvider';
import {DEFAULT_PAGE_TEMPLATE, DEFAULT_PARTIAL_TEMPLATE} from './templates';
import {buildLspArgsFromConfig, LspConfig, resolvePlatformBinaryPath, validateFileName} from './extensionUtils';
import {resolveGoBinPath, buildPathWithGo} from './goPathResolver';
import {extractPRefNames, generatePKDeclaration, generatePKCDeclaration} from './pRefExtractor';
import {ensureFallbackTypes} from './fallbackTypes';

/** Default TCP port for external LSP server. */
const DEFAULT_EXTERNAL_LSP_PORT = 4389;

/** File permission mode for executable binaries. */
const EXECUTABLE_MODE = 0o755;

/** Maximum attempts to connect to the LSP server after starting it. */
const MAX_CONNECTION_ATTEMPTS = 50;

/** Delay between connection attempts in milliseconds. */
const CONNECTION_RETRY_DELAY_MS = 100;

/** Grace period in milliseconds for process termination. */
const PROCESS_TERMINATION_GRACE_MS = 100;

/** Default pprof port when enabled. */
const DEFAULT_PPROF_PORT = 6060;

/** Glob pattern for Piko type definition files. */
const PIKO_TYPES_GLOB = '**/dist/ts/piko{-ide,-actions}.d.ts';

/**
 * Global LSP client instance.
 */
let client: LanguageClient | undefined;

/**
 * Event emitter for signalling virtual TypeScript document changes.
 * Fired when piko-ide.d.ts or piko-actions.d.ts files are created/modified.
 */
let virtualTsChangeEmitter: vscode.EventEmitter<vscode.Uri>;

/**
 * Output channel for extension logging.
 */
let outputChannel: vscode.OutputChannel;

/**
 * Reference to the spawned LSP process for proper cleanup.
 */
let lspProcess: import('child_process').ChildProcess | undefined;

/**
 * Settings that require an LSP restart when changed.
 */
const SETTINGS_REQUIRING_RESTART = [
    'piko.lspPath',
    'piko.useExternalServer',
    'piko.externalServerPort',
    'piko.useStdioTransport',
    'piko.enablePprof',
    'piko.pprofPort',
    'piko.enableFormatting',
    'piko.enableFileLogging',
    'piko.enableTemplateSupport',
    'piko.goBinPath',
    'piko.detectGoFromExtension',
    'piko.searchGlobalGoLocations',
] as const;

/**
 * Creates a virtual document content provider for serving Go script content.
 *
 * Resolves piko-virtual-go:///path/to/doc.pk.go URIs to script block content.
 *
 * @returns The text document content provider.
 */
function createVirtualGoProvider(): vscode.TextDocumentContentProvider {
    return {
        provideTextDocumentContent: uri => {
            const originalPath = uri.path.slice(1).replace(/\.go$/, '');
            const originalUri = vscode.Uri.file(originalPath);

            const doc = vscode.workspace.textDocuments.find(d => d.uri.fsPath === originalUri.fsPath);
            if (!doc) {
                outputChannel.appendLine(`[VirtualGoProvider] Could not find open document for ${originalUri.fsPath}`);
                return readScriptFromDisk(originalPath, uri);
            }

            const sfcBlocks = parseSfc(doc.getText());
            const scriptBlock = sfcBlocks.find(b => b.type === 'script');

            outputChannel.appendLine(`[VirtualGoProvider] Serving content for ${uri.toString()}`);
            return scriptBlock?.content ?? '';
        }
    };
}

/**
 * Creates a virtual document content provider for serving TypeScript script content.
 *
 * Resolves piko-virtual-ts:///path/to/doc.pk.ts URIs to TypeScript script block content.
 * Includes a reference directive to the Piko types for IDE integration.
 *
 * The provider listens for changes to piko-ide.d.ts files via the virtualTsChangeEmitter
 * to invalidate cached content when types are updated.
 *
 * @returns The text document content provider.
 */
function createVirtualTsProvider(): vscode.TextDocumentContentProvider {
    return {
        onDidChange: virtualTsChangeEmitter.event,
        provideTextDocumentContent: uri => {
            const originalPath = uri.path.slice(1).replace(/\.ts$/, '');
            const originalUri = vscode.Uri.file(originalPath);

            const doc = vscode.workspace.textDocuments.find(d => d.uri.fsPath === originalUri.fsPath);
            if (!doc) {
                outputChannel.appendLine(`[VirtualTsProvider] Could not find open document for ${originalUri.fsPath}`);
                return readTsScriptFromDisk(originalPath, uri);
            }

            const sfcBlocks = parseSfc(doc.getText());
            const scriptBlock = sfcBlocks.find(b => b.type === 'script-ts');
            const templateBlock = sfcBlocks.find(b => b.type === 'template');

            outputChannel.appendLine(`[VirtualTsProvider] Serving content for ${uri.toString()}`);
            return buildTsContent(scriptBlock?.content ?? '', originalPath, templateBlock?.content ?? '');
        }
    };
}

/**
 * Reads TypeScript script content from disk when document is not open.
 *
 * @param originalPath - The path to the .pk file.
 * @param uri - The virtual URI for logging.
 * @returns The TypeScript script block content with type references.
 */
function readTsScriptFromDisk(originalPath: string, uri: vscode.Uri): string {
    try {
        const content = fs.readFileSync(originalPath, 'utf-8');
        const sfcBlocks = parseSfc(content);
        const scriptBlock = sfcBlocks.find(b => b.type === 'script-ts');
        const templateBlock = sfcBlocks.find(b => b.type === 'template');
        const scriptContent = scriptBlock?.content ?? '';

        outputChannel.appendLine(`[VirtualTsProvider] Serving content for ${uri.toString()}`);
        return buildTsContent(scriptContent, originalPath, templateBlock?.content ?? '');
    } catch (_error) {
        outputChannel.appendLine(`[VirtualTsProvider] Failed to read file from disk: ${originalPath}`);
        return '';
    }
}

/**
 * Builds TypeScript content with Piko type references and per-file pk context.
 *
 * Prepends triple-slash reference directives to include Piko types
 * so the TypeScript language service can provide intellisense. Also
 * generates a per-file `declare const pk` declaration from p-ref
 * attributes found in the template block, including lifecycle methods.
 *
 * @param scriptContent - The raw TypeScript content from the script block.
 * @param originalPath - The path to the .pk file for resolving types.
 * @param templateContent - The raw HTML content from the template block.
 * @returns The TypeScript content with type references and pk declaration prepended.
 */
function buildTsContent(scriptContent: string, originalPath: string, templateContent: string): string {
    const projectRoot = findProjectRoot(originalPath);
    if (!projectRoot) {
        return scriptContent;
    }

    ensureFallbackTypes(projectRoot, msg => outputChannel.appendLine(msg));

    const pikoTypesPath = path.join(projectRoot, 'dist', 'ts', 'piko-ide.d.ts');
    const actionsTypesPath = path.join(projectRoot, 'dist', 'ts', 'piko-actions.d.ts');

    const references: string[] = [
        '/// <reference lib="esnext" />',
        '/// <reference lib="dom" />',
    ];
    if (fs.existsSync(pikoTypesPath)) {
        references.push(`/// <reference path="${pikoTypesPath}" />`);
    }
    if (fs.existsSync(actionsTypesPath)) {
        references.push(`/// <reference path="${actionsTypesPath}" />`);
    }

    const refNames = extractPRefNames(templateContent);
    const isPKC = originalPath.endsWith('.pkc');
    const contextDecl = isPKC ? generatePKCDeclaration(refNames) : generatePKDeclaration(refNames);

    return `${references.join('\n')}\n${contextDecl}\n${scriptContent}`;
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

    let dir = currentDir;
    while (dir !== root) {
        if (fs.existsSync(path.join(dir, 'dist', 'ts', 'piko-ide.d.ts'))) {
            return dir;
        }
        dir = path.dirname(dir);
    }

    dir = currentDir;
    while (dir !== root) {
        if (fs.existsSync(path.join(dir, 'go.mod'))) {
            return dir;
        }
        dir = path.dirname(dir);
    }

    return undefined;
}

/**
 * Reads script content from disk when document is not open.
 *
 * @param originalPath - The path to the .pk file.
 * @param uri - The virtual URI for logging.
 * @returns The script block content, or empty string on error.
 */
function readScriptFromDisk(originalPath: string, uri: vscode.Uri): string {
    try {
        const content = fs.readFileSync(originalPath, 'utf-8');
        const sfcBlocks = parseSfc(content);
        const scriptBlock = sfcBlocks.find(b => b.type === 'script');
        const scriptContent = scriptBlock?.content ?? '';

        outputChannel.appendLine(`[VirtualGoProvider] Serving content for ${uri.toString()}`);
        outputChannel.appendLine('--- BEGIN VIRTUAL GO CONTENT ---');
        outputChannel.appendLine(scriptContent);
        outputChannel.appendLine('---  END VIRTUAL GO CONTENT  ---');

        return scriptContent;
    } catch (_error) {
        outputChannel.appendLine(`[VirtualGoProvider] Failed to read file from disk: ${originalPath}`);
        return '';
    }
}

/**
 * Registers language feature providers (symbols, folding, selection).
 *
 * @param context - The extension context.
 */
function registerLanguageProviders(context: vscode.ExtensionContext): void {
    const symbolProvider = vscode.languages.registerDocumentSymbolProvider(
        {language: 'piko'},
        new PikoDocumentSymbolProvider(),
        {label: 'Piko'}
    );
    context.subscriptions.push(symbolProvider);

    const foldingProvider = vscode.languages.registerFoldingRangeProvider(
        {language: 'piko'},
        new PikoFoldingRangeProvider()
    );
    context.subscriptions.push(foldingProvider);

    const selectionProvider = registerSelectionRangeProvider(context);
    context.subscriptions.push(selectionProvider);
}

/**
 * Registers a configuration change watcher to restart LSP when needed.
 *
 * @param context - The extension context.
 */
function registerConfigWatcher(context: vscode.ExtensionContext): void {
    const configWatcher = vscode.workspace.onDidChangeConfiguration(async (event) => {
        const needsRestart = SETTINGS_REQUIRING_RESTART.some(setting =>
            event.affectsConfiguration(setting)
        );

        if (needsRestart && client) {
            outputChannel.appendLine('Configuration changed, restarting Piko LSP...');
            await restartLspClient(context);
            outputChannel.appendLine('Piko LSP restarted due to configuration change');
        }
    });
    context.subscriptions.push(configWatcher);
}

/**
 * Registers debug commands for development.
 *
 * @param context - The extension context.
 */
function registerDebugCommands(context: vscode.ExtensionContext): void {
    const debugCommand = vscode.commands.registerCommand('piko.debug.showVirtualGo', async () => {
        const activeEditor = vscode.window.activeTextEditor;
        if (activeEditor?.document.languageId === 'piko') {
            const virtualUri = vscode.Uri.parse(`piko-virtual-go:///${activeEditor.document.uri.fsPath}.go`);

            try {
                const doc = await vscode.workspace.openTextDocument(virtualUri);
                await vscode.window.showTextDocument(doc, {preview: false});
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to open virtual document: ${error}`);
            }
        } else {
            vscode.window.showWarningMessage('Please open a .pk file first.');
        }
    });
    context.subscriptions.push(debugCommand);
}

/**
 * Registers a file watcher for Piko type definition files.
 *
 * Watches for changes to dist/ts/piko-ide.d.ts and dist/ts/piko-actions.d.ts.
 * When these files are created, modified, or deleted, invalidates all virtual
 * TypeScript documents so the TypeScript language service picks up the new types.
 *
 * @param context - The extension context.
 */
function registerPikoTypesWatcher(context: vscode.ExtensionContext): void {
    const typesWatcher = vscode.workspace.createFileSystemWatcher(PIKO_TYPES_GLOB);

    const handleTypesChange = (uri: vscode.Uri) => {
        outputChannel.appendLine(`[PikoTypesWatcher] Type definitions changed: ${uri.fsPath}`);

        invalidateAllVirtualTsDocuments();

        void restartTypeScriptServer();
    };

    typesWatcher.onDidCreate(handleTypesChange);
    typesWatcher.onDidChange(handleTypesChange);
    typesWatcher.onDidDelete(handleTypesChange);

    context.subscriptions.push(typesWatcher);
    outputChannel.appendLine('[PikoTypesWatcher] Watching for Piko type definition changes');
}

/**
 * Invalidates all open virtual TypeScript documents.
 *
 * Fires the change event for each open .pk file that has a corresponding
 * virtual TypeScript document, causing VSCode to re-fetch the content.
 */
function invalidateAllVirtualTsDocuments(): void {
    for (const doc of vscode.workspace.textDocuments) {
        if (doc.languageId === 'piko') {
            const virtualUri = vscode.Uri.parse(`piko-virtual-ts:///${doc.uri.fsPath}.ts`);
            virtualTsChangeEmitter.fire(virtualUri);
        }
    }
}

/**
 * Restarts the TypeScript language server to pick up type changes.
 *
 * Executes the TypeScript "Restart TS Server" command if available.
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
 * Extension activation entry point.
 *
 * Called when VS Code activates the extension (when a .pk file is opened).
 *
 * @param context - The extension context.
 */
export async function activate(context: vscode.ExtensionContext): Promise<void> {
    outputChannel = vscode.window.createOutputChannel('Piko Language Server');
    context.subscriptions.push(outputChannel);
    outputChannel.appendLine('Piko extension activating...');

    const config = vscode.workspace.getConfiguration('piko');
    if (!config.get<boolean>('enableTemplateSupport', true)) {
        outputChannel.appendLine('Piko LSP is disabled in settings. Template support will not be available.');
        vscode.window.showInformationMessage('Piko: Template support is disabled. Enable it in settings.');
        return;
    }

    const virtualGoProvider = vscode.workspace.registerTextDocumentContentProvider(
        'piko-virtual-go',
        createVirtualGoProvider()
    );
    context.subscriptions.push(virtualGoProvider);

    virtualTsChangeEmitter = new vscode.EventEmitter<vscode.Uri>();
    context.subscriptions.push(virtualTsChangeEmitter);

    const virtualTsProvider = vscode.workspace.registerTextDocumentContentProvider(
        'piko-virtual-ts',
        createVirtualTsProvider()
    );
    context.subscriptions.push(virtualTsProvider);

    registerPikoTypesWatcher(context);

    try {
        const serverOptions = createServerOptions(context);
        const clientOptions = createClientOptions();

        client = new LanguageClient('piko', 'Piko Language Server', serverOptions, clientOptions);
        context.subscriptions.push(client);

        outputChannel.appendLine('Starting Piko LSP client...');
        await client.start();
        outputChannel.appendLine('Piko LSP client started successfully');
        vscode.window.showInformationMessage('Piko: Language server activated');

        registerCommands(context);
        registerCommentCommands(context);
        registerLanguageProviders(context);
        registerConfigWatcher(context);

    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        outputChannel.appendLine(`Failed to start Piko LSP: ${errorMessage}`);
        vscode.window.showErrorMessage(`Piko: Failed to start language server. Check the "Piko Language Server" output for details.`);
        throw error;
    }

    registerDebugCommands(context);
}

/**
 * Extension deactivation.
 *
 * Called when the extension is deactivated or VS Code is shutting down.
 */
export async function deactivate(): Promise<void> {
    await stopLspClient();
}

/**
 * Stops the LSP client and cleans up the process.
 */
async function stopLspClient(): Promise<void> {
    if (client) {
        outputChannel.appendLine('Stopping Piko LSP client...');
        try {
            await client.stop();
        } catch (error) {
            outputChannel.appendLine(`Error stopping client: ${error}`);
        }
        outputChannel.appendLine('Piko LSP client stopped.');
    }

    if (lspProcess) {
        outputChannel.appendLine('Cleaning up LSP process...');
        try {
            lspProcess.kill('SIGTERM');
            await new Promise(resolve => setTimeout(resolve, PROCESS_TERMINATION_GRACE_MS));
            if (!lspProcess.killed) {
                lspProcess.kill('SIGKILL');
            }
        } catch (error) {
            outputChannel.appendLine(`Error killing LSP process: ${error}`);
        }
        lspProcess = undefined;
        outputChannel.appendLine('LSP process cleaned up.');
    }
}

/**
 * Restarts the LSP client with fresh configuration.
 *
 * @param context - The extension context.
 */
async function restartLspClient(context: vscode.ExtensionContext): Promise<void> {
    await stopLspClient();

    const config = vscode.workspace.getConfiguration('piko');
    if (!config.get<boolean>('enableTemplateSupport', true)) {
        outputChannel.appendLine('Piko LSP is now disabled in settings.');
        client = undefined;
        return;
    }

    try {
        const serverOptions = createServerOptions(context);
        const clientOptions = createClientOptions();

        client = new LanguageClient('piko', 'Piko Language Server', serverOptions, clientOptions);
        await client.start();
        vscode.window.showInformationMessage('Piko: Language server restarted');
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        outputChannel.appendLine(`Failed to restart Piko LSP: ${errorMessage}`);
        vscode.window.showErrorMessage(`Piko: Failed to restart language server. Check the output for details.`);
    }
}

/**
 * Finds a free port by briefly opening a server socket.
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
 * @param host - The host to connect to.
 * @param port - The port to connect to.
 * @param maxAttempts - Maximum connection attempts.
 * @returns The connected socket.
 */
function waitForConnection(host: string, port: number, maxAttempts: number): Promise<net.Socket> {
    return new Promise((resolve, reject) => {
        let attempts = 0;

        function tryConnect() {
            attempts++;
            const socket = new net.Socket();

            socket.connect(port, host, () => {
                outputChannel.appendLine(`Connected to LSP on attempt ${attempts}`);
                resolve(socket);
            });

            socket.on('error', (err) => {
                socket.destroy();
                if (attempts >= maxAttempts) {
                    reject(new Error(`Failed to connect to LSP after ${maxAttempts} attempts: ${err.message}`));
                } else {
                    setTimeout(tryConnect, CONNECTION_RETRY_DELAY_MS);
                }
            });
        }

        tryConnect();
    });
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
        socket.connect(port, '127.0.0.1', () => {
            resolve({reader: socket, writer: socket});
        });
        socket.on('error', (err) => {
            outputChannel.appendLine(`External LSP connection error: ${err.message}`);
            reject(err);
        });
    });
}

/**
 * Resolves the LSP binary path from configuration or bundled binary.
 *
 * @param context - The extension context.
 * @returns The path to the LSP binary.
 */
function resolveLspBinaryPath(context: vscode.ExtensionContext): string {
    const config = vscode.workspace.getConfiguration('piko');
    const customPath = config.get<string>('lspPath');

    if (customPath && customPath.trim() !== '') {
        const command = customPath.trim();
        outputChannel.appendLine(`Using custom LSP binary: ${command}`);
        if (!fs.existsSync(command)) {
            throw new Error(`Custom piko-lsp binary not found at: ${command}\nPlease check your "piko.lspPath" setting.`);
        }
        return command;
    }

    const platform = process.platform;
    const arch = process.arch;
    const relativeBinaryPath = resolvePlatformBinaryPath(platform, arch);
    const binaryPath = context.asAbsolutePath(relativeBinaryPath);

    outputChannel.appendLine(`Using bundled LSP binary: ${binaryPath}`);

    if (!fs.existsSync(binaryPath)) {
        throw new Error(
            `Bundled piko-lsp binary not found at: ${binaryPath}\n` +
            `Expected path: ${relativeBinaryPath} (Node.js reports: ${platform}-${arch})\n` +
            `The extension may not be installed correctly.`
        );
    }

    if (platform !== 'win32') {
        try {
            fs.chmodSync(binaryPath, EXECUTABLE_MODE);
        } catch (_error) {
            outputChannel.appendLine(`Warning: Could not make binary executable`);
        }
    }

    return binaryPath;
}

/**
 * Builds command-line arguments for the LSP server.
 *
 * @param port - The TCP port for the server.
 * @param config - The workspace configuration.
 * @returns The command-line arguments.
 */
function buildLspArgs(port: number, config: vscode.WorkspaceConfiguration): string[] {
    const lspConfig: LspConfig = {
        enablePprof: config.get<boolean>('enablePprof', false),
        pprofPort: config.get<number>('pprofPort', DEFAULT_PPROF_PORT),
        enableFormatting: config.get<boolean>('enableFormatting', false),
        enableFileLogging: config.get<boolean>('enableFileLogging', true),
    };

    if (lspConfig.enablePprof) {
        outputChannel.appendLine(`pprof enabled on port ${lspConfig.pprofPort}`);
    }
    if (lspConfig.enableFormatting) {
        outputChannel.appendLine('formatting enabled');
    }
    if (lspConfig.enableFileLogging) {
        outputChannel.appendLine('file logging enabled');
    }

    return buildLspArgsFromConfig(port, lspConfig);
}

/**
 * Spawns the LSP process and sets up event handlers.
 *
 * @param command - The command to run.
 * @param args - The command arguments.
 * @returns The spawned child process.
 */
async function spawnLspProcess(command: string, args: string[]): Promise<import('child_process').ChildProcess> {
    const {spawn} = await import('child_process');

    const goBinPath = resolveGoBinPath(outputChannel);
    const modifiedPath = buildPathWithGo(goBinPath);

    const spawnedProcess = spawn(command, args, {
        env: {
            ...process.env,
            PIKO_DISABLE_CONSOLE_LOG: 'true',
            PATH: modifiedPath
        }
    });

    lspProcess = spawnedProcess;

    spawnedProcess.stderr.on('data', (data: Buffer) => {
        outputChannel.appendLine(`[LSP stderr] ${data.toString()}`);
    });

    spawnedProcess.on('error', (err) => {
        outputChannel.appendLine(`LSP process error: ${err.message}`);
    });

    spawnedProcess.on('exit', (code) => {
        outputChannel.appendLine(`LSP process exited with code ${code}`);
        if (lspProcess === spawnedProcess) {
            lspProcess = undefined;
        }
    });

    return spawnedProcess;
}

/**
 * Creates server options for the LSP client.
 *
 * @param context - The extension context.
 * @returns The server options.
 */
function createServerOptions(context: vscode.ExtensionContext): ServerOptions {
    const config = vscode.workspace.getConfiguration('piko');

    if (config.get<boolean>('useExternalServer', false)) {
        const externalPort = config.get<number>('externalServerPort', DEFAULT_EXTERNAL_LSP_PORT);
        return createExternalServerOptions(externalPort);
    }

    const command = resolveLspBinaryPath(context);

    if (config.get<boolean>('useStdioTransport', false)) {
        outputChannel.appendLine('WARNING: Using experimental stdio transport. This may cause IDE freezes.');
        vscode.window.showWarningMessage(
            'Piko: Using experimental stdio transport. This may cause IDE freezes due to pipe buffer issues. ' +
            'Disable "piko.useStdioTransport" if you experience problems.'
        );

        const goBinPath = resolveGoBinPath(outputChannel);
        const modifiedPath = buildPathWithGo(goBinPath);

        return {
            command,
            args: [],
            options: {
                env: {
                    ...process.env,
                    PIKO_DISABLE_CONSOLE_LOG: 'true',
                    PATH: modifiedPath
                }
            }
        };
    }

    return async () => {
        const port = await findFreePort();
        outputChannel.appendLine(`Starting LSP in TCP mode on port ${port}...`);

        const args = buildLspArgs(port, config);
        await spawnLspProcess(command, args);

        const socket = await waitForConnection('127.0.0.1', port, MAX_CONNECTION_ATTEMPTS);
        socket.setNoDelay(true);

        return {reader: socket, writer: socket};
    };
}

/**
 * Creates client options for the LSP client.
 *
 * Configures communication and includes the middleware for request routing.
 *
 * @returns The client options.
 */
function createClientOptions(): LanguageClientOptions {
    const config = vscode.workspace.getConfiguration('piko');
    const traceLevel = config.get<string>('trace.server', 'off');

    return {
        documentSelector: [
            {scheme: 'file', language: 'piko'},
            {scheme: 'untitled', language: 'piko'}
        ],
        synchronize: {
            fileEvents: vscode.workspace.createFileSystemWatcher('**/*.pk')
        },
        middleware: createPikoLspMiddleware(),
        outputChannel,
        traceOutputChannel: traceLevel !== 'off' ? outputChannel : undefined,
        initializationOptions: {},
        errorHandler: {
            error: (error, message, _count) => {
                outputChannel.appendLine(`LSP Error: ${error.message}`);
                if (message) {
                    outputChannel.appendLine(`Message: ${JSON.stringify(message)}`);
                }
                return {action: 1};
            },
            closed: () => {
                outputChannel.appendLine('LSP connection closed. It will not be restarted automatically.');
                return {action: 1};
            }
        }
    };
}

/**
 * Determines the target folder for a new file.
 *
 * Uses the active editor's directory, workspace root, or prompts the user.
 *
 * @returns The target folder URI, or undefined if cancelled.
 */
async function determineTargetFolder(): Promise<vscode.Uri | undefined> {
    const workspaceFolders = vscode.workspace.workspaceFolders;

    if (workspaceFolders && workspaceFolders.length > 0) {
        const activeEditor = vscode.window.activeTextEditor;
        if (activeEditor) {
            const activeDir = path.dirname(activeEditor.document.uri.fsPath);
            return vscode.Uri.file(activeDir);
        }
        return workspaceFolders[0].uri;
    }

    const selected = await vscode.window.showOpenDialog({
        canSelectFiles: false,
        canSelectFolders: true,
        canSelectMany: false,
        openLabel: 'Select folder'
    });

    return selected && selected.length > 0 ? selected[0] : undefined;
}

/**
 * Creates a new Piko file with the selected template.
 *
 * Prompts the user for a template type, file name, and target folder.
 */
async function createNewPikoFile(): Promise<void> {
    const templateType = await vscode.window.showQuickPick(
        [
            {label: 'Page', description: 'Piko page with NoProps and i18n', template: DEFAULT_PAGE_TEMPLATE},
            {label: 'Partial', description: 'Piko partial with Props struct', template: DEFAULT_PARTIAL_TEMPLATE}
        ],
        {placeHolder: 'Select template type'}
    );

    if (!templateType) {
        return;
    }

    const fileName = await vscode.window.showInputBox({
        prompt: 'Enter file name (without extension)',
        placeHolder: 'my-component',
        validateInput: validateFileName
    });

    if (!fileName) {
        return;
    }

    const targetFolder = await determineTargetFolder();
    if (!targetFolder) {
        vscode.window.showErrorMessage('No folder selected for new file');
        return;
    }

    const filePath = vscode.Uri.joinPath(targetFolder, `${fileName}.pk`);

    try {
        await vscode.workspace.fs.writeFile(filePath, Buffer.from(templateType.template, 'utf-8'));
        const doc = await vscode.workspace.openTextDocument(filePath);
        await vscode.window.showTextDocument(doc);
        outputChannel.appendLine(`Created new Piko file: ${filePath.fsPath}`);
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        vscode.window.showErrorMessage(`Failed to create file: ${errorMessage}`);
        outputChannel.appendLine(`Failed to create Piko file: ${errorMessage}`);
    }
}

/**
 * Registers custom commands for the extension.
 *
 * @param context - The extension context.
 */
function registerCommands(context: vscode.ExtensionContext): void {
    const restartCommand = vscode.commands.registerCommand('piko.restartServer', async () => {
        outputChannel.appendLine('Manual restart requested...');
        await restartLspClient(context);
    });

    const showOutputCommand = vscode.commands.registerCommand('piko.showOutput', () => {
        outputChannel.show();
    });

    const openLogsCommand = vscode.commands.registerCommand('piko.openLogs', async () => {
        const logPath = '/tmp/piko-lsp-main.log';
        if (fs.existsSync(logPath)) {
            await vscode.window.showTextDocument(vscode.Uri.file(logPath));
        } else {
            await vscode.window.showWarningMessage(`Piko LSP log file not found at ${logPath}`);
        }
    });

    const newFileCommand = vscode.commands.registerCommand('piko.newFile', createNewPikoFile);

    context.subscriptions.push(restartCommand, showOutputCommand, openLogsCommand, newFileCommand);
}
