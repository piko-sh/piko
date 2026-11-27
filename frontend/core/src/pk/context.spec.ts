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

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { _createPKContext, _createPagePKContext } from '@/pk/context';
import {
    _executeConnected,
    _executeDisconnected,
    _executeBeforeRender,
    _executeAfterRender,
    _executeUpdated,
    _disconnectCleanupObserver,
    _runPageCleanup,
    _hasLifecycleCallbacks,
    _endPageVisit,
    _currentPageVisit,
} from '@/pk/lifecycle';

describe('_createPKContext()', () => {
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
    });

    afterEach(() => {
        _disconnectCleanupObserver();
        _runPageCleanup();
        testContainer.remove();
        vi.clearAllMocks();
    });

    it('should return an object with refs, lifecycle hooks, and cleanup', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-1');

        const pk = _createPKContext(scope);

        expect(pk).toHaveProperty('refs');
        expect(pk).toHaveProperty('createRefs');
        expect(pk).toHaveProperty('onConnected');
        expect(pk).toHaveProperty('onDisconnected');
        expect(pk).toHaveProperty('onBeforeRender');
        expect(pk).toHaveProperty('onAfterRender');
        expect(pk).toHaveProperty('onUpdated');
        expect(pk).toHaveProperty('onCleanup');
    });

    it('should register lifecycle state on the scope element', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-2');

        const pk = _createPKContext(scope);
        pk.onConnected(() => {});

        expect(_hasLifecycleCallbacks(scope)).toBe(true);
    });

    it('should fire onConnected callback when _executeConnected is called', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-3');

        const pk = _createPKContext(scope);
        const cb = vi.fn();
        pk.onConnected(cb);

        _executeConnected(scope);

        expect(cb).toHaveBeenCalledTimes(1);
    });

    it('should fire onDisconnected callback when _executeDisconnected is called', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-4');

        const pk = _createPKContext(scope);
        const cb = vi.fn();
        pk.onDisconnected(cb);

        _executeDisconnected(scope);

        expect(cb).toHaveBeenCalledTimes(1);
    });

    it('should fire onBeforeRender callback when _executeBeforeRender is called', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-5');

        const pk = _createPKContext(scope);
        const cb = vi.fn();
        pk.onBeforeRender(cb);

        _executeBeforeRender(scope);

        expect(cb).toHaveBeenCalledTimes(1);
    });

    it('should fire onAfterRender callback when _executeAfterRender is called', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-6');

        const pk = _createPKContext(scope);
        const cb = vi.fn();
        pk.onAfterRender(cb);

        _executeAfterRender(scope);

        expect(cb).toHaveBeenCalledTimes(1);
    });

    it('should fire onUpdated callback with context when _executeUpdated is called', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-7');

        const pk = _createPKContext(scope);
        const cb = vi.fn();
        pk.onUpdated(cb);

        const context = { reason: 'test' };
        _executeUpdated(scope, context);

        expect(cb).toHaveBeenCalledWith(context);
    });

    it('should allow multiple registrations of the same hook', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-8');
        const order: number[] = [];

        const pk = _createPKContext(scope);
        pk.onConnected(() => order.push(1));
        pk.onConnected(() => order.push(2));
        pk.onConnected(() => order.push(3));

        _executeConnected(scope);

        expect(order).toEqual([1, 2, 3]);
    });

    it('should provide refs proxy scoped to the scope element', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-9');
        const child = document.createElement('span');
        child.setAttribute('p-ref', 'mySpan');
        scope.appendChild(child);

        const pk = _createPKContext(scope);

        expect(pk.refs.mySpan).toBe(child);
        expect(pk.refs.nonexistent).toBeNull();
    });

    it('should provide createRefs that defaults to the scope element', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-10');
        const child = document.createElement('span');
        child.setAttribute('p-ref', 'mySpan');
        scope.appendChild(child);

        const pk = _createPKContext(scope);
        const refs = pk.createRefs();

        expect(refs.mySpan).toBe(child);
    });

    it('should provide createRefs that accepts a custom scope', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-11');

        const inner = document.createElement('div');
        const innerChild = document.createElement('span');
        innerChild.setAttribute('p-ref', 'innerRef');
        inner.appendChild(innerChild);
        scope.appendChild(inner);

        const pk = _createPKContext(scope);
        const refs = pk.createRefs(inner);

        expect(refs.innerRef).toBe(innerChild);
    });

    it('should register cleanup scoped to the scope element', () => {
        const scope = document.createElement('div');
        scope.setAttribute('partial', 'ctx-12');

        const pk = _createPKContext(scope);
        const cb = vi.fn();
        pk.onConnected(() => {});
        pk.onCleanup(cb);

        _executeConnected(scope);
        _executeDisconnected(scope);

    });
});

