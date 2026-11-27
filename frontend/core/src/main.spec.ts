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

import {describe, it, expect} from 'vitest';
import {existsSync, readdirSync, readFileSync} from 'node:fs';
import {basename, join, resolve} from 'node:path';

const REPO_ROOT = resolve(__dirname, '../../..');

const MAIN_SOURCE = join(REPO_ROOT, 'frontend/core/src/main.ts');

const RUNTIME_SHIM_SOURCE = join(REPO_ROOT, 'frontend/core/test/pkRuntimeShim.ts');

const BUILT_DIRECTORY = join(REPO_ROOT, 'internal/daemon/daemon_frontend/built');

const BUILT_BUNDLE = join(BUILT_DIRECTORY, 'ppframework.core.es.js');

const BUILT_MINIFIED_ES_BUNDLE = join(BUILT_DIRECTORY, 'ppframework.core.min.es.js');

const BUILT_MINIFIED_UMD_BUNDLE = join(BUILT_DIRECTORY, 'ppframework.core.min.umd.js');

const BUILT_COMPONENTS_BUNDLE = join(BUILT_DIRECTORY, 'ppframework.components.es.js');

const BUILT_MINIFIED_COMPONENTS_BUNDLE = join(BUILT_DIRECTORY, 'ppframework.components.min.es.js');

const EXTENSION_NAMES = ['analytics', 'animation', 'dev', 'modals', 'toasts'];

const EXTENSION_WINDOW_PROPERTIES = ['piko', 'dataLayer', 'gtag', '__piko_animation', '__piko_registerTimelineAction'];

const DROPPED_CONSOLE_METHODS = ['log', 'warn', 'info', 'debug'];

const COMPILER_GOLDEN_CASES = join(REPO_ROOT, 'internal/compiler/compiler_test/testdata');

const SERVER_DTO_DIRECTORY = join(REPO_ROOT, 'internal/daemon/daemon_dto');

const COMPILED_HANDLER_PREFIXES = ['_dir_', '_hof_', '_once_'];

const COMPILED_PROPERTY_PATTERN = /\.(_[A-Za-z][\w$]*)|['"](_[A-Za-z][\w$]*)['"]|(?<![\w$.])(_[A-Za-z][\w$]*)\s*:/g;

const MODULE_PROTOCOL_PROPERTIES = ['__reinit__', '__PARTIAL_ID__'];

const CORE_WINDOW_PROPERTIES = ['__pikoRuntime', '__pikoPageContext', '__pikoDebug', '__PP_HOOKS_QUEUE__'];

const COMPONENTS_WINDOW_PROPERTIES = ['__piko_animation'];

const COMPONENT_ELEMENT_PROTOCOL_METHODS = ['hasSelfAddedAttribute'];

const SHARED_COMPONENT_PROPERTIES = ['_updateFormState'];

const GENERATOR_GOLDEN_CASES = join(REPO_ROOT, 'internal/generator/generator_test/generator/testdata');

const BUNDLE_URL = '/_piko/dist/ppframework.core.es.js';

function exportedNames(source: string): Set<string> {
    const names = new Set<string>();
    for (const match of source.matchAll(/^export\s*\{([^}]*)\}/gm)) {
        for (const entry of match[1].split(',')) {
            const name = entry.trim().split(/\s+as\s+/).pop();
            if (name) {
                names.add(name);
            }
        }
    }
    for (const match of source.matchAll(/^export\s+(?:async\s+)?(?:function|const|let|class)\s+([A-Za-z_$][\w$]*)/gm)) {
        names.add(match[1]);
    }
    return names;
}

function minifiedExportedNames(source: string): Set<string> {
    const names = new Set<string>();
    for (const match of source.matchAll(/\bexport\s*\{([^}]*)\}/g)) {
        for (const entry of match[1].split(',')) {
            const name = entry.trim().split(/\s+as\s+/).pop();
            if (name) {
                names.add(name);
            }
        }
    }
    return names;
}

function namesMissingFromUmd(source: string, names: Set<string>): string[] {
    return sorted(names).filter(name => {
        const escaped = name.replaceAll('$', '\\$');
        return !new RegExp(`\\.${escaped}=`).test(source);
    });
}

function jsFilesUnder(directory: string): string[] {
    if (!existsSync(directory)) {
        return [];
    }
    return readdirSync(directory, {recursive: true, encoding: 'utf8'})
        .filter(entry => entry.endsWith('.js'))
        .map(entry => join(directory, entry));
}

