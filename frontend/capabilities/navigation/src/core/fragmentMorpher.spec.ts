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

import { describe, it, expect, afterEach, vi } from 'vitest';
import fragmentMorpher from './fragmentMorpher';

describe('fragmentMorpher', () => {
    let fromEl: HTMLElement;
    let host: HTMLElement;

    const setup = (html: string) => {
        host = document.createElement('div');
        host.innerHTML = html.trim();
        fromEl = host.firstChild as HTMLElement;
        document.body.appendChild(host);
    };

    afterEach(() => {
        if (host && host.parentNode) {
            host.parentNode.removeChild(host);
        }
    });

    describe('Basic Node Morphing', () => {
        it('should change the tag name of the root element', () => {
            setup('<div>Old</div>');
            fragmentMorpher(fromEl, '<span>New</span>');
            expect(host.innerHTML).toBe('<span>New</span>');
        });

        it('should update the text content of a text node', () => {
            setup('<p>Old text</p>');
            const fromP = fromEl;
            fragmentMorpher(fromP, '<p>New text</p>');
            expect(fromP.textContent).toBe('New text');
            expect(fromP.tagName).toBe('P');
        });

        it('should not change the element if from and to are identical', () => {
            setup('<div><p>Content</p></div>');
            const originalChild = fromEl.querySelector('p');
            fragmentMorpher(fromEl, '<div><p>Content</p></div>');
            expect(fromEl.isEqualNode(host.firstChild as HTMLElement)).toBe(true);
            expect(fromEl.querySelector('p')).toBe(originalChild);
        });

        it('should replace an element with a text node', () => {
            setup('<div>Element</div>');
            fragmentMorpher(fromEl, 'Just Text');
            expect(host.innerHTML).toBe('Just Text');
        });
    });

    describe('Attribute Morphing', () => {
        it('should add new attributes', () => {
            setup('<div id="a"></div>');
            fragmentMorpher(fromEl, '<div id="a" class="foo" data-val="bar"></div>');
            expect(fromEl.id).toBe('a');
            expect(fromEl.className).toBe('foo');
            expect(fromEl.getAttribute('data-val')).toBe('bar');
        });

        it('should remove old attributes', () => {
            setup('<div id="a" class="foo" data-val="bar"></div>');
            fragmentMorpher(fromEl, '<div id="a"></div>');
            expect(fromEl.hasAttribute('class')).toBe(false);
            expect(fromEl.hasAttribute('data-val')).toBe(false);
            expect(fromEl.id).toBe('a');
        });

        it('should update existing attributes', () => {
            setup('<div id="a" class="foo"></div>');
            fragmentMorpher(fromEl, '<div id="a" class="bar"></div>');
            expect(fromEl.id).toBe('a');
            expect(fromEl.className).toBe('bar');
        });
    });

    describe('Children Morphing (keyed)', () => {
        const getKey = (node: Node) => {
            if (node.nodeType === 1) {
                const id = (node as HTMLElement).id;
                return id ? id : null;
            }
            return null;
        };
        const opts = { getNodeKey: getKey };

        it('should append nodes to the end', () => {
            setup('<div><p id="1"></p></div>');
            fragmentMorpher(fromEl, '<div><p id="1"></p><p id="2"></p><p id="3"></p></div>', opts);
            expect(fromEl.children.length).toBe(3);
            expect(fromEl.children[0].id).toBe('1');
            expect(fromEl.children[1].id).toBe('2');
            expect(fromEl.children[2].id).toBe('3');
        });

        it('should prepend nodes to the beginning', () => {
            setup('<div><p id="3"></p></div>');
            fragmentMorpher(fromEl, '<div><p id="1"></p><p id="2"></p><p id="3"></p></div>', opts);
            expect(fromEl.children.length).toBe(3);
            expect(fromEl.children[0].id).toBe('1');
            expect(fromEl.children[1].id).toBe('2');
        });

        it('should remove nodes from the end', () => {
            setup('<div><p id="1"></p><p id="2"></p><p id="3"></p></div>');
            fragmentMorpher(fromEl, '<div><p id="1"></p></div>', opts);
            expect(fromEl.children.length).toBe(1);
            expect(fromEl.children[0].id).toBe('1');
        });

        it('should remove nodes from the beginning', () => {
            setup('<div><p id="1"></p><p id="2"></p><p id="3"></p></div>');
            fragmentMorpher(fromEl, '<div><p id="3"></p></div>', opts);
            expect(fromEl.children.length).toBe(1);
            expect(fromEl.children[0].id).toBe('3');
        });

        it('should handle complex reordering and additions/removals', () => {
            setup('<div><p id="a"></p><b id="b"></b><i id="c"></i><span id="d"></span></div>');
            const p = fromEl.querySelector('#a');
            const b = fromEl.querySelector('#b');
            const i = fromEl.querySelector('#c');

            fragmentMorpher(fromEl, '<div><section id="e"></section><i id="c"></i><b id="b"></b><p id="a"></p></div>', opts);

            expect(fromEl.children.length).toBe(4);
            expect(fromEl.children[0].tagName).toBe('SECTION');
            expect(fromEl.children[1]).toBe(i);
            expect(fromEl.children[2]).toBe(b);
            expect(fromEl.children[3]).toBe(p);
        });
    });

    describe('Mixed Keyed and Un-keyed Children', () => {
        const getKey = (node: Node) => {
            if (node.nodeType === 1) {
                const id = (node as HTMLElement).id;
                return id ? id : null;
            }
            return null;
        };
        const opts = { getNodeKey: getKey };

        it('should handle reordering keyed nodes around un-keyed nodes', () => {
            setup(`
                <div>
                    <p id="a">A</p>
                    <span>un-keyed 1</span>
                    <p id="b">B</p>
                    <span>un-keyed 2</span>
                    <p id="c">C</p>
                </div>
            `);
            const pA = fromEl.querySelector('#a');
            const pB = fromEl.querySelector('#b');
            const pC = fromEl.querySelector('#c');
            const span1 = fromEl.children[1];
            const span2 = fromEl.children[3];

            fragmentMorpher(fromEl, `
                <div>
                    <p id="c">C</p>
                    <span>un-keyed 1</span>
                    <p id="a">A</p>
                    <span>un-keyed 2</span>
                    <p id="b">B</p>
                </div>
            `, opts);

            expect(fromEl.children.length).toBe(5);
            expect(fromEl.children[0]).toBe(pC);
            expect(fromEl.children[1]).toBe(span1);
            expect(fromEl.children[2]).toBe(pA);
            expect(fromEl.children[3]).toBe(span2);
            expect(fromEl.children[4]).toBe(pB);
        });
    });

    describe('childrenOnly Mode', () => {
        it('should only morph children, leaving root element and its attributes intact', () => {
            setup('<div id="host" class="old-class"><p>Old child</p></div>');
            const toNode = '<div id="new-host" class="new-class"><span>New child</span></div>';
            fragmentMorpher(fromEl, toNode, { childrenOnly: true });

            expect(fromEl.id).toBe('host');
            expect(fromEl.className).toBe('old-class');
            expect(fromEl.innerHTML).toBe('<span>New child</span>');
        });

        it('should add children to an empty root', () => {
            setup('<div id="host"></div>');
            const toNode = '<div><p>Child 1</p><p>Child 2</p></div>';
            fragmentMorpher(fromEl, toNode, { childrenOnly: true });

            expect(fromEl.id).toBe('host');
            expect(fromEl.children.length).toBe(2);
            expect(fromEl.children[0].textContent).toBe('Child 1');
        });

        it('should remove all children from a root', () => {
            setup('<div id="host"><p>Child 1</p><p>Child 2</p></div>');
            const toNode = '<div></div>';
            fragmentMorpher(fromEl, toNode, { childrenOnly: true });

            expect(fromEl.id).toBe('host');
            expect(fromEl.children.length).toBe(0);
        });
    });

    describe('Lifecycle Callbacks', () => {
        it('should call onNodeAdded for new nodes', () => {
            setup('<div></div>');
            const onNodeAdded = vi.fn();
            fragmentMorpher(fromEl, '<div><p></p><span></span></div>', { onNodeAdded });
            expect(onNodeAdded).toHaveBeenCalledTimes(2);
            expect((onNodeAdded.mock.calls[0][0] as HTMLElement).tagName).toBe('P');
            expect((onNodeAdded.mock.calls[1][0] as HTMLElement).tagName).toBe('SPAN');
        });

        it('should call onNodeDiscarded for removed nodes', () => {
            setup('<div><p></p><span></span></div>');
            const onNodeDiscarded = vi.fn();
            fragmentMorpher(fromEl, '<div></div>', { onNodeDiscarded });
            expect(onNodeDiscarded).toHaveBeenCalledTimes(2);
            expect((onNodeDiscarded.mock.calls[0][0] as HTMLElement).tagName).toBe('P');
            expect((onNodeDiscarded.mock.calls[1][0] as HTMLElement).tagName).toBe('SPAN');
        });

        it('should call onBeforeElUpdated before updating an element', () => {
            setup('<div class="old"></div>');
            const onBeforeElUpdated = vi.fn();
            fragmentMorpher(fromEl, '<div class="new"></div>', { onBeforeElUpdated });
            expect(onBeforeElUpdated).toHaveBeenCalledTimes(1);
        });

        it('should respect onBeforeElUpdated returning false', () => {
            setup('<div class="old"></div>');
            const onBeforeElUpdated = vi.fn(() => false);
            fragmentMorpher(fromEl, '<div class="new"></div>', { onBeforeElUpdated });
            expect(fromEl.className).toBe('old');
        });

        it('should respect onBeforeNodeDiscarded returning false', () => {
            setup('<div><p>Keep me</p></div>');
            const onBeforeNodeDiscarded = vi.fn(() => false);
            fragmentMorpher(fromEl, '<div></div>', { onBeforeNodeDiscarded });
            expect(fromEl.children.length).toBe(1);
            expect(fromEl.textContent).toBe('Keep me');
        });

        it('should respect onBeforeNodeAdded returning false', () => {
            setup('<div></div>');
            const onBeforeNodeAdded = vi.fn(() => false);
            fragmentMorpher(fromEl, '<div><p>Do not add me</p></div>', { onBeforeNodeAdded });
            expect(fromEl.children.length).toBe(0);
        });

        it('should respect onBeforeElChildrenUpdated returning false', () => {
            setup('<div><p>Old</p></div>');
            const onBeforeElChildrenUpdated = vi.fn((from, _to) => from.tagName !== 'DIV');
            fragmentMorpher(fromEl, '<div><span>New</span></div>', { onBeforeElChildrenUpdated });
            expect(fromEl.innerHTML).toBe('<p>Old</p>');
        });
    });

    describe('Special Element Handling (Form elements)', () => {
        it('should correctly update INPUT value', () => {
            setup('<input type="text" value="old">');
            fragmentMorpher(fromEl, '<input type="text" value="new">');
            expect((fromEl as HTMLInputElement).value).toBe('new');
        });

        it('should correctly update INPUT checked state', () => {
            setup('<input type="checkbox">');
            expect((fromEl as HTMLInputElement).checked).toBe(false);
            fragmentMorpher(fromEl, '<input type="checkbox" checked>');
            expect((fromEl as HTMLInputElement).checked).toBe(true);
            fragmentMorpher(fromEl, '<input type="checkbox">');
            expect((fromEl as HTMLInputElement).checked).toBe(false);
        });

        it('should correctly update TEXTAREA value', () => {
            setup('<textarea>old</textarea>');
            fragmentMorpher(fromEl, '<textarea>new</textarea>');
            expect((fromEl as HTMLTextAreaElement).value).toBe('new');
        });

        it('should correctly update SELECT value', () => {
            setup(`
        <select>
          <option value="a">A</option>
          <option value="b" selected>B</option>
        </select>
      `);
            expect((fromEl as HTMLSelectElement).value).toBe('b');

            fragmentMorpher(fromEl, `
        <select>
          <option value="a" selected>A</option>
          <option value="b">B</option>
        </select>
      `);
            expect((fromEl as HTMLSelectElement).value).toBe('a');
        });

        it('should correctly update multi-SELECT values', () => {
            setup(`
        <select multiple>
          <option value="a">A</option>
          <option value="b" selected>B</option>
          <option value="c">C</option>
        </select>
      `);
            fragmentMorpher(fromEl, `
        <select multiple>
          <option value="a" selected>A</option>
          <option value="b">B</option>
          <option value="c" selected>C</option>
        </select>
      `);
            const selectedOptions = Array.from((fromEl as HTMLSelectElement).options)
                .filter(o => o.selected)
                .map(o => o.value);
            expect(selectedOptions).toEqual(['a', 'c']);
        });
    });

    describe('Advanced Scenarios & Edge Cases', () => {
        it('should preserve focus on a morphed element', () => {
            const getKey = (node: Node) => {
                if (node.nodeType === 1) {
                    const id = (node as HTMLElement).id;
                    return id ? id : null;
                }
                return null;
            };
            setup('<div><input id="focusable" value="one"></div>');
            const input = fromEl.querySelector('#focusable') as HTMLInputElement;

            input.focus();
            input.value = "user-edited";

            expect(document.activeElement).toBe(input);

            fragmentMorpher(fromEl, '<div><input id="focusable" value="one" class="new"></div>', { getNodeKey: getKey });

            expect(document.activeElement).toBe(input);
            expect((document.activeElement as HTMLInputElement).value).toBe("user-edited");
        });

        it('should respect the "pk-no-refresh" attribute on an element and its children', () => {
            setup('<div pk-no-refresh><p>original</p></div>');
            fragmentMorpher(fromEl, '<div><p>updated</p></div>');
            expect(fromEl.innerHTML).toBe('<p>original</p>');
        });

        it('should allow a "pk-refresh" attribute to override a "pk-no-refresh" parent', () => {
            setup('<div pk-no-refresh><p pk-refresh>original</p></div>');
            const originalP = fromEl.querySelector('p');

            fragmentMorpher(fromEl, '<div pk-no-refresh><p pk-refresh class="new">updated</p></div>');

            const newP = fromEl.querySelector('p');
            expect(newP).toBe(originalP);
            expect(newP?.textContent).toBe('updated');
            expect(newP?.classList.contains('new')).toBe(true);
        });

        it('should handle deeply nested morphs correctly', () => {
            const getKey = (node: Node) => {
                if (node.nodeType === 1) {
                    const id = (node as HTMLElement).id;
                    return id ? id : null;
                }
                return null;
            };
            setup(`
                <ul id="list-1">
                    <li id="item-1">Item 1 <p>Nested old</p></li>
                    <li id="item-2">Item 2</li>
                </ul>
            `);
            const item1 = fromEl.querySelector('#item-1');

            fragmentMorpher(fromEl, `
                <ul id="list-1" class="changed">
                    <li id="item-2">Item 2 changed</li>
                    <li id="item-1">Item 1 changed <p>Nested new</p></li>
                </ul>
            `, { getNodeKey: getKey });

            expect(fromEl.className).toBe('changed');
            expect(fromEl.children.length).toBe(2);
            expect(fromEl.children[0].id).toBe('item-2');
            expect(fromEl.children[0].textContent).toBe('Item 2 changed');
            expect(fromEl.children[1]).toBe(item1);
            expect(fromEl.children[1].textContent).toBe('Item 1 changed Nested new');
        });

        it('should correctly handle SVG attributes', () => {
            setup('<svg viewBox="0 0 10 10"></svg>');
            fragmentMorpher(fromEl, '<svg viewBox="0 0 20 20" class="new-svg"></svg>');

            expect(fromEl.getAttribute('viewBox')).toBe('0 0 20 20');
            expect(fromEl.getAttribute('class')).toBe('new-svg');
        });
    });

    describe('childrenOnly Mode (Advanced)', () => {
        it('should correctly handle keyed children reordering within childrenOnly mode', () => {
            const getKey = (node: Node) => {
                if (node.nodeType === 1) {
                    const id = (node as HTMLElement).id;
                    return id ? id : null;
                }
                return null;
            };
            setup('<div id="host"><p id="a">A</p><p id="b">B</p></div>');
            const pA = fromEl.querySelector('#a');
            const pB = fromEl.querySelector('#b');

            fragmentMorpher(fromEl, '<div><p id="b">B</p><p id="a">A</p></div>', { childrenOnly: true, getNodeKey: getKey });

            expect(fromEl.id).toBe('host');
            expect(fromEl.children.length).toBe(2);
            expect(fromEl.children[0]).toBe(pB);
            expect(fromEl.children[1]).toBe(pA);
        });

        it('should handle morphing from children to a single text node in childrenOnly mode', () => {
            setup('<div><p>Old</p></div>');
            fragmentMorpher(fromEl, '<div>New Text</div>', { childrenOnly: true });
            expect(fromEl.innerHTML).toBe('New Text');
        });

        it('should handle morphing from a text node to elements in childrenOnly mode', () => {
            setup('<div>Old Text</div>');
            fragmentMorpher(fromEl, '<div><p>New</p></div>', { childrenOnly: true });
            expect(fromEl.innerHTML).toBe('<p>New</p>');
        });
    });

    describe('Focus Management', () => {
        it('should preserve focus on an element that is moved within the DOM', () => {
            const getKey = (node: Node) => {
                if (node.nodeType === 1) {
                    const id = (node as HTMLElement).id;
                    return id ? id : null;
                }
                return null;
            };
            setup('<div><input id="a" value="A"><input id="b" value="B"></div>');
            const inputB = fromEl.querySelector('#b') as HTMLInputElement;

            inputB.focus();
            expect(document.activeElement).toBe(inputB);

            fragmentMorpher(fromEl, '<div><input id="b" value="B"><input id="a" value="A"></div>', { getNodeKey: getKey });

            expect(document.activeElement).toBe(inputB);
            expect(fromEl.children[0]).toBe(inputB);
        });

        it('should NOT preserve focus if the focused element is removed from the DOM', () => {
            const getKey = (node: Node) => {
                if (node.nodeType === 1) {
                    const id = (node as HTMLElement).id;
                    return id ? id : null;
                }
                return null;
            };
            setup('<div><input id="a" value="A"><input id="b" value="B"></div>');
            const inputB = fromEl.querySelector('#b') as HTMLInputElement;

            inputB.focus();
            expect(document.activeElement).toBe(inputB);

            fragmentMorpher(fromEl, '<div><input id="a" value="A"></div>', { getNodeKey: getKey });

            expect(document.activeElement).not.toBe(inputB);
            expect(document.activeElement).toBe(document.body);
        });
    });

    describe('Whitespace and Text Node Handling', () => {
        it('should correctly handle morphing between text and elements with surrounding text', () => {
            setup('<div>Before <b>middle</b> After</div>');
            fragmentMorpher(fromEl, '<div>Before <i>middle</i> After</div>');

            expect(fromEl.childNodes.length).toBe(3);
            expect(fromEl.childNodes[0].nodeValue).toBe('Before ');
            expect((fromEl.childNodes[1] as HTMLElement).tagName).toBe('I');
            expect(fromEl.childNodes[2].nodeValue).toBe(' After');
        });

        it('should ignore insignificant whitespace differences', () => {
            setup('<div>  <p>Hello</p>  </div>');
            const originalP = fromEl.querySelector('p');

            fragmentMorpher(fromEl, '<div><p>Hello</p></div>');

            expect(fromEl.querySelector('p')).toBe(originalP);
        });
    });

    class MockFormElement extends HTMLElement {
        static formAssociated = true;
        _value = '';
        _updateFormState = vi.fn();

        get value() { return this._value; }
        set value(val) { this._value = val; }
    }
    customElements.define('mock-form-element', MockFormElement);

    describe('Custom Element Contract', () => {
        it('should call syncCustomElementState for a form-associated custom element', () => {
            setup('<mock-form-element></mock-form-element>');
            const syncSpy = vi.spyOn(fromEl as any, '_updateFormState');

            fragmentMorpher(fromEl, '<mock-form-element></mock-form-element>');

            expect(syncSpy).toHaveBeenCalled();
        });

        it('should update the value property of a custom element', () => {
            setup('<mock-form-element value="old"></mock-form-element>');
            const toEl = document.createElement('mock-form-element') as MockFormElement;
            toEl.value = "new";

            fragmentMorpher(fromEl, toEl);

            expect((fromEl as MockFormElement).value).toBe('new');
        });
    });

    describe('Malformed and Empty Inputs', () => {
        it('should handle morphing to an empty string', () => {
            setup('<div><p>Content</p></div>');
            fragmentMorpher(fromEl, '<div></div>');
            expect(fromEl.innerHTML).toBe('');
        });

        it('should handle morphing from an empty element', () => {
            setup('<div></div>');
            fragmentMorpher(fromEl, '<div><p>Content</p></div>');
            expect(fromEl.innerHTML).toBe('<p>Content</p>');
        });

        it('should handle multiple root nodes in the "to" fragment by wrapping them', () => {
            setup('<div>Old content</div>');
            fragmentMorpher(fromEl, '<p>1</p><p>2</p>');
            expect(host.innerHTML).toBe('<div><p>1</p><p>2</p></div>');
        });
    });
    describe('SVG Elements and Namespaced Attributes', () => {
        it('should correctly handle SVG elements with namespaced attributes', () => {
            setup('<svg xmlns:xlink="http://www.w3.org/1999/xlink"></svg>');
            fragmentMorpher(fromEl, '<svg xmlns:xlink="http://www.w3.org/1999/xlink"><path d="M10 10" fill="red"></path></svg>');

            expect(fromEl.namespaceURI).toBe('http://www.w3.org/2000/svg');
            expect(fromEl.firstChild?.nodeName).toBe('path');
            expect((fromEl.firstChild as SVGPathElement).getAttribute('fill')).toBe('red');
        });

        it('should correctly update namespaced attributes', () => {
            setup('<svg><use xlink:href="#icon-1"></use></svg>');
            fragmentMorpher(fromEl, '<svg><use xlink:href="#icon-2"></use></svg>');

            const useEl = fromEl.querySelector('use');
            expect(useEl?.getAttributeNS('http://www.w3.org/1999/xlink', 'href')).toBe('#icon-2');
        });
    });

    describe('Comment Nodes', () => {
        it('should correctly morph comment nodes', () => {
            setup('<div><!-- Old comment --></div>');
            fragmentMorpher(fromEl, '<div><!-- New comment --></div>');

            expect(fromEl.firstChild?.nodeType).toBe(8);
            expect(fromEl.firstChild?.nodeValue).toBe(' New comment ');
        });

        it('should preserve comment nodes when morphing', () => {
            setup('<div><!-- Comment -->Text</div>');
            fragmentMorpher(fromEl, '<div><!-- Comment -->Updated Text</div>');

            expect(fromEl.childNodes.length).toBe(2);
            expect(fromEl.childNodes[0].nodeType).toBe(8);
            expect(fromEl.childNodes[1].nodeValue).toBe('Updated Text');
        });
    });

    describe('Document Fragments', () => {
        it('should correctly morph from a document fragment', () => {
            setup('<div>Original</div>');

            const fragment = document.createDocumentFragment();
            const p1 = document.createElement('p');
            p1.textContent = 'Fragment 1';
            const p2 = document.createElement('p');
            p2.textContent = 'Fragment 2';
            fragment.appendChild(p1);
            fragment.appendChild(p2);

            fragmentMorpher(fromEl, fragment);

            expect(fromEl.childNodes.length).toBe(2);
            expect(fromEl.firstChild?.textContent).toBe('Fragment 1');
            expect(fromEl.lastChild?.textContent).toBe('Fragment 2');
        });
    });

    describe('Advanced Form Element Scenarios', () => {
        it('should handle file input elements correctly', () => {
            setup('<input type="file">');
            fragmentMorpher(fromEl, '<input type="file" class="new-class">');

            expect(fromEl.className).toBe('new-class');
            expect(fromEl.hasAttribute('value')).toBe(false);
        });

        it('should handle radio button groups correctly', () => {
            setup(`
                <div>
                    <input type="radio" name="group" value="a">
                    <input type="radio" name="group" value="b" checked>
                </div>
            `);

            const radioA = fromEl.querySelector('input[value="a"]') as HTMLInputElement;
            const radioB = fromEl.querySelector('input[value="b"]') as HTMLInputElement;

            fragmentMorpher(fromEl, `
                <div>
                    <input type="radio" name="group" value="a" checked>
                    <input type="radio" name="group" value="b">
                </div>
            `);

            expect(radioA.checked).toBe(true);
            expect(radioB.checked).toBe(false);
        });

        it('should handle disabled state on form elements', () => {
            setup('<input type="text">');
            fragmentMorpher(fromEl, '<input type="text" disabled>');

            expect((fromEl as HTMLInputElement).disabled).toBe(true);

            fragmentMorpher(fromEl, '<input type="text">');
            expect((fromEl as HTMLInputElement).disabled).toBe(false);
        });
    });

    describe('Complex Custom Elements', () => {
        class ComplexCustomElement extends HTMLElement {
            static formAssociated = true;
            _value = '';
            _checked = false;
            _disabled = false;
            _updateFormState = vi.fn();

            get value() { return this._value; }
            set value(val) { this._value = val; }

            get checked() { return this._checked; }
            set checked(val) { this._checked = val; }

            get disabled() { return this._disabled; }
            set disabled(val) { this._disabled = val; }
        }
        customElements.define('complex-custom-element', ComplexCustomElement);

        it('should sync multiple properties of a complex custom element', () => {
            setup('<complex-custom-element></complex-custom-element>');

            const toEl = document.createElement('complex-custom-element') as ComplexCustomElement;
            toEl.value = 'test value';
            toEl.checked = true;
            toEl.disabled = true;

            fragmentMorpher(fromEl, toEl);

            const fromCustomEl = fromEl as ComplexCustomElement;
            expect(fromCustomEl.value).toBe('test value');
            expect(fromCustomEl.checked).toBe(true);
            expect(fromCustomEl._updateFormState).toHaveBeenCalled();
        });
    });

    describe('Initial State Option', () => {
        it('should respect initialState="pk-no-refresh" option', () => {
            setup('<div><p>Original</p></div>');
            fragmentMorpher(fromEl, '<div><p>Updated</p></div>', { initialState: 'pk-no-refresh' });

            expect(fromEl.innerHTML).toBe('<p>Original</p>');
        });

        it('should allow initialState="pk-refresh" option', () => {
            setup('<div><p>Original</p></div>');
            fragmentMorpher(fromEl, '<div><p>Updated</p></div>', { initialState: 'pk-refresh' });

            expect(fromEl.innerHTML).toBe('<p>Updated</p>');
        });
    });

    describe('Error Handling', () => {
        it('should handle null or undefined inputs gracefully', () => {
            setup('<div>Original</div>');

            fragmentMorpher(fromEl, null);
            expect(fromEl.textContent).toBe('Original');

            fragmentMorpher(null, '<div>New</div>');
            expect(fromEl.textContent).toBe('Original');
        });
    });

    describe('Nested SVG Elements', () => {
        it('should correctly handle nested SVG elements', () => {
            setup(`
                <svg width="100" height="100">
                    <circle cx="50" cy="50" r="40" fill="blue"></circle>
                </svg>
            `);

            fragmentMorpher(fromEl, `
                <svg width="200" height="200">
                    <circle cx="50" cy="50" r="40" fill="red"></circle>
                    <rect x="10" y="10" width="30" height="30" fill="green"></rect>
                </svg>
            `);

            expect(fromEl.getAttribute('width')).toBe('200');
            expect(fromEl.getAttribute('height')).toBe('200');
            expect(fromEl.children.length).toBe(2);

            const circle = fromEl.querySelector('circle');
            expect(circle?.getAttribute('fill')).toBe('red');

            const rect = fromEl.querySelector('rect');
            expect(rect?.getAttribute('fill')).toBe('green');
        });
    });

    describe('Complex Attribute Changes', () => {
        it('should handle data attributes with JSON content', () => {
            setup('<div data-config=\'{"key":"old-value"}\'></div>');
            fragmentMorpher(fromEl, '<div data-config=\'{"key":"new-value","added":"property"}\'></div>');

            const config = JSON.parse(fromEl.getAttribute('data-config') || '{}');
            expect(config.key).toBe('new-value');
            expect(config.added).toBe('property');
        });

        it('should handle style attribute changes', () => {
            setup('<div style="color: red; font-size: 12px;"></div>');
            fragmentMorpher(fromEl, '<div style="color: blue; font-weight: bold;"></div>');

            expect(fromEl.style.color).toBe('blue');
            expect(fromEl.style.fontSize).toBe(''); 
            expect(fromEl.style.fontWeight).toBe('bold'); 
        });
    });

    describe('HTML Entities and Special Characters', () => {
        it('should correctly handle HTML entities in text nodes', () => {
            setup('<div>Regular text</div>');
            fragmentMorpher(fromEl, '<div>Text with &lt;entities&gt; &amp; special chars</div>');

            expect(fromEl.innerHTML).toBe('Text with &lt;entities&gt; &amp; special chars');
            expect(fromEl.textContent).toBe('Text with <entities> & special chars');
        });

        it('should correctly handle special characters in attributes', () => {
            setup('<div title="Old title"></div>');
            fragmentMorpher(fromEl, '<div title="Title with &quot;quotes&quot; and &lt;brackets&gt;"></div>');

            expect(fromEl.getAttribute('title')).toBe('Title with "quotes" and <brackets>');
        });
    });

    describe('Refresh Attribute Behaviour', () => {
        it('should pk-refresh elements by default (without any pk-refresh attributes)', () => {
            setup('<div><p>Original</p></div>');
            fragmentMorpher(fromEl, '<div><p>Updated</p></div>');
            expect(fromEl.innerHTML).toBe('<p>Updated</p>');
        });

        it('should respect nested pk-no-refresh attributes at multiple levels', () => {
            setup(`
                <div>
                    <section>
                        <article pk-no-refresh>
                            <h1>Original Title</h1>
                            <p>Original paragraph</p>
                        </article>
                        <aside>
                            <p>Original sidebar</p>
                        </aside>
                    </section>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div>
                    <section>
                        <article pk-no-refresh>
                            <h1>Updated Title</h1>
                            <p>Updated paragraph</p>
                        </article>
                        <aside>
                            <p>Updated sidebar</p>
                        </aside>
                    </section>
                </div>
            `);

            const article = fromEl.querySelector('article');
            expect(article?.querySelector('h1')?.textContent).toBe('Original Title');
            expect(article?.querySelector('p')?.textContent).toBe('Original paragraph');

            expect(fromEl.querySelector('aside p')?.textContent).toBe('Updated sidebar');
        });

        it('should allow pk-refresh attributes to override pk-no-refresh at multiple levels of nesting', () => {
            setup(`
                <div pk-no-refresh>
                    <section>
                        <article>
                            <h1>Original Title</h1>
                            <p pk-refresh>Original paragraph</p>
                            <div>
                                <span>Original span</span>
                                <em pk-refresh>Original emphasis</em>
                            </div>
                        </article>
                    </section>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div pk-no-refresh>
                    <section>
                        <article>
                            <h1>Updated Title</h1>
                            <p pk-refresh>Updated paragraph</p>
                            <div>
                                <span>Updated span</span>
                                <em pk-refresh>Updated emphasis</em>
                            </div>
                        </article>
                    </section>
                </div>
            `);

            expect(fromEl.querySelector('p')?.textContent).toBe('Updated paragraph');
            expect(fromEl.querySelector('em')?.textContent).toBe('Updated emphasis');

            expect(fromEl.querySelector('h1')?.textContent).toBe('Original Title');
            expect(fromEl.querySelector('span')?.textContent).toBe('Original span');
        });

        it('should handle refresh attributes on form elements correctly', () => {
            setup(`
                <form pk-no-refresh>
                    <input type="text" value="original" pk-refresh>
                    <input type="checkbox" checked>
                    <select>
                        <option value="1" selected>Option 1</option>
                        <option value="2">Option 2</option>
                    </select>
                    <textarea pk-refresh>Original text</textarea>
                </form>
            `);

            fragmentMorpher(fromEl, `
                <form pk-no-refresh>
                    <input type="text" value="updated" pk-refresh>
                    <input type="checkbox">
                    <select>
                        <option value="1">Option 1</option>
                        <option value="2" selected>Option 2</option>
                    </select>
                    <textarea pk-refresh>Updated text</textarea>
                </form>
            `);

            expect((fromEl.querySelector('input[type="text"]') as HTMLInputElement).value).toBe('updated');
            expect(fromEl.querySelector('textarea')?.textContent).toBe('Updated text');

            expect((fromEl.querySelector('input[type="checkbox"]') as HTMLInputElement).checked).toBe(true);
            expect((fromEl.querySelector('select') as HTMLSelectElement).value).toBe('1');
        });

        it('should handle dynamically added pk-refresh attributes', () => {
            setup('<div pk-no-refresh><p>Original</p></div>');

            fragmentMorpher(fromEl, '<div pk-no-refresh><p>Updated</p></div>');
            expect(fromEl.querySelector('p')?.textContent).toBe('Original');

            fromEl.querySelector('p')?.setAttribute('pk-refresh', '');

            fragmentMorpher(fromEl, '<div pk-no-refresh><p pk-refresh>Updated again</p></div>');
            expect(fromEl.querySelector('p')?.textContent).toBe('Updated again');
        });

        it('should handle dynamically removed refresh attributes', () => {
            setup('<div pk-no-refresh><p pk-refresh>Original</p></div>');

            fragmentMorpher(fromEl, '<div pk-no-refresh><p pk-refresh>Updated</p></div>');
            expect(fromEl.querySelector('p')?.textContent).toBe('Updated');

            fromEl.querySelector('p')?.removeAttribute('pk-refresh');

            fragmentMorpher(fromEl, '<div pk-no-refresh><p>Updated again</p></div>');
            expect(fromEl.querySelector('p')?.textContent).toBe('Updated');
        });

        it('should handle pk-refresh attributes with keyed elements', () => {
            const getKey = (node: Node) => {
                if (node.nodeType === 1) {
                    const id = (node as HTMLElement).id;
                    return id ? id : null;
                }
                return null;
            };

            setup(`
                <div pk-no-refresh>
                    <p id="p1" pk-refresh>Original 1</p>
                    <p id="p2">Original 2</p>
                    <p id="p3" pk-refresh>Original 3</p>
                </div>
            `);

            const p1 = fromEl.querySelector('#p1');
            const p2 = fromEl.querySelector('#p2');
            const p3 = fromEl.querySelector('#p3');

            fragmentMorpher(fromEl, `
                <div pk-no-refresh>
                    <p id="p3" pk-refresh>Updated 3</p>
                    <p id="p1" pk-refresh>Updated 1</p>
                    <p id="p2">Updated 2</p>
                </div>
            `, { getNodeKey: getKey });

            expect(fromEl.children[0]).toBe(p3);
            expect(fromEl.children[1]).toBe(p1);
            expect(fromEl.children[2]).toBe(p2);

            expect(p1?.textContent).toBe('Updated 1');
            expect(p2?.textContent).toBe('Original 2');
            expect(p3?.textContent).toBe('Updated 3');
        });

        it('should handle pk-refresh attributes with childrenOnly option', () => {
            setup(`
                <div id="root" pk-no-refresh>
                    <p>Original paragraph</p>
                    <section pk-refresh>
                        <h2>Original heading</h2>
                    </section>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div id="new-root">
                    <p>Updated paragraph</p>
                    <section>
                        <h2>Updated heading</h2>
                    </section>
                </div>
            `, { childrenOnly: true });

            expect(fromEl.id).toBe('root');

            expect(fromEl.querySelector('p')?.textContent).toBe('Original paragraph');

            expect(fromEl.querySelector('section h2')?.textContent).toBe('Updated heading');
        });

        it('should handle pk-refresh attributes with initialState option', () => {
            setup(`
                <div>
                    <p>Original paragraph</p>
                    <section pk-refresh>
                        <h2>Original heading</h2>
                    </section>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div>
                    <p>Updated paragraph</p>
                    <section>
                        <h2>Updated heading</h2>
                    </section>
                </div>
            `, { initialState: 'pk-no-refresh' });

            expect(fromEl.querySelector('p')?.textContent).toBe('Original paragraph');

            expect(fromEl.querySelector('section h2')?.textContent).toBe('Updated heading');
        });

        it('should handle complex nested structures with mixed pk-refresh attributes', () => {
            setup(`
                <div class="container">
                    <header pk-no-refresh>
                        <h1>Original Title</h1>
                        <nav pk-refresh>
                            <ul>
                                <li>Original Item 1</li>
                                <li pk-no-refresh>Original Item 2</li>
                                <li>Original Item 3</li>
                            </ul>
                        </nav>
                    </header>
                    <main>
                        <article pk-no-refresh>
                            <h2>Original Article</h2>
                            <p pk-refresh>Original paragraph 1</p>
                            <p>Original paragraph 2</p>
                        </article>
                        <aside>
                            <div pk-no-refresh>
                                <h3>Original Sidebar</h3>
                                <ul pk-refresh>
                                    <li>Original Sidebar Item 1</li>
                                    <li>Original Sidebar Item 2</li>
                                </ul>
                            </div>
                        </aside>
                    </main>
                    <footer pk-refresh>
                        <p>Original Footer</p>
                        <div pk-no-refresh>
                            <p>Original Copyright</p>
                        </div>
                    </footer>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div class="container updated">
                    <header pk-no-refresh>
                        <h1>Updated Title</h1>
                        <nav pk-refresh>
                            <ul>
                                <li>Updated Item 1</li>
                                <li pk-no-refresh>Updated Item 2</li>
                                <li>Updated Item 3</li>
                            </ul>
                        </nav>
                    </header>
                    <main>
                        <article pk-no-refresh>
                            <h2>Updated Article</h2>
                            <p pk-refresh>Updated paragraph 1</p>
                            <p>Updated paragraph 2</p>
                        </article>
                        <aside>
                            <div pk-no-refresh>
                                <h3>Updated Sidebar</h3>
                                <ul pk-refresh>
                                    <li>Updated Sidebar Item 1</li>
                                    <li>Updated Sidebar Item 2</li>
                                </ul>
                            </div>
                        </aside>
                    </main>
                    <footer pk-refresh>
                        <p>Updated Footer</p>
                        <div pk-no-refresh>
                            <p>Updated Copyright</p>
                        </div>
                    </footer>
                </div>
            `);

            expect(fromEl.className).toBe('container updated');
            expect(fromEl.querySelector('header h1')?.textContent).toBe('Original Title');
            expect(fromEl.querySelector('nav li:first-child')?.textContent).toBe('Updated Item 1');
            expect(fromEl.querySelector('nav li:nth-child(2)')?.textContent).toBe('Original Item 2');
            expect(fromEl.querySelector('nav li:nth-child(3)')?.textContent).toBe('Updated Item 3');
            expect(fromEl.querySelector('article h2')?.textContent).toBe('Original Article');
            expect(fromEl.querySelector('article p:first-of-type')?.textContent).toBe('Updated paragraph 1');
            expect(fromEl.querySelector('article p:nth-of-type(2)')?.textContent).toBe('Original paragraph 2');
            expect(fromEl.querySelector('aside h3')?.textContent).toBe('Original Sidebar');
            expect(fromEl.querySelector('aside li:first-child')?.textContent).toBe('Updated Sidebar Item 1');
            expect(fromEl.querySelector('footer > p')?.textContent).toBe('Updated Footer');
            expect(fromEl.querySelector('footer div p')?.textContent).toBe('Original Copyright');
        });

        it('should handle refresh attributes with attribute changes only', () => {
            setup(`
                <div pk-no-refresh>
                    <button 
                        id="btn1" 
                        class="btn" 
                        data-count="0" 
                        aria-pressed="false"
                        pk-refresh
                    >
                        Click me
                    </button>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div pk-no-refresh>
                    <button 
                        id="btn1" 
                        class="btn active" 
                        data-count="1" 
                        aria-pressed="true"
                        pk-refresh
                    >
                        Click me
                    </button>
                </div>
            `);

            const button = fromEl.querySelector('button');

            expect(button?.textContent?.trim()).toBe('Click me');
            expect(button?.className).toBe('btn active');
            expect(button?.getAttribute('data-count')).toBe('1');
            expect(button?.getAttribute('aria-pressed')).toBe('true');
        });

        it('should handle refresh attributes with structural changes only', () => {
            setup(`
                <div pk-no-refresh>
                    <ul pk-refresh>
                        <li>Item 1</li>
                        <li>Item 2</li>
                    </ul>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div pk-no-refresh>
                    <ul pk-refresh>
                        <li>Item 1</li>
                        <li>Item 2</li>
                        <li>Item 3</li>
                    </ul>
                </div>
            `);

            const ul = fromEl.querySelector('ul');

            expect(ul?.children.length).toBe(3);
            expect(ul?.lastElementChild?.textContent).toBe('Item 3');
        });

        it('should handle refresh attributes with SVG elements', () => {
            setup(`
                <div pk-no-refresh>
                    <svg pk-refresh viewBox="0 0 100 100">
                        <circle cx="50" cy="50" r="40" fill="blue"></circle>
                    </svg>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div pk-no-refresh>
                    <svg pk-refresh viewBox="0 0 200 200">
                        <circle cx="50" cy="50" r="40" fill="red"></circle>
                        <rect x="10" y="10" width="30" height="30" fill="green"></rect>
                    </svg>
                </div>
            `);

            const svg = fromEl.querySelector('svg');
            const circle = fromEl.querySelector('circle');
            const rect = fromEl.querySelector('rect');

            expect(svg?.getAttribute('viewBox')).toBe('0 0 200 200');
            expect(circle?.getAttribute('fill')).toBe('red');
            expect(rect).not.toBeNull();
            expect(rect?.getAttribute('fill')).toBe('green');
        });

        it('should handle refresh attributes with event listeners', () => {
            setup(`
                <div pk-no-refresh>
                    <button pk-refresh>Click me</button>
                </div>
            `);

            const button = fromEl.querySelector('button');
            const clickHandler = vi.fn();
            button?.addEventListener('click', clickHandler);

            fragmentMorpher(fromEl, `
                <div pk-no-refresh>
                    <button pk-refresh class="updated">Updated button</button>
                </div>
            `);

            const updatedButton = fromEl.querySelector('button');

            expect(updatedButton?.className).toBe('updated');
            expect(updatedButton?.textContent).toBe('Updated button');

            updatedButton?.click();
            expect(clickHandler).toHaveBeenCalledTimes(1);
        });
    });

    describe('SVG and Namespaced Attribute Morphing', () => {
        const setupWithDefs = (html: string) => {
            host = document.createElement('div');
            host.innerHTML = `
            <svg style="display: none;">
                <defs>
                    <symbol id="icon-1" viewBox="0 0 24 24"><path d="M1..."></path></symbol>
                    <symbol id="icon-2" viewBox="0 0 24 24"><path d="M2..."></path></symbol>
                    <symbol id="icon-3" viewBox="0 0 24 24"><path d="M3..."></path></symbol>
                </defs>
            </svg>
            ${html.trim()}
        `;
            fromEl = host.lastElementChild as HTMLElement;
            document.body.appendChild(host);
        };

        it('should correctly ADD a new namespaced attribute', () => {
            setupWithDefs('<svg><use></use></svg>');
            const useEl = fromEl.querySelector('use');

            expect(useEl?.hasAttributeNS('http://www.w3.org/1999/xlink', 'href')).toBe(false);

            fragmentMorpher(fromEl, '<svg><use xlink:href="#icon-3"></use></svg>');

            expect(useEl?.getAttributeNS('http://www.w3.org/1999/xlink', 'href')).toBe('#icon-3');
        });

        it('should correctly REMOVE an existing namespaced attribute', () => {
            setupWithDefs('<svg><use xlink:href="#icon-1" class="old"></use></svg>');
            const useEl = fromEl.querySelector('use');

            expect(useEl?.getAttributeNS('http://www.w3.org/1999/xlink', 'href')).toBe('#icon-1');

            fragmentMorpher(fromEl, '<svg><use class="new"></use></svg>');

            expect(useEl?.hasAttributeNS('http://www.w3.org/1999/xlink', 'href')).toBe(false);
            expect(useEl?.getAttribute('class')).toBe('new');
        });

        it('should handle a mix of regular and namespaced attribute updates', () => {
            setupWithDefs('<svg viewBox="0 0 10 10"><use xlink:href="#icon-1" fill="red"></use></svg>');
            const useEl = fromEl.querySelector('use');

            fragmentMorpher(fromEl, '<svg viewBox="0 0 20 20"><use xlink:href="#icon-2" fill="blue"></use></svg>');

            expect(fromEl.getAttribute('viewBox')).toBe('0 0 20 20');

            expect(useEl?.getAttributeNS('http://www.w3.org/1999/xlink', 'href')).toBe('#icon-2');

            expect(useEl?.getAttribute('fill')).toBe('blue');
        });
    });

    describe('Mixed Content Nodes', () => {
        it('should correctly handle mixed content (text and elements)', () => {
            setup('<div>Start <b>bold</b> middle <i>italic</i> end</div>');
            fragmentMorpher(fromEl, '<div>New start <b>new bold</b> new middle <i>new italic</i> new end</div>');

            expect(fromEl.childNodes.length).toBe(5);
            expect(fromEl.childNodes[0].nodeValue).toBe('New start ');
            expect((fromEl.childNodes[1] as HTMLElement).tagName).toBe('B');
            expect(fromEl.childNodes[1].textContent).toBe('new bold');
            expect(fromEl.childNodes[2].nodeValue).toBe(' new middle ');
            expect((fromEl.childNodes[3] as HTMLElement).tagName).toBe('I');
            expect(fromEl.childNodes[3].textContent).toBe('new italic');
            expect(fromEl.childNodes[4].nodeValue).toBe(' new end');
        });

        it('should handle complex mixed content with nested elements', () => {
            setup(`
                <article>
                    <h1>Original Title</h1>
                    Text before
                    <section>
                        <h2>Section Title</h2>
                        <p>Paragraph <em>with emphasis</em> and <strong>strong text</strong></p>
                    </section>
                    Text after
                </article>
            `);

            fragmentMorpher(fromEl, `
                <article>
                    <h1>Updated Title</h1>
                    New text before
                    <section>
                        <h2>Updated Section</h2>
                        <p>New paragraph <em>with new emphasis</em> and <strong>new strong text</strong></p>
                        <p>Additional paragraph</p>
                    </section>
                    New text after
                </article>
            `);

            expect(fromEl.querySelector('h1')?.textContent).toBe('Updated Title');
            expect(fromEl.querySelector('h2')?.textContent).toBe('Updated Section');
            expect(fromEl.querySelector('em')?.textContent).toBe('with new emphasis');
            expect(fromEl.querySelector('section')?.children.length).toBe(3);

            const articleChildNodes = Array.from(fromEl.childNodes).filter(node =>
                node.nodeType === Node.TEXT_NODE && node.textContent?.trim());
            expect(articleChildNodes.length).toBe(2);
            expect(articleChildNodes[0].textContent?.trim()).toBe('New text before');
            expect(articleChildNodes[1].textContent?.trim()).toBe('New text after');
        });
    });

    describe('Performance and Real-World Edge Cases', () => {
        it('should handle tables with complex structure', () => {
            setup(`
                <table>
                    <thead>
                        <tr><th>Header 1</th><th>Header 2</th></tr>
                    </thead>
                    <tbody>
                        <tr><td>Cell 1</td><td>Cell 2</td></tr>
                    </tbody>
                </table>
            `);

            fragmentMorpher(fromEl, `
                <table>
                    <thead>
                        <tr><th>New Header 1</th><th>New Header 2</th><th>New Header 3</th></tr>
                    </thead>
                    <tbody>
                        <tr><td>New Cell 1</td><td>New Cell 2</td><td>New Cell 3</td></tr>
                        <tr><td>New Cell 4</td><td>New Cell 5</td><td>New Cell 6</td></tr>
                    </tbody>
                    <tfoot>
                        <tr><td colspan="3">Footer</td></tr>
                    </tfoot>
                </table>
            `);

            expect(fromEl.querySelectorAll('th').length).toBe(3);
            expect(fromEl.querySelectorAll('tbody tr').length).toBe(2);
            expect(fromEl.querySelector('tfoot')).not.toBeNull();
            expect(fromEl.querySelector('th')?.textContent).toBe('New Header 1');
            expect(fromEl.querySelector('tfoot td')?.getAttribute('colspan')).toBe('3');
        });

        it('should handle script and style elements correctly', () => {
            setup(`
                <div>
                    <script type="text/javascript">
                        var x = 1;
                    </script>
                    <style>
                        body { color: red; }
                    </style>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div>
                    <script type="text/javascript">
                        var x = 2;
                    </script>
                    <style>
                        body { color: blue; }
                    </style>
                </div>
            `);

            const script = fromEl.querySelector('script');
            const style = fromEl.querySelector('style');

            expect(script?.textContent?.includes('var x = 2')).toBe(true);
            expect(style?.textContent?.includes('color: blue')).toBe(true);
        });

        it('should handle empty text nodes and whitespace correctly', () => {
            setup(`
                <div>
                    <p>  Text with spaces  </p>
                    <p>
                        Text with
                        line breaks
                    </p>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div>
                    <p>New text</p>
                    <p>
                        New text with
                        new line breaks
                    </p>
                </div>
            `);

            const paragraphs = fromEl.querySelectorAll('p');
            expect(paragraphs[0].textContent?.trim()).toBe('New text');
            expect(paragraphs[1].textContent?.trim().replace(/\s+/g, ' ')).toBe('New text with new line breaks');
        });

        it('should handle deeply nested DOM structures', () => {
            let nestedHtml = '<div class="level-0">';
            for (let i = 1; i <= 10; i++) {
                nestedHtml += `<div class="level-${i}">`;
            }
            nestedHtml += 'Inner content';
            for (let i = 10; i >= 0; i--) {
                nestedHtml += '</div>';
            }

            setup(nestedHtml);

            let newNestedHtml = '<div class="level-0 modified">';
            for (let i = 1; i <= 10; i++) {
                newNestedHtml += `<div class="level-${i} modified">`;
            }
            newNestedHtml += 'Modified inner content';
            for (let i = 10; i >= 0; i--) {
                newNestedHtml += '</div>';
            }

            fragmentMorpher(fromEl, newNestedHtml);

            expect(fromEl.className).toBe('level-0 modified');

            let currentEl = fromEl;
            for (let i = 1; i <= 10; i++) {
                currentEl = currentEl.firstElementChild as HTMLElement;
                expect(currentEl.className).toBe(`level-${i} modified`);
            }

            expect(currentEl.textContent).toBe('Modified inner content');
        });

        it('should handle large DOM structures efficiently', () => {
            let largeListHtml = '<ul>';
            for (let i = 0; i < 100; i++) {
                largeListHtml += `<li id="item-${i}">Item ${i}</li>`;
            }
            largeListHtml += '</ul>';

            setup(largeListHtml);

            let newLargeListHtml = '<ul class="modified">';
            for (let i = 100; i < 150; i++) {
                newLargeListHtml += `<li id="item-${i}">New Item ${i}</li>`;
            }
            for (let i = 99; i >= 0; i--) {
                newLargeListHtml += `<li id="item-${i}">Updated Item ${i}</li>`;
            }
            newLargeListHtml += '</ul>';

            const getNodeKey = (node: Node) => {
                if (node.nodeType === 1) {
                    return (node as HTMLElement).id;
                }
                return null;
            };

            fragmentMorpher(fromEl, newLargeListHtml, { getNodeKey });

            expect(fromEl.className).toBe('modified');
            expect(fromEl.childNodes.length).toBe(150);

            for (let i = 0; i < 100; i++) {
                const item = fromEl.querySelector(`#item-${i}`);
                expect(item?.textContent).toBe(`Updated Item ${i}`);
            }

            for (let i = 100; i < 150; i++) {
                const item = fromEl.querySelector(`#item-${i}`);
                expect(item?.textContent).toBe(`New Item ${i}`);
            }
        });
    });

    describe('Complex Web Page Structures', () => {
        it('should handle a complete web page layout with semantic elements', () => {
            setup(`
                <div class="page-container">
                    <header>
                        <h1>Original Site Title</h1>
                        <nav>
                            <ul>
                                <li><a href="#home">Home</a></li>
                                <li><a href="#about">About</a></li>
                                <li><a href="#contact">Contact</a></li>
                            </ul>
                        </nav>
                    </header>
                    <main>
                        <article>
                            <h2>Original Article Title</h2>
                            <p>Original paragraph 1.</p>
                            <p>Original paragraph 2.</p>
                        </article>
                        <aside>
                            <h3>Sidebar</h3>
                            <ul>
                                <li>Sidebar item 1</li>
                                <li>Sidebar item 2</li>
                            </ul>
                        </aside>
                    </main>
                    <footer>
                        <p>&copy; 2023 Original Footer</p>
                    </footer>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div class="page-container updated">
                    <header class="new-header">
                        <h1>Updated Site Title</h1>
                        <nav>
                            <ul>
                                <li><a href="#home">Home</a></li>
                                <li><a href="#about">About</a></li>
                                <li><a href="#contact">Contact</a></li>
                                <li><a href="#blog">Blog</a></li>
                            </ul>
                        </nav>
                    </header>
                    <main>
                        <article>
                            <h2>Updated Article Title</h2>
                            <p>Updated paragraph 1.</p>
                            <p>Updated paragraph 2.</p>
                            <p>New paragraph 3.</p>
                        </article>
                        <aside>
                            <h3>Updated Sidebar</h3>
                            <ul>
                                <li>Updated sidebar item 1</li>
                                <li>Updated sidebar item 2</li>
                                <li>New sidebar item 3</li>
                            </ul>
                        </aside>
                    </main>
                    <footer>
                        <p>&copy; 2023 Updated Footer</p>
                        <div class="social-links">
                            <a href="#">Twitter</a>
                            <a href="#">Facebook</a>
                        </div>
                    </footer>
                </div>
            `);

            expect(fromEl.className).toBe('page-container updated');
            expect(fromEl.querySelector('header')?.className).toBe('new-header');
            expect(fromEl.querySelector('h1')?.textContent).toBe('Updated Site Title');

            expect(fromEl.querySelectorAll('nav li').length).toBe(4);
            expect(fromEl.querySelector('nav li:last-child a')?.textContent).toBe('Blog');
            expect(fromEl.querySelectorAll('article p').length).toBe(3);
            expect(fromEl.querySelectorAll('aside li').length).toBe(3);
            expect(fromEl.querySelector('.social-links')).not.toBeNull();

            expect(fromEl.querySelector('article h2')?.textContent).toBe('Updated Article Title');
            expect(fromEl.querySelector('aside h3')?.textContent).toBe('Updated Sidebar');
            expect(fromEl.querySelector('footer p')?.textContent).toBe('© 2023 Updated Footer');
        });

        it('should handle complex nested forms with various input types', () => {
            setup(`
                <form id="registration">
                    <fieldset>
                        <legend>Personal Information</legend>
                        <div class="form-group">
                            <label for="name">Name:</label>
                            <input type="text" id="name" name="name" required>
                        </div>
                        <div class="form-group">
                            <label for="email">Email:</label>
                            <input type="email" id="email" name="email" required>
                        </div>
                    </fieldset>
                    <fieldset>
                        <legend>Preferences</legend>
                        <div class="checkbox-group">
                            <input type="checkbox" id="pref1" name="preferences" value="1">
                            <label for="pref1">Preference 1</label>
                        </div>
                        <div class="checkbox-group">
                            <input type="checkbox" id="pref2" name="preferences" value="2" checked>
                            <label for="pref2">Preference 2</label>
                        </div>
                    </fieldset>
                    <button type="submit">Register</button>
                </form>
            `);

            fragmentMorpher(fromEl, `
                <form id="registration" class="enhanced-form">
                    <fieldset>
                        <legend>Personal Information</legend>
                        <div class="form-group">
                            <label for="name">Full Name:</label>
                            <input type="text" id="name" name="name" required placeholder="Enter your full name">
                            <span class="help-text">Please enter your legal name</span>
                        </div>
                        <div class="form-group">
                            <label for="email">Email Address:</label>
                            <input type="email" id="email" name="email" required placeholder="your@email.com">
                        </div>
                        <div class="form-group">
                            <label for="phone">Phone Number:</label>
                            <input type="tel" id="phone" name="phone" pattern="[0-9]{3}-[0-9]{3}-[0-9]{4}">
                        </div>
                    </fieldset>
                    <fieldset>
                        <legend>Preferences</legend>
                        <div class="checkbox-group">
                            <input type="checkbox" id="pref1" name="preferences" value="1" checked>
                            <label for="pref1">Preference 1 (updated)</label>
                        </div>
                        <div class="checkbox-group">
                            <input type="checkbox" id="pref2" name="preferences" value="2">
                            <label for="pref2">Preference 2 (updated)</label>
                        </div>
                        <div class="checkbox-group">
                            <input type="checkbox" id="pref3" name="preferences" value="3">
                            <label for="pref3">Preference 3 (new)</label>
                        </div>
                    </fieldset>
                    <fieldset>
                        <legend>Additional Information</legend>
                        <div class="form-group">
                            <label for="comments">Comments:</label>
                            <textarea id="comments" name="comments" rows="4"></textarea>
                        </div>
                        <div class="form-group">
                            <label for="source">How did you hear about us?</label>
                            <select id="source" name="source">
                                <option value="">Please select</option>
                                <option value="search">Search Engine</option>
                                <option value="social">Social Media</option>
                                <option value="friend">Friend</option>
                            </select>
                        </div>
                    </fieldset>
                    <div class="form-actions">
                        <button type="submit" class="primary">Register Now</button>
                        <button type="reset" class="secondary">Reset Form</button>
                    </div>
                </form>
            `);

            expect(fromEl.className).toBe('enhanced-form');
            expect(fromEl.querySelectorAll('fieldset').length).toBe(3);
            expect(fromEl.querySelectorAll('.form-group').length).toBe(5);
            expect(fromEl.querySelectorAll('.checkbox-group').length).toBe(3);

            expect(fromEl.querySelector('input[type="text"]')?.getAttribute('placeholder')).toBe('Enter your full name');
            expect(fromEl.querySelector('input[type="email"]')?.getAttribute('placeholder')).toBe('your@email.com');
            expect(fromEl.querySelector('input[type="tel"]')).not.toBeNull();
            expect(fromEl.querySelector('textarea')).not.toBeNull();
            expect(fromEl.querySelector('select')).not.toBeNull();
            expect(fromEl.querySelectorAll('option').length).toBe(4);

            expect((fromEl.querySelector('#pref1') as HTMLInputElement).checked).toBe(true);
            expect((fromEl.querySelector('#pref2') as HTMLInputElement).checked).toBe(false);

            expect(fromEl.querySelectorAll('button').length).toBe(2);
            expect(fromEl.querySelector('button.primary')?.textContent).toBe('Register Now');
            expect(fromEl.querySelector('button.secondary')?.textContent).toBe('Reset Form');
        });

        it('should handle accessibility-rich structures with ARIA attributes', () => {
            setup(`
                <div role="dialog" aria-labelledby="dialog-title" aria-describedby="dialog-desc">
                    <h2 id="dialog-title">Original Dialog Title</h2>
                    <p id="dialog-desc">Original dialog description.</p>
                    <div role="tablist">
                        <button role="tab" aria-selected="true" aria-controls="panel1" id="tab1">Tab 1</button>
                        <button role="tab" aria-selected="false" aria-controls="panel2" id="tab2">Tab 2</button>
                    </div>
                    <div role="tabpanel" id="panel1" aria-labelledby="tab1">
                        <p>Original Tab 1 content</p>
                    </div>
                    <div role="tabpanel" id="panel2" aria-labelledby="tab2" hidden>
                        <p>Original Tab 2 content</p>
                    </div>
                    <button aria-label="Close dialog">×</button>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div role="dialog" aria-labelledby="dialog-title" aria-describedby="dialog-desc" aria-modal="true">
                    <h2 id="dialog-title">Updated Dialog Title</h2>
                    <p id="dialog-desc">Updated dialog description with <a href="#" aria-label="Learn more about accessibility">more information</a>.</p>
                    <div role="tablist" aria-orientation="horizontal">
                        <button role="tab" aria-selected="false" aria-controls="panel1" id="tab1">Tab 1 Updated</button>
                        <button role="tab" aria-selected="true" aria-controls="panel2" id="tab2">Tab 2 Updated</button>
                        <button role="tab" aria-selected="false" aria-controls="panel3" id="tab3">Tab 3 New</button>
                    </div>
                    <div role="tabpanel" id="panel1" aria-labelledby="tab1" hidden>
                        <p>Updated Tab 1 content</p>
                    </div>
                    <div role="tabpanel" id="panel2" aria-labelledby="tab2">
                        <p>Updated Tab 2 content</p>
                        <ul aria-label="Options list">
                            <li>Option 1</li>
                            <li>Option 2</li>
                        </ul>
                    </div>
                    <div role="tabpanel" id="panel3" aria-labelledby="tab3" hidden>
                        <p>New Tab 3 content</p>
                    </div>
                    <div class="actions">
                        <button aria-label="Save changes">Save</button>
                        <button aria-label="Close dialog" class="close-btn">×</button>
                    </div>
                </div>
            `);

            expect(fromEl.getAttribute('aria-modal')).toBe('true');
            expect(fromEl.querySelector('#dialog-title')?.textContent).toBe('Updated Dialog Title');

            expect(fromEl.querySelector('[role="tablist"]')?.getAttribute('aria-orientation')).toBe('horizontal');
            expect(fromEl.querySelectorAll('[role="tab"]').length).toBe(3);

            expect(fromEl.querySelector('#tab1')?.getAttribute('aria-selected')).toBe('false');
            expect(fromEl.querySelector('#tab2')?.getAttribute('aria-selected')).toBe('true');

            expect(fromEl.querySelectorAll('[role="tabpanel"]').length).toBe(3);
            expect(fromEl.querySelector('#panel1')?.hasAttribute('hidden')).toBe(true);
            expect(fromEl.querySelector('#panel2')?.hasAttribute('hidden')).toBe(false);

            expect(fromEl.querySelector('a[aria-label="Learn more about accessibility"]')).not.toBeNull();
            expect(fromEl.querySelector('ul[aria-label="Options list"]')).not.toBeNull();
            expect(fromEl.querySelector('.actions')).not.toBeNull();
            expect(fromEl.querySelector('button[aria-label="Save changes"]')).not.toBeNull();
        });

        it('should handle dynamic content structures like pagination', () => {
            setup(`
                <div class="paginated-content">
                    <div class="items-container">
                        <div class="item" id="item1">Item 1 content</div>
                        <div class="item" id="item2">Item 2 content</div>
                        <div class="item" id="item3">Item 3 content</div>
                    </div>
                    <div class="pagination">
                        <button class="prev" disabled>Previous</button>
                        <span class="page-indicator">Page 1 of 1</span>
                        <button class="next" disabled>Next</button>
                    </div>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div class="paginated-content">
                    <div class="items-container">
                        <div class="item" id="item4">Item 4 content (new page)</div>
                        <div class="item" id="item5">Item 5 content (new page)</div>
                        <div class="item" id="item6">Item 6 content (new page)</div>
                    </div>
                    <div class="pagination">
                        <button class="prev">Previous</button>
                        <div class="page-numbers">
                            <button class="page" data-page="1">1</button>
                            <button class="page active" data-page="2">2</button>
                            <button class="page" data-page="3">3</button>
                        </div>
                        <span class="page-indicator">Page 2 of 3</span>
                        <button class="next">Next</button>
                    </div>
                </div>
            `);

            expect(fromEl.querySelectorAll('.item').length).toBe(3);
            expect(fromEl.querySelector('#item1')).toBeNull();
            expect(fromEl.querySelector('#item4')).not.toBeNull();

            expect(fromEl.querySelector('.prev')?.hasAttribute('disabled')).toBe(false);
            expect(fromEl.querySelector('.next')?.hasAttribute('disabled')).toBe(false);
            expect(fromEl.querySelector('.page-indicator')?.textContent).toBe('Page 2 of 3');

            expect(fromEl.querySelector('.page-numbers')).not.toBeNull();
            expect(fromEl.querySelectorAll('.page').length).toBe(3);
            expect(fromEl.querySelector('.page.active')?.getAttribute('data-page')).toBe('2');
        });

        it('should handle complex nested components with conditional rendering', () => {
            setup(`
                <div class="app-container">
                    <div class="user-profile" data-state="logged-out">
                        <div class="login-prompt">
                            <h3>Please log in</h3>
                            <button class="login-btn">Log In</button>
                        </div>
                    </div>
                    <div class="content-area">
                        <div class="placeholder-content">
                            <p>Content will appear after login</p>
                        </div>
                    </div>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div class="app-container">
                    <div class="user-profile" data-state="logged-in">
                        <div class="user-info">
                            <img src="avatar.jpg" alt="User avatar" class="avatar">
                            <div class="user-details">
                                <h3>Welcome, Jane Doe</h3>
                                <p>jane.doe@example.com</p>
                                <span class="badge">Premium</span>
                            </div>
                            <div class="user-actions">
                                <button class="settings-btn">Settings</button>
                                <button class="logout-btn">Log Out</button>
                            </div>
                        </div>
                    </div>
                    <div class="content-area">
                        <div class="user-dashboard">
                            <div class="dashboard-header">
                                <h2>Your Dashboard</h2>
                                <div class="dashboard-controls">
                                    <select class="view-selector">
                                        <option value="grid">Grid View</option>
                                        <option value="list" selected>List View</option>
                                    </select>
                                    <button class="refresh-btn">Refresh</button>
                                </div>
                            </div>
                            <div class="dashboard-content list-view">
                                <div class="dashboard-item">
                                    <h4>Item One</h4>
                                    <p>Description for item one</p>
                                    <div class="item-actions">
                                        <button>Edit</button>
                                        <button>Delete</button>
                                    </div>
                                </div>
                                <div class="dashboard-item">
                                    <h4>Item Two</h4>
                                    <p>Description for item two</p>
                                    <div class="item-actions">
                                        <button>Edit</button>
                                        <button>Delete</button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            `);

            expect(fromEl.querySelector('.user-profile')?.getAttribute('data-state')).toBe('logged-in');

            expect(fromEl.querySelector('.login-prompt')).toBeNull();
            expect(fromEl.querySelector('.user-info')).not.toBeNull();
            expect(fromEl.querySelector('.avatar')).not.toBeNull();
            expect(fromEl.querySelector('.user-details h3')?.textContent).toBe('Welcome, Jane Doe');

            expect(fromEl.querySelector('.placeholder-content')).toBeNull();
            expect(fromEl.querySelector('.user-dashboard')).not.toBeNull();

            expect(fromEl.querySelector('.dashboard-header')).not.toBeNull();
            expect(fromEl.querySelector('.dashboard-controls')).not.toBeNull();
            expect(fromEl.querySelector('.view-selector')).not.toBeNull();
            expect(fromEl.querySelectorAll('.dashboard-item').length).toBe(2);
            expect(fromEl.querySelectorAll('.item-actions button').length).toBe(4);

            const select = fromEl.querySelector('.view-selector') as HTMLSelectElement;
            expect(select.value).toBe('list');
        });

        it('should handle extremely complex nested structures with multiple levels', () => {
            let complexHtml = `
                <div class="mega-structure" id="root">
                    <header class="mega-header">
                        <div class="logo-container">
                            <img src="logo.png" alt="Logo" class="logo">
                            <span class="logo-text">MegaCorp</span>
                        </div>
                        <nav class="mega-nav">
                            <ul class="nav-list">`;

            for (let i = 1; i <= 10; i++) {
                complexHtml += `
                                <li class="nav-item" id="nav-${i}">
                                    <a href="#section-${i}" class="nav-link">Section ${i}</a>
                                    <div class="dropdown">
                                        <ul class="dropdown-list">`;

                for (let j = 1; j <= 5; j++) {
                    complexHtml += `
                                            <li class="dropdown-item" id="dropdown-${i}-${j}">
                                                <a href="#subsection-${i}-${j}">Subsection ${i}.${j}</a>
                                            </li>`;
                }

                complexHtml += `
                                        </ul>
                                    </div>
                                </li>`;
            }

            complexHtml += `
                            </ul>
                        </nav>
                    </header>
                    <main class="mega-main">
                        <div class="content-grid">`;

            for (let i = 1; i <= 5; i++) {
                complexHtml += `
                            <section class="content-section" id="section-${i}">
                                <h2 class="section-title">Section ${i} Title</h2>
                                <div class="section-content">`;

                for (let j = 1; j <= 3; j++) {
                    complexHtml += `
                                    <article class="content-article" id="article-${i}-${j}">
                                        <h3 class="article-title">Article ${i}.${j} Title</h3>
                                        <div class="article-body">
                                            <p>Paragraph 1 for article ${i}.${j}</p>
                                            <p>Paragraph 2 for article ${i}.${j}</p>
                                            <ul class="article-list">`;

                    for (let k = 1; k <= 4; k++) {
                        complexHtml += `
                                                <li class="article-list-item" id="list-${i}-${j}-${k}">List item ${i}.${j}.${k}</li>`;
                    }

                    complexHtml += `
                                            </ul>
                                        </div>
                                    </article>`;
                }

                complexHtml += `
                                </div>
                            </section>`;
            }

            complexHtml += `
                        </div>
                        <aside class="mega-sidebar">
                            <div class="sidebar-widgets">`;

            for (let i = 1; i <= 3; i++) {
                complexHtml += `
                                <div class="widget" id="widget-${i}">
                                    <h4 class="widget-title">Widget ${i}</h4>
                                    <div class="widget-content">`;

                for (let j = 1; j <= 3; j++) {
                    complexHtml += `
                                        <div class="widget-item" id="widget-item-${i}-${j}">
                                            <span class="item-label">Item ${i}.${j}</span>
                                            <span class="item-value">Value ${i}.${j}</span>
                                        </div>`;
                }

                complexHtml += `
                                    </div>
                                </div>`;
            }

            complexHtml += `
                            </div>
                        </aside>
                    </main>
                    <footer class="mega-footer">
                        <div class="footer-sections">`;

            for (let i = 1; i <= 4; i++) {
                complexHtml += `
                            <div class="footer-section" id="footer-${i}">
                                <h5 class="footer-title">Footer Section ${i}</h5>
                                <ul class="footer-links">`;

                for (let j = 1; j <= 5; j++) {
                    complexHtml += `
                                    <li class="footer-link-item">
                                        <a href="#footer-link-${i}-${j}" class="footer-link">Footer Link ${i}.${j}</a>
                                    </li>`;
                }

                complexHtml += `
                                </ul>
                            </div>`;
            }

            complexHtml += `
                        </div>
                        <div class="copyright">
                            <p>&copy; 2023 MegaCorp. All rights reserved.</p>
                        </div>
                    </footer>
                </div>
            `;

            setup(complexHtml);

            let modifiedHtml = complexHtml
                .replace('class="mega-structure"', 'class="mega-structure modified"')
                .replace('List item 3.2.4', 'MODIFIED List item 3.2.4')
                .replace('<h4 class="widget-title">Widget 2</h4>', '<h4 class="widget-title">MODIFIED Widget 2</h4>')
                .replace('</ul>\n                        </nav>', '<li class="nav-item" id="nav-11"><a href="#section-11" class="nav-link">NEW Section 11</a></li>\n                            </ul>\n                        </nav>')
                .replace('<li class="footer-link-item">\n                                        <a href="#footer-link-4-3" class="footer-link">Footer Link 4.3</a>\n                                    </li>', '')
                .replace('<p>&copy; 2023 MegaCorp. All rights reserved.</p>', '<p>&copy; 2023 MegaCorp. All rights reserved.</p><p class="privacy-policy">Privacy Policy | Terms of Service</p>');

            fragmentMorpher(fromEl, modifiedHtml);

            expect(fromEl.className).toBe('mega-structure modified');

            const modifiedListItem = fromEl.querySelector('#list-3-2-4');
            expect(modifiedListItem?.textContent).toBe('MODIFIED List item 3.2.4');

            const widgetTitle = fromEl.querySelector('#widget-2 .widget-title');
            expect(widgetTitle?.textContent).toBe('MODIFIED Widget 2');

            const navItems = fromEl.querySelectorAll('.nav-item');
            expect(navItems.length).toBe(11);
            expect(navItems[10].id).toBe('nav-11');

            const footerSection4Links = fromEl.querySelectorAll('#footer-4 .footer-link-item');
            expect(footerSection4Links.length).toBe(4);

            const privacyPolicy = fromEl.querySelector('.privacy-policy');
            expect(privacyPolicy).not.toBeNull();
            expect(privacyPolicy?.textContent).toBe('Privacy Policy | Terms of Service');

            expect(fromEl.querySelectorAll('section.content-section').length).toBe(5);
            expect(fromEl.querySelectorAll('article.content-article').length).toBe(15);
            expect(fromEl.querySelectorAll('.article-list-item').length).toBe(60);
            expect(fromEl.querySelectorAll('.widget').length).toBe(3);
            expect(fromEl.querySelectorAll('.footer-section').length).toBe(4);
        });

        it('should handle complex interactive components with dynamic content', () => {
            setup(`
                <div class="interactive-widget" id="data-explorer">
                    <header class="widget-header">
                        <h2 class="widget-title">Data Explorer</h2>
                        <div class="widget-controls">
                            <button class="refresh-btn" aria-label="Refresh data">↻</button>
                            <button class="settings-btn" aria-label="Settings">⚙</button>
                        </div>
                    </header>
                    <div class="widget-body">
                        <div class="filters-panel">
                            <div class="filter-group">
                                <label for="date-range">Date Range:</label>
                                <select id="date-range" name="date-range">
                                    <option value="today">Today</option>
                                    <option value="week" selected>This Week</option>
                                    <option value="month">This Month</option>
                                    <option value="year">This Year</option>
                                </select>
                            </div>
                            <div class="filter-group">
                                <label for="category">Category:</label>
                                <select id="category" name="category">
                                    <option value="all" selected>All Categories</option>
                                    <option value="sales">Sales</option>
                                    <option value="marketing">Marketing</option>
                                    <option value="support">Support</option>
                                </select>
                            </div>
                            <button class="apply-filters-btn">Apply Filters</button>
                        </div>
                        <div class="data-visualization">
                            <div class="chart-container">
                                <div class="chart-placeholder">
                                    <div class="bar-chart">
                                        <div class="chart-bar" style="height: 30%;" data-value="30">
                                            <span class="bar-label">Mon</span>
                                        </div>
                                        <div class="chart-bar" style="height: 45%;" data-value="45">
                                            <span class="bar-label">Tue</span>
                                        </div>
                                        <div class="chart-bar" style="height: 70%;" data-value="70">
                                            <span class="bar-label">Wed</span>
                                        </div>
                                        <div class="chart-bar" style="height: 90%;" data-value="90">
                                            <span class="bar-label">Thu</span>
                                        </div>
                                        <div class="chart-bar" style="height: 60%;" data-value="60">
                                            <span class="bar-label">Fri</span>
                                        </div>
                                    </div>
                                </div>
                                <div class="chart-legend">
                                    <div class="legend-item">
                                        <span class="legend-color" style="background-color: blue;"></span>
                                        <span class="legend-label">Visitors</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="data-table">
                            <table>
                                <thead>
                                    <tr>
                                        <th>Day</th>
                                        <th>Visitors</th>
                                        <th>Conversion</th>
                                        <th>Revenue</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr>
                                        <td>Monday</td>
                                        <td>1,234</td>
                                        <td>5.7%</td>
                                        <td>$1,500</td>
                                    </tr>
                                    <tr>
                                        <td>Tuesday</td>
                                        <td>2,100</td>
                                        <td>6.2%</td>
                                        <td>$2,300</td>
                                    </tr>
                                    <tr>
                                        <td>Wednesday</td>
                                        <td>3,590</td>
                                        <td>7.1%</td>
                                        <td>$3,700</td>
                                    </tr>
                                    <tr>
                                        <td>Thursday</td>
                                        <td>4,200</td>
                                        <td>8.4%</td>
                                        <td>$4,800</td>
                                    </tr>
                                    <tr>
                                        <td>Friday</td>
                                        <td>3,500</td>
                                        <td>6.8%</td>
                                        <td>$3,200</td>
                                    </tr>
                                </tbody>
                                <tfoot>
                                    <tr>
                                        <td>Total</td>
                                        <td>14,624</td>
                                        <td>6.84%</td>
                                        <td>$15,500</td>
                                    </tr>
                                </tfoot>
                            </table>
                        </div>
                    </div>
                    <footer class="widget-footer">
                        <div class="pagination">
                            <button class="page-btn" disabled>Previous</button>
                            <span class="page-indicator">Page 1 of 1</span>
                            <button class="page-btn" disabled>Next</button>
                        </div>
                        <div class="widget-status">
                            <span class="status-indicator">Last updated: 5 minutes ago</span>
                        </div>
                    </footer>
                </div>
            `);

            fragmentMorpher(fromEl, `
                <div class="interactive-widget" id="data-explorer">
                    <header class="widget-header">
                        <h2 class="widget-title">Data Explorer</h2>
                        <div class="widget-controls">
                            <button class="refresh-btn" aria-label="Refresh data">↻</button>
                            <button class="settings-btn" aria-label="Settings">⚙</button>
                            <button class="export-btn" aria-label="Export data">↓</button>
                        </div>
                    </header>
                    <div class="widget-body">
                        <div class="filters-panel expanded">
                            <div class="filter-group">
                                <label for="date-range">Date Range:</label>
                                <select id="date-range" name="date-range">
                                    <option value="today">Today</option>
                                    <option value="week">This Week</option>
                                    <option value="month" selected>This Month</option>
                                    <option value="year">This Year</option>
                                    <option value="custom">Custom Range</option>
                                </select>
                            </div>
                            <div class="filter-group">
                                <label for="category">Category:</label>
                                <select id="category" name="category">
                                    <option value="all">All Categories</option>
                                    <option value="sales" selected>Sales</option>
                                    <option value="marketing">Marketing</option>
                                    <option value="support">Support</option>
                                </select>
                            </div>
                            <div class="filter-group">
                                <label for="region">Region:</label>
                                <select id="region" name="region">
                                    <option value="all" selected>All Regions</option>
                                    <option value="north">North</option>
                                    <option value="south">South</option>
                                    <option value="east">East</option>
                                    <option value="west">West</option>
                                </select>
                            </div>
                            <button class="apply-filters-btn">Apply Filters</button>
                            <button class="reset-filters-btn">Reset</button>
                        </div>
                        <div class="data-visualization">
                            <div class="chart-container">
                                <div class="chart-placeholder">
                                    <div class="bar-chart">
                                        <div class="chart-bar" style="height: 20%;" data-value="20">
                                            <span class="bar-label">Week 1</span>
                                        </div>
                                        <div class="chart-bar" style="height: 35%;" data-value="35">
                                            <span class="bar-label">Week 2</span>
                                        </div>
                                        <div class="chart-bar" style="height: 50%;" data-value="50">
                                            <span class="bar-label">Week 3</span>
                                        </div>
                                        <div class="chart-bar active" style="height: 80%;" data-value="80">
                                            <span class="bar-label">Week 4</span>
                                        </div>
                                    </div>
                                </div>
                                <div class="chart-legend">
                                    <div class="legend-item">
                                        <span class="legend-color" style="background-color: blue;"></span>
                                        <span class="legend-label">Visitors</span>
                                    </div>
                                    <div class="legend-item">
                                        <span class="legend-color" style="background-color: green;"></span>
                                        <span class="legend-label">Conversions</span>
                                    </div>
                                </div>
                            </div>
                            <div class="chart-tabs">
                                <button class="chart-tab active" data-chart="bar">Bar Chart</button>
                                <button class="chart-tab" data-chart="line">Line Chart</button>
                                <button class="chart-tab" data-chart="pie">Pie Chart</button>
                            </div>
                        </div>
                        <div class="data-table">
                            <table>
                                <thead>
                                    <tr>
                                        <th>Week</th>
                                        <th>Visitors</th>
                                        <th>Conversion</th>
                                        <th>Revenue</th>
                                        <th>Average. Order</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr>
                                        <td>Week 1</td>
                                        <td>5,234</td>
                                        <td>4.2%</td>
                                        <td>$6,500</td>
                                        <td>$29.50</td>
                                    </tr>
                                    <tr>
                                        <td>Week 2</td>
                                        <td>7,100</td>
                                        <td>5.1%</td>
                                        <td>$9,300</td>
                                        <td>$25.70</td>
                                    </tr>
                                    <tr>
                                        <td>Week 3</td>
                                        <td>8,590</td>
                                        <td>6.3%</td>
                                        <td>$12,700</td>
                                        <td>$23.40</td>
                                    </tr>
                                    <tr class="highlighted">
                                        <td>Week 4</td>
                                        <td>10,200</td>
                                        <td>7.8%</td>
                                        <td>$18,800</td>
                                        <td>$23.70</td>
                                    </tr>
                                </tbody>
                                <tfoot>
                                    <tr>
                                        <td>Total</td>
                                        <td>31,124</td>
                                        <td>5.85%</td>
                                        <td>$47,300</td>
                                        <td>$25.58</td>
                                    </tr>
                                </tfoot>
                            </table>
                        </div>
                    </div>
                    <footer class="widget-footer">
                        <div class="pagination">
                            <button class="page-btn">Previous</button>
                            <div class="page-numbers">
                                <button class="page-number active">1</button>
                                <button class="page-number">2</button>
                                <button class="page-number">3</button>
                            </div>
                            <button class="page-btn">Next</button>
                        </div>
                        <div class="widget-status">
                            <span class="status-indicator">Last updated: Just now</span>
                            <button class="auto-refresh-toggle active" title="Auto-refresh is on">
                                <span class="toggle-label">Auto</span>
                            </button>
                        </div>
                    </footer>
                </div>
            `);

            expect(fromEl.querySelectorAll('.widget-controls button').length).toBe(3);
            expect(fromEl.querySelector('.export-btn')).not.toBeNull();

            expect(fromEl.querySelector('.filters-panel')?.classList.contains('expanded')).toBe(true);
            expect(fromEl.querySelectorAll('.filter-group').length).toBe(3);
            expect(fromEl.querySelector('#region')).not.toBeNull();
            expect(fromEl.querySelector('.reset-filters-btn')).not.toBeNull();

            expect((fromEl.querySelector('#date-range') as HTMLSelectElement).value).toBe('month');
            expect((fromEl.querySelector('#category') as HTMLSelectElement).value).toBe('sales');

            expect(fromEl.querySelectorAll('.chart-bar').length).toBe(4);
            expect(fromEl.querySelector('.chart-bar.active')).not.toBeNull();
            expect(fromEl.querySelectorAll('.legend-item').length).toBe(2);
            expect(fromEl.querySelector('.chart-tabs')).not.toBeNull();

            expect(fromEl.querySelectorAll('thead th').length).toBe(5);
            expect(fromEl.querySelectorAll('tbody tr').length).toBe(4);
            expect(fromEl.querySelector('tr.highlighted')).not.toBeNull();

            expect(fromEl.querySelector('.page-numbers')).not.toBeNull();
            expect(fromEl.querySelectorAll('.page-number').length).toBe(3);
            expect(fromEl.querySelector('.auto-refresh-toggle')).not.toBeNull();
            expect(fromEl.querySelector('.status-indicator')?.textContent).toBe('Last updated: Just now');
        });
    });

    describe('Real-World Scenarios with Custom Elements and Slots', () => {
        class SlottedContainer extends HTMLElement {
            constructor() {
                super();
                const shadow = this.attachShadow({ mode: 'open' });
                const slot = document.createElement('slot');
                shadow.appendChild(slot);
            }
        }
        customElements.define('slotted-container', SlottedContainer);

        it('should correctly morph children of a custom element container using a slot', () => {
            setup(`
            <slotted-container>
                <div class="split">
                    <p id="a">Original A</p>
                    <p id="b">Original B</p>
                </div>
            </slotted-container>
        `);
            const originalSplit = fromEl.querySelector('.split');
            const originalP_A = fromEl.querySelector('#a');

            const toHtml = `
            <slotted-container>
                <div class="split" data-changed="true">
                    <p id="b">Original B</p>
                    <p id="a">Updated A</p>
                </div>
                <div class="footer">New Footer</div>
            </slotted-container>
        `;
            const toEl = toElement(toHtml) as HTMLElement;

            fragmentMorpher(fromEl, toEl, {
                childrenOnly: true,
                getNodeKey: (node: Node) => {
                    if (node.nodeType === 1) return (node as HTMLElement).id || null;
                    return null;
                }
            });

            expect(fromEl.children.length).toBe(2);

            const newSplit = fromEl.querySelector('.split');
            const newP_A = fromEl.querySelector('#a');
            expect(newSplit).toBe(originalSplit);
            expect(newP_A).toBe(originalP_A);

            expect(newSplit?.getAttribute('data-changed')).toBe('true');
            expect(newP_A?.textContent).toBe('Updated A');
            expect(fromEl.querySelector('.footer')).not.toBeNull();
            expect(fromEl.lastElementChild?.textContent).toBe('New Footer');

            const splitChildren = Array.from(newSplit?.children || []);
            expect(splitChildren[0].id).toBe('b');
            expect(splitChildren[1].id).toBe('a');
        });
    });

    describe('preservePartialScopes', () => {
        it('should preserve parent scopes when updating existing element with partial', () => {
            setup('<div partial="list_scope modal_scope page_scope">Content</div>');
            fragmentMorpher(fromEl, '<div partial="list_scope_new">Updated</div>', {
                preservePartialScopes: true
            });
            expect(fromEl.getAttribute('partial')).toBe('list_scope_new modal_scope page_scope');
            expect(fromEl.textContent).toBe('Updated');
        });

        it('should preserve parent scopes in childrenOnly mode', () => {
            setup(`<div class="container">
                <div class="item" partial="item_scope modal_scope page_scope">Item 1</div>
            </div>`);
            const container = fromEl;
            fragmentMorpher(container, `<div class="container">
                <div class="item" partial="item_scope_new">Item 1 Updated</div>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });
            const item = container.querySelector('.item');
            expect(item?.getAttribute('partial')).toBe('item_scope_new modal_scope page_scope');
        });

        it('should inherit parent scopes from existing children when adding new elements', () => {
            setup(`<div class="container">
                <div class="existing" partial="existing_scope modal_scope page_scope">Existing</div>
            </div>`);
            const container = fromEl;
            fragmentMorpher(container, `<div class="container">
                <div class="existing" partial="existing_scope">Existing</div>
                <div class="new-item" partial="new_scope">New Item</div>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });
            const existing = container.querySelector('.existing');
            const newItem = container.querySelector('.new-item');
            expect(existing?.getAttribute('partial')).toBe('existing_scope modal_scope page_scope');
            expect(newItem?.getAttribute('partial')).toBe('new_scope modal_scope page_scope');
        });

        it('should NOT modify scopes when no existing children have partial attribute', () => {
            setup('<div class="container"></div>');
            const container = fromEl;
            fragmentMorpher(container, `<div class="container">
                <div class="item" partial="server_scope">New Item</div>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });
            const item = container.querySelector('.item');
            expect(item?.getAttribute('partial')).toBe('server_scope');
        });

        it('should handle nested partials correctly', () => {
            setup(`<div class="list" partial="list_scope modal_scope page_scope">
                <div class="row" partial="row1_scope list_scope modal_scope page_scope">Row 1</div>
                <div class="row" partial="row2_scope list_scope modal_scope page_scope">Row 2</div>
            </div>`);
            const list = fromEl;
            fragmentMorpher(list, `<div class="list" partial="list_scope">
                <div class="row" partial="row1_scope">Row 1 Updated</div>
                <div class="row" partial="row2_scope">Row 2 Updated</div>
                <div class="row" partial="row3_scope">Row 3 New</div>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });
            const rows = list.querySelectorAll('.row');
            expect(rows[0].getAttribute('partial')).toBe('row1_scope list_scope modal_scope page_scope');
            expect(rows[1].getAttribute('partial')).toBe('row2_scope list_scope modal_scope page_scope');
            expect(rows[2].getAttribute('partial')).toBe('row3_scope list_scope modal_scope page_scope');
        });

        it('should handle single-scope partial attribute correctly', () => {
            setup('<div partial="self_scope">Content</div>');
            fragmentMorpher(fromEl, '<div partial="new_scope">Updated</div>', {
                preservePartialScopes: true
            });
            expect(fromEl.getAttribute('partial')).toBe('new_scope');
        });

        it('should not affect elements without partial attribute', () => {
            setup('<div class="no-partial">Content</div>');
            fragmentMorpher(fromEl, '<div class="no-partial updated">Updated</div>', {
                preservePartialScopes: true
            });
            expect(fromEl.hasAttribute('partial')).toBe(false);
            expect(fromEl.className).toBe('no-partial updated');
        });

        it('should preserve scopes deeply in the tree', () => {
            setup(`<div class="container">
                <div class="wrapper">
                    <div class="deep" partial="deep_scope parent1 parent2">Deep Content</div>
                </div>
            </div>`);
            fragmentMorpher(fromEl, `<div class="container">
                <div class="wrapper">
                    <div class="deep" partial="deep_new">Deep Updated</div>
                </div>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });
            const deep = fromEl.querySelector('.deep');
            expect(deep?.getAttribute('partial')).toBe('deep_new parent1 parent2');
        });

        it('should NOT inherit scopes from slotted content', () => {
            setup(`<div class="modal">
                <button slot="actions" partial="page_scope">Cancel</button>
                <button slot="actions" partial="page_scope">Continue</button>
            </div>`);

            fragmentMorpher(fromEl, `<div class="modal">
                <form partial="modal_content_scope">
                    <input type="text" />
                </form>
                <button slot="actions" partial="new_action_scope">Save</button>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });

            const form = fromEl.querySelector('form');
            expect(form?.getAttribute('partial')).toBe('modal_content_scope');

            const saveButton = fromEl.querySelector('button[slot="actions"]');
            expect(saveButton?.getAttribute('partial')).toBe('new_action_scope');
        });

        it('should inherit scopes from non-slotted children only', () => {
            setup(`<div class="container">
                <button slot="header" partial="page_scope">Header Button</button>
                <div class="content" partial="content_scope parent_scope">Existing Content</div>
            </div>`);

            fragmentMorpher(fromEl, `<div class="container">
                <button slot="header" partial="new_header">New Header</button>
                <div class="content" partial="new_content">New Content</div>
                <div class="added" partial="added_scope">Added Element</div>
            </div>`, {
                childrenOnly: true,
                preservePartialScopes: true
            });

            const content = fromEl.querySelector('.content');
            expect(content?.getAttribute('partial')).toBe('new_content parent_scope');

            const added = fromEl.querySelector('.added');
            expect(added?.getAttribute('partial')).toBe('added_scope parent_scope');
        });
    });
});

function toElement(html: string): Node {
    const template = document.createElement('template');

    template.innerHTML = html.trim();

    if (template.content.childNodes.length === 1) {
        return template.content.firstChild!;
    }

    return template.content;
}