describe('_createPagePKContext()', () => {
    let pageRoot: HTMLDivElement;

    beforeEach(() => {
        pageRoot = document.createElement('div');
        pageRoot.setAttribute('data-pageid', 'pages/main');
        pageRoot.innerHTML = '<input p-ref="pageInput"><div partial="card_1" partial_name="card"><input p-ref="cardInput"></div>';
        document.body.appendChild(pageRoot);
    });

    afterEach(() => {
        _endPageVisit();
        pageRoot.remove();
        vi.clearAllMocks();
    });

    it('resolves refs against the page root, including those inside partials', () => {
        const pk = _createPagePKContext(pageRoot);

        expect(pk.refs.pageInput).toBe(pageRoot.querySelector('[p-ref="pageInput"]'));
        expect(pk.refs.cardInput).toBe(pageRoot.querySelector('[p-ref="cardInput"]'));
        expect(pk.createRefs(pageRoot.querySelector('[partial]')!).cardInput).toBe(pageRoot.querySelector('[p-ref="cardInput"]'));
    });

    it('fires onConnected straight away when the page root is in the DOM', () => {
        const connected = vi.fn();

        _createPagePKContext(pageRoot).onConnected(connected);

        expect(connected).toHaveBeenCalledTimes(1);
    });

    it('fires onDisconnected and its cleanups once when the page visit ends', () => {
        const pk = _createPagePKContext(pageRoot);
        const disconnected = vi.fn();
        const cleanup = vi.fn();
        pk.onConnected(() => pk.onCleanup(cleanup));
        pk.onDisconnected(disconnected);

        _endPageVisit();
        _endPageVisit();

        expect(disconnected).toHaveBeenCalledTimes(1);
        expect(cleanup).toHaveBeenCalledTimes(1);
    });

    it('disconnects the first visit and connects the next one on the same page root', () => {
        const firstConnected = vi.fn();
        const firstDisconnected = vi.fn();
        const first = _createPagePKContext(pageRoot);
        first.onConnected(firstConnected);
        first.onDisconnected(firstDisconnected);

        _endPageVisit();
        const secondConnected = vi.fn();
        _createPagePKContext(pageRoot).onConnected(secondConnected);

        expect(firstConnected).toHaveBeenCalledTimes(1);
        expect(firstDisconnected).toHaveBeenCalledTimes(1);
        expect(secondConnected).toHaveBeenCalledTimes(1);
    });

    it('ignores callbacks registered after its page visit ended', () => {
        const pk = _createPagePKContext(pageRoot);
        _endPageVisit();
        const late = vi.fn();

        pk.onConnected(late);
        _createPagePKContext(pageRoot).onConnected(vi.fn());

        expect(late).not.toHaveBeenCalled();
    });

    it('connects when created for the page visit shown now', () => {
        const connected = vi.fn();

        _createPagePKContext(pageRoot, _currentPageVisit()).onConnected(connected);

        expect(connected).toHaveBeenCalledTimes(1);
    });

    it('never connects when created for a page visit that has ended', () => {
        const endedVisit = _currentPageVisit();
        _endPageVisit();
        const connected = vi.fn();
        const disconnected = vi.fn();
        const cleanup = vi.fn();

        const pk = _createPagePKContext(pageRoot, endedVisit);
        pk.onConnected(connected);
        pk.onDisconnected(disconnected);
        pk.onCleanup(cleanup);
        _endPageVisit();

        expect(connected).not.toHaveBeenCalled();
        expect(disconnected).not.toHaveBeenCalled();
        expect(cleanup).toHaveBeenCalledTimes(1);
    });

    it('does not share lifecycle state with a partial scoped to the same element', () => {
        const partialConnected = vi.fn();
        const partialDisconnected = vi.fn();
        const partial = _createPKContext(pageRoot);
        partial.onConnected(partialConnected);
        partial.onDisconnected(partialDisconnected);

        const pageConnected = vi.fn();
        _createPagePKContext(pageRoot).onConnected(pageConnected);
        _endPageVisit();

        expect(partialConnected).toHaveBeenCalledTimes(1);
        expect(pageConnected).toHaveBeenCalledTimes(1);
        expect(partialDisconnected).not.toHaveBeenCalled();
    });

    it('runs its render hooks when a partial on the page re-renders', () => {
        const partial = pageRoot.querySelector('[partial]')!;
        const pk = _createPagePKContext(pageRoot);
        const before = vi.fn();
        const after = vi.fn();
        const updated = vi.fn();
        pk.onBeforeRender(before);
        pk.onAfterRender(after);
        pk.onUpdated(updated);

        _executeBeforeRender(partial);
        _executeAfterRender(partial);
        _executeUpdated(partial, {patchMethod: 'replace'});

        expect(before).toHaveBeenCalledWith();
        expect(after).toHaveBeenCalledWith();
        expect(updated).toHaveBeenCalledWith({patchMethod: 'replace'});
    });
});