function namesGeneratedCodeImports(): Map<string, string> {
    const imported = new Map<string, string>();
    const importFromBundle = new RegExp(`import\\s*\\{([^}]*)\\}\\s*from\\s*["']${BUNDLE_URL.replaceAll('.', '\\.')}["']`, 'g');
    for (const caseName of readdirSync(GENERATOR_GOLDEN_CASES)) {
        for (const file of jsFilesUnder(join(GENERATOR_GOLDEN_CASES, caseName, 'golden', 'pk-js'))) {
            for (const match of readFileSync(file, 'utf8').matchAll(importFromBundle)) {
                for (const entry of match[1].split(',')) {
                    const name = entry.trim().split(/\s+as\s+/)[0];
                    if (name && !imported.has(name)) {
                        imported.set(name, file.slice(REPO_ROOT.length + 1));
                    }
                }
            }
        }
    }
    return imported;
}

function usesName(source: string, name: string): boolean {
    const escaped = name.replaceAll('$', '\\$');
    return new RegExp(`(?<![\\w$])${escaped}(?![\\w$])`).test(source);
}

function namesRenamedByMinifier(unminified: string, minified: string, names: Iterable<string>): string[] {
    return sorted(new Set(names)).filter(name => usesName(unminified, name) && !usesName(minified, name));
}

function underscoreNamesGeneratedModulesExport(): Set<string> {
    const names = new Set<string>();
    for (const caseName of readdirSync(GENERATOR_GOLDEN_CASES)) {
        for (const file of jsFilesUnder(join(GENERATOR_GOLDEN_CASES, caseName, 'golden', 'pk-js'))) {
            for (const name of exportedNames(readFileSync(file, 'utf8'))) {
                if (name.startsWith('_')) {
                    names.add(name);
                }
            }
        }
    }
    return names;
}

function underscorePropertiesCompiledComponentsUse(): Set<string> {
    const names = new Set<string>();
    for (const caseName of readdirSync(COMPILER_GOLDEN_CASES)) {
        const golden = join(COMPILER_GOLDEN_CASES, caseName, 'golden.js');
        if (!existsSync(golden)) {
            continue;
        }
        const source = readFileSync(golden, 'utf8');
        for (const match of source.matchAll(COMPILED_PROPERTY_PATTERN)) {
            const name = match[1] ?? match[2] ?? match[3];
            if (!COMPILED_HANDLER_PREFIXES.some(prefix => name.startsWith(prefix))) {
                names.add(name);
            }
        }
    }
    return names;
}

