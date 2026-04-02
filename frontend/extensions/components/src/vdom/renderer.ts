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

import type {VirtualNode, ElementVNode} from '@/vdom/types';
import {
    isUndef,
    isVNodeHidden,
    areSameVNode,
    createKeyToOldIdxMap,
    getHiddenNodeType
} from '@/vdom/patchComputation';
import {
    parseClassData,
    parseStyleData,
    combineClasses,
    combineStyles
} from '@/vdom/propDiff';

/** SVG namespace URI used when creating SVG elements. */
const SVG_NS = "http://www.w3.org/2000/svg";

/**
 * Options for element replacement tracking.
 */
export interface ReplacementOptions {
    /** Props to watch for changes that should invalidate the replacement. */
    watchProps?: string[];
}

/**
 * Internal entry stored in the replacement registry.
 */
interface ReplacementEntry {
    /** The replacement DOM node. */
    element: Node;
    /** The prop names to monitor for invalidation. */
    watchedProps?: string[];
    /** The prop values captured at registration time. */
    watchedValues?: Record<string, unknown>;
}

/**
 * Registry for elements that replace themselves at runtime.
 * Custom elements like piko-svg-inline can register their replacement so the VDOM
 * can adopt the new element instead of recreating on every update.
 *
 * Prefer using replaceElementWithTracking() which handles both registration and
 * replacement atomically.
 */
const elementReplacements = new WeakMap<Node, ReplacementEntry>();

/**
 * Registers an element replacement for VDOM tracking.
 *
 * Prefer using replaceElementWithTracking() which handles both registration
 * and DOM replacement atomically.
 *
 * @param originalElement - The original element being replaced.
 * @param replacementElement - The new element to track.
 * @param options - Optional replacement options.
 */
export function registerElementReplacement(
    originalElement: Node,
    replacementElement: Node,
    options?: ReplacementOptions
): void {
    const entry: ReplacementEntry = {
        element: replacementElement,
    };

    if (options?.watchProps && originalElement instanceof Element) {
        entry.watchedProps = options.watchProps;
        entry.watchedValues = {};
        for (const prop of options.watchProps) {
            entry.watchedValues[prop] = originalElement.getAttribute(prop);
        }
    }

    elementReplacements.set(originalElement, entry);
}

/**
 * Replaces an element and registers the replacement for VDOM tracking.
 *
 * Performs both VDOM registration and DOM replacement atomically.
 *
 * @param originalElement - The original element to replace.
 * @param replacementElement - The new element to insert.
 * @param options - Optional replacement options including watched props.
 */
export function replaceElementWithTracking(
    originalElement: Element,
    replacementElement: Node,
    options?: ReplacementOptions
): void {
    registerElementReplacement(originalElement, replacementElement, options);
    originalElement.replaceWith(replacementElement);
}

/** Event handler type for DOM event listeners. */
type EventHandler = EventListenerOrEventListenerObject;
/** Record type for props objects. */
type PropsRecord = Record<string, unknown>;

/**
 * Checks whether any watched props have changed between registration and new virtual node props.
 *
 * @param entry - The replacement entry with watched values.
 * @param newProps - The new props from the virtual node.
 * @returns True if any watched prop value has changed.
 */
function hasWatchedPropsChanged(
    entry: ReplacementEntry,
    newProps: PropsRecord | undefined
): boolean {
    if (!entry.watchedProps || !entry.watchedValues) {
        return false;
    }

    for (const prop of entry.watchedProps) {
        const oldValue = entry.watchedValues[prop];
        const newValue = newProps?.[prop];

        if (oldValue !== newValue) {
            return true;
        }
    }

    return false;
}

/** Radix for random key generation in fallback identifiers. */
const RANDOM_KEY_RADIX = 36;
/** Slice offset for random key string extraction. */
const RANDOM_KEY_SLICE = 2;
/** Length of the 'pe:' event prefix. */
const PREFIX_PE_LENGTH = 3;
/** Length of the 'on' event prefix. */
const PREFIX_ON_LENGTH = 2;

/**
 * Patches a fragment virtual node that matches an existing fragment.
 *
 * @param oldVNode - The previous fragment virtual node.
 * @param newVNode - The new fragment virtual node.
 * @param oldElm - The existing DOM node for the old fragment.
 * @param parentElement - The parent DOM node.
 * @param nextSiblingNode - The sibling to insert before, or null for append.
 * @param refs - Object for storing element references.
 */
function patchSameFragment(
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    oldElm: Node | undefined,
    parentElement: Node,
    nextSiblingNode: Node | null,
    refs: Record<string, Node> | null
): void {
    const newIsHidden = isVNodeHidden(newVNode);
    const oldIsHidden = isVNodeHidden(oldVNode);
    newVNode.elm = oldElm;

    if (newIsHidden) {
        patchHiddenFragment(oldVNode, newVNode, oldElm, oldIsHidden, parentElement, nextSiblingNode, refs);
    } else if (oldIsHidden && oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
        parentElement.replaceChild(createElm(newVNode, refs) as DocumentFragment, oldElm);
        newVNode.elm = undefined;
    } else if (!oldIsHidden) {
        updateChildren(parentElement, oldVNode.children ?? [], newVNode.children ?? [], refs, nextSiblingNode);
        newVNode.elm = undefined;
    } else {
        removeVNode(oldVNode, parentElement, refs);
        parentElement.insertBefore(createElm(newVNode, refs) as DocumentFragment, nextSiblingNode);
        newVNode.elm = undefined;
    }
}