describe('_createPagePKContext() in generated page scripts', () => {
    interface GeneratedPageScript {
        __reinit__(visit?: object): void;
        recordClick?(event: unknown): Promise<void>;
    }

    const loadScriptWithFunctions = async (): Promise<GeneratedPageScript> =>
        await import('@generator-goldens/181_pk_page_script_lifecycle/golden/pk-js/pages/main.js') as GeneratedPageScript;

    const loadScriptWithoutFunctions = async (): Promise<GeneratedPageScript> =>
        await import('@generator-goldens/182_pk_page_script_without_functions/golden/pk-js/pages/main.js') as GeneratedPageScript;

    let endPageVisit: () => void;
    let currentPageVisit: () => object;
    let content: HTMLDivElement;

    function pageLog(): string[] {
        return (window as unknown as { __pageScriptLog?: string[] }).__pageScriptLog ?? [];
    }

    function renderPage(): HTMLButtonElement {
        content.innerHTML = '<h1>Page script</h1><button p-ref="counter" type="button">Count</button>';
        return content.querySelector('button')!;
    }

    function navigateAwayAndBack(script: GeneratedPageScript): HTMLButtonElement {
        endPageVisit();
        const button = renderPage();
        script.__reinit__();
        return button;
    }

    beforeEach(async () => {
        vi.resetModules();
        delete (window as unknown as { __pageScriptLog?: string[] }).__pageScriptLog;
        document.body.setAttribute('data-pageid', 'pages/main');
        content = document.createElement('div');
        document.body.appendChild(content);

        const lifecycle = await import('@/pk/lifecycle');
        endPageVisit = lifecycle._endPageVisit;
        currentPageVisit = lifecycle._currentPageVisit;
    });

    afterEach(() => {
        endPageVisit();
        content.remove();
        document.body.removeAttribute('data-pageid');
    });

    describe('with functions', () => {
        it('does nothing until the loader re-initialises it, then connects once', async () => {
            renderPage();

            const script = await loadScriptWithFunctions();
            const logBeforeReinit = [...pageLog()];
            script.__reinit__();
            script.__reinit__();

            expect(logBeforeReinit).toEqual([]);
            expect(pageLog()).toEqual(['connected']);
        });

        it('disconnects when the page is left and binds the new page element on every visit', async () => {
            const firstButton = renderPage();
            const script = await loadScriptWithFunctions();
            script.__reinit__();
            firstButton.click();

            const secondButton = navigateAwayAndBack(script);
            firstButton.click();
            secondButton.click();

            expect(pageLog()).toEqual(['connected', 'click 1', 'disconnected', 'connected', 'click 1']);
        });

        it('sends exported handlers to the instance of the current visit', async () => {
            const firstButton = renderPage();
            const script = await loadScriptWithFunctions();
            script.__reinit__();
            await script.recordClick!({currentTarget: firstButton});
            await script.recordClick!({currentTarget: firstButton});

            const secondButton = navigateAwayAndBack(script);
            await script.recordClick!({currentTarget: secondButton});

            expect(pageLog()).toEqual(['connected', 'click 1', 'click 2', 'disconnected', 'connected', 'click 1']);
        });

        it('keeps connecting on repeated visits without stacking listeners', async () => {
            let button = renderPage();
            const script = await loadScriptWithFunctions();
            script.__reinit__();
            for (let visit = 0; visit < 3; visit++) {
                button = navigateAwayAndBack(script);
            }
            button.click();

            expect(pageLog().filter(entry => entry === 'connected')).toHaveLength(4);
            expect(pageLog().filter(entry => entry === 'disconnected')).toHaveLength(3);
            expect(pageLog().filter(entry => entry.startsWith('click'))).toEqual(['click 1']);
        });

        it('does nothing when old code calls an exported function after the page is left', async () => {
            const firstButton = renderPage();
            const script = await loadScriptWithFunctions();
            script.__reinit__(currentPageVisit());

            endPageVisit();
            const nextPageButton = renderPage();
            const result = await script.recordClick!({currentTarget: firstButton});
            nextPageButton.click();

            expect(result).toBeUndefined();
            expect(pageLog()).toEqual(['connected', 'disconnected']);
        });

        it('does not connect when re-initialised for a page visit that has ended', async () => {
            renderPage();
            const script = await loadScriptWithFunctions();
            const endedVisit = currentPageVisit();

            endPageVisit();
            const nextPageButton = renderPage();
            script.__reinit__(endedVisit);
            nextPageButton.click();

            expect(pageLog()).toEqual([]);
        });

        it('connects when re-initialised for the page visit shown now', async () => {
            const button = renderPage();
            const script = await loadScriptWithFunctions();

            script.__reinit__(currentPageVisit());
            button.click();

            expect(pageLog()).toEqual(['connected', 'click 1']);
        });
    });

    describe('without functions', () => {
        it('connects when the loader re-initialises it and counts clicks on the page element', async () => {
            const button = renderPage();

            const script = await loadScriptWithoutFunctions();
            const logBeforeReinit = [...pageLog()];
            script.__reinit__();
            button.click();

            expect(logBeforeReinit).toEqual([]);
            expect(pageLog()).toEqual(['connected', 'click 1']);
        });

        it('disconnects when the page is left and connects again on the next visit', async () => {
            const firstButton = renderPage();
            const script = await loadScriptWithoutFunctions();
            script.__reinit__();

            const secondButton = navigateAwayAndBack(script);
            firstButton.click();
            secondButton.click();

            expect(pageLog()).toEqual(['connected', 'disconnected', 'connected', 'click 1']);
        });

        it('does not connect when re-initialised for a page visit that has ended', async () => {
            renderPage();
            const script = await loadScriptWithoutFunctions();
            const endedVisit = currentPageVisit();

            endPageVisit();
            const nextPageButton = renderPage();
            script.__reinit__(endedVisit);
            nextPageButton.click();

            expect(pageLog()).toEqual([]);
        });
    });
});

