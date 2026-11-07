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
import * as net from 'net';
import * as os from 'os';
import * as path from 'path';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {resolvePlatformBinaryPath} from './extensionUtils';

/**
 * Shared state for the vscode and language client mocks.
 */
const state = vi.hoisted(() => ({
    settings: {} as Record<string, unknown>,
    commands: new Map<string, (...args: unknown[]) => unknown>(),
    order: [] as string[],
    clients: [] as Array<{ serverOptions: unknown; clientOptions: Record<string, unknown>; start: ReturnType<typeof import('vitest').vi.fn>; dispose: ReturnType<typeof import('vitest').vi.fn> }>,
    startError: undefined as Error | undefined,
    configListeners: [] as Array<(event: { affectsConfiguration: (key: string) => boolean }) => unknown>,
    contentProviders: new Map<string, { provideTextDocumentContent: (uri: unknown) => string }>(),
    registeredProviders: [] as string[],
    disposedProviders: 0,
    messages: [] as string[],
    quickPick: undefined as unknown,
    inputBox: undefined as string | undefined,
    warningChoice: undefined as string | undefined,
    writtenFiles: [] as string[],
    existingFiles: new Set<string>(),
    activeEditor: undefined as unknown,
    workspaceFolders: undefined as unknown,
    textDocuments: [] as unknown[],
    closeListeners: [] as Array<(document: { uri: unknown }) => unknown>,
    logLines: [] as string[],
}));

/**
 * Creates a URI mock compatible with the extension's use of Uri.
 *
 * @param scheme - The scheme.
 * @param uriPath - The path.
 * @returns The URI mock.
 */
function makeUri(scheme: string, uriPath: string): Record<string, unknown> {
    return {
        scheme,
        path: uriPath,
        fsPath: uriPath,
        with: (change: { scheme?: string; path?: string }) => makeUri(change.scheme ?? scheme, change.path ?? uriPath),
        toString: () => `${scheme}://${uriPath}`,
    };
}

vi.mock('vscode', () => {
    const disposable = (label?: string) => ({
        dispose: () => {
            if (label === 'provider') {
                state.disposedProviders++;
            }
        },
    });
    const uriFile = (p: string) => makeUri('file', p);
    return {
        Uri: {
            file: uriFile,
            joinPath: (base: { fsPath: string }, ...parts: string[]) => uriFile(path.join(base.fsPath, ...parts)),
        },
        Disposable: {from: (...items: Array<{ dispose: () => void }>) => ({dispose: () => items.forEach(item => item.dispose())})},
        EventEmitter: class {
            event = () => disposable();
            fire = vi.fn();
            dispose = vi.fn();
        },
        RelativePattern: class {
            constructor(public base: unknown, public pattern: string) {
            }
        },
        SymbolKind: {Namespace: 2, Module: 1, Property: 6, Function: 11},
        FoldingRangeKind: {Comment: 1, Imports: 2, Region: 3},
        window: {
            createOutputChannel: () => ({
                appendLine: vi.fn((line: string) => state.logLines.push(line)),
                show: vi.fn(),
                dispose: vi.fn(),
            }),
            showErrorMessage: vi.fn(async (message: string) => {
                state.messages.push(`error:${message}`);
                return undefined;
            }),
            showWarningMessage: vi.fn(async (message: string) => {
                state.messages.push(`warning:${message}`);
                return state.warningChoice;
            }),
            showInformationMessage: vi.fn(async (message: string) => {
                state.messages.push(`info:${message}`);
                return undefined;
            }),
            setStatusBarMessage: vi.fn(),
            showQuickPick: vi.fn(async () => state.quickPick),
            showInputBox: vi.fn(async () => state.inputBox),
            showOpenDialog: vi.fn(async () => undefined),
            showTextDocument: vi.fn(async () => undefined),
            get activeTextEditor() {
                return state.activeEditor;
            },
        },
        workspace: {
            getConfiguration: () => ({
                get: (key: string, fallback?: unknown) => (key in state.settings ? state.settings[key] : fallback),
            }),
            get workspaceFolders() {
                return state.workspaceFolders;
            },
            get textDocuments() {
                return state.textDocuments;
            },
            registerTextDocumentContentProvider: (scheme: string, provider: { provideTextDocumentContent: (uri: unknown) => string }) => {
                state.contentProviders.set(scheme, provider);
                state.order.push(`content:${scheme}`);
                return disposable();
            },
            createFileSystemWatcher: () => ({onDidCreate: vi.fn(), onDidChange: vi.fn(), onDidDelete: vi.fn(), dispose: vi.fn()}),
            onDidChangeConfiguration: (listener: (event: { affectsConfiguration: (key: string) => boolean }) => unknown) => {
                state.configListeners.push(listener);
                state.order.push('configWatcher');
                return disposable();
            },
            onDidChangeTextDocument: () => disposable(),
            onDidCloseTextDocument: (listener: (document: { uri: unknown }) => unknown) => {
                state.closeListeners.push(listener);
                return disposable();
            },
            openTextDocument: vi.fn(async (uri: unknown) => uri),
            fs: {
                stat: vi.fn(async (uri: { fsPath: string }) => {
                    if (!state.existingFiles.has(uri.fsPath)) {
                        throw new Error('ENOENT');
                    }
                    return {};
                }),
                writeFile: vi.fn(async (uri: { fsPath: string }) => {
                    state.writtenFiles.push(uri.fsPath);
                }),
            },
        },
        languages: {
            registerDocumentSymbolProvider: () => {
                state.registeredProviders.push('symbols');
                return disposable('provider');
            },
            registerFoldingRangeProvider: () => {
                state.registeredProviders.push('folding');
                return disposable('provider');
            },
            registerSelectionRangeProvider: () => disposable(),
        },
        commands: {
            registerCommand: (name: string, handler: (...args: unknown[]) => unknown) => {
                state.commands.set(name, handler);
                state.order.push(`command:${name}`);
                return disposable();
            },
            executeCommand: vi.fn(),
        },
    };
});

