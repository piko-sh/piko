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

/**
 * Base virtual node structure.
 */
export interface VirtualNodeBase {
    /** The node type discriminator. */
    _type: 'text' | 'element' | 'fragment' | 'comment';
    /** The text content for text and comment nodes. */
    text?: string;
    /** The HTML tag name for element nodes. */
    tag?: string;
    /** The props associated with this node. */
    props?: Record<string, unknown>;
    /** The child virtual nodes. */
    children?: VirtualNode[] | null;
}

/**
 * Virtual node with DOM element reference and key.
 */
export interface VirtualNode extends VirtualNodeBase {
    /** The real DOM node backing this virtual node. */
    elm?: Node;
    /** The stable key used for diffing. */
    key?: string;
    /** Raw HTML content to inject via innerHTML. */
    html?: string;
}

/**
 * An element virtual node with a required tag name.
 *
 * Narrows VirtualNode for cases where the node is known to be an element,
 * making `tag` non-optional and eliminating the need for non-null assertions.
 */
export interface ElementVNode extends VirtualNode {
    /** Element nodes always have type "element". */
    _type: 'element';
    /** The HTML tag name (required for element nodes). */
    tag: string;
}

/**
 * DOM API for creating virtual nodes.
 *
 * Used by compiler-generated code.
 */
export interface DomAPI {
    /** Creates a whitespace text node. */
    ws: (id: string) => VirtualNode;
    /** Creates a text node. */
    txt: (content: unknown, id: string, props?: Record<string, unknown> | null) => VirtualNode;
    /** Creates a raw HTML node. */
    html: (content: string, id: string) => VirtualNode;
    /** Creates a comment node. */
    cmt: (content: unknown, id: string, props?: Record<string, unknown> | null) => VirtualNode;
    /** Creates an element node. */
    el: (
        tag: string,
        id: string,
        props?: Record<string, unknown>,
        children?: VirtualNode | VirtualNode[] | null
    ) => ElementVNode;
    /** Creates a fragment node. */
    frag: (id: string, children?: VirtualNode | VirtualNode[] | null, props?: Record<string, unknown>) => VirtualNode;
    /** Validates and resolves a dynamic tag from piko:element :is. */
    resolveTag: (tag: unknown) => string;
    /** Creates an element from a piko:element with dynamic :is, adding link attributes when needed. */
    pikoEl: (
        rawTag: unknown,
        id: string,
        props?: Record<string, unknown>,
        children?: VirtualNode | VirtualNode[] | null,
        moduleName?: string
    ) => ElementVNode;
}