/**
 * Patches an element that has a registered replacement in the DOM.
 *
 * @param oldVNode - The previous virtual node.
 * @param newVNode - The new virtual node.
 * @param oldElm - The existing DOM node.
 * @param parentElement - The parent DOM node.
 * @param nextSiblingNode - The sibling to insert before, or null for append.
 * @param refs - Object for storing element references.
 */
function patchElementWithReplacement(
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    oldElm: Node,
    parentElement: Node,
    nextSiblingNode: Node | null,
    refs: Record<string, Node> | null
): void {
    const entry = elementReplacements.get(oldElm);
    if (entry?.element.parentNode === parentElement) {
        if (hasWatchedPropsChanged(entry, newVNode.props)) {
            parentElement.removeChild(entry.element);
            clearElmRefsRecursive(oldVNode, refs);
            parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
        } else {
            newVNode.elm = entry.element;
        }
    } else {
        clearElmRefsRecursive(oldVNode, refs);
        parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
    }
}

/**
 * Patches a virtual node that matches an existing node by key and type.
 *
 * @param oldVNode - The previous virtual node.
 * @param newVNode - The new virtual node.
 * @param oldElm - The existing DOM node.
 * @param parentElement - The parent DOM node.
 * @param nextSiblingNode - The sibling to insert before, or null for append.
 * @param refs - Object for storing element references.
 */
