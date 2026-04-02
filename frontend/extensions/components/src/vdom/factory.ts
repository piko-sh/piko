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

import type {DomAPI, VirtualNode} from "./types";

/** Access the piko namespace set by core on window. */
function getPiko(): PikoNamespace {
    return window.piko;
}

/**
 * DOM API object for creating virtual nodes.
 *
 * Used by compiler-generated code.
 */
export const dom: DomAPI = {
    ws(id) {
        return dom.txt(" ", id, {_isWhitespace: true});
    },

    txt(content, id, props = null) {
        const finalProps = {...(props ?? {})};
        return {
            _type: "text",
            text: String(content ?? ""),
            props: finalProps,
            children: null,
            key: id,
        };
    },

    html(content, id) {
        return {
            _type: "element",
            tag: "div",
            props: {},
            children: null,
            html: content,
            key: id,
        };
    },

    cmt(content, id, props = null) {
        const finalProps = {...(props ?? {})};
        return {
            _type: "comment",
            text: String(content ?? ""),
            props: finalProps,
            children: null,
            key: id,
        };
    },

    el(tag, id, props = {}, children = []) {
        const finalProps = {...props};
        const childArray = normaliseChildren(children);
        return {
            _type: "element",
            tag,
            props: finalProps,
            children: childArray as VirtualNode[],
            key: id,
        };
    },

    frag(id, children = [], props = {}) {
        const finalProps = {...props};
        const childArray = normaliseChildren(children);
        return {
            _type: "fragment",
            props: finalProps,
            children: childArray as VirtualNode[],
            key: id,
        };
    },

    resolveTag(tag) {
        const s = String(tag ?? "");
        if (s === "") {
            console.warn("<piko:element> resolved to an empty tag name, falling back to <div>");
            return "div";
        }
        if (rejectedPikoElementTargets[s]) {
            console.warn(`<piko:element> cannot target '${s}', falling back to <div>`);
            return "div";
        }
        if (pikoTagMap[s]) {
            return pikoTagMap[s];
        }
        return s;
    },

    pikoEl(rawTag, id, props = {}, children = [], moduleName = "") {
        const tag = dom.resolveTag(rawTag);
        const finalProps = {...props};
        const rawTagStr = String(rawTag ?? "");
        const pikoNs = getPiko();

        if (pikoLinkTags[rawTagStr]) {
            finalProps["piko:a"] = "";
            const href = String(finalProps["href"] ?? "");
            if (href && pikoNs?.nav) {
                finalProps["onClick"] = (e: Event) => {
                    pikoNs.nav.navigateTo(href, e);
                };
            }
        }

        if (pikoAssetTags[rawTagStr] && typeof finalProps["src"] === "string" && pikoNs?.assets) {
            finalProps["src"] = pikoNs.assets.resolve(finalProps["src"], moduleName || undefined);
        }

        return dom.el(tag, id, finalProps, children);
    },
};

/** Tag names that piko:element refuses to target, falling back to div. */
const rejectedPikoElementTargets: Record<string, boolean> = {
    "piko:partial": true,
    "piko:slot": true,
    "piko:element": true,
};

/** Map from piko-prefixed tag names to their native HTML equivalents. */
const pikoTagMap: Record<string, string> = {
    "piko:a": "a",
    "piko:img": "img",
    "piko:svg": "piko-svg-inline",
    "piko:picture": "picture",
    "piko:video": "video",
};

/** Piko tags that behave as navigational links and receive click handlers. */
const pikoLinkTags: Record<string, boolean> = {
    "piko:a": true,
};

/** Piko tags whose src attribute is resolved through the asset pipeline. */
const pikoAssetTags: Record<string, boolean> = {
    "piko:img": true,
    "piko:svg": true,
    "piko:picture": true,
    "piko:video": true,
};

/**
 * Normalises children into a flat array of virtual nodes.
 *
 * Flattens nested arrays produced by p-for with siblings and filters out
 * null or undefined entries.
 *
 * @param children - A single node, array of nodes, or null.
 * @returns A flat array of defined virtual nodes.
 */
function normaliseChildren(children: VirtualNode | VirtualNode[] | null | undefined): VirtualNode[] {
    if (children == null) {
        return [];
    }
    if (Array.isArray(children)) {
        return children.flat(Infinity).filter(isDef) as VirtualNode[];
    }
    return [children].filter(isDef);
}

/**
 * Checks whether a value is neither undefined nor null.
 *
 * @param x - The value to check.
 * @returns True if the value is defined.
 */
function isDef<T>(x: T | undefined | null): x is T {
    return x != null;
}
