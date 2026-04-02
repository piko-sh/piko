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

import {describe, it, expect, beforeEach, vi} from 'vitest';
import {createPageContext, resetGlobalPageContext, getGlobalPageContext, findClosestMatch} from './PageContext';

describe('PageContext', () => {
    beforeEach(() => {
        resetGlobalPageContext();
    });

    describe('createPageContext', () => {
        it('should create a page context with empty exports', () => {
            const ctx = createPageContext();
            expect(ctx.getExportedFunctions()).toEqual([]);
        });

        it('should return undefined for non-existent functions', () => {
            const ctx = createPageContext();
            expect(ctx.getFunction('nonExistent')).toBeUndefined();
        });

        it('should report hasFunction as false for non-existent functions', () => {
            const ctx = createPageContext();
            expect(ctx.hasFunction('nonExistent')).toBe(false);
        });
    });

    describe('setExports', () => {
        it('should register functions that can be retrieved', () => {
            const ctx = createPageContext();
            const testFn = () => 'test';
            ctx.setExports({testFn});

            expect(ctx.hasFunction('testFn')).toBe(true);
            expect(ctx.getFunction('testFn')).toBe(testFn);
        });

        it('should merge exports instead of replacing them', () => {
            const ctx = createPageContext();
            const funcA = () => 'a';
            const funcB = () => 'b';

            ctx.setExports({funcA});
            ctx.setExports({funcB});

            expect(ctx.hasFunction('funcA')).toBe(true);
            expect(ctx.hasFunction('funcB')).toBe(true);
            expect(ctx.getFunction('funcA')).toBe(funcA);
            expect(ctx.getFunction('funcB')).toBe(funcB);
        });

        it('should allow overwriting existing functions with same name', () => {
            const ctx = createPageContext();
            const originalFn = () => 'original';
            const newFn = () => 'new';

            ctx.setExports({myFunc: originalFn});
            ctx.setExports({myFunc: newFn});

            expect(ctx.getFunction('myFunc')).toBe(newFn);
        });

        it('should handle multiple partials registering their exports', () => {
            const ctx = createPageContext();

            const partialAFunc = () => 'partialA';
            ctx.setExports({partialAFunc});

            const partialBFunc = () => 'partialB';
            ctx.setExports({partialBFunc});

            const partialCFunc = () => 'partialC';
            ctx.setExports({partialCFunc});

            expect(ctx.hasFunction('partialAFunc')).toBe(true);
            expect(ctx.hasFunction('partialBFunc')).toBe(true);
            expect(ctx.hasFunction('partialCFunc')).toBe(true);
        });

        it('should not include non-function exports in getExportedFunctions', () => {
            const ctx = createPageContext();
            ctx.setExports({
                myFunc: () => 'function',
                myString: 'string',
                myNumber: 42,
                myObject: {key: 'value'}
            });

            expect(ctx.getExportedFunctions()).toEqual(['myFunc']);
        });
    });

    describe('clear', () => {
        it('should remove all exports', () => {
            const ctx = createPageContext();
            ctx.setExports({funcA: () => 'a', funcB: () => 'b'});

            ctx.clear();

            expect(ctx.hasFunction('funcA')).toBe(false);
            expect(ctx.hasFunction('funcB')).toBe(false);
            expect(ctx.getExportedFunctions()).toEqual([]);
        });
    });

    describe('getGlobalPageContext', () => {
        it('should return the same instance on multiple calls', () => {
            const ctx1 = getGlobalPageContext();
            const ctx2 = getGlobalPageContext();
            expect(ctx1).toBe(ctx2);
        });

        it('should preserve exports across calls', () => {
            const ctx1 = getGlobalPageContext();
            ctx1.setExports({testFn: () => 'test'});

            const ctx2 = getGlobalPageContext();
            expect(ctx2.hasFunction('testFn')).toBe(true);
        });
    });

    describe('resetGlobalPageContext', () => {
        it('should create a new instance after reset', () => {
            const ctx1 = getGlobalPageContext();
            ctx1.setExports({testFn: () => 'test'});

            resetGlobalPageContext();

            const ctx2 = getGlobalPageContext();
            expect(ctx2.hasFunction('testFn')).toBe(false);
        });
    });

    describe('island architecture scenarios', () => {
        it('should handle partial reload without losing other partial exports', () => {
            const ctx = getGlobalPageContext();

            ctx.setExports({mainPageFunc: () => 'main'});
            ctx.setExports({partialAFunc: () => 'partialA'});
            ctx.setExports({partialBFunc: () => 'partialB'});

            const newPartialAFunc = () => 'partialA-reloaded';
            ctx.setExports({partialAFunc: newPartialAFunc});

            expect(ctx.hasFunction('mainPageFunc')).toBe(true);
            expect(ctx.hasFunction('partialAFunc')).toBe(true);
            expect(ctx.hasFunction('partialBFunc')).toBe(true);

            expect(ctx.getFunction('partialAFunc')).toBe(newPartialAFunc);
        });

        it('should handle modal navigation step1 -> step2 -> step3 -> step2', () => {
            const ctx = getGlobalPageContext();

            const step1Func = () => 'step1';
            ctx.setExports({step1Func});

            const selectField = () => 'selectField';
            ctx.setExports({selectField});

            const goBack = () => 'goBack';
            const submitField = () => 'submitField';
            ctx.setExports({goBack, submitField});

            ctx.setExports({selectField});

            expect(ctx.hasFunction('step1Func')).toBe(true);
            expect(ctx.hasFunction('selectField')).toBe(true);
            expect(ctx.hasFunction('goBack')).toBe(true);
            expect(ctx.hasFunction('submitField')).toBe(true);
        });

        it('should handle same partial appearing twice on page', () => {
            const ctx = getGlobalPageContext();

            const sharedFunc = () => 'shared';
            ctx.setExports({sharedFunc});

            ctx.setExports({sharedFunc});

            expect(ctx.hasFunction('sharedFunc')).toBe(true);
            expect(ctx.getFunction('sharedFunc')).toBe(sharedFunc);
        });
    });

    describe('findClosestMatch', () => {
        it('should find exact matches', () => {
            const result = findClosestMatch('handleClick', ['handleClick', 'handleSubmit']);
            expect(result).toBe('handleClick');
        });

        it('should find close matches within threshold', () => {
            const result = findClosestMatch('handleClck', ['handleClick', 'handleSubmit']);
            expect(result).toBe('handleClick');
        });

        it('should return undefined for no close matches', () => {
            const result = findClosestMatch('xyz', ['handleClick', 'handleSubmit']);
            expect(result).toBeUndefined();
        });

        it('should return undefined for empty candidates', () => {
            const result = findClosestMatch('handleClick', []);
            expect(result).toBeUndefined();
        });

        it('should be case insensitive', () => {
            const result = findClosestMatch('HandleClick', ['handleclick', 'handleSubmit']);
            expect(result).toBe('handleclick');
        });
    });

    describe('findClosestMatch (extended)', () => {
        it('should pick the closest match among multiple close candidates', () => {
            const result = findClosestMatch('handleClck', ['handleClick', 'handleCluck', 'handleClock']);
            expect(result).toBe('handleClick');
        });

        it('should respect a custom threshold', () => {
            const result = findClosestMatch('abc', ['xyz'], 2);
            expect(result).toBeUndefined();
        });

        it('should return a match when distance equals the threshold exactly', () => {
            const result = findClosestMatch('abc', ['axc'], 1);
            expect(result).toBe('axc');
        });

        it('should handle single-character strings', () => {
            const result = findClosestMatch('a', ['b', 'c', 'a']);
            expect(result).toBe('a');
        });

        it('should handle one empty string as target', () => {
            const result = findClosestMatch('', ['ab', 'abc', 'abcd']);
            expect(result).toBe('ab');
        });

        it('should handle empty string candidates', () => {
            const result = findClosestMatch('ab', ['', 'abcdef']);
            expect(result).toBe('');
        });

        it('should return undefined when all candidates exceed threshold', () => {
            const result = findClosestMatch('hello', ['completelyDifferent', 'somethingElse'], 1);
            expect(result).toBeUndefined();
        });

        it('should prefer the first candidate found at the same distance', () => {
            const result = findClosestMatch('ab', ['ac', 'ad'], 3);
            expect(result).toBe('ac');
        });
    });

    describe('loadModule', () => {
        it('should call onError callback when import fails', async () => {
            const onError = vi.fn();
            const ctx = createPageContext({onError});

            await ctx.loadModule('http://nonexistent.invalid/module.js');

            expect(onError).toHaveBeenCalledOnce();
            expect(onError.mock.calls[0][0]).toBeInstanceOf(Error);
            expect(onError.mock.calls[0][1]).toBe('loadModule(http://nonexistent.invalid/module.js)');
        });

        it('should log to console.error when import fails and no onError is provided', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const ctx = createPageContext();

            await ctx.loadModule('http://nonexistent.invalid/module.js');

            expect(consoleSpy).toHaveBeenCalledOnce();
            expect(consoleSpy.mock.calls[0][0]).toBe('[PageContext] Failed to load module:');
            expect(consoleSpy.mock.calls[0][1]).toHaveProperty('url', 'http://nonexistent.invalid/module.js');
            consoleSpy.mockRestore();
        });

        it('should not call onModuleLoaded when import fails', async () => {
            const onModuleLoaded = vi.fn();
            const onError = vi.fn();
            const ctx = createPageContext({onError, onModuleLoaded});

            await ctx.loadModule('http://nonexistent.invalid/module.js');

            expect(onModuleLoaded).not.toHaveBeenCalled();
        });

        it('should load module into global scope when no partialName is given', async () => {
            const onModuleLoaded = vi.fn();
            const ctx = createPageContext({onModuleLoaded});

            await ctx.loadModule('./__fixtures__/test-module-no-id.ts');

            expect(ctx.hasFunction('noIdFunc')).toBe(true);
            expect(ctx.getFunction('noIdFunc')).toBeTypeOf('function');
            expect(onModuleLoaded).toHaveBeenCalledOnce();
            expect(onModuleLoaded.mock.calls[0][0]).toBe('./__fixtures__/test-module-no-id.ts');
            expect(onModuleLoaded.mock.calls[0][1]).toContain('noIdFunc');
        });

        it('should load module into partial scope using __PARTIAL_ID__', async () => {
            const onModuleLoaded = vi.fn();
            const ctx = createPageContext({onModuleLoaded});

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.getScopedFunction('fixtureFunc', 'fixture-partial-hash')).toBeTypeOf('function');
            expect(ctx.getRegisteredPartialNames()).toContain('my-partial');
            expect(onModuleLoaded).toHaveBeenCalledOnce();
        });

        it('should use partialName as fallback when module has no __PARTIAL_ID__', async () => {
            const onError = vi.fn();
            const ctxWithErr = createPageContext({onError});

            await ctxWithErr.loadModule('./__fixtures__/test-module-no-id.ts', 'fallback-name');

            expect(onError).not.toHaveBeenCalled();
            expect(ctxWithErr.getScopedFunction('noIdFunc', 'fallback-name')).toBeTypeOf('function');
            expect(ctxWithErr.getRegisteredPartialNames()).toContain('fallback-name');
        });

        it('should merge scoped exports when loadModule is called twice for the same partial', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');
            await ctx.loadModule('./__fixtures__/test-module-no-id.ts', 'my-partial');

            expect(ctx.getScopedFunction('fixtureFunc', 'fixture-partial-hash')).toBeTypeOf('function');
            expect(ctx.getScopedFunction('noIdFunc', 'my-partial')).toBeTypeOf('function');
        });

        it('should merge global exports when loadModule is called multiple times without partialName', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts');
            await ctx.loadModule('./__fixtures__/test-module-no-id.ts');

            expect(ctx.hasFunction('fixtureFunc')).toBe(true);
            expect(ctx.hasFunction('noIdFunc')).toBe(true);
        });
    });

    describe('getScopedFunction', () => {
        it('should return undefined when no scoped exports exist for the given partial ID', () => {
            const ctx = createPageContext();
            expect(ctx.getScopedFunction('myFn', 'nonexistent-id')).toBeUndefined();
        });

        it('should return a scoped function after loadModule populates the scope', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            const scopedFunction = ctx.getScopedFunction('fixtureFunc', 'fixture-partial-hash');
            expect(scopedFunction).toBeTypeOf('function');
        });

        it('should return undefined when the scope exists but the function name is wrong', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.getScopedFunction('nonExistentFunc', 'fixture-partial-hash')).toBeUndefined();
        });

        it('should return undefined for a non-function export in the scope', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.getScopedFunction('nonFunctionExport', 'fixture-partial-hash')).toBeUndefined();
        });
    });

    describe('getFunctionsByPartialName', () => {
        it('should return an empty array when the partial name is not registered', () => {
            const ctx = createPageContext();
            const fns = ctx.getFunctionsByPartialName('unknown-partial', 'someFunc');
            expect(fns).toEqual([]);
        });

        it('should return an empty array when partial is registered but has no scoped exports', () => {
            const ctx = createPageContext();
            ctx.registerPartialInstance('my-partial', 'id-1');
            const fns = ctx.getFunctionsByPartialName('my-partial', 'someFunc');
            expect(fns).toEqual([]);
        });

        it('should return matching functions from all instances of a partial', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            await ctx.loadModule('./__fixtures__/test-module-no-id.ts', 'my-partial');
            const fixtureFns = ctx.getFunctionsByPartialName('my-partial', 'fixtureFunc');
            expect(fixtureFns).toHaveLength(1);
            expect(fixtureFns[0]).toBeTypeOf('function');

            const noIdFns = ctx.getFunctionsByPartialName('my-partial', 'noIdFunc');
            expect(noIdFns).toHaveLength(1);
            expect(noIdFns[0]).toBeTypeOf('function');
        });

        it('should filter out non-function values from results', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            const fns = ctx.getFunctionsByPartialName('my-partial', 'nonFunctionExport');
            expect(fns).toEqual([]);
        });

        it('should return empty array for a function name that does not exist in any instance', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            const fns = ctx.getFunctionsByPartialName('my-partial', 'totallyFakeFunction');
            expect(fns).toEqual([]);
        });
    });

    describe('registerPartialInstance', () => {
        it('should register a partial instance and make it discoverable', () => {
            const ctx = createPageContext();
            ctx.registerPartialInstance('my-partial', 'hash-abc');
            expect(ctx.getRegisteredPartialNames()).toContain('my-partial');
        });

        it('should not duplicate the same partial ID when registered twice', () => {
            const ctx = createPageContext();
            ctx.registerPartialInstance('my-partial', 'hash-abc');
            ctx.registerPartialInstance('my-partial', 'hash-abc');
            expect(ctx.getRegisteredPartialNames()).toEqual(['my-partial']);
        });

        it('should allow multiple distinct IDs for the same partial name', () => {
            const ctx = createPageContext();
            ctx.registerPartialInstance('my-partial', 'hash-abc');
            ctx.registerPartialInstance('my-partial', 'hash-def');
            expect(ctx.getRegisteredPartialNames()).toEqual(['my-partial']);
        });

        it('should track multiple different partial names', () => {
            const ctx = createPageContext();
            ctx.registerPartialInstance('partial-a', 'id-1');
            ctx.registerPartialInstance('partial-b', 'id-2');
            expect(ctx.getRegisteredPartialNames()).toContain('partial-a');
            expect(ctx.getRegisteredPartialNames()).toContain('partial-b');
        });

        it('should return empty array when no partials are registered', () => {
            const ctx = createPageContext();
            expect(ctx.getRegisteredPartialNames()).toEqual([]);
        });
    });

    describe('clear (extended)', () => {
        it('should clear registered partial names', () => {
            const ctx = createPageContext();
            ctx.registerPartialInstance('my-partial', 'hash-abc');
            expect(ctx.getRegisteredPartialNames()).toContain('my-partial');

            ctx.clear();

            expect(ctx.getRegisteredPartialNames()).toEqual([]);
        });

        it('should clear scoped function lookups after loadModule', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');
            expect(ctx.getScopedFunction('fixtureFunc', 'fixture-partial-hash')).toBeTypeOf('function');

            ctx.clear();

            expect(ctx.getScopedFunction('fixtureFunc', 'fixture-partial-hash')).toBeUndefined();
            expect(ctx.getFunctionsByPartialName('my-partial', 'fixtureFunc')).toEqual([]);
            expect(ctx.getRegisteredPartialNames()).toEqual([]);
        });
    });

    describe('getFunction with scoped exports', () => {
        it('should find a function in scoped exports when not in global', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.getFunction('fixtureFunc')).toBeTypeOf('function');
        });

        it('should prefer global function over scoped function of the same name', async () => {
            const ctx = createPageContext();
            const globalFn = () => 'global-version';
            ctx.setExports({fixtureFunc: globalFn});

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.getFunction('fixtureFunc')).toBe(globalFn);
        });

        it('should return undefined when function exists in neither global nor scoped', () => {
            const ctx = createPageContext();
            expect(ctx.getFunction('completelyUnknown')).toBeUndefined();
        });
    });

    describe('hasFunction with scoped exports', () => {
        it('should return true for a function only in scoped exports', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.hasFunction('fixtureFunc')).toBe(true);
        });

        it('should return false for a non-function value in scoped exports', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            expect(ctx.hasFunction('nonFunctionExport')).toBe(false);
        });
    });

    describe('getExportedFunctions with scoped exports', () => {
        it('should include function names from both global and scoped exports', async () => {
            const ctx = createPageContext();
            ctx.setExports({globalFunc: () => 'global'});

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            const exported = ctx.getExportedFunctions();
            expect(exported).toContain('globalFunc');
            expect(exported).toContain('fixtureFunc');
        });

        it('should deduplicate function names present in both global and scoped', async () => {
            const ctx = createPageContext();
            ctx.setExports({fixtureFunc: () => 'global-version'});

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            const exported = ctx.getExportedFunctions();
            const fixtureCount = exported.filter(n => n === 'fixtureFunc').length;
            expect(fixtureCount).toBe(1);
        });

        it('should not include non-function exports from scoped modules', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module.ts', 'my-partial');

            const exported = ctx.getExportedFunctions();
            expect(exported).not.toContain('nonFunctionExport');
            expect(exported).not.toContain('__PARTIAL_ID__');
        });

        it('should return only function names, not non-function values from global exports', () => {
            const ctx = createPageContext();
            ctx.setExports({
                aFunc: () => 'a',
                aString: 'hello',
                aNumber: 123,
                bFunc: () => 'b',
                anObject: {key: 'value'},
                aNull: null,
                anUndefined: undefined,
                aBool: true
            });

            const exported = ctx.getExportedFunctions();
            expect(exported).toContain('aFunc');
            expect(exported).toContain('bFunc');
            expect(exported).toHaveLength(2);
        });
    });

    describe('__reinit__ support', () => {
        it('should call __reinit__ on module after import', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module-reinit.ts');

            const getCount = ctx.getFunction('getReinitCallCount') as (() => number) | undefined;
            expect(getCount).toBeTypeOf('function');
            expect(getCount!()).toBe(1);
        });

        it('should call __reinit__ on each loadModule invocation', async () => {
            const ctx = createPageContext();

            await ctx.loadModule('./__fixtures__/test-module-reinit.ts');
            const getCount = ctx.getFunction('getReinitCallCount') as (() => number) | undefined;
            expect(getCount).toBeTypeOf('function');
            const countAfterFirst = getCount!();

            await ctx.loadModule('./__fixtures__/test-module-reinit.ts');
            const countAfterSecond = getCount!();

            expect(countAfterSecond - countAfterFirst).toBe(1);
        });

        it('should not fail when module has no __reinit__', async () => {
            const ctx = createPageContext();

            await expect(ctx.loadModule('./__fixtures__/test-module-no-id.ts')).resolves.not.toThrow();
            expect(ctx.hasFunction('noIdFunc')).toBe(true);
        });
    });

    describe('hasFunction edge cases', () => {
        it('should return false for a non-function value stored in exports', () => {
            const ctx = createPageContext();
            ctx.setExports({notAFunction: 'string-value'});
            expect(ctx.hasFunction('notAFunction')).toBe(false);
        });

        it('should return true after setting and false after clearing', () => {
            const ctx = createPageContext();
            const testFunction = () => 'test';
            ctx.setExports({fn: testFunction});
            expect(ctx.hasFunction('fn')).toBe(true);
            ctx.clear();
            expect(ctx.hasFunction('fn')).toBe(false);
        });
    });
});