function patchSameVNode(
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    oldElm: Node | undefined,
    parentElement: Node,
    nextSiblingNode: Node | null,
    refs: Record<string, Node> | null
): void {
    newVNode.elm = oldElm;

    if (isVNodeHidden(newVNode)) {
        patchHiddenElement(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
        return;
    }

    if (oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
        const newContent = createElm(newVNode, refs);
        if (oldElm.parentNode === parentElement) {
            parentElement.replaceChild(newContent, oldElm);
        } else {
            parentElement.insertBefore(newContent, nextSiblingNode);
        }
        return;
    }

    if (oldElm?.parentNode === parentElement) {
        patchVNode(oldElm, oldVNode, newVNode, refs);
        return;
    }

    if (oldElm) {
        patchElementWithReplacement(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
        return;
    }

    parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
}

/**
 * Patches the DOM to match the virtual DOM representation.
 *
 * @param oldVNode - The previous virtual node, or null for initial render.
 * @param newVNode - The new virtual node to render, or null to remove.
 * @param refs - Object for storing element references.
 * @param parentElement - The DOM parent element.
 * @param nextSiblingNode - The sibling to insert before, or null for append.
 */
export function patch(
    oldVNode: VirtualNode | null,
    newVNode: VirtualNode | null,
    refs: Record<string, Node> | null,
    parentElement: Node,
    nextSiblingNode: Node | null
): void {
    if (newVNode == null) {
        if (oldVNode != null) {
            removeVNode(oldVNode, parentElement, refs);
        }
        return;
    }

    if (oldVNode === newVNode) {
        return;
    }

    const oldElm = oldVNode?.elm;

    if (newVNode._type === "fragment") {
        if (oldVNode?._type === "fragment" && oldVNode.key === newVNode.key) {
            patchSameFragment(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
        } else {
            if (oldVNode) {
                removeVNode(oldVNode, parentElement, refs);
            }
            const newIsHidden = isVNodeHidden(newVNode);
            const newContent = createElm(newVNode, refs);
            parentElement.insertBefore(newContent, nextSiblingNode);
            newVNode.elm = newIsHidden ? newContent : undefined;
        }
        return;
    }

    if (oldVNode && areSameVNode(oldVNode, newVNode)) {
        patchSameVNode(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
    } else {
        if (oldVNode) {
            removeVNode(oldVNode, parentElement, refs);
        }
        parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
    }
}

/**
 * Patches a hidden fragment by replacing its children with a comment placeholder.
 *
 * @param oldVNode - The previous fragment virtual node.
 * @param newVNode - The new fragment virtual node.
 * @param oldElm - The existing DOM node for the old fragment.
 * @param oldIsHidden - Whether the old fragment was hidden.
 * @param parentElement - The parent DOM node.
 * @param nextSiblingNode - The sibling to insert before, or null for append.
 * @param refs - Object for storing element references.
 */
function patchHiddenFragment(
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    oldElm: Node | undefined,
    oldIsHidden: boolean,
    parentElement: Node,
    nextSiblingNode: Node | null,
    refs: Record<string, Node> | null
): void {
    if (!oldIsHidden) {
        removeVNode(oldVNode, parentElement, refs);
        const placeholder = createElm(newVNode, refs);
        parentElement.insertBefore(placeholder, nextSiblingNode);
        newVNode.elm = placeholder;
    } else if (oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
        const message = `hidden fragment _k=${newVNode.key ?? `err-H2H`}`;
        if (oldElm.nodeValue !== message) {
            oldElm.nodeValue = message;
        }
    } else {
        if (oldElm?.parentNode === parentElement) {
            parentElement.removeChild(oldElm);
        }
        clearElmRefsRecursive(oldVNode, refs);
        const placeholder = createElm(newVNode, refs);
        parentElement.insertBefore(placeholder, nextSiblingNode);
        newVNode.elm = placeholder;
    }
}

/**
 * Patches a hidden element by replacing it with a comment placeholder.
 *
 * @param _oldVNode - The previous virtual node (unused).
 * @param newVNode - The new virtual node.
 * @param oldElm - The existing DOM node.
 * @param parentElement - The parent DOM node.
 * @param nextSiblingNode - The sibling to insert before, or null for append.
 * @param refs - Object for storing element references.
 */
function patchHiddenElement(
    _oldVNode: VirtualNode,
    newVNode: VirtualNode,
    oldElm: Node | undefined,
    parentElement: Node,
    nextSiblingNode: Node | null,
    refs: Record<string, Node> | null
): void {
    if (oldElm && oldElm.nodeType !== Node.COMMENT_NODE) {
        const placeholder = createElm(newVNode, refs);
        if (oldElm.parentNode === parentElement) {
            parentElement.replaceChild(placeholder, oldElm);
        } else {
            parentElement.insertBefore(placeholder, nextSiblingNode);
        }
        newVNode.elm = placeholder;
    } else if (oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
        const message = `hidden node _k=${newVNode.key ?? `err-elemH2H`}`;
        if (oldElm.nodeValue !== message) {
            oldElm.nodeValue = message;
        }
    } else if (!oldElm) {
        const placeholder = createElm(newVNode, refs);
        parentElement.insertBefore(placeholder, nextSiblingNode);
        newVNode.elm = placeholder;
    }
}

/**
 * Creates a real DOM node from a virtual node.
 *
 * @param vnode - The virtual node to create a DOM node from.
 * @param refs - Object for storing element references.
 * @param isSvg - Whether the element is inside an SVG context.
 * @returns The created DOM node.
 */
function createElm(vnode: VirtualNode, refs: Record<string, Node> | null, isSvg?: boolean): Node {
    if (isVNodeHidden(vnode)) {
        vnode.elm = document.createComment(`hidden ${getHiddenNodeType(vnode)} _k=${vnode.key ?? `err-no-key-hidden-${Math.random().toString(RANDOM_KEY_RADIX).slice(RANDOM_KEY_SLICE)}`}`);
        return vnode.elm;
    }

    switch (vnode._type) {
        case "text":
            vnode.elm = document.createTextNode(vnode.text ?? "");
            return vnode.elm;
        case "comment":
            vnode.elm = document.createComment(vnode.text ?? "");
            return vnode.elm;
        case "fragment": {
            const fragmentDoc = document.createDocumentFragment();
            if (vnode.children) {
                for (const child of vnode.children) {
                    fragmentDoc.appendChild(createElm(child, refs, isSvg));
                }
            }
            return fragmentDoc;
        }
        case "element":
            return createElementNode(vnode as ElementVNode, refs, isSvg);
        default:
            vnode.elm = document.createTextNode("");
            return vnode.elm;
    }
}

/**
 * Creates a real DOM element from an element virtual node, handling SVG namespace
 * propagation, property patching, and child or innerHTML content.
 *
 * @param elVNode - The element virtual node to create a DOM element from.
 * @param refs - Object for storing element references.
 * @param isSvg - Whether the element is inside an SVG context.
 * @returns The created DOM element.
 */
function createElementNode(elVNode: ElementVNode, refs: Record<string, Node> | null, isSvg?: boolean): Element {
    const tagIsSvg = elVNode.tag === "svg";
    const childSvg = isSvg === true || tagIsSvg === true;
    const elementNode = childSvg
        ? document.createElementNS(SVG_NS, elVNode.tag)
        : document.createElement(elVNode.tag);
    elVNode.elm = elementNode;
    patchProps(elementNode as HTMLElement, {}, elVNode.props ?? {}, refs);
    if (elVNode.html != null) {
        (elementNode as HTMLElement).innerHTML = elVNode.html;
    } else if (elVNode.children) {
        for (const child of elVNode.children) {
            elementNode.appendChild(createElm(child, refs, childSvg));
        }
    }
    return elementNode;
}

/**
 * Recursively clears DOM element references and ref entries from a virtual node tree.
 *
 * @param vnode - The virtual node to clear references from.
 * @param refs - Object storing element references.
 */
function clearElmRefsRecursive(vnode: VirtualNode | null, refs: Record<string, Node> | null) {
    if (!vnode) {
        return;
    }
    if (vnode.children) {
        for (const child of vnode.children) {
            clearElmRefsRecursive(child, refs);
        }
    }
    if (refs && vnode.props?._ref && typeof vnode.props._ref === 'string') {
        const refKey = vnode.props._ref;
        if (refs[refKey] === vnode.elm) {
            delete refs[refKey];
        }
    }
    vnode.elm = undefined;
}

/**
 * Removes a virtual node and its DOM elements from the parent.
 *
 * @param vnodeToRemove - The virtual node to remove.
 * @param parentElement - The parent DOM node.
 * @param refs - Object storing element references.
 */
function removeVNode(vnodeToRemove: VirtualNode | null, parentElement: Node, refs: Record<string, Node> | null = null) {
    if (!vnodeToRemove) {
        return;
    }

    if (vnodeToRemove._type === "fragment" && !isVNodeHidden(vnodeToRemove)) {
        if (vnodeToRemove.children) {
            for (const child of vnodeToRemove.children) {
                removeVNode(child, parentElement, refs);
            }
        }
    } else if (vnodeToRemove.elm?.parentNode === parentElement) {
        parentElement.removeChild(vnodeToRemove.elm);
    }

    clearElmRefsRecursive(vnodeToRemove, refs);
}

/**
 * Inserts new virtual nodes into the DOM as children of the parent element.
 *
 * @param parentElement - The parent DOM node.
 * @param referenceNode - The sibling to insert before, or null for append.
 * @param vnodesToAdd - The array of virtual nodes to insert.
 * @param startIndex - The start index in the array.
 * @param endIndex - The end index in the array (inclusive).
 * @param refs - Object for storing element references.
 */
function addVNodes(
    parentElement: Node,
    referenceNode: Node | null,
    vnodesToAdd: (VirtualNode | undefined)[],
    startIndex: number,
    endIndex: number,
    refs: Record<string, Node> | null
) {
    for (let i = startIndex; i <= endIndex; i++) {
        const childVNode = vnodesToAdd[i];
        if (childVNode) {
            const createdDomMaterial = createElm(childVNode, refs);
            parentElement.insertBefore(createdDomMaterial, referenceNode);

            if (childVNode._type === "fragment" && !isVNodeHidden(childVNode)) {
                childVNode.elm = undefined;
            }
        }
    }
}

/**
 * Finds the first real DOM node in a virtual node tree.
 *
 * @param vnode - The virtual node to search from.
 * @returns The first real DOM node, or null if none found.
 */
function getFirstDomElementRecursive(vnode: VirtualNode | undefined | null): Node | null {
    if (!vnode) {
        return null;
    }
    if (vnode.elm && vnode.elm.nodeType !== Node.DOCUMENT_FRAGMENT_NODE) {
        return vnode.elm;
    }
    if (vnode._type !== "fragment" || isVNodeHidden(vnode) || !vnode.children) {
        return null;
    }
    for (const child of vnode.children) {
        const firstChildDom = getFirstDomElementRecursive(child);
        if (firstChildDom) {
            return firstChildDom;
        }
    }
    return null;
}

/**
 * Patch an element-type virtual node by syncing props, innerHTML, and
 * recursively updating children.
 *
 * @param domElement - The live DOM element.
 * @param oldVNode - The previous virtual node.
 * @param newVNode - The new virtual node.
 * @param refs - Object for storing element references.
 */
function patchElementVNode(
    domElement: HTMLElement,
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    refs: Record<string, Node> | null
): void {
    if (domElement.nodeType === Node.COMMENT_NODE && !isVNodeHidden(newVNode)) {
        console.error("PPElement Error: patchVNode attempting to patch a comment as a visible element.", {
            oldVNode,
            newVNode,
            domElement
        });
        return;
    }

    patchProps(domElement, oldVNode.props ?? {}, newVNode.props ?? {}, refs);

    if (newVNode.html != null) {
        if (oldVNode.children && oldVNode.children.length > 0) {
            domElement.textContent = '';
        }
        if (oldVNode.html !== newVNode.html) {
            domElement.innerHTML = newVNode.html;
        }
    } else {
        const oldChildren = oldVNode.children ?? [];
        const newChildren = newVNode.children ?? [];
        if (oldChildren.length > 0 || newChildren.length > 0) {
            updateChildren(domElement, oldChildren, newChildren, refs, null);
        }
    }
}

/**
 * Patches a single DOM element to match a new virtual node.
 *
 * @param domElement - The DOM node to patch.
 * @param oldVNode - The previous virtual node.
 * @param newVNode - The new virtual node.
 * @param refs - Object for storing element references.
 */
function patchVNode(
    domElement: Node,
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    refs: Record<string, Node> | null
) {
    newVNode.elm = domElement;
    if (oldVNode === newVNode) {
        return;
    }

    if (isVNodeHidden(newVNode) && newVNode._type !== "comment") {
        if (domElement.nodeType === Node.COMMENT_NODE) {
            const message = `hidden node _k=${newVNode.key ?? "no-key"}`;
            if (domElement.nodeValue !== message) {
                domElement.nodeValue = message;
            }
        }
        return;
    }

    if (newVNode._type === "text") {
        if (oldVNode.text !== newVNode.text) {
            domElement.nodeValue = newVNode.text ?? "";
        }
    } else if (newVNode._type === "comment") {
        if (oldVNode.text !== newVNode.text) {
            domElement.nodeValue = newVNode.text ?? "";
        }
    } else if (newVNode._type === "element") {
        patchElementVNode(domElement as HTMLElement, oldVNode, newVNode, refs);
    }
}

/**
 * Patches two matched nodes and relocates the DOM element to the end of the processed range.
 *
 * Used when the old-start pointer matches the new-end pointer, meaning the node
 * needs to move rightward in the DOM.
 *
 * @param oldVNode - The old virtual node being patched.
 * @param newVNode - The new virtual node to patch against.
 * @param nextOldSibling - The vnode after oldStart (for insertion reference).
 * @param oldEndVNode - The current old-end vnode (for computing the move target).
 * @param refs - Object for storing element references.
 * @param parentElement - The parent DOM node.
 * @param overallInsertBeforeNode - Fallback sibling to insert before.
 */
function patchAndRelocateToEnd(
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    nextOldSibling: VirtualNode | undefined,
    oldEndVNode: VirtualNode,
    refs: Record<string, Node> | null,
    parentElement: Node,
    overallInsertBeforeNode: Node | null,
): void {
    const nextOldSiblingElm = getFirstDomElementRecursive(nextOldSibling);
    patch(oldVNode, newVNode, refs, parentElement, nextOldSiblingElm ?? overallInsertBeforeNode);

    const domToMove = newVNode.elm ?? getFirstDomElementRecursive(newVNode);
    const referenceForMove = getFirstDomElementRecursive(oldEndVNode)?.nextSibling ?? overallInsertBeforeNode;
    if (domToMove) {
        parentElement.insertBefore(domToMove, referenceForMove);
    }
}

/**
 * Patches two matched nodes and relocates the DOM element to the start of the processed range.
 *
 * Used when the old-end pointer matches the new-start pointer, meaning the node
 * needs to move leftward in the DOM.
 *
 * @param oldVNode - The old virtual node being patched.
 * @param newVNode - The new virtual node to patch against.
 * @param oldStartVNode - The current old-start vnode (for computing the insertion point).
 * @param refs - Object for storing element references.
 * @param parentElement - The parent DOM node.
 * @param overallInsertBeforeNode - Fallback sibling to insert before.
 */
function patchAndRelocateToStart(
    oldVNode: VirtualNode,
    newVNode: VirtualNode,
    oldStartVNode: VirtualNode,
    refs: Record<string, Node> | null,
    parentElement: Node,
    overallInsertBeforeNode: Node | null,
): void {
    const referenceForPatch = getFirstDomElementRecursive(oldStartVNode);
    patch(oldVNode, newVNode, refs, parentElement, referenceForPatch ?? overallInsertBeforeNode);

    const domToMove = newVNode.elm ?? getFirstDomElementRecursive(newVNode);
    if (domToMove) {
        parentElement.insertBefore(domToMove, referenceForPatch ?? overallInsertBeforeNode);
    }
}

/**
 * Removes old child vnodes that were not matched during the two-pointer diff.
 *
 * @param oldChildren - The old child vnode array.
 * @param startIdx - First index in the unmatched range.
 * @param endIdx - Last index in the unmatched range.
 * @param parentElement - The parent DOM node.
 * @param refs - Object for storing element references.
 */
function removeRemainingOldChildren(
    oldChildren: (VirtualNode | undefined)[],
    startIdx: number,
    endIdx: number,
    parentElement: Node,
    refs: Record<string, Node> | null,
): void {
    for (let i = startIdx; i <= endIdx; i++) {
        const child = oldChildren[i];
        if (child) {
            removeVNode(child, parentElement, refs);
        }
    }
}

/**
 * Inserts new child vnodes that had no match in the old children array.
 *
 * @param newChildren - The new child vnode array.
 * @param startIdx - First index in the unmatched range.
 * @param endIdx - Last index in the unmatched range.
 * @param parentElement - The parent DOM node.
 * @param refs - Object for storing element references.
 * @param overallInsertBeforeNode - Fallback sibling to insert before.
 */
function addRemainingNewChildren(
    newChildren: (VirtualNode | undefined)[],
    startIdx: number,
    endIdx: number,
    parentElement: Node,
    refs: Record<string, Node> | null,
    overallInsertBeforeNode: Node | null,
): void {
    const insertRef = getFirstDomElementRecursive(newChildren[endIdx + 1]) ?? overallInsertBeforeNode;
    addVNodes(parentElement, insertRef, newChildren, startIdx, endIdx, refs);
}

/**
 * Resolves a new child node against the keyed map of old children.
 *
 * Looks up the new node's key in the old-children index. When a match is found
 * the old node is patched in place and its DOM element is repositioned; when no
 * match exists a fresh element is created. Stale matches (same key but different
 * tag/type) are replaced and the old node is removed.
 *
 * @param oldChildren - The mutable array of old child vnodes (entries may be set to undefined).
 * @param oldKeyToIdxMap - Map from key to index in oldChildren.
 * @param newStartVNode - The new child being resolved.
 * @param oldStartVNode - The current old-start pointer (used for insertion reference).
 * @param refs - Object for storing element references.
 * @param parentElement - The parent DOM node.
 * @param overallInsertBeforeNode - Fallback sibling to insert before.
 */
function resolveKeyedChild(
    oldChildren: (VirtualNode | undefined)[],
    oldKeyToIdxMap: Map<string, number>,
    newStartVNode: VirtualNode,
    oldStartVNode: VirtualNode,
    refs: Record<string, Node> | null,
    parentElement: Node,
    overallInsertBeforeNode: Node | null,
): void {
    const idxInOld = newStartVNode.key != null ? oldKeyToIdxMap.get(newStartVNode.key) : undefined;
    const referenceForInsert = getFirstDomElementRecursive(oldStartVNode);

    if (isUndef(idxInOld)) {
        patch(null, newStartVNode, refs, parentElement, referenceForInsert ?? overallInsertBeforeNode);
        return;
    }

    const vnodeToMove = oldChildren[idxInOld];
    if (vnodeToMove && areSameVNode(vnodeToMove, newStartVNode)) {
        patch(vnodeToMove, newStartVNode, refs, parentElement, referenceForInsert ?? overallInsertBeforeNode);
        oldChildren[idxInOld] = undefined;

        const domToActuallyMove = newStartVNode.elm ?? getFirstDomElementRecursive(newStartVNode);
        if (domToActuallyMove && (referenceForInsert !== domToActuallyMove)) {
            parentElement.insertBefore(domToActuallyMove, referenceForInsert ?? overallInsertBeforeNode);
        }
        return;
    }

    patch(null, newStartVNode, refs, parentElement, referenceForInsert ?? overallInsertBeforeNode);
    if (vnodeToMove) {
        removeVNode(vnodeToMove, parentElement, refs);
    }
}

/**
 * Reconciles old and new child virtual node arrays using a two-pointer diffing algorithm.
 *
 * Compares nodes at four pointer positions (old-start, old-end, new-start, new-end),
 * patches matches in place, relocates moved nodes, and falls back to keyed lookup
 * when no pointer match is found. After the loop, removes unmatched old nodes or
 * inserts remaining new nodes.
 *
 * @param parentElement - The parent DOM node.
 * @param oldChildren - The previous array of child virtual nodes.
 * @param newChildren - The new array of child virtual nodes.
 * @param refs - Object for storing element references.
 * @param overallInsertBeforeNode - The sibling to insert before, or null for append.
 */
function updateChildren(
    parentElement: Node,
    oldChildren: (VirtualNode | undefined)[],
    newChildren: (VirtualNode | undefined)[],
    refs: Record<string, Node> | null,
    overallInsertBeforeNode: Node | null
) {
    let oldStartIdx = 0, newStartIdx = 0;
    let oldEndIdx = oldChildren.length - 1;
    let newEndIdx = newChildren.length - 1;
    let oldStartVNode = oldChildren[0];
    let oldEndVNode = oldChildren[oldEndIdx];
    let newStartVNode = newChildren[0];
    let newEndVNode = newChildren[newEndIdx];
    let oldKeyToIdxMap: Map<string, number> | undefined;

    while (oldStartIdx <= oldEndIdx && newStartIdx <= newEndIdx) {
        if (isUndef(oldStartVNode)) { oldStartVNode = oldChildren[++oldStartIdx]; continue; }
        if (isUndef(oldEndVNode)) { oldEndVNode = oldChildren[--oldEndIdx]; continue; }
        if (isUndef(newStartVNode)) { newStartVNode = newChildren[++newStartIdx]; continue; }
        if (isUndef(newEndVNode)) { newEndVNode = newChildren[--newEndIdx]; continue; }

        if (areSameVNode(oldStartVNode, newStartVNode)) {
            const ref = getFirstDomElementRecursive(oldChildren[oldStartIdx + 1]);
            patch(oldStartVNode, newStartVNode, refs, parentElement, ref ?? overallInsertBeforeNode);
            oldStartVNode = oldChildren[++oldStartIdx];
            newStartVNode = newChildren[++newStartIdx];
        } else if (areSameVNode(oldEndVNode, newEndVNode)) {
            patch(oldEndVNode, newEndVNode, refs, parentElement, overallInsertBeforeNode);
            oldEndVNode = oldChildren[--oldEndIdx];
            newEndVNode = newChildren[--newEndIdx];
        } else if (areSameVNode(oldStartVNode, newEndVNode)) {
            patchAndRelocateToEnd(
                oldStartVNode, newEndVNode, oldChildren[oldStartIdx + 1], oldEndVNode,
                refs, parentElement, overallInsertBeforeNode,
            );
            oldStartVNode = oldChildren[++oldStartIdx];
            newEndVNode = newChildren[--newEndIdx];
        } else if (areSameVNode(oldEndVNode, newStartVNode)) {
            patchAndRelocateToStart(
                oldEndVNode, newStartVNode, oldStartVNode,
                refs, parentElement, overallInsertBeforeNode,
            );
            oldEndVNode = oldChildren[--oldEndIdx];
            newStartVNode = newChildren[++newStartIdx];
        } else {
            oldKeyToIdxMap ??= createKeyToOldIdxMap(oldChildren, oldStartIdx, oldEndIdx);
            resolveKeyedChild(
                oldChildren, oldKeyToIdxMap, newStartVNode, oldStartVNode,
                refs, parentElement, overallInsertBeforeNode,
            );
            newStartVNode = newChildren[++newStartIdx];
        }
    }

    if (oldStartIdx <= oldEndIdx) {
        removeRemainingOldChildren(oldChildren, oldStartIdx, oldEndIdx, parentElement, refs);
    } else if (newStartIdx <= newEndIdx) {
        addRemainingNewChildren(newChildren, newStartIdx, newEndIdx, parentElement, refs, overallInsertBeforeNode);
    }
}

/**
 * Remove old props that are absent or changed in the new props.
 *
 * @param htmlElement - The target HTML element.
 * @param oldProps - The previous props record.
 * @param newProps - The incoming props record.
 * @param refs - An object for storing element references, or null.
 */
function removeStaleProps(
    htmlElement: HTMLElement,
    oldProps: PropsRecord,
    newProps: PropsRecord,
    refs: Record<string, Node> | null
): void {
    for (const propName in oldProps) {
        if (!(propName in newProps) || oldProps[propName] !== newProps[propName]) {
            if (propName.startsWith("?")) {
                htmlElement.removeAttribute(propName.slice(1));
            } else if (propName.startsWith("on")) {
                const parsed = parseEventPropKey(propName, PREFIX_ON_LENGTH);
                toggleListener(htmlElement, parsed.eventName, oldProps[propName] as EventHandler | EventHandler[], false, parsed.listenerOptions);
            } else if (propName.startsWith("pe:")) {
                const parsed = parseEventPropKey(propName, PREFIX_PE_LENGTH);
                toggleListener(htmlElement, parsed.eventName, oldProps[propName] as EventHandler | EventHandler[], false, parsed.listenerOptions);
            } else if (propName === "_ref" && refs && oldProps[propName] && refs[oldProps[propName] as string] === htmlElement) {
                delete refs[oldProps[propName] as string];
            } else if (!["_k", "_c", "_s", "class", "_class", "style", "_style"].includes(propName)) {
                htmlElement.removeAttribute(propName);
            }
        }
    }
}

/**
 * Set a DOM attribute from a prop value, handling null removal, form element
 * value properties, object serialisation, and plain string attributes.
 *
 * @param htmlElement - The target HTML element.
 * @param propName - The attribute name to set.
 * @param newValue - The incoming prop value.
 */
function applyAttributeValue(htmlElement: HTMLElement, propName: string, newValue: unknown): void {
    if (newValue == null || newValue === false) {
        htmlElement.removeAttribute(propName);
        return;
    }

    if ((htmlElement.tagName === "INPUT" || htmlElement.tagName === "TEXTAREA" || htmlElement.tagName === "SELECT") && propName === "value") {
        if ((htmlElement as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value !== String(newValue)) {
            (htmlElement as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value = String(newValue);
        }
        return;
    }

    if (typeof newValue === "object") {
        try {
            htmlElement.setAttribute(propName, JSON.stringify(newValue));
        } catch (error) {
            console.warn('[renderer] Failed to JSON.stringify prop, falling back to String():', {
                propName,
                value: newValue,
                error
            });
            htmlElement.setAttribute(propName, String(newValue));
        }
        return;
    }

    htmlElement.setAttribute(propName, String(newValue));
}

/**
 * Apply a single prop value to an HTML element, dispatching by prop-name
 * prefix to the appropriate handler (boolean attributes, event listeners,
 * refs, or plain attributes).
 *
 * @param htmlElement - The target HTML element.
 * @param propName - The prop name being applied.
 * @param oldValue - The previous value of the prop.
 * @param newValue - The incoming value of the prop.
 * @param refs - An object for storing element references, or null.
 */
function applyPropValue(
    htmlElement: HTMLElement,
    propName: string,
    oldValue: unknown,
    newValue: unknown,
    refs: Record<string, Node> | null
): void {
    if (propName.startsWith("?")) {
        const realAttrName = propName.slice(1);
        if (newValue) {
            htmlElement.setAttribute(realAttrName, "");
        } else {
            htmlElement.removeAttribute(realAttrName);
        }
        return;
    }

    if (propName.startsWith("on")) {
        const parsed = parseEventPropKey(propName, PREFIX_ON_LENGTH);
        if (oldValue !== newValue) {
            toggleListener(htmlElement, parsed.eventName, oldValue as EventHandler | EventHandler[], false, parsed.listenerOptions);
            toggleListener(htmlElement, parsed.eventName, newValue as EventHandler | EventHandler[], true, parsed.listenerOptions);
        }
        return;
    }

    if (propName.startsWith("pe:")) {
        const parsed = parseEventPropKey(propName, PREFIX_PE_LENGTH);
        if (oldValue !== newValue) {
            toggleListener(htmlElement, parsed.eventName, oldValue as EventHandler | EventHandler[], false, parsed.listenerOptions);
            toggleListener(htmlElement, parsed.eventName, newValue as EventHandler | EventHandler[], true, parsed.listenerOptions);
        }
        return;
    }

    if (propName === "_ref") {
        if (!refs) { return; }
        if (oldValue && refs[oldValue as string] === htmlElement) {
            delete refs[oldValue as string];
        }
        if (newValue && typeof newValue === 'string') {
            refs[newValue] = htmlElement;
        }
        return;
    }

    applyAttributeValue(htmlElement, propName, newValue);
}

/**
 * Apply new or changed prop values to an HTML element.
 *
 * @param htmlElement - The target HTML element.
 * @param oldProps - The previous props record.
 * @param newProps - The incoming props record.
 * @param refs - An object for storing element references, or null.
 */
function applyNewProps(
    htmlElement: HTMLElement,
    oldProps: PropsRecord,
    newProps: PropsRecord,
    refs: Record<string, Node> | null
): void {
    for (const propName in newProps) {
        const newValue = newProps[propName];
        const oldValue = oldProps[propName];

        if (["_k", "_c", "_s", "class", "_class", "style", "_style"].includes(propName)) {
            continue;
        }
        if (propName !== "value" && newValue === oldValue && typeof newValue !== 'function') {
            continue;
        }
        if (propName !== "value" && newValue === oldValue) {
            continue;
        }

        applyPropValue(htmlElement, propName, oldValue, newValue, refs);
    }
}

/**
 * Reconcile the class attribute from static and dynamic sources.
 *
 * @param htmlElement - The target HTML element.
 * @param newProps - The incoming props record containing class data.
 */
function reconcileClassAttribute(htmlElement: HTMLElement, newProps: PropsRecord): void {
    const staticClassValue = (newProps.class ?? "") as string;
    const dynamicClassValue = newProps._class ?? null;
    const finalClass = combineClasses(staticClassValue, parseClassData(dynamicClassValue));

    if (finalClass.trim()) {
        if (htmlElement.getAttribute('class') !== finalClass.trim()) {
            htmlElement.setAttribute("class", finalClass.trim());
        }
    } else {
        htmlElement.removeAttribute("class");
    }
}

/**
 * Reconcile the style attribute from static and dynamic sources.
 *
 * @param htmlElement - The target HTML element.
 * @param newProps - The incoming props record containing style data.
 */
function reconcileStyleAttribute(htmlElement: HTMLElement, newProps: PropsRecord): void {
    const staticStyleValue = (newProps.style ?? "") as string;
    const dynamicStyleValue = newProps._style ?? null;
    const finalStyle = combineStyles(staticStyleValue, parseStyleData(dynamicStyleValue));

    if (finalStyle.trim()) {
        if (htmlElement.getAttribute('style') !== finalStyle.trim()) {
            htmlElement.setAttribute("style", finalStyle.trim());
        }
    } else {
        htmlElement.removeAttribute("style");
    }
}

/**
 * Apply visibility behaviour based on the _s show/hide prop.
 *
 * @param htmlElement - The target HTML element.
 * @param newProps - The incoming props record containing the _s flag.
 */
function applyVisibility(htmlElement: HTMLElement, newProps: PropsRecord): void {
    const shouldShow = newProps._s !== false;

    if (!shouldShow) {
        if (htmlElement.style.display !== 'none') {
            htmlElement.style.display = 'none';
        }
    } else if (htmlElement.style.display === 'none') {
        htmlElement.style.display = '';
    }
}

/**
 * Apply prop changes to an HTML element by diffing old and new props.
 *
 * @param htmlElement - The HTML element to update.
 * @param oldProps - The previous props.
 * @param newProps - The new props.
 * @param refs - An object for storing element references.
 */
function patchProps(
    htmlElement: HTMLElement | null,
    oldProps: PropsRecord,
    newProps: PropsRecord,
    refs: Record<string, Node> | null
): void {
    if (!htmlElement) {
        console.error("PPElement Error: patchProps called with undefined htmlElement.", {oldProps, newProps});
        return;
    }

    removeStaleProps(htmlElement, oldProps, newProps, refs);
    applyNewProps(htmlElement, oldProps, newProps, refs);
    reconcileClassAttribute(htmlElement, newProps);
    reconcileStyleAttribute(htmlElement, newProps);
    applyVisibility(htmlElement, newProps);
}

/**
 * Parses a VDOM event prop key (after the prefix has been stripped) into the
 * event name and optional listener options encoded as $-delimited suffixes.
 *
 * @param propName - The full prop name (e.g. "onClick$capture").
 * @param prefixLen - Length of the prefix to strip (2 for "on", 3 for "pe:").
 * @returns The lowercased event name and optional listener options.
 */
function parseEventPropKey(propName: string, prefixLen: number): { eventName: string; listenerOptions?: AddEventListenerOptions } {
    const raw = propName.slice(prefixLen);
    const delimiterIndex = raw.indexOf('$');
    if (delimiterIndex === -1) {
        return {eventName: raw.toLowerCase()};
    }
    const eventName = raw.slice(0, delimiterIndex).toLowerCase();
    const opts: AddEventListenerOptions = {};
    for (const s of raw.slice(delimiterIndex + 1).split('$')) {
        if (s === 'capture') { opts.capture = true; }
        if (s === 'passive') { opts.passive = true; }
    }
    return {eventName, listenerOptions: opts};
}

/**
 * Adds or removes an event listener on an HTML element.
 *
 * @param htmlElement - The HTML element to modify.
 * @param eventName - The event name to listen for.
 * @param handler - The event handler or array of handlers.
 * @param add - Whether to add (true) or remove (false) the listener.
 * @param listenerOptions - Optional addEventListener options (capture, passive).
 */
function toggleListener(
    htmlElement: HTMLElement,
    eventName: string,
    handler: EventHandler | EventHandler[] | undefined,
    add: boolean,
    listenerOptions?: AddEventListenerOptions
) {
    if (!handler) {
        return;
    }
    const options: AddEventListenerOptions = {...listenerOptions};
    if ((eventName === "focus" || eventName === "blur") && options.capture === undefined) {
        options.capture = true;
    }
    const method = add ? 'addEventListener' : 'removeEventListener';
    if (Array.isArray(handler)) {
        for (const func of handler) {
            if (typeof func === "function") {
                htmlElement[method](eventName, func, options);
            }
        }
    } else if (typeof handler === "function") {
        htmlElement[method](eventName, handler, options);
    }
}
