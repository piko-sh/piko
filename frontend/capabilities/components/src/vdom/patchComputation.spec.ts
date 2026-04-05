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

import { describe, it, expect } from 'vitest';
import {
    isUndef,
    isVNodeHidden,
    areSameVNode,
    createKeyToOldIdxMap,
    getHiddenNodeType,
    createHiddenPlaceholderMessage,
    computePatchDecision,
    compareChildrenPointers
} from '@/vdom/patchComputation';
import type { VirtualNode } from '@/vdom/types';

function createVNode(overrides: Partial<VirtualNode> = {}): VirtualNode {
    return {
        _type: 'element',
        tag: 'div',
        ...overrides
    };
}

describe('patchComputation', () => {
    describe('isUndef()', () => {
        it('should return true for undefined', () => {
            expect(isUndef(undefined)).toBe(true);
        });

        it('should return true for null', () => {
            expect(isUndef(null)).toBe(true);
        });

        it('should return false for zero', () => {
            expect(isUndef(0)).toBe(false);
        });

        it('should return false for empty string', () => {
            expect(isUndef('')).toBe(false);
        });

        it('should return false for false', () => {
            expect(isUndef(false)).toBe(false);
        });

        it('should return false for objects', () => {
            expect(isUndef({})).toBe(false);
        });

        it('should return false for arrays', () => {
            expect(isUndef([])).toBe(false);
        });

        it('should return false for strings', () => {
            expect(isUndef('hello')).toBe(false);
        });

        it('should return false for numbers', () => {
            expect(isUndef(42)).toBe(false);
        });
    });

    describe('isVNodeHidden()', () => {
        it('should return true when _c prop is false', () => {
            const vnode = createVNode({ props: { _c: false } });
            expect(isVNodeHidden(vnode)).toBe(true);
        });

        it('should return false when _c prop is true', () => {
            const vnode = createVNode({ props: { _c: true } });
            expect(isVNodeHidden(vnode)).toBe(false);
        });

        it('should return false when _c prop is missing', () => {
            const vnode = createVNode({ props: { other: 'value' } });
            expect(isVNodeHidden(vnode)).toBe(false);
        });

        it('should return false when props is undefined', () => {
            const vnode = createVNode({ props: undefined });
            expect(isVNodeHidden(vnode)).toBe(false);
        });

        it('should return false when props is empty object', () => {
            const vnode = createVNode({ props: {} });
            expect(isVNodeHidden(vnode)).toBe(false);
        });

        it('should handle _c with truthy non-boolean values', () => {
            const vnode = createVNode({ props: { _c: 1 } });
            expect(isVNodeHidden(vnode)).toBe(false);
        });

        it('should handle _c with falsy non-boolean values', () => {
            const vnode = createVNode({ props: { _c: 0 } });
            expect(isVNodeHidden(vnode)).toBe(true);
        });

        it('should handle _c with empty string', () => {
            const vnode = createVNode({ props: { _c: '' } });
            expect(isVNodeHidden(vnode)).toBe(true);
        });
    });

    describe('areSameVNode()', () => {
        it('should return true for identical vnodes', () => {
            const vnode1 = createVNode({ key: 'a', _type: 'element', tag: 'div' });
            const vnode2 = createVNode({ key: 'a', _type: 'element', tag: 'div' });
            expect(areSameVNode(vnode1, vnode2)).toBe(true);
        });

        it('should return false when keys differ', () => {
            const vnode1 = createVNode({ key: 'a', _type: 'element', tag: 'div' });
            const vnode2 = createVNode({ key: 'b', _type: 'element', tag: 'div' });
            expect(areSameVNode(vnode1, vnode2)).toBe(false);
        });

        it('should return false when types differ', () => {
            const vnode1 = createVNode({ key: 'a', _type: 'element', tag: 'div' });
            const vnode2 = createVNode({ key: 'a', _type: 'text', tag: 'div' });
            expect(areSameVNode(vnode1, vnode2)).toBe(false);
        });

        it('should return false when tags differ', () => {
            const vnode1 = createVNode({ key: 'a', _type: 'element', tag: 'div' });
            const vnode2 = createVNode({ key: 'a', _type: 'element', tag: 'span' });
            expect(areSameVNode(vnode1, vnode2)).toBe(false);
        });

        it('should return true when both keys are undefined', () => {
            const vnode1 = createVNode({ _type: 'element', tag: 'div' });
            const vnode2 = createVNode({ _type: 'element', tag: 'div' });
            expect(areSameVNode(vnode1, vnode2)).toBe(true);
        });

        it('should return false when one key is undefined', () => {
            const vnode1 = createVNode({ key: 'a', _type: 'element', tag: 'div' });
            const vnode2 = createVNode({ _type: 'element', tag: 'div' });
            expect(areSameVNode(vnode1, vnode2)).toBe(false);
        });

        it('should handle text nodes', () => {
            const vnode1: VirtualNode = { _type: 'text', text: 'hello' };
            const vnode2: VirtualNode = { _type: 'text', text: 'world' };
            expect(areSameVNode(vnode1, vnode2)).toBe(true);
        });

        it('should handle fragment nodes', () => {
            const vnode1 = createVNode({ key: 'frag1', _type: 'fragment' });
            const vnode2 = createVNode({ key: 'frag1', _type: 'fragment' });
            expect(areSameVNode(vnode1, vnode2)).toBe(true);
        });

        it('should handle comment nodes', () => {
            const vnode1: VirtualNode = { _type: 'comment', key: 'c1', text: 'comment' };
            const vnode2: VirtualNode = { _type: 'comment', key: 'c1', text: 'different' };
            expect(areSameVNode(vnode1, vnode2)).toBe(true);
        });
    });

    describe('createKeyToOldIdxMap()', () => {
        it('should create map from keyed children', () => {
            const children: VirtualNode[] = [
                createVNode({ key: 'a' }),
                createVNode({ key: 'b' }),
                createVNode({ key: 'c' })
            ];

            const map = createKeyToOldIdxMap(children, 0, 2);

            expect(map.get('a')).toBe(0);
            expect(map.get('b')).toBe(1);
            expect(map.get('c')).toBe(2);
        });

        it('should only include specified range', () => {
            const children: VirtualNode[] = [
                createVNode({ key: 'a' }),
                createVNode({ key: 'b' }),
                createVNode({ key: 'c' }),
                createVNode({ key: 'd' })
            ];

            const map = createKeyToOldIdxMap(children, 1, 2);

            expect(map.has('a')).toBe(false);
            expect(map.get('b')).toBe(1);
            expect(map.get('c')).toBe(2);
            expect(map.has('d')).toBe(false);
        });

        it('should skip children without keys', () => {
            const children: VirtualNode[] = [
                createVNode({ key: 'a' }),
                createVNode({}),
                createVNode({ key: 'c' })
            ];

            const map = createKeyToOldIdxMap(children, 0, 2);

            expect(map.get('a')).toBe(0);
            expect(map.size).toBe(2);
            expect(map.get('c')).toBe(2);
        });

        it('should skip children with null keys', () => {
            const children: VirtualNode[] = [
                createVNode({ key: 'a' }),
                createVNode({ key: undefined }),
                createVNode({ key: 'c' })
            ];

            const map = createKeyToOldIdxMap(children, 0, 2);

            expect(map.size).toBe(2);
        });

        it('should handle empty range', () => {
            const children: VirtualNode[] = [
                createVNode({ key: 'a' })
            ];

            const map = createKeyToOldIdxMap(children, 0, -1);

            expect(map.size).toBe(0);
        });

        it('should handle single element range', () => {
            const children: VirtualNode[] = [
                createVNode({ key: 'a' }),
                createVNode({ key: 'b' }),
                createVNode({ key: 'c' })
            ];

            const map = createKeyToOldIdxMap(children, 1, 1);

            expect(map.size).toBe(1);
            expect(map.get('b')).toBe(1);
        });

        it('should handle sparse arrays', () => {
            const children: VirtualNode[] = [];
            children[0] = createVNode({ key: 'a' });
            children[2] = createVNode({ key: 'c' });

            const map = createKeyToOldIdxMap(children, 0, 2);

            expect(map.get('a')).toBe(0);
            expect(map.get('c')).toBe(2);
            expect(map.size).toBe(2);
        });
    });

    describe('getHiddenNodeType()', () => {
        it('should return "fragment" for fragment vnodes', () => {
            const vnode = createVNode({ _type: 'fragment' });
            expect(getHiddenNodeType(vnode)).toBe('fragment');
        });

        it('should return "node" for element vnodes', () => {
            const vnode = createVNode({ _type: 'element' });
            expect(getHiddenNodeType(vnode)).toBe('node');
        });

        it('should return "node" for text vnodes', () => {
            const vnode: VirtualNode = { _type: 'text', text: 'hello' };
            expect(getHiddenNodeType(vnode)).toBe('node');
        });

        it('should return "node" for comment vnodes', () => {
            const vnode: VirtualNode = { _type: 'comment', text: 'comment' };
            expect(getHiddenNodeType(vnode)).toBe('node');
        });
    });

    describe('createHiddenPlaceholderMessage()', () => {
        it('should create message with provided key', () => {
            const message = createHiddenPlaceholderMessage('node', 'myKey', 'fallback');
            expect(message).toBe('hidden node _k=myKey');
        });

        it('should create message with fallback when key is undefined', () => {
            const message = createHiddenPlaceholderMessage('node', undefined, 'item-0');
            expect(message).toBe('hidden node _k=err-item-0');
        });

        it('should create message for fragment type', () => {
            const message = createHiddenPlaceholderMessage('fragment', 'fragKey', 'prefix');
            expect(message).toBe('hidden fragment _k=fragKey');
        });

        it('should create fragment message with fallback', () => {
            const message = createHiddenPlaceholderMessage('fragment', undefined, 'frag-1');
            expect(message).toBe('hidden fragment _k=err-frag-1');
        });

        it('should handle empty string key', () => {
            const message = createHiddenPlaceholderMessage('node', '', 'fallback');
            expect(message).toBe('hidden node _k=');
        });
    });

    describe('computePatchDecision()', () => {
        describe('null/undefined handling', () => {
            it('should return "remove" when newVNode is null and oldVNode exists', () => {
                const oldVNode = createVNode({ key: 'a' });
                const result = computePatchDecision(oldVNode, null);
                expect(result).toEqual({ action: 'remove' });
            });

            it('should return "noop" when both are null', () => {
                const result = computePatchDecision(null, null);
                expect(result).toEqual({ action: 'noop' });
            });

            it('should return "create" when oldVNode is null', () => {
                const newVNode = createVNode({ key: 'a' });
                const result = computePatchDecision(null, newVNode);
                expect(result).toEqual({ action: 'create' });
            });
        });

        describe('same reference', () => {
            it('should return "noop" when old and new are same reference', () => {
                const vnode = createVNode({ key: 'a' });
                const result = computePatchDecision(vnode, vnode);
                expect(result).toEqual({ action: 'noop' });
            });
        });

        describe('different vnodes', () => {
            it('should return "replace" when keys differ', () => {
                const oldVNode = createVNode({ key: 'a', _type: 'element', tag: 'div' });
                const newVNode = createVNode({ key: 'b', _type: 'element', tag: 'div' });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'replace' });
            });

            it('should return "replace" when types differ', () => {
                const oldVNode = createVNode({ key: 'a', _type: 'element', tag: 'div' });
                const newVNode = createVNode({ key: 'a', _type: 'text' });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'replace' });
            });

            it('should return "replace" when tags differ', () => {
                const oldVNode = createVNode({ key: 'a', _type: 'element', tag: 'div' });
                const newVNode = createVNode({ key: 'a', _type: 'element', tag: 'span' });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'replace' });
            });
        });

        describe('hidden state transitions', () => {
            it('should return "update-hidden" when both are hidden', () => {
                const oldVNode = createVNode({ key: 'a', props: { _c: false } });
                const newVNode = createVNode({ key: 'a', props: { _c: false } });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update-hidden' });
            });

            it('should return "update-hidden" when becoming hidden', () => {
                const oldVNode = createVNode({ key: 'a', props: { _c: true } });
                const newVNode = createVNode({ key: 'a', props: { _c: false } });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update-hidden' });
            });

            it('should return "show-hidden" when becoming visible', () => {
                const oldVNode = createVNode({ key: 'a', props: { _c: false } });
                const newVNode = createVNode({ key: 'a', props: { _c: true } });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'show-hidden' });
            });

            it('should return "update" when both are visible', () => {
                const oldVNode = createVNode({ key: 'a', props: { _c: true } });
                const newVNode = createVNode({ key: 'a', props: { _c: true } });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update' });
            });

            it('should return "update" when neither has _c prop', () => {
                const oldVNode = createVNode({ key: 'a', props: {} });
                const newVNode = createVNode({ key: 'a', props: {} });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update' });
            });
        });

        describe('edge cases', () => {
            it('should handle vnodes without keys', () => {
                const oldVNode = createVNode({ _type: 'element', tag: 'div' });
                const newVNode = createVNode({ _type: 'element', tag: 'div' });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update' });
            });

            it('should handle text nodes', () => {
                const oldVNode: VirtualNode = { _type: 'text', text: 'old' };
                const newVNode: VirtualNode = { _type: 'text', text: 'new' };
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update' });
            });

            it('should handle fragment nodes', () => {
                const oldVNode = createVNode({ key: 'frag', _type: 'fragment' });
                const newVNode = createVNode({ key: 'frag', _type: 'fragment' });
                const result = computePatchDecision(oldVNode, newVNode);
                expect(result).toEqual({ action: 'update' });
            });
        });
    });

    describe('compareChildrenPointers()', () => {
        const divA = createVNode({ key: 'a', _type: 'element', tag: 'div' });
        const divB = createVNode({ key: 'b', _type: 'element', tag: 'div' });
        const divC = createVNode({ key: 'c', _type: 'element', tag: 'div' });
        const divD = createVNode({ key: 'd', _type: 'element', tag: 'div' });

        describe('start-start match', () => {
            it('should return "start-start" when old and new start match', () => {
                const result = compareChildrenPointers(divA, divB, divA, divC);
                expect(result).toBe('start-start');
            });

            it('should prioritise start-start over other matches', () => {
                const result = compareChildrenPointers(divA, divB, divA, divB);
                expect(result).toBe('start-start');
            });
        });

        describe('end-end match', () => {
            it('should return "end-end" when old and new end match', () => {
                const result = compareChildrenPointers(divA, divB, divC, divB);
                expect(result).toBe('end-end');
            });
        });

        describe('start-end match', () => {
            it('should return "start-end" when old start matches new end', () => {
                const result = compareChildrenPointers(divA, divB, divC, divA);
                expect(result).toBe('start-end');
            });
        });

        describe('end-start match', () => {
            it('should return "end-start" when old end matches new start', () => {
                const result = compareChildrenPointers(divA, divB, divB, divC);
                expect(result).toBe('end-start');
            });
        });

        describe('no match', () => {
            it('should return "none" when no pointers match', () => {
                const result = compareChildrenPointers(divA, divB, divC, divD);
                expect(result).toBe('none');
            });
        });

        describe('undefined handling', () => {
            it('should return "none" when oldStart is undefined', () => {
                const result = compareChildrenPointers(undefined, divB, divA, divC);
                expect(result).toBe('none');
            });

            it('should return "none" when newStart is undefined', () => {
                const result = compareChildrenPointers(divA, divB, undefined, divC);
                expect(result).toBe('none');
            });

            it('should return "end-end" when starts are undefined but ends match', () => {
                const result = compareChildrenPointers(undefined, divB, undefined, divB);
                expect(result).toBe('end-end');
            });

            it('should return "none" when all are undefined', () => {
                const result = compareChildrenPointers(undefined, undefined, undefined, undefined);
                expect(result).toBe('none');
            });

            it('should return "start-end" when oldEnd undefined but oldStart matches newEnd', () => {
                const result = compareChildrenPointers(divA, undefined, divB, divA);
                expect(result).toBe('start-end');
            });

            it('should return "end-start" when newEnd undefined but oldEnd matches newStart', () => {
                const result = compareChildrenPointers(divA, divB, divB, undefined);
                expect(result).toBe('end-start');
            });
        });

        describe('match priority', () => {
            it('should check start-start before end-end', () => {
                const result = compareChildrenPointers(divA, divB, divA, divB);
                expect(result).toBe('start-start');
            });

            it('should check end-end before start-end', () => {
                const result = compareChildrenPointers(divA, divB, divC, divB);
                expect(result).toBe('end-end');
            });

            it('should check start-end before end-start', () => {
                const result = compareChildrenPointers(divA, divB, divB, divA);
                expect(result).toBe('start-end');
            });
        });
    });
});
