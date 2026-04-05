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
    isReservedProp,
    categoriseProp,
    shouldRemoveProp,
    shouldUpdateProp,
    parseClassData,
    parseStyleData,
    combineClasses,
    combineStyles,
    computeClassStyle,
    computeAttrValue,
    computePropsToRemove,
    computePropsToUpdate,
    computePropDiff,
    RESERVED_PROPS
} from '@/vdom/propDiff';

describe('propDiff', () => {
    describe('isReservedProp', () => {
        it('should return true for all reserved props', () => {
            for (const prop of RESERVED_PROPS) {
                expect(isReservedProp(prop)).toBe(true);
            }
        });

        it('should return false for non-reserved props', () => {
            expect(isReservedProp('id')).toBe(false);
            expect(isReservedProp('onClick')).toBe(false);
            expect(isReservedProp('_ref')).toBe(false);
            expect(isReservedProp('custom')).toBe(false);
        });
    });

    describe('categoriseProp', () => {
        it('should categorize reserved props', () => {
            expect(categoriseProp('class')).toEqual({ type: 'reserved' });
            expect(categoriseProp('_class')).toEqual({ type: 'reserved' });
            expect(categoriseProp('style')).toEqual({ type: 'reserved' });
            expect(categoriseProp('_style')).toEqual({ type: 'reserved' });
            expect(categoriseProp('_k')).toEqual({ type: 'reserved' });
            expect(categoriseProp('_c')).toEqual({ type: 'reserved' });
            expect(categoriseProp('_s')).toEqual({ type: 'reserved' });
        });

        it('should categorize boolean attributes (? prefix)', () => {
            expect(categoriseProp('?disabled')).toEqual({ type: 'boolean-attr', attrName: 'disabled' });
            expect(categoriseProp('?checked')).toEqual({ type: 'boolean-attr', attrName: 'checked' });
            expect(categoriseProp('?hidden')).toEqual({ type: 'boolean-attr', attrName: 'hidden' });
        });

        it('should categorize standard event handlers (on prefix)', () => {
            expect(categoriseProp('onClick')).toEqual({ type: 'event', eventName: 'click', prefix: 'on' });
            expect(categoriseProp('onMouseOver')).toEqual({ type: 'event', eventName: 'mouseover', prefix: 'on' });
            expect(categoriseProp('onKeyDown')).toEqual({ type: 'event', eventName: 'keydown', prefix: 'on' });
        });

        it('should categorize custom event handlers (pe: prefix)', () => {
            expect(categoriseProp('pe:customevent')).toEqual({ type: 'event', eventName: 'customevent', prefix: 'pe' });
            expect(categoriseProp('pe:MyEvent')).toEqual({ type: 'event', eventName: 'myevent', prefix: 'pe' });
        });

        it('should categorize refs', () => {
            expect(categoriseProp('_ref')).toEqual({ type: 'ref' });
        });

        it('should categorize standard props', () => {
            expect(categoriseProp('id')).toEqual({ type: 'standard' });
            expect(categoriseProp('data-value')).toEqual({ type: 'standard' });
            expect(categoriseProp('aria-label')).toEqual({ type: 'standard' });
        });
    });

    describe('shouldRemoveProp', () => {
        it('should return true when prop is not in newProps', () => {
            expect(shouldRemoveProp('id', 'test', {})).toBe(true);
            expect(shouldRemoveProp('onClick', () => {}, {})).toBe(true);
        });

        it('should return true when value has changed', () => {
            expect(shouldRemoveProp('id', 'old', { id: 'new' })).toBe(true);
        });

        it('should return false when value is the same', () => {
            expect(shouldRemoveProp('id', 'same', { id: 'same' })).toBe(false);
        });
    });

    describe('shouldUpdateProp', () => {
        it('should return false for reserved props', () => {
            expect(shouldUpdateProp('class', 'old', 'new')).toBe(false);
            expect(shouldUpdateProp('_class', null, { active: true })).toBe(false);
            expect(shouldUpdateProp('style', '', 'colour: red')).toBe(false);
        });

        it('should return true for value prop (special handling)', () => {
            expect(shouldUpdateProp('value', 'same', 'same')).toBe(true);
        });

        it('should compare functions by reference', () => {
            const fn1 = () => {};
            const fn2 = () => {};
            expect(shouldUpdateProp('onClick', fn1, fn1)).toBe(false);
            expect(shouldUpdateProp('onClick', fn1, fn2)).toBe(true);
        });

        it('should return true when values differ', () => {
            expect(shouldUpdateProp('id', 'old', 'new')).toBe(true);
            expect(shouldUpdateProp('count', 1, 2)).toBe(true);
        });

        it('should return false when values are the same', () => {
            expect(shouldUpdateProp('id', 'same', 'same')).toBe(false);
            expect(shouldUpdateProp('count', 5, 5)).toBe(false);
        });
    });

    describe('parseClassData', () => {
        it('should handle string values', () => {
            expect(parseClassData('foo bar')).toBe('foo bar');
            expect(parseClassData('single')).toBe('single');
        });

        it('should handle empty/falsy values', () => {
            expect(parseClassData('')).toBe('');
            expect(parseClassData(null)).toBe('');
            expect(parseClassData(undefined)).toBe('');
        });

        it('should handle arrays', () => {
            expect(parseClassData(['foo', 'bar'])).toBe('foo bar');
            expect(parseClassData(['a', '', 'b'])).toBe('a b');
        });

        it('should handle nested arrays', () => {
            expect(parseClassData(['a', ['b', 'c'], 'd'])).toBe('a b c d');
        });

        it('should handle objects (truthy values become classes)', () => {
            expect(parseClassData({ active: true, disabled: false, visible: true })).toBe('active visible');
            expect(parseClassData({ foo: 1, bar: 0, baz: 'yes' })).toBe('foo baz');
        });

        it('should handle mixed nested structures', () => {
            expect(parseClassData(['base', { active: true, hidden: false }])).toBe('base active');
        });
    });

    describe('parseStyleData', () => {
        it('should handle string values', () => {
            expect(parseStyleData('colour: red')).toBe('colour: red');
            expect(parseStyleData('colour: red; font-size: 12px')).toBe('colour: red; font-size: 12px');
        });

        it('should handle empty/falsy values', () => {
            expect(parseStyleData('')).toBe('');
            expect(parseStyleData(null)).toBe('');
            expect(parseStyleData(undefined)).toBe('');
        });

        it('should handle object values with camelCase conversion', () => {
            expect(parseStyleData({ color: 'red' })).toBe('color: red;');
            expect(parseStyleData({ fontSize: '12px' })).toBe('font-size: 12px;');
            expect(parseStyleData({ backgroundColor: 'blue', marginTop: '10px' })).toBe('background-color: blue; margin-top: 10px;');
        });

        it('should filter out null/undefined values', () => {
            expect(parseStyleData({ color: 'red', display: null, opacity: undefined })).toBe('color: red;');
        });

        it('should return empty string for empty object', () => {
            expect(parseStyleData({})).toBe('');
            expect(parseStyleData({ display: null })).toBe('');
        });
    });

    describe('combineClasses', () => {
        it('should combine static and dynamic classes', () => {
            expect(combineClasses('static', 'dynamic')).toBe('static dynamic');
        });

        it('should handle empty static class', () => {
            expect(combineClasses('', 'dynamic')).toBe('dynamic');
            expect(combineClasses('   ', 'dynamic')).toBe('dynamic');
        });

        it('should handle empty dynamic class', () => {
            expect(combineClasses('static', '')).toBe('static');
            expect(combineClasses('static', '   ')).toBe('static');
        });

        it('should handle both empty', () => {
            expect(combineClasses('', '')).toBe('');
            expect(combineClasses('   ', '   ')).toBe('');
        });
    });

    describe('combineStyles', () => {
        it('should combine static and dynamic styles', () => {
            expect(combineStyles('colour: red', 'font-size: 12px')).toBe('colour: red; font-size: 12px;');
        });

        it('should handle trailing semicolons correctly', () => {
            expect(combineStyles('colour: red;', 'font-size: 12px;')).toBe('colour: red; font-size: 12px;');
            expect(combineStyles('colour: red;', 'font-size: 12px')).toBe('colour: red; font-size: 12px;');
        });

        it('should handle empty static style', () => {
            expect(combineStyles('', 'font-size: 12px')).toBe('font-size: 12px;');
        });

        it('should handle empty dynamic style', () => {
            expect(combineStyles('colour: red', '')).toBe('colour: red;');
        });

        it('should handle both empty', () => {
            expect(combineStyles('', '')).toBe('');
        });
    });

    describe('computeClassStyle', () => {
        it('should compute final class from static and dynamic values', () => {
            const result = computeClassStyle({ class: 'static', _class: { active: true } });
            expect(result.finalClass).toBe('static active');
        });

        it('should compute final style from static and dynamic values', () => {
            const result = computeClassStyle({ style: 'colour: red', _style: { fontSize: '12px' } });
            expect(result.finalStyle).toBe('colour: red; font-size: 12px;');
        });

        it('should determine shouldShow from _s prop', () => {
            expect(computeClassStyle({ _s: true }).shouldShow).toBe(true);
            expect(computeClassStyle({ _s: false }).shouldShow).toBe(false);
            expect(computeClassStyle({}).shouldShow).toBe(true);
        });

        it('should handle missing props', () => {
            const result = computeClassStyle({});
            expect(result.finalClass).toBe('');
            expect(result.finalStyle).toBe('');
            expect(result.shouldShow).toBe(true);
        });
    });

    describe('computeAttrValue', () => {
        it('should return null for null/undefined/false', () => {
            expect(computeAttrValue(null)).toBe(null);
            expect(computeAttrValue(undefined)).toBe(null);
            expect(computeAttrValue(false)).toBe(null);
        });

        it('should stringify objects', () => {
            expect(computeAttrValue({ a: 1 })).toBe('{"a":1}');
            expect(computeAttrValue([1, 2, 3])).toBe('[1,2,3]');
        });

        it('should convert primitives to strings', () => {
            expect(computeAttrValue('hello')).toBe('hello');
            expect(computeAttrValue(42)).toBe('42');
            expect(computeAttrValue(true)).toBe('true');
        });
    });

    describe('computePropsToRemove', () => {
        it('should generate remove-attr for standard props', () => {
            const ops = computePropsToRemove({ id: 'test' }, {});
            expect(ops).toContainEqual({ type: 'remove-attr', attrName: 'id' });
        });

        it('should generate remove-boolean-attr for boolean attrs', () => {
            const ops = computePropsToRemove({ '?disabled': true }, {});
            expect(ops).toContainEqual({ type: 'remove-boolean-attr', attrName: 'disabled' });
        });

        it('should generate remove-event for event handlers', () => {
            const handler = () => {};
            const ops = computePropsToRemove({ onClick: handler }, {});
            expect(ops).toContainEqual({ type: 'remove-event', eventName: 'click', handler });
        });

        it('should generate remove-ref for refs', () => {
            const ops = computePropsToRemove({ _ref: 'myRef' }, {});
            expect(ops).toContainEqual({ type: 'remove-ref', refName: 'myRef' });
        });

        it('should not remove props that still exist with same value', () => {
            const ops = computePropsToRemove({ id: 'same' }, { id: 'same' });
            expect(ops).toHaveLength(0);
        });

        it('should not generate operations for reserved props', () => {
            const ops = computePropsToRemove({ class: 'test', _class: { active: true } }, {});
            expect(ops.filter(op => op.type === 'remove-attr' && (op.attrName === 'class' || op.attrName === '_class'))).toHaveLength(0);
        });
    });

    describe('computePropsToUpdate', () => {
        it('should generate set-attr for standard props', () => {
            const ops = computePropsToUpdate({}, { id: 'test' });
            expect(ops).toContainEqual({ type: 'set-attr', attrName: 'id', value: 'test' });
        });

        it('should generate set-boolean-attr for boolean attrs', () => {
            const ops = computePropsToUpdate({}, { '?disabled': true });
            expect(ops).toContainEqual({ type: 'set-boolean-attr', attrName: 'disabled', value: true });
        });

        it('should generate set-event for event handlers', () => {
            const newHandler = () => {};
            const ops = computePropsToUpdate({}, { onClick: newHandler });
            expect(ops).toContainEqual({
                type: 'set-event',
                eventName: 'click',
                oldHandler: undefined,
                newHandler
            });
        });

        it('should generate set-ref for refs', () => {
            const ops = computePropsToUpdate({}, { _ref: 'myRef' });
            expect(ops).toContainEqual({ type: 'set-ref', oldRefName: undefined, newRefName: 'myRef' });
        });

        it('should generate remove-null-attr for null/false values', () => {
            const ops = computePropsToUpdate({ id: 'test' }, { id: null });
            expect(ops).toContainEqual({ type: 'remove-null-attr', attrName: 'id' });
        });

        it('should not update unchanged props', () => {
            const ops = computePropsToUpdate({ id: 'same' }, { id: 'same' });
            expect(ops.filter(op => 'attrName' in op && op.attrName === 'id')).toHaveLength(0);
        });

        it('should not generate operations for reserved props', () => {
            const ops = computePropsToUpdate({}, { class: 'test', style: 'colour: red' });
            expect(ops).toHaveLength(0);
        });
    });

    describe('listener options ($-suffixed event props)', () => {
        describe('categoriseProp with listener options', () => {
            it('should parse $capture suffix on standard events', () => {
                expect(categoriseProp('onClick$capture')).toEqual({
                    type: 'event',
                    eventName: 'click',
                    prefix: 'on',
                    listenerOptions: { capture: true }
                });
            });

            it('should parse $passive suffix on standard events', () => {
                expect(categoriseProp('onClick$passive')).toEqual({
                    type: 'event',
                    eventName: 'click',
                    prefix: 'on',
                    listenerOptions: { passive: true }
                });
            });

            it('should parse multiple $-delimited options', () => {
                expect(categoriseProp('onClick$capture$passive')).toEqual({
                    type: 'event',
                    eventName: 'click',
                    prefix: 'on',
                    listenerOptions: { capture: true, passive: true }
                });
            });

            it('should leave listenerOptions undefined when no $ suffix is present (backward compat)', () => {
                const result = categoriseProp('onClick');
                expect(result).toEqual({ type: 'event', eventName: 'click', prefix: 'on' });
                expect(result.type === 'event' && result.listenerOptions).toBeFalsy();
            });

            it('should parse $passive suffix on pe: custom events', () => {
                expect(categoriseProp('pe:update$passive')).toEqual({
                    type: 'event',
                    eventName: 'update',
                    prefix: 'pe',
                    listenerOptions: { passive: true }
                });
            });

            it('should parse multiple options on pe: custom events', () => {
                expect(categoriseProp('pe:update$capture$passive')).toEqual({
                    type: 'event',
                    eventName: 'update',
                    prefix: 'pe',
                    listenerOptions: { capture: true, passive: true }
                });
            });
        });

        describe('computePropsToRemove with listener options', () => {
            it('should carry listenerOptions on remove-event operations', () => {
                const handler = () => {};
                const ops = computePropsToRemove({ 'onClick$capture': handler }, {});
                expect(ops).toContainEqual({
                    type: 'remove-event',
                    eventName: 'click',
                    handler,
                    listenerOptions: { capture: true }
                });
            });
        });

        describe('computePropsToUpdate with listener options', () => {
            it('should carry listenerOptions on set-event operations', () => {
                const newHandler = () => {};
                const ops = computePropsToUpdate({}, { 'onClick$passive': newHandler });
                expect(ops).toContainEqual({
                    type: 'set-event',
                    eventName: 'click',
                    oldHandler: undefined,
                    newHandler,
                    listenerOptions: { passive: true }
                });
            });
        });
    });

    describe('computePropDiff', () => {
        it('should compute full diff with removals, updates, and classStyle', () => {
            const oldProps = { id: 'old', class: 'base', onClick: () => {} };
            const newProps = { id: 'new', class: 'base active', title: 'hello' };

            const result = computePropDiff(oldProps, newProps);

            expect(result.removals.length).toBeGreaterThan(0);
            expect(result.updates.length).toBeGreaterThan(0);
            expect(result.classStyle).toBeDefined();
            expect(result.classStyle.finalClass).toBe('base active');
        });

        it('should handle empty props', () => {
            const result = computePropDiff({}, {});
            expect(result.removals).toHaveLength(0);
            expect(result.updates).toHaveLength(0);
            expect(result.classStyle.shouldShow).toBe(true);
        });
    });
});