function underscoreJSONNamesServerUses(): Set<string> {
    const names = new Set<string>();
    for (const entry of readdirSync(SERVER_DTO_DIRECTORY)) {
        if (!entry.endsWith('.go') || entry.endsWith('_test.go')) {
            continue;
        }
        const source = readFileSync(join(SERVER_DTO_DIRECTORY, entry), 'utf8');
        for (const match of source.matchAll(/json:"(_[\w$]+)/g)) {
            names.add(match[1]);
        }
    }
    return names;
}

function sorted(names: Set<string>): string[] {
    return [...names].sort((first, second) => first.localeCompare(second));
}

function droppedConsoleCallsLeft(source: string): string[] {
    return DROPPED_CONSOLE_METHODS
        .map(method => `console.${method}`)
        .filter(call => source.includes(`${call}(`));
}

function builtExtensionFile(extension: string, suffix: string): string {
    return join(BUILT_DIRECTORY, `ppframework.${extension}.${suffix}`);
}

describe('main.ts bundle entry', () => {
    const mainExports = exportedNames(readFileSync(MAIN_SOURCE, 'utf8'));

    it('exports every name that generated modules import from the bundle', () => {
        const imported = namesGeneratedCodeImports();

        expect(imported.has('_createPKContext'), 'no golden imports _createPKContext').toBe(true);
        expect(imported.has('_createPagePKContext'), 'no golden imports _createPagePKContext').toBe(true);
        expect(imported.has('_currentPageVisit'), 'no golden imports _currentPageVisit').toBe(true);
        const missing = [...imported].filter(([name]) => !mainExports.has(name)).map(([name, file]) => `${name} (imported by ${file})`);
        expect(missing).toEqual([]);
    });

    it('claims the window before any other runtime module is evaluated', () => {
        const firstImport = readFileSync(MAIN_SOURCE, 'utf8').match(/^import\s+[^;]*;/m);

        expect(firstImport?.[0]).toBe(`import '@/claimRuntime';`);
    });

    it('records the page token before the runtime starts', () => {
        const source = readFileSync(MAIN_SOURCE, 'utf8');
        const captureIndex = source.indexOf('capturePageCSPToken(document);');
        const startIndex = source.indexOf('PPFramework.init();');

        expect(captureIndex, 'main.ts does not record the page token').toBeGreaterThan(-1);
        expect(captureIndex).toBeLessThan(startIndex);
    });

    it('puts the runtime claim first in the built bundle', () => {
        const source = readFileSync(BUILT_BUNDLE, 'utf8');
        const claimIndex = source.search(/claimRuntime\(\s*\{/);
        const pageContextIndex = source.indexOf('__pikoPageContext =');

        expect(claimIndex, 'the built bundle does not claim the window').toBeGreaterThan(-1);
        expect(pageContextIndex, 'the built bundle does not set the page context').toBeGreaterThan(-1);
        expect(claimIndex).toBeLessThan(pageContextIndex);
    });

    it('exports everything the test stand-in for the bundle exports', () => {
        const shimExports = exportedNames(readFileSync(RUNTIME_SHIM_SOURCE, 'utf8'));

        expect([...shimExports].filter(name => !mainExports.has(name))).toEqual([]);
    });

    it('has the same exports as the built bundle', () => {
        expect(existsSync(BUILT_BUNDLE), `${BUILT_BUNDLE} is missing`).toBe(true);
        const builtExports = exportedNames(readFileSync(BUILT_BUNDLE, 'utf8'));

        expect(sorted(builtExports)).toEqual(sorted(mainExports));
    });

    it('keeps every export name in the minified ES build', () => {
        expect(existsSync(BUILT_MINIFIED_ES_BUNDLE), `${BUILT_MINIFIED_ES_BUNDLE} is missing`).toBe(true);
        const builtExports = minifiedExportedNames(readFileSync(BUILT_MINIFIED_ES_BUNDLE, 'utf8'));

        expect(sorted(builtExports)).toEqual(sorted(mainExports));
    });

    it('keeps every export name in the minified UMD build', () => {
        expect(existsSync(BUILT_MINIFIED_UMD_BUNDLE), `${BUILT_MINIFIED_UMD_BUNDLE} is missing`).toBe(true);
        const source = readFileSync(BUILT_MINIFIED_UMD_BUNDLE, 'utf8');

        expect(mainExports.has('_createPagePKContext')).toBe(true);
        expect(namesMissingFromUmd(source, mainExports)).toEqual([]);
    });
});

describe('minified builds', () => {
    const coreSource = readFileSync(BUILT_BUNDLE, 'utf8');
    const minifiedBuilds = [
        ['ES', BUILT_MINIFIED_ES_BUNDLE],
        ['UMD', BUILT_MINIFIED_UMD_BUNDLE],
    ] as const;

    describe.each(minifiedBuilds)('the core %s build', (_label, file) => {
        const minifiedSource = readFileSync(file, 'utf8');

        it('reads the module protocol properties by their real names', () => {
            for (const name of MODULE_PROTOCOL_PROPERTIES) {
                expect(minifiedSource.includes(`.${name}`), `the build does not read .${name}`).toBe(true);
            }
        });

        it('keeps every name generated modules export', () => {
            const exported = underscoreNamesGeneratedModulesExport();

            expect(exported.has('__reinit__'), 'no golden exports __reinit__').toBe(true);
            expect(namesRenamedByMinifier(coreSource, minifiedSource, exported)).toEqual([]);
        });

        it('keeps the window properties other code relies on', () => {
            expect(namesRenamedByMinifier(coreSource, minifiedSource, CORE_WINDOW_PROPERTIES)).toEqual([]);
        });

        it('calls the component element methods by their real names', () => {
            for (const name of COMPONENT_ELEMENT_PROTOCOL_METHODS) {
                expect(usesName(coreSource, name), `the core runtime does not call ${name}`).toBe(true);
            }
            expect(namesRenamedByMinifier(coreSource, minifiedSource, COMPONENT_ELEMENT_PROTOCOL_METHODS)).toEqual([]);
        });

        it('keeps the JSON names the server sends and reads', () => {
            const serverNames = underscoreJSONNamesServerUses();

            expect(serverNames.has('_helpers'), 'no server type uses _helpers').toBe(true);
            expect(namesRenamedByMinifier(coreSource, minifiedSource, serverNames)).toEqual([]);
        });

        it('keeps the properties compiled components share with the runtime', () => {
            const names = [...underscorePropertiesCompiledComponentsUse(), ...SHARED_COMPONENT_PROPERTIES];

            expect(namesRenamedByMinifier(coreSource, minifiedSource, names)).toEqual([]);
        });
    });

    describe('the component runtime build', () => {
        const componentsSource = readFileSync(BUILT_COMPONENTS_BUNDLE, 'utf8');
        const minifiedSource = readFileSync(BUILT_MINIFIED_COMPONENTS_BUNDLE, 'utf8');

        it('keeps every property compiled components use', () => {
            const names = underscorePropertiesCompiledComponentsUse();

            expect(names.has('_updateFormState'), 'no compiled golden uses _updateFormState').toBe(true);
            expect(namesRenamedByMinifier(componentsSource, minifiedSource, names)).toEqual([]);
        });

        it('keeps the window properties other bundles set', () => {
            expect(namesRenamedByMinifier(componentsSource, minifiedSource, COMPONENTS_WINDOW_PROPERTIES)).toEqual([]);
        });

        it('keeps the element methods the core runtime calls', () => {
            for (const name of COMPONENT_ELEMENT_PROTOCOL_METHODS) {
                expect(usesName(componentsSource, name), `the component runtime does not define ${name}`).toBe(true);
            }
            expect(namesRenamedByMinifier(componentsSource, minifiedSource, COMPONENT_ELEMENT_PROTOCOL_METHODS)).toEqual([]);
        });

        it('keeps the JSON names the server sends and reads', () => {
            expect(namesRenamedByMinifier(componentsSource, minifiedSource, underscoreJSONNamesServerUses())).toEqual([]);
        });
    });
});

describe('minified builds drop the quiet console calls', () => {
    const minifiedFiles = [
        BUILT_MINIFIED_ES_BUNDLE,
        BUILT_MINIFIED_UMD_BUNDLE,
        BUILT_MINIFIED_COMPONENTS_BUNDLE,
        ...EXTENSION_NAMES.flatMap(extension => [
            builtExtensionFile(extension, 'min.es.js'),
            builtExtensionFile(extension, 'min.js'),
        ]),
    ].map(file => [basename(file), file] as const);

    it.each(minifiedFiles)('%s calls no console method but console.error', (_name, file) => {
        expect(droppedConsoleCallsLeft(readFileSync(file, 'utf8'))).toEqual([]);
    });

    it.each(EXTENSION_NAMES)('the %s build keeps its console.error calls', extension => {
        const unminifiedSource = readFileSync(builtExtensionFile(extension, 'es.js'), 'utf8');
        if (!unminifiedSource.includes('console.error(')) {
            return;
        }
        for (const suffix of ['min.es.js', 'min.js']) {
            expect(readFileSync(builtExtensionFile(extension, suffix), 'utf8')).toContain('console.error(');
        }
    });
});

describe('minified builds compare with true and false strictly', () => {
    const looseBooleanComparison = /[^-=!0-9.][01]==[A-Za-z_$]|[A-Za-z_$)\]]==[01](?![0-9.])/g;
    const minifiedFiles = [
        BUILT_MINIFIED_ES_BUNDLE,
        BUILT_MINIFIED_UMD_BUNDLE,
        BUILT_MINIFIED_COMPONENTS_BUNDLE,
        ...EXTENSION_NAMES.flatMap(extension => [
            builtExtensionFile(extension, 'min.es.js'),
            builtExtensionFile(extension, 'min.js'),
        ]),
    ].map(file => [basename(file), file] as const);

    it.each(minifiedFiles)('%s never compares a value loosely with 0 or 1', (_name, file) => {
        expect(readFileSync(file, 'utf8').match(looseBooleanComparison) ?? []).toEqual([]);
    });
});

describe.each(EXTENSION_NAMES)('the %s extension build', extension => {
    const unminifiedSource = readFileSync(builtExtensionFile(extension, 'es.js'), 'utf8');
    const minifiedSources = ['min.es.js', 'min.js'].map(suffix => readFileSync(builtExtensionFile(extension, suffix), 'utf8'));

    it('has the same exports in its minified ES build', () => {
        const minifiedExports = minifiedExportedNames(readFileSync(builtExtensionFile(extension, 'min.es.js'), 'utf8'));

        expect(sorted(minifiedExports)).toEqual(sorted(exportedNames(unminifiedSource)));
    });

    it('keeps the window properties other code relies on', () => {
        for (const minifiedSource of minifiedSources) {
            expect(namesRenamedByMinifier(unminifiedSource, minifiedSource, EXTENSION_WINDOW_PROPERTIES)).toEqual([]);
        }
    });
});