vi.mock('vscode-languageclient/node', () => ({
    TransportKind: {stdio: 0, ipc: 1, pipe: 2, socket: 3},
    LanguageClient: class {
        start = vi.fn(async () => {
            state.order.push('client:start');
            if (state.startError) {
                throw state.startError;
            }
        });
        dispose = vi.fn(async () => undefined);

        constructor(_id: string, _name: string, public serverOptions: unknown, public clientOptions: Record<string, unknown>) {
            state.clients.push(this as never);
        }
    },
}));

vi.mock('./fallbackTypes/piko-ide.txt', () => ({default: ''}));
vi.mock('./fallbackTypes/piko-actions.txt', () => ({default: ''}));

describe('extension activation', () => {
    let extensionRoot: string;
    let context: { subscriptions: Array<{ dispose: () => void }>; asAbsolutePath: (p: string) => string; logUri: { fsPath: string } };

    beforeEach(() => {
        vi.resetModules();
        state.settings = {};
        state.commands.clear();
        state.order = [];
        state.clients = [];
        state.startError = undefined;
        state.configListeners = [];
        state.contentProviders.clear();
        state.registeredProviders = [];
        state.disposedProviders = 0;
        state.messages = [];
        state.writtenFiles = [];
        state.existingFiles = new Set();
        state.activeEditor = undefined;
        state.workspaceFolders = undefined;
        state.textDocuments = [];
        state.closeListeners = [];
        state.logLines = [];

        extensionRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'piko-ext-'));
        const binary = path.join(extensionRoot, resolvePlatformBinaryPath(process.platform, process.arch));
        fs.mkdirSync(path.dirname(binary), {recursive: true});
        fs.writeFileSync(binary, '');
        context = {
            subscriptions: [],
            asAbsolutePath: (p: string) => path.join(extensionRoot, p),
            logUri: {fsPath: path.join(extensionRoot, 'logs')},
        };
    });

    afterEach(() => {
        fs.rmSync(extensionRoot, {recursive: true, force: true});
    });

    it('registers commands and watchers before starting the client, over stdio by default', async () => {
        const extension = await import('./extension');

        await extension.activate(context as never);

        const startIndex = state.order.indexOf('client:start');
        expect(startIndex).toBeGreaterThan(state.order.indexOf('command:piko.restartServer'));
        expect(startIndex).toBeGreaterThan(state.order.indexOf('command:piko.toggleLineComment'));
        expect(startIndex).toBeGreaterThan(state.order.indexOf('configWatcher'));
        expect(state.order).not.toContain('content:piko-virtual-go');

        const serverOptions = state.clients[0].serverOptions as { args: string[]; transport: number };
        expect(serverOptions.transport).toBe(0);
        expect(serverOptions.args).not.toContain('--tcp');
        expect(serverOptions.args).not.toContain('--file-logging');
        expect(state.clients[0].clientOptions['errorHandler']).toBeUndefined();
        expect(state.clients[0].clientOptions['initializationOptions']).toEqual({goBridge: true});

        expect(state.registeredProviders).toEqual(['symbols', 'folding']);
        expect(state.disposedProviders).toBe(2);

        await extension.deactivate();
        expect(state.clients[0].dispose).toHaveBeenCalled();
    });

    it('reports a start failure without throwing and keeps the fallback providers', async () => {
        state.startError = new Error('boom');
        const extension = await import('./extension');

        await expect(extension.activate(context as never)).resolves.toBeUndefined();

        expect(state.messages.some(message => message.startsWith('error:'))).toBe(true);
        expect(state.disposedProviders).toBe(0);
        expect(state.commands.has('piko.newFile')).toBe(true);
    });

    it('restarts even when no client was started, and on relevant setting changes', async () => {
        state.settings = {enableTemplateSupport: false};
        const extension = await import('./extension');
        await extension.activate(context as never);
        expect(state.clients).toHaveLength(0);

        state.settings = {};
        await state.commands.get('piko.restartServer')?.();
        expect(state.clients).toHaveLength(1);

        state.settings = {transport: 'tcp', goplsDisable: true, enableFileLogging: true};
        await state.configListeners[0]({affectsConfiguration: key => key === 'piko.transport'});
        expect(state.clients).toHaveLength(2);
        expect(state.clients[0].dispose).toHaveBeenCalled();
        expect(typeof state.clients[1].serverOptions).toBe('function');
        expect(state.clients[1].clientOptions['initializationOptions']).toEqual({goBridge: false});

        await state.configListeners[0]({affectsConfiguration: () => false});
        expect(state.clients).toHaveLength(2);

        await extension.deactivate();
    });

    it('reports a missing custom lspPath', async () => {
        state.settings = {lspPath: path.join(extensionRoot, 'missing-pikopls')};
        const extension = await import('./extension');

        await extension.activate(context as never);

        expect(state.clients).toHaveLength(0);
        expect(state.messages.some(message => message.startsWith('error:'))).toBe(true);
    });

    it('opens logs only when file logging is enabled', async () => {
        const extension = await import('./extension');
        await extension.activate(context as never);

        await state.commands.get('piko.openLogs')?.();
        expect(state.messages.some(message => message.includes('File logging is off'))).toBe(true);

        state.settings = {enableFileLogging: true};
        await state.commands.get('piko.openLogs')?.();
        expect(state.messages.some(message => message.includes(path.join(extensionRoot, 'logs', 'pikopls.log')))).toBe(true);

        await extension.deactivate();
    });

    it('creates new files, asking before overwriting', async () => {
        const extension = await import('./extension');
        await extension.activate(context as never);
        const folder = path.join(extensionRoot, 'site');
        state.workspaceFolders = [{uri: makeUri('file', folder)}];
        state.activeEditor = {document: {uri: makeUri('untitled', 'Untitled-1')}};
        state.inputBox = 'my-card';

        const {PIKO_FILE_TEMPLATES} = await import('./templates');
        const component = PIKO_FILE_TEMPLATES.find(entry => entry.extension === '.pkc');
        state.quickPick = {label: 'Component', entry: component};

        await state.commands.get('piko.newFile')?.();
        expect(state.writtenFiles).toEqual([path.join(folder, 'my-card.pkc')]);

        state.existingFiles.add(path.join(folder, 'my-card.pkc'));
        state.warningChoice = undefined;
        await state.commands.get('piko.newFile')?.();
        expect(state.writtenFiles).toHaveLength(1);

        state.warningChoice = 'Overwrite';
        await state.commands.get('piko.newFile')?.();
        expect(state.writtenFiles).toHaveLength(2);

        await extension.deactivate();
    });

    it('serves virtual TypeScript content with the header recorded', async () => {
        const extension = await import('./extension');
        await extension.activate(context as never);
        const project = path.join(extensionRoot, 'project');
        fs.mkdirSync(path.join(project, 'pages'), {recursive: true});
        fs.writeFileSync(path.join(project, 'go.mod'), 'module x\n');
        const source = path.join(project, 'pages', 'card.pkc');
        fs.writeFileSync(source, '<template name="x-card"><p p-ref="title"></p></template>\n<script lang="ts">\nconst a = 1;\n</script>\n');
        state.workspaceFolders = [{uri: makeUri('file', extensionRoot)}];

        const provider = state.contentProviders.get('piko-virtual-ts');
        const virtualUri = makeUri('piko-virtual-ts', `${source}.ts`);
        const content = provider?.provideTextDocumentContent(virtualUri) ?? '';

        const {getHeaderLineCount} = await import('./virtualTs');
        const headerLineCount = getHeaderLineCount(virtualUri as never);
        expect(content.split('\n')[headerLineCount + 1]).toBe('const a = 1;');
        expect(content).toContain('readonly title: HTMLElement | null');
        expect(fs.existsSync(path.join(project, 'dist', 'ts', 'piko-ide.d.ts'))).toBe(true);

        expect(provider?.provideTextDocumentContent(makeUri('piko-virtual-ts', `${project}/missing.pk.ts`))).toBe('');

        await extension.deactivate();
    });

    it('does not write fallback types outside the workspace', async () => {
        const extension = await import('./extension');
        await extension.activate(context as never);
        const project = path.join(extensionRoot, 'outside');
        fs.mkdirSync(project, {recursive: true});
        fs.writeFileSync(path.join(project, 'go.mod'), 'module x\n');
        const source = path.join(project, 'page.pk');
        fs.writeFileSync(source, '<script lang="ts">\nconst a = 1;\n</script>\n');
        state.workspaceFolders = [{uri: makeUri('file', path.join(extensionRoot, 'elsewhere'))}];

        state.contentProviders.get('piko-virtual-ts')?.provideTextDocumentContent(makeUri('piko-virtual-ts', `${source}.ts`));

        expect(fs.existsSync(path.join(project, 'dist'))).toBe(false);

        await extension.deactivate();
    });

    it('falls back to defaults for settings that are out of range', async () => {
        state.settings = {enablePprof: true, pprofPort: 70_000, goplsMaxChildren: -1, goplsMaxOverlays: 2.5};
        const extension = await import('./extension');

        await extension.activate(context as never);

        const serverOptions = state.clients[0].serverOptions as { args: string[] };
        expect(serverOptions.args).toContain('--pprof-port=6060');
        expect(serverOptions.args.some(arg => arg.startsWith('--gopls-max'))).toBe(false);
        expect(state.logLines.some(line => line.startsWith('Ignoring piko.pprofPort'))).toBe(true);
        expect(state.logLines.some(line => line.startsWith('Ignoring piko.goplsMaxChildren'))).toBe(true);
        expect(state.logLines.some(line => line.startsWith('Ignoring piko.goplsMaxOverlays'))).toBe(true);

        await extension.deactivate();
    });

    it('connects to an external server on the configured port', async () => {
        const server = net.createServer(socket => socket.end());
        await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
        const address = server.address();
        const port = typeof address === 'object' && address !== null ? address.port : 0;
        state.settings = {useExternalServer: true, externalServerPort: port};
        const extension = await import('./extension');
        await extension.activate(context as never);

        const connect = state.clients[0].serverOptions as () => Promise<{ reader: net.Socket }>;
        const streams = await connect();
        expect(streams.reader.remotePort).toBe(port);
        streams.reader.destroy();

        await extension.deactivate();
        await new Promise(resolve => server.close(resolve));
    });

    it('rejects when an external server refuses the connection', async () => {
        const server = net.createServer();
        await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
        const address = server.address();
        const port = typeof address === 'object' && address !== null ? address.port : 0;
        await new Promise(resolve => server.close(resolve));
        state.settings = {useExternalServer: true, externalServerPort: port};
        const extension = await import('./extension');
        await extension.activate(context as never);

        const connect = state.clients[0].serverOptions as () => Promise<unknown>;
        await expect(connect()).rejects.toThrow();

        await extension.deactivate();
    });

    it.skipIf(process.platform === 'win32')('gives up on TCP mode when pikopls exits before listening', async () => {
        state.settings = {transport: 'tcp'};
        const extension = await import('./extension');
        await extension.activate(context as never);

        const connect = state.clients[0].serverOptions as () => Promise<unknown>;
        await expect(connect()).rejects.toThrow(/exited before accepting a connection/);

        await extension.deactivate();
    });

    it('does not read source files larger than the limit', async () => {
        const extension = await import('./extension');
        await extension.activate(context as never);
        const source = path.join(extensionRoot, 'huge.pk');
        fs.writeFileSync(source, '');
        fs.truncateSync(source, 17 * 1024 * 1024);

        const content = state.contentProviders.get('piko-virtual-ts')?.provideTextDocumentContent(makeUri('piko-virtual-ts', `${source}.ts`));

        expect(content).toBe('');
        expect(state.logLines.some(line => line.includes('byte limit'))).toBe(true);

        await extension.deactivate();
    });

    it('forgets the header length when a virtual document closes', async () => {
        const extension = await import('./extension');
        await extension.activate(context as never);
        const {getHeaderLineCount, recordHeaderLineCount} = await import('./virtualTs');
        const virtualUri = makeUri('piko-virtual-ts', '/site/page.pk.ts');
        recordHeaderLineCount(virtualUri as never, 4);

        for (const listener of state.closeListeners) {
            listener({uri: makeUri('file', '/site/page.pk')});
        }
        expect(getHeaderLineCount(virtualUri as never)).toBe(4);

        for (const listener of state.closeListeners) {
            listener({uri: virtualUri});
        }
        expect(getHeaderLineCount(virtualUri as never)).toBe(0);

        await extension.deactivate();
    });
});
