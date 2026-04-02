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
import { dom } from './';
import type { VirtualNode } from './';

describe('DOM Factory Functions', () => {
  describe('dom.txt()', () => {
    it('should create a text VNode with correct properties', () => {
      const node = dom.txt('Hello World', 'txt1');
      expect(node._type).toBe('text');
      expect(node.text).toBe('Hello World');
      expect(node.key).toBe('txt1');
      expect(node.props).toEqual({});
      expect(node.children).toBeNull();
    });

    it('should create a text VNode with given props', () => {
      const props = { 'data-test': 'my-text' };
      const node = dom.txt('With Props', 'txt2', props);
      expect(node.props).toEqual(props);
      expect(node.key).toBe('txt2');
    });

    it('should stringify non-string content', () => {
      const node = dom.txt(123, 'txt3');
      expect(node.text).toBe('123');
      const node2 = dom.txt(null, 'txt4');
      expect(node2.text).toBe('');
      const node3 = dom.txt(undefined, 'txt5');
      expect(node3.text).toBe('');
    });

    it('should handle _isWhitespace prop for dom.ws via dom.txt', () => {
      const node = dom.ws('ws1');
      expect(node._type).toBe('text');
      expect(node.text).toBe(' ');
      expect(node.key).toBe('ws1');
      expect(node.props).toEqual({ _isWhitespace: true });
    });
  });

  describe('dom.cmt()', () => {
    it('should create a comment VNode with correct properties', () => {
      const node = dom.cmt('This is a comment', 'cmt1');
      expect(node._type).toBe('comment');
      expect(node.text).toBe('This is a comment');
      expect(node.key).toBe('cmt1');
      expect(node.props).toEqual({});
      expect(node.children).toBeNull();
    });

    it('should create a comment VNode with given props', () => {
      const props = { 'data-info': 'hidden' };
      const node = dom.cmt('Another comment', 'cmt2', props);
      expect(node.props).toEqual(props);
      expect(node.key).toBe('cmt2');
    });

    it('should stringify non-string content for comments', () => {
      const node = dom.cmt(456, 'cmt3');
      expect(node.text).toBe('456');
    });
  });

  describe('dom.el()', () => {
    it('should create an element VNode with correct properties', () => {
      const node = dom.el('div', 'el1');
      expect(node._type).toBe('element');
      expect(node.tag).toBe('div');
      expect(node.key).toBe('el1');
      expect(node.props).toEqual({});
      expect(node.children).toEqual([]);
    });

    it('should create an element VNode with given props', () => {
      const props = { id: 'myDiv', class: 'container' };
      const node = dom.el('div', 'el2', props);
      expect(node.props).toEqual(props);
      expect(node.key).toBe('el2');
    });

    it('should create an element VNode with a single child', () => {
      const childNode = dom.txt('child text', 'child1');
      const node = dom.el('p', 'el3', {}, [childNode]);
      expect(node.children).toEqual([childNode]);
    });

    it('should create an element VNode with multiple children', () => {
      const child1 = dom.el('span', 'span1');
      const child2 = dom.txt('text', 'txt-in-el');
      const node = dom.el('div', 'el4', {}, [child1, child2]);
      expect(node.children).toEqual([child1, child2]);
    });

    it('should filter out null/undefined children', () => {
      const child1 = dom.el('span', 'span2');
      const node = dom.el('div', 'el5', {}, [child1, null, undefined, dom.txt('valid', 'txt-valid')] as unknown as VirtualNode[]);
      expect(node.children?.length).toBe(2);
      expect((node.children![0] as VirtualNode).key).toBe('span2');
      expect((node.children![1] as VirtualNode).key).toBe('txt-valid');
    });

    it('should handle single child not in an array', () => {
      const childNode = dom.txt('single child', 'single-child-txt');
      const node = dom.el('p', 'el6', {}, childNode);
      expect(node.children).toEqual([childNode]);
    });

    it('should handle null children argument', () => {
      const node = dom.el('div', 'el7', {}, null);
      expect(node.children).toEqual([]);
    });
  });

  describe('dom.frag()', () => {
    it('should create a fragment VNode with correct properties', () => {
      const node = dom.frag('frag1');
      expect(node._type).toBe('fragment');
      expect(node.key).toBe('frag1');
      expect(node.props).toEqual({});
      expect(node.children).toEqual([]);
    });

    it('should create a fragment VNode with given props', () => {
      const props = { _c: true, 'data-group': 'groupA' };
      const node = dom.frag('frag2', [], props);
      expect(node.props).toEqual(props);
      expect(node.key).toBe('frag2');
    });

    it('should create a fragment VNode with children', () => {
      const child1 = dom.el('span', 'span-in-frag');
      const child2 = dom.txt('text in frag', 'txt-in-frag');
      const node = dom.frag('frag3', [child1, child2]);
      expect(node.children).toEqual([child1, child2]);
    });

    it('should filter out null/undefined children for fragments', () => {
      const child1 = dom.el('h1', 'h1-in-frag');
      const node = dom.frag('frag4', [null, child1, undefined] as unknown as VirtualNode[]);
      expect(node.children?.length).toBe(1);
      expect((node.children![0] as VirtualNode).key).toBe('h1-in-frag');
    });

    it('should handle single child not in an array for fragments', () => {
      const childNode = dom.txt('single child for frag', 'single-child-frag-txt');
      const node = dom.frag('frag5', childNode);
      expect(node.children).toEqual([childNode]);
    });

    it('should handle null children argument for fragments', () => {
      const node = dom.frag('frag6', null, { _c: false });
      expect(node.children).toEqual([]);
      expect(node.props).toEqual({ _c: false });
    });
  });

  describe('dom.resolveTag()', () => {
    it('should return plain HTML tags unchanged', () => {
      expect(dom.resolveTag('div')).toBe('div');
      expect(dom.resolveTag('span')).toBe('span');
      expect(dom.resolveTag('h1')).toBe('h1');
    });

    it('should map piko: prefixed tags to HTML equivalents', () => {
      expect(dom.resolveTag('piko:a')).toBe('a');
      expect(dom.resolveTag('piko:img')).toBe('img');
      expect(dom.resolveTag('piko:svg')).toBe('piko-svg-inline');
      expect(dom.resolveTag('piko:picture')).toBe('picture');
      expect(dom.resolveTag('piko:video')).toBe('video');
    });

    it('should fall back to div for empty tag', () => {
      expect(dom.resolveTag('')).toBe('div');
      expect(dom.resolveTag(null)).toBe('div');
      expect(dom.resolveTag(undefined)).toBe('div');
    });

    it('should fall back to div for rejected targets', () => {
      expect(dom.resolveTag('piko:partial')).toBe('div');
      expect(dom.resolveTag('piko:slot')).toBe('div');
      expect(dom.resolveTag('piko:element')).toBe('div');
    });

    it('should pass through unknown piko: prefixed tags', () => {
      expect(dom.resolveTag('piko:unknown')).toBe('piko:unknown');
    });
  });

  describe('dom.pikoEl()', () => {
    it('should create an element with resolved tag', () => {
      const node = dom.pikoEl('div', 'pe1');
      expect(node._type).toBe('element');
      expect(node.tag).toBe('div');
      expect(node.key).toBe('pe1');
    });

    it('should resolve piko:a to a and add piko:a attribute', () => {
      const node = dom.pikoEl('piko:a', 'pe2', { href: '/test' });
      expect(node.tag).toBe('a');
      expect(node.props).toHaveProperty('href', '/test');
      expect(node.props).toHaveProperty('piko:a', '');
      expect(node.props).toHaveProperty('onClick');
    });

    it('should not add piko:a attribute for non-link tags', () => {
      const node = dom.pikoEl('h1', 'pe3', { id: 'heading' });
      expect(node.tag).toBe('h1');
      expect(node.props).toEqual({ id: 'heading' });
      expect(node.props).not.toHaveProperty('piko:a');
    });

    it('should resolve piko:img without adding piko:a attribute', () => {
      const node = dom.pikoEl('piko:img', 'pe4', { src: '/photo.jpg' });
      expect(node.tag).toBe('img');
      expect(node.props).not.toHaveProperty('piko:a');
    });

    it('should resolve piko:svg to piko-svg-inline tag', () => {
      const node = dom.pikoEl('piko:svg', 'pe-svg', { viewBox: '0 0 24 24' });
      expect(node.tag).toBe('piko-svg-inline');
      expect(node.props).not.toHaveProperty('piko:a');
    });

    it('should add onClick handler for piko:a with href', () => {
      const node = dom.pikoEl('piko:a', 'pe-link', { href: '/test' });
      expect(node.tag).toBe('a');
      expect(node.props).toHaveProperty('piko:a', '');
      expect(node.props).toHaveProperty('onClick');
      expect(typeof node.props!['onClick']).toBe('function');
    });

    it('should not add onClick handler for piko:a without href', () => {
      const node = dom.pikoEl('piko:a', 'pe-link-nohref', {});
      expect(node.tag).toBe('a');
      expect(node.props).toHaveProperty('piko:a', '');
      expect(node.props).not.toHaveProperty('onClick');
    });

    it('should fall back to div for empty tag', () => {
      const node = dom.pikoEl('', 'pe5');
      expect(node.tag).toBe('div');
    });

    it('should fall back to div for rejected targets', () => {
      const node = dom.pikoEl('piko:element', 'pe6');
      expect(node.tag).toBe('div');
      expect(node.props).not.toHaveProperty('piko:a');
    });

    it('should handle children correctly', () => {
      const child = dom.txt('Hello', 'child1');
      const node = dom.pikoEl('section', 'pe7', {}, [child]);
      expect(node.children).toEqual([child]);
    });

    it('should not mutate the original props object', () => {
      const props = { href: '/test' };
      dom.pikoEl('piko:a', 'pe8', props);
      expect(props).toEqual({ href: '/test' });
      expect(props).not.toHaveProperty('piko:a');
    });
  });

  describe('Nested Array Flattening (p-for with siblings)', () => {

    describe('Basic Nested Arrays', () => {
      it('should flatten p-for result array with single sibling after', () => {
        const item1 = dom.el('div', 'item-1', { class: 'file-item' });
        const item2 = dom.el('div', 'item-2', { class: 'file-item' });
        const addMoreBtn = dom.el('label', 'add-more', { class: 'add-more' });

        const node = dom.el('div', 'container', {}, [
          [item1, item2],
          addMoreBtn
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![0]).toBe(item1);
        expect(node.children![1]).toBe(item2);
        expect(node.children![2]).toBe(addMoreBtn);
      });

      it('should flatten p-for result array with single sibling before', () => {
        const title = dom.el('h2', 'title');
        const item1 = dom.el('div', 'item-1');
        const item2 = dom.el('div', 'item-2');

        const node = dom.el('div', 'container', {}, [
          title,
          [item1, item2]
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![0]).toBe(title);
        expect(node.children![1]).toBe(item1);
        expect(node.children![2]).toBe(item2);
      });

      it('should flatten p-for result sandwiched between siblings', () => {
        const header = dom.el('h2', 'header');
        const item1 = dom.el('div', 'item-1');
        const item2 = dom.el('div', 'item-2');
        const footer = dom.el('button', 'submit');

        const node = dom.el('div', 'container', {}, [
          header,
          [item1, item2],
          footer
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(4);
        expect(node.children![0]).toBe(header);
        expect(node.children![1]).toBe(item1);
        expect(node.children![2]).toBe(item2);
        expect(node.children![3]).toBe(footer);
      });

      it('should flatten multiple adjacent p-for result arrays', () => {
        const a1 = dom.el('div', 'a-1');
        const a2 = dom.el('div', 'a-2');
        const b1 = dom.el('div', 'b-1');
        const b2 = dom.el('div', 'b-2');

        const node = dom.el('div', 'container', {}, [
          [a1, a2],
          [b1, b2]
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(4);
        expect(node.children![0]).toBe(a1);
        expect(node.children![1]).toBe(a2);
        expect(node.children![2]).toBe(b1);
        expect(node.children![3]).toBe(b2);
      });

      it('should flatten multiple p-for arrays with elements between', () => {
        const a1 = dom.el('div', 'a-1');
        const a2 = dom.el('div', 'a-2');
        const separator = dom.el('hr', 'separator');
        const b1 = dom.el('div', 'b-1');
        const b2 = dom.el('div', 'b-2');

        const node = dom.el('div', 'container', {}, [
          [a1, a2],
          separator,
          [b1, b2]
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(5);
        expect(node.children![0]).toBe(a1);
        expect(node.children![1]).toBe(a2);
        expect(node.children![2]).toBe(separator);
        expect(node.children![3]).toBe(b1);
        expect(node.children![4]).toBe(b2);
      });
    });

    describe('Deeply Nested Arrays (nested p-for)', () => {
      it('should flatten double-nested arrays from nested p-for', () => {
        const deep1 = dom.el('span', 'deep-1');
        const deep2 = dom.el('span', 'deep-2');
        const sibling = dom.el('button', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [[deep1, deep2]],
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![0]).toBe(deep1);
        expect(node.children![1]).toBe(deep2);
        expect(node.children![2]).toBe(sibling);
      });

      it('should flatten triple-nested arrays', () => {
        const veryDeep = dom.el('span', 'very-deep');

        const node = dom.el('div', 'container', {}, [
          [[[veryDeep]]]
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(1);
        expect(node.children![0]).toBe(veryDeep);
      });

      it('should flatten mixed nesting depths in same children array', () => {
        const shallow1 = dom.el('div', 'shallow-1');
        const shallow2 = dom.el('div', 'shallow-2');
        const deep1 = dom.el('div', 'deep-1');
        const deep2 = dom.el('div', 'deep-2');
        const regular = dom.el('div', 'regular');

        const node = dom.el('div', 'container', {}, [
          [shallow1, shallow2],
          [[deep1, deep2]],
          regular
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(5);
        expect(node.children![0]).toBe(shallow1);
        expect(node.children![1]).toBe(shallow2);
        expect(node.children![2]).toBe(deep1);
        expect(node.children![3]).toBe(deep2);
        expect(node.children![4]).toBe(regular);
      });
    });

    describe('Empty/Null Handling with Nesting', () => {
      it('should handle empty array from p-for with sibling', () => {
        const sibling = dom.el('label', 'add-first');

        const node = dom.el('div', 'container', {}, [
          [],
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(1);
        expect(node.children![0]).toBe(sibling);
      });

      it('should filter nulls from nested arrays', () => {
        const item1 = dom.el('div', 'item-1');
        const item3 = dom.el('div', 'item-3');
        const sibling = dom.el('button', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [item1, null, item3],
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![0]).toBe(item1);
        expect(node.children![1]).toBe(item3);
        expect(node.children![2]).toBe(sibling);
      });

      it('should handle nested array containing only nulls', () => {
        const sibling = dom.el('div', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [null, null, null],
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(1);
        expect(node.children![0]).toBe(sibling);
      });

      it('should handle mix of nulls and nested arrays at top level', () => {
        const item1 = dom.el('div', 'item-1');
        const item2 = dom.el('div', 'item-2');
        const sibling = dom.el('button', 'sibling');

        const node = dom.el('div', 'container', {}, [
          null,
          [item1, item2],
          null,
          sibling,
          undefined
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![0]).toBe(item1);
        expect(node.children![1]).toBe(item2);
        expect(node.children![2]).toBe(sibling);
      });

      it('should handle multiple empty nested arrays', () => {
        const sibling = dom.el('div', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [],
          [],
          sibling,
          []
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(1);
        expect(node.children![0]).toBe(sibling);
      });

      it('should handle nested empty arrays in deep structure', () => {
        const item = dom.el('div', 'item');
        const sibling = dom.el('button', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [[], [item]],
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(2);
        expect(node.children![0]).toBe(item);
        expect(node.children![1]).toBe(sibling);
      });
    });

    describe('Real-World UI Patterns', () => {
      it('should handle file uploader pattern (p-for files + add button)', () => {
        const file1 = dom.el('div', 'file-1', { class: 'file-item' }, [
          dom.el('img', 'thumb-1', { src: 'preview1.jpg' }),
          dom.txt('document.pdf', 'name-1')
        ]);
        const file2 = dom.el('div', 'file-2', { class: 'file-item' }, [
          dom.el('img', 'thumb-2', { src: 'preview2.jpg' }),
          dom.txt('image.png', 'name-2')
        ]);
        const addMoreBtn = dom.el('label', 'add-more', { class: 'upload-area add-more' }, [
          dom.el('span', 'icon'),
          dom.txt('Add more', 'add-text')
        ]);

        const fileList = dom.el('div', 'file-list', { class: 'file-list' }, [
          [file1, file2],
          addMoreBtn
        ] as unknown as VirtualNode[]);

        expect(fileList.children).toHaveLength(3);
        expect(fileList.children![0]).toBe(file1);
        expect(fileList.children![1]).toBe(file2);
        expect(fileList.children![2]).toBe(addMoreBtn);
      });

      it('should handle todo list pattern (p-for todos + input)', () => {
        const todo1 = dom.el('li', 'todo-1', {}, [dom.txt('Buy milk', 'text-1')]);
        const todo2 = dom.el('li', 'todo-2', {}, [dom.txt('Walk dog', 'text-2')]);
        const inputField = dom.el('input', 'new-todo', { placeholder: 'Add todo...' });

        const todoList = dom.el('ul', 'todos', {}, [
          [todo1, todo2],
          dom.el('li', 'input-wrapper', {}, [inputField])
        ] as unknown as VirtualNode[]);

        expect(todoList.children).toHaveLength(3);
      });

      it('should handle tab bar pattern (p-for tabs + indicator)', () => {
        const tab1 = dom.el('button', 'tab-1', { class: 'tab' });
        const tab2 = dom.el('button', 'tab-2', { class: 'tab' });
        const tab3 = dom.el('button', 'tab-3', { class: 'tab' });
        const indicator = dom.el('div', 'indicator', { class: 'active-indicator' });

        const tabBar = dom.el('div', 'tab-bar', {}, [
          [tab1, tab2, tab3],
          indicator
        ] as unknown as VirtualNode[]);

        expect(tabBar.children).toHaveLength(4);
        expect(tabBar.children![3]).toBe(indicator);
      });

      it('should handle comment thread pattern (p-for comments + reply form)', () => {
        const comment1 = dom.el('div', 'comment-1', { class: 'comment' });
        const comment2 = dom.el('div', 'comment-2', { class: 'comment' });
        const replyForm = dom.el('form', 'reply-form', { class: 'reply-form' }, [
          dom.el('textarea', 'reply-input'),
          dom.el('button', 'submit-btn', {}, [dom.txt('Reply', 'btn-text')])
        ]);

        const thread = dom.el('div', 'thread', {}, [
          [comment1, comment2],
          replyForm
        ] as unknown as VirtualNode[]);

        expect(thread.children).toHaveLength(3);
      });

      it('should handle card grid with conditional empty state', () => {
        const card1 = dom.el('div', 'card-1', { class: 'card' });
        const card2 = dom.el('div', 'card-2', { class: 'card' });
        const emptyState = null;

        const grid = dom.el('div', 'grid', {}, [
          [card1, card2],
          emptyState
        ] as unknown as VirtualNode[]);

        expect(grid.children).toHaveLength(2);
        expect(grid.children![0]).toBe(card1);
        expect(grid.children![1]).toBe(card2);
      });
    });

    describe('Fragment Interactions', () => {
      it('should flatten nested arrays inside fragments', () => {
        const item1 = dom.el('div', 'item-1');
        const item2 = dom.el('div', 'item-2');
        const sibling = dom.el('span', 'sibling');

        const frag = dom.frag('my-frag', [
          [item1, item2],
          sibling
        ] as unknown as VirtualNode[]);

        expect(frag.children).toHaveLength(3);
        expect(frag.children![0]).toBe(item1);
        expect(frag.children![1]).toBe(item2);
        expect(frag.children![2]).toBe(sibling);
      });

      it('should handle nested fragments with p-for results', () => {
        const innerItem1 = dom.el('span', 'inner-1');
        const innerItem2 = dom.el('span', 'inner-2');
        const outerSibling = dom.el('div', 'outer-sibling');

        const innerFrag = dom.frag('inner-frag', [
          [innerItem1, innerItem2]
        ] as unknown as VirtualNode[]);

        const outerFrag = dom.frag('outer-frag', [
          innerFrag,
          outerSibling
        ]);

        expect(outerFrag.children).toHaveLength(2);
        expect(innerFrag.children).toHaveLength(2);
      });
    });

    describe('Performance and Scale', () => {
      it('should handle large arrays without stack overflow', () => {
        const largeArray: VirtualNode[] = [];
        for (let i = 0; i < 1000; i++) {
          largeArray.push(dom.el('div', `item-${i}`));
        }
        const sibling = dom.el('button', 'load-more');

        const node = dom.el('div', 'container', {}, [
          largeArray,
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(1001);
        expect(node.children![0].key).toBe('item-0');
        expect(node.children![999].key).toBe('item-999');
        expect(node.children![1000]).toBe(sibling);
      });

      it('should handle many small nested arrays', () => {
        const arrays: VirtualNode[][] = [];
        for (let i = 0; i < 50; i++) {
          arrays.push([
            dom.el('span', `group-${i}-a`),
            dom.el('span', `group-${i}-b`)
          ]);
        }
        const finalSibling = dom.el('div', 'final');

        const node = dom.el('div', 'container', {}, [
          ...arrays,
          finalSibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(101);
        expect(node.children![100]).toBe(finalSibling);
      });

      it('should handle deeply nested structure without stack overflow', () => {
        let deeplyNested: unknown = dom.el('span', 'deepest');
        for (let i = 0; i < 10; i++) {
          deeplyNested = [deeplyNested];
        }
        const sibling = dom.el('div', 'sibling');

        const node = dom.el('div', 'container', {}, [
          deeplyNested,
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(2);
        expect(node.children![0].key).toBe('deepest');
        expect(node.children![1]).toBe(sibling);
      });
    });

    describe('Whitespace and Special Nodes', () => {
      it('should preserve whitespace nodes with nested arrays', () => {
        const item1 = dom.el('span', 'item-1');
        const item2 = dom.el('span', 'item-2');
        const whitespace = dom.ws('ws-1');
        const sibling = dom.el('div', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [item1, item2],
          whitespace,
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(4);
        expect(node.children![2]).toBe(whitespace);
      });

      it('should preserve comment nodes with nested arrays', () => {
        const item1 = dom.el('span', 'item-1');
        const comment = dom.cmt('Generated by p-for', 'comment-1');
        const sibling = dom.el('div', 'sibling');

        const node = dom.el('div', 'container', {}, [
          [item1],
          comment,
          sibling
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![1]).toBe(comment);
      });

      it('should handle text nodes mixed with nested arrays', () => {
        const item1 = dom.el('li', 'item-1');
        const item2 = dom.el('li', 'item-2');
        const textNode = dom.txt('No more items', 'end-text');

        const node = dom.el('ul', 'list', {}, [
          [item1, item2],
          textNode
        ] as unknown as VirtualNode[]);

        expect(node.children).toHaveLength(3);
        expect(node.children![2]).toBe(textNode);
      });
    });

    describe('Edge Cases from Bug Report', () => {
      it('should correctly render the exact pp-file-uploader structure', () => {
        const createFileItem = (id: string, name: string, isImage: boolean) => {
          const children: VirtualNode[] = [];
          if (isImage) {
            children.push(dom.el('img', `thumb-${id}`, { class: 'thumbnail image', src: 'blob:...' }));
          } else {
            children.push(dom.el('div', `placeholder-${id}`, { class: 'thumbnail document' }));
          }
          children.push(dom.el('p', `name-${id}`, { class: 'filename' }, [dom.txt(name, `name-text-${id}`)]));
          children.push(dom.el('button', `remove-${id}`, { class: 'remove-btn' }));

          return dom.el('div', id, { class: 'file-item', 'data-file-id': id }, children);
        };

        const file1 = createFileItem('file-1', 'test-red.png', true);
        const file2 = createFileItem('file-2', 'document.pdf', false);

        const addMoreLabel = dom.el('label', 'add-more', { class: 'upload-area add-more' }, [
          dom.el('span', 'icon', { class: 'upload-icon' }),
          dom.el('span', 'text', { class: 'add-text' }, [dom.txt('Add more', 'add-text-content')])
        ]);

        const fileList = dom.el('div', 'file-list', { class: 'file-list' }, [
          [file1, file2],
          addMoreLabel
        ] as unknown as VirtualNode[]);

        expect(fileList.children).toHaveLength(3);
        expect(fileList.children![0]).toBe(file1);
        expect(fileList.children![0].key).toBe('file-1');
        expect(fileList.children![1]).toBe(file2);
        expect(fileList.children![1].key).toBe('file-2');
        expect(fileList.children![2]).toBe(addMoreLabel);
        expect(fileList.children![2].key).toBe('add-more');
      });

      it('should handle single file selected case', () => {
        const file1 = dom.el('div', 'file-1', { class: 'file-item' });
        const addMoreLabel = dom.el('label', 'add-more', { class: 'upload-area add-more' });

        const fileList = dom.el('div', 'file-list', { class: 'file-list' }, [
          [file1],
          addMoreLabel
        ] as unknown as VirtualNode[]);

        expect(fileList.children).toHaveLength(2);
        expect(fileList.children![0]).toBe(file1);
        expect(fileList.children![1]).toBe(addMoreLabel);
      });

      it('should handle no files selected case (empty p-for)', () => {
        const addMoreLabel = dom.el('label', 'add-more', { class: 'upload-area add-more' });

        const fileList = dom.el('div', 'file-list', { class: 'file-list' }, [
          [],
          addMoreLabel
        ] as unknown as VirtualNode[]);

        expect(fileList.children).toHaveLength(1);
        expect(fileList.children![0]).toBe(addMoreLabel);
      });
    });
  });
});
