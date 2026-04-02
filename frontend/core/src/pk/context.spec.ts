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
import { _createPKContext } from '@/pk/context';
import {
    _executeConnected,
    _executeDisconnected,
    _executeBeforeRender,
    _executeAfterRender,
    _executeUpdated,
    _disconnectCleanupObserver,
    _runPageCleanup,
    _hasLifecycleCallbacks,
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
