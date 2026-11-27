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

import {describe, it, expect, beforeEach, afterEach} from 'vitest';
import {PPElement, type PropTypeDefinition} from '@/element';
import {dom} from '@/vdom';
import type {VirtualNode} from '@/vdom';
import {makeReactive, type ReactiveContext} from '@/reactivity';

const SERVER_PAGINATOR_TAG_WITHOUT_PAGE_INFO = '<paginate-markup-test-element partial="partials_lists_item_table_3a9c1d42" ' +
    'partial_name="lists/item_table" partial_src="/_piko/partials/lists/item_table" p-key="r.0:0:1:2" ' +
    'filter_input="#item_filter" current_page="0" filter="" items_per_page="25" total_pages="141">' +
    '</paginate-markup-test-element>';

interface PaginatorState {
    filter: string;
    filter_input: string;
    current_page: number;
    items_per_page: number;
    total_pages: number;
    show_page_info: boolean;
}

class PaginateMarkupTestElement extends PPElement {
    static override get propTypes(): Record<string, PropTypeDefinition> | undefined {
        return {
            filter: {type: 'string'},
            filter_input: {type: 'string'},
            current_page: {type: 'number'},
            items_per_page: {type: 'number'},
            total_pages: {type: 'number'},
            show_page_info: {type: 'boolean', default: true},
        };
    }

    initLikeCompiledComponent(): void {
        const initialState: PaginatorState = {
            filter: '',
            filter_input: '',
            current_page: 0,
            items_per_page: 0,
            total_pages: 0,
            show_page_info: true,
        };
        this.init({
            state: makeReactive(initialState, this as unknown as ReactiveContext) as unknown as Record<string, unknown>,
            $$initialState: {...initialState},
        });
    }

    override renderVDOM(): VirtualNode {
        return dom.el('div', 'paginator-root', {});
    }
}
customElements.define('paginate-markup-test-element', PaginateMarkupTestElement);

function parseServerElement(html: string): Element {
    const template = document.createElement('template');
    template.innerHTML = html;
    return template.content.firstElementChild as Element;
}

describe('PPElement attributes it adds from its own state', () => {
    let paginator: PaginateMarkupTestElement;

    beforeEach(() => {
        document.body.innerHTML = SERVER_PAGINATOR_TAG_WITHOUT_PAGE_INFO;
        paginator = document.body.firstElementChild as PaginateMarkupTestElement;
        paginator.initLikeCompiledComponent();
    });

    afterEach(() => {
        document.body.innerHTML = '';
    });

    it('reports the attribute of a default it reflects as added by itself', () => {
        expect(paginator.getAttribute('show_page_info')).toBe('');
        expect(paginator.hasSelfAddedAttribute('show_page_info')).toBe(true);
    });

    it('does not report the attributes the server markup set', () => {
        const server = parseServerElement(SERVER_PAGINATOR_TAG_WITHOUT_PAGE_INFO);

        for (const attributeName of server.getAttributeNames()) {
            expect(paginator.hasSelfAddedAttribute(attributeName), attributeName).toBe(false);
        }
    });

    it('stops reporting an attribute once something else sets it', () => {
        paginator.setAttribute('show_page_info', 'false');

        expect(paginator.hasSelfAddedAttribute('show_page_info')).toBe(false);
        expect(paginator.state?.show_page_info).toBe(false);
    });

    it('still takes the removal of an attribute it added as false for a boolean', () => {
        paginator.removeAttribute('show_page_info');

        expect(paginator.state?.show_page_info).toBe(false);
        expect(paginator.hasSelfAddedAttribute('show_page_info')).toBe(false);
    });
});