describe('_createPKContext() in generated partial scripts', () => {
    interface GeneratedPartialScript {
        recordClick(event: unknown): Promise<void>;
    }

    const loadCounterScript = async (): Promise<GeneratedPartialScript> =>
        await import('@generator-goldens/185_pk_partial_script_handler_in_slot_content/golden/pk-js/partials/counter.js') as GeneratedPartialScript;

    const counterMarkup =
        '<div partial_name="counter">' +
        '<button class="counter-direct" type="button">Direct</button>' +
        '<div partial_name="card"><button class="counter-slotted" type="button">Slotted</button></div>' +
        '</div>';

    let content: HTMLDivElement;

    function clickLog(): string[] {
        return (window as unknown as { __partialScriptLog?: string[] }).__partialScriptLog ?? [];
    }

    function counterButton(counterIndex: number, buttonClass: string): HTMLButtonElement {
        const counter = content.querySelectorAll('[partial_name="counter"]')[counterIndex];
        return counter.querySelector<HTMLButtonElement>('.' + buttonClass)!;
    }

    beforeEach(() => {
        vi.resetModules();
        delete (window as unknown as { __partialScriptLog?: string[] }).__partialScriptLog;
        document.body.setAttribute('data-pageid', 'pages/main');
        content = document.createElement('div');
        document.body.appendChild(content);
    });

    afterEach(() => {
        _disconnectCleanupObserver();
        _runPageCleanup();
        content.remove();
        document.body.removeAttribute('data-pageid');
    });

    it('runs a handler in the slot of another partial on the instance of its own partial', async () => {
        content.innerHTML = counterMarkup;
        const script = await loadCounterScript();

        await script.recordClick({currentTarget: counterButton(0, 'counter-direct')});
        await script.recordClick({currentTarget: counterButton(0, 'counter-slotted')});

        expect(clickLog()).toEqual(['click 1', 'click 2']);
    });

    it('keeps a separate instance for each root element of the partial', async () => {
        content.innerHTML = counterMarkup + counterMarkup;
        const script = await loadCounterScript();

        await script.recordClick({currentTarget: counterButton(0, 'counter-slotted')});
        await script.recordClick({currentTarget: counterButton(1, 'counter-slotted')});
        await script.recordClick({currentTarget: counterButton(1, 'counter-direct')});

        expect(clickLog()).toEqual(['click 1', 'click 1', 'click 2']);
    });
});
