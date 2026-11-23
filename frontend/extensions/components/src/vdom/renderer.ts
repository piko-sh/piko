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
import {toRaw} from '@/reactivity';
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
    combineStyles,
    isReservedProp
} from '@/vdom/propDiff';

/** SVG namespace URI used when creating SVG elements. */
const SVG_NS = "http://www.w3.org/2000/svg";

/** Lowercased tag name whose subtree returns to the HTML namespace within an SVG. */
const FOREIGN_OBJECT_TAG = "foreignobject";

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

/** Tracks replacement nodes so VDOM updates can adopt them. */
const elementReplacements = new WeakMap<Node, ReplacementEntry>();

/**
 * Tracks a replacement node for subsequent VDOM updates.
 *
 * @param originalElement - The node the VDOM originally created.
 * @param replacementElement - The node to adopt during updates.
 * @param options - Optional props that invalidate the replacement when changed.
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
 * Replaces a DOM node and tracks the replacement for subsequent VDOM updates.
 *
 * @param originalElement - The element to replace.
 * @param replacementElement - The node to insert.
 * @param options - Optional props that invalidate the replacement when changed.
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
 * Checks whether a watched prop differs from its value at registration.
 *
 * @param entry - The replacement and its recorded prop values.
 * @param newProps - The incoming virtual node props.
 * @returns True when a watched prop differs.
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
    } else if (oldIsHidden && oldElm?.nodeType === Node.COMMENT_NODE) {
        parentElement.replaceChild(createElm(newVNode, refs, isSvgParent(parentElement)) as DocumentFragment, oldElm);
        newVNode.elm = undefined;
    } else if (!oldIsHidden) {
        updateChildren(parentElement, oldVNode.children ?? [], newVNode.children ?? [], refs, nextSiblingNode);
        newVNode.elm = undefined;
    } else {
        removeVNode(oldVNode, parentElement, refs);
        parentElement.insertBefore(createElm(newVNode, refs, isSvgParent(parentElement)) as DocumentFragment, nextSiblingNode);
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
            parentElement.insertBefore(createElm(newVNode, refs, isSvgParent(parentElement)), nextSiblingNode);
        } else {
            newVNode.elm = entry.element;
        }
    } else {
        clearElmRefsRecursive(oldVNode, refs);
        parentElement.insertBefore(createElm(newVNode, refs, isSvgParent(parentElement)), nextSiblingNode);
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

    if (oldElm?.nodeType === Node.COMMENT_NODE) {
        const newContent = createElm(newVNode, refs, isSvgParent(parentElement));
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

    parentElement.insertBefore(createElm(newVNode, refs, isSvgParent(parentElement)), nextSiblingNode);
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
            const newContent = createElm(newVNode, refs, isSvgParent(parentElement));
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
        parentElement.insertBefore(createElm(newVNode, refs, isSvgParent(parentElement)), nextSiblingNode);
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
        const placeholder = createElm(newVNode, refs, isSvgParent(parentElement));
        parentElement.insertBefore(placeholder, nextSiblingNode);
        newVNode.elm = placeholder;
    } else if (oldElm?.nodeType === Node.COMMENT_NODE) {
        const message = `hidden fragment _k=${newVNode.key ?? `err-H2H`}`;
        if (oldElm.nodeValue !== message) {
            oldElm.nodeValue = message;
        }
    } else {
        if (oldElm?.parentNode === parentElement) {
            parentElement.removeChild(oldElm);
        }
        clearElmRefsRecursive(oldVNode, refs);
        const placeholder = createElm(newVNode, refs, isSvgParent(parentElement));
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
        const placeholder = createElm(newVNode, refs, isSvgParent(parentElement));
        if (oldElm.parentNode === parentElement) {
            parentElement.replaceChild(placeholder, oldElm);
        } else {
            parentElement.insertBefore(placeholder, nextSiblingNode);
        }
        newVNode.elm = placeholder;
    } else if (oldElm?.nodeType === Node.COMMENT_NODE) {
        const message = `hidden node _k=${newVNode.key ?? `err-elemH2H`}`;
        if (oldElm.nodeValue !== message) {
            oldElm.nodeValue = message;
        }
    } else if (!oldElm) {
        const placeholder = createElm(newVNode, refs, isSvgParent(parentElement));
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
 * Creates a DOM element from an element virtual node.
 *
 * @param elVNode - The virtual node to materialise.
 * @param refs - The object that stores element references, if any.
 * @param isSvg - Whether to inherit the SVG namespace.
 * @returns The created element.
 */
function createElementNode(elVNode: ElementVNode, refs: Record<string, Node> | null, isSvg?: boolean): Element {
    const tagIsSvg = elVNode.tag === "svg";
    const inSvg = isSvg === true || tagIsSvg === true;
    const elementNode = inSvg
        ? document.createElementNS(SVG_NS, elVNode.tag)
        : document.createElement(elVNode.tag);
    elVNode.elm = elementNode;
    patchProps(elementNode as HTMLElement, {}, elVNode.props ?? {}, refs);
    const childSvg = inSvg && elVNode.tag.toLowerCase() !== FOREIGN_OBJECT_TAG;
    if (elVNode.html != null) {
        (elementNode as HTMLElement).innerHTML = elVNode.html;
    } else if (elVNode.children) {
        for (const child of elVNode.children) {
            elementNode.appendChild(createElm(child, refs, childSvg));
        }
    }
    syncSelectValue(elementNode, elVNode.props);
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
            const createdDomMaterial = createElm(childVNode, refs, isSvgParent(parentElement));
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
 * Updates a live element unless its memo dependencies are unchanged.
 *
 * @param domElement - The live DOM element.
 * @param oldVNode - The previous virtual node.
 * @param newVNode - The incoming virtual node.
 * @param refs - The object that stores element references, if any.
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

    if (canSkipPatchByMemo(oldVNode, newVNode)) {
        newVNode.children = oldVNode.children;
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
    syncSelectValue(domElement, newVNode.props);
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
 * Patches matched nodes and moves the live element after the old end node.
 *
 * @param oldVNode - The virtual node to patch.
 * @param newVNode - The incoming virtual node.
 * @param nextOldSibling - The first sibling to check for a patch anchor.
 * @param oldEndVNode - The last old node in the active range.
 * @param refs - The object that stores element references, if any.
 * @param parentElement - The parent whose children are moving.
 * @param overallInsertBeforeNode - The fallback anchor, or null to append.
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
 * Patches matched nodes and moves the live element before the old start node.
 *
 * @param oldVNode - The virtual node to patch.
 * @param newVNode - The incoming virtual node.
 * @param oldStartVNode - The first old node in the active range.
 * @param refs - The object that stores element references, if any.
 * @param parentElement - The parent whose children are moving.
 * @param overallInsertBeforeNode - The fallback anchor, or null to append.
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
 * Reuses a matching keyed child or inserts a new node. A matching key with a different
 * node type or tag replaces the old node.
 *
 * @param oldChildren - The old child array, mutated to clear a reused entry.
 * @param oldKeyToIdxMap - The index of old children by key.
 * @param newStartVNode - The incoming child to resolve.
 * @param oldStartVNode - The old node that anchors insertion.
 * @param refs - The object that stores element references, if any.
 * @param parentElement - The parent whose children are changing.
 * @param overallInsertBeforeNode - The fallback anchor, or null to append.
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
 * Reconciles children in place, reusing and moving nodes with matching keys.
 *
 * @param parentElement - The parent whose children are changing.
 * @param oldChildren - The previous children, with reused entries cleared as needed.
 * @param newChildren - The incoming children.
 * @param refs - The object that stores element references, if any.
 * @param overallInsertBeforeNode - The fallback anchor, or null to append.
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
 * Reports whether a row's entire patch can be skipped.
 *
 * @param oldVNode - The previous virtual node.
 * @param newVNode - The incoming virtual node.
 * @returns True when the patch can be skipped entirely.
 */
function canSkipPatchByMemo(oldVNode: VirtualNode, newVNode: VirtualNode): boolean {
    const oldMemo = oldVNode.props ? oldVNode.props["_memo"] : undefined;
    const newMemo = newVNode.props ? newVNode.props["_memo"] : undefined;
    return oldMemo !== undefined && newMemo !== undefined && memoDependenciesEqual(oldMemo, newMemo);
}

/**
 * Compares memo dependencies after unwrapping reactive values. Arrays with the same
 * length are compared element by element using strict equality.
 *
 * @param a - The previous dependency value.
 * @param b - The incoming dependency value.
 * @returns True when the unwrapped values or array entries are strictly equal.
 */
function memoDependenciesEqual(a: unknown, b: unknown): boolean {
    if (toRaw(a) === toRaw(b)) {
        return true;
    }
    if (Array.isArray(a) && Array.isArray(b) && a.length === b.length) {
        for (let i = 0; i < a.length; i++) {
            if (toRaw(a[i]) !== toRaw(b[i])) {
                return false;
            }
        }
        return true;
    }
    return false;
}

/**
 * Removes the listener when its event prop is absent from the new virtual node. Keeping
 * the listener for retained props lets pending handlers run after a synchronous render.
 *
 * @param htmlElement - The element with the listener.
 * @param propName - The event prop to remove.
 * @param newProps - The incoming props.
 */
function removeStaleEventListener(
    htmlElement: HTMLElement,
    propName: string,
    newProps: PropsRecord
): void {
    if (propName in newProps) {
        return;
    }
    const isCustom = propName.startsWith("pe:");
    const parsed = parseEventPropKey(propName, isCustom ? PREFIX_PE_LENGTH : PREFIX_ON_LENGTH);
    setManagedListener(htmlElement, eventPropNamespace(propName), parsed.eventName, undefined, parsed.listenerOptions);
}

/**
 * Clears stale attributes, listeners and refs.
 *
 * @param htmlElement - The element to update.
 * @param oldProps - The previous props.
 * @param newProps - The incoming props.
 * @param refs - The object that stores element references, if any.
 */
function removeStaleProps(
    htmlElement: HTMLElement,
    oldProps: PropsRecord,
    newProps: PropsRecord,
    refs: Record<string, Node> | null
): void {
    for (const propName in oldProps) {
        if ((propName in newProps) && oldProps[propName] === newProps[propName]) {
            continue;
        }
        if (propName.startsWith("?")) {
            if (!(propName in newProps)) {
                setBooleanAttribute(htmlElement, propName.slice(1), false);
            }
        } else if (propName.startsWith("on") || propName.startsWith("pe:")) {
            removeStaleEventListener(htmlElement, propName, newProps);
        } else if (propName === "_ref" && refs && oldProps[propName] && refs[oldProps[propName] as string] === htmlElement) {
            delete refs[oldProps[propName] as string];
        } else if (!isReservedProp(propName)) {
            htmlElement.removeAttribute(propName);
        }
    }
}

/**
 * Applies a select's bound value after its options are reconciled.
 *
 * @param element - The element whose children were reconciled.
 * @param props - The incoming props, including the value binding when present.
 */
function syncSelectValue(element: Element, props: PropsRecord | undefined): void {
    if (element.tagName !== "SELECT" || !props || !("value" in props)) {
        return;
    }
    const select = element as HTMLSelectElement;
    if (!select.multiple || !Array.isArray(props.value)) {
        applyAttributeValue(select, "value", props.value);
        return;
    }
    const selected = new Set(props.value.map(String));
    for (const option of Array.from(select.options)) {
        const included = selected.has(option.value);
        if (option.selected !== included) {
            option.selected = included;
        }
    }
}

/**
 * Checks whether a binding must be reapplied even when unchanged. Value bindings always
 * control the live value. A checked binding controls the live state on every patch only
 * with p-model.
 *
 * @param propName - The binding to check.
 * @param props - The incoming props.
 * @returns True when the binding controls the live value on every patch.
 */
function isControlledProp(propName: string, props: PropsRecord): boolean {
    return propName === "value" || (propName === "?checked" && props._model === true);
}

/**
 * Sets a boolean attribute and keeps an input's live checked state in sync.
 *
 * @param element - The element receiving the binding.
 * @param name - The attribute name.
 * @param enabled - Whether to keep the attribute present.
 */
function setBooleanAttribute(element: HTMLElement, name: string, enabled: boolean): void {
    element.toggleAttribute(name, enabled);
    if (name === "checked" && element.tagName === "INPUT") {
        const input = element as HTMLInputElement;
        if (input.checked !== enabled) {
            input.checked = enabled;
        }
    }
}

/**
 * Clears a control's live text or selection.
 *
 * @param control - The form control whose value binding became empty.
 */
function clearFormValue(control: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement): void {
    if (control instanceof HTMLSelectElement) {
        if (control.selectedIndex !== -1) {
            control.selectedIndex = -1;
        }
        return;
    }
    if (control.value !== "") {
        control.value = "";
    }
}

/**
 * Applies a prop to an attribute or a form control's live value. Null, undefined and
 * false remove the attribute and clear any live form value.
 *
 * @param htmlElement - The element to update.
 * @param propName - The attribute or value binding name.
 * @param newValue - The incoming prop value.
 */
function applyAttributeValue(htmlElement: HTMLElement, propName: string, newValue: unknown): void {
    const formValue = propName === "value" &&
        (htmlElement.tagName === "INPUT" || htmlElement.tagName === "TEXTAREA" || htmlElement.tagName === "SELECT");
    if (newValue == null || newValue === false) {
        htmlElement.removeAttribute(propName);
        if (formValue) {
            clearFormValue(htmlElement as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement);
        }
        return;
    }

    if (formValue) {
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
 * Applies a virtual DOM prop to a live element.
 *
 * @param htmlElement - The element to patch.
 * @param propName - The prop name.
 * @param oldValue - The previous prop value.
 * @param newValue - The incoming prop value.
 * @param refs - The object that stores element references, if any.
 */
function applyPropValue(
    htmlElement: HTMLElement,
    propName: string,
    oldValue: unknown,
    newValue: unknown,
    refs: Record<string, Node> | null
): void {
    if (propName.startsWith("?")) {
        setBooleanAttribute(htmlElement, propName.slice(1), !!newValue);
        return;
    }

    if (propName.startsWith("on") || propName.startsWith("pe:")) {
        const isCustom = propName.startsWith("pe:");
        const parsed = parseEventPropKey(propName, isCustom ? PREFIX_PE_LENGTH : PREFIX_ON_LENGTH);
        setManagedListener(
            htmlElement,
            eventPropNamespace(propName),
            parsed.eventName,
            newValue as EventHandler | EventHandler[] | undefined,
            parsed.listenerOptions
        );
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
 * Applies changed and controlled props. Select values are applied after the option nodes
 * are reconciled.
 *
 * @param htmlElement - The element to update.
 * @param oldProps - The previous props.
 * @param newProps - The incoming props.
 * @param refs - The object that stores element references, if any.
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

        if (isReservedProp(propName)) {
            continue;
        }
        if (propName === "value" && htmlElement.tagName === "SELECT") {
            continue;
        }
        if (newValue === oldValue && !isControlledProp(propName, newProps)) {
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
 * Applies the visibility flag using the element's display style.
 *
 * @param htmlElement - The element to show or hide.
 * @param newProps - The incoming props, including the visibility flag when present.
 */
function applyVisibility(htmlElement: HTMLElement, newProps: PropsRecord): void {
    const shouldShow = newProps._s !== false;

    if (!shouldShow) {
        if (htmlElement.style.display !== 'none') {
            htmlElement.style.display = 'none';
        }
    } else if (htmlElement.style.display === 'none') {
        htmlElement.style.display = '';
        if (htmlElement.getAttribute('style') === '') {
            htmlElement.removeAttribute('style');
        }
    }
}

/**
 * Patches an element's attributes, listeners and refs.
 *
 * @param htmlElement - The element to patch, if available.
 * @param oldProps - The previous props.
 * @param newProps - The incoming props.
 * @param refs - The object that stores element references, if any.
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
    // Reconcile only bound class and style attributes so external updates survive.
    if ("class" in newProps || "_class" in newProps || "class" in oldProps || "_class" in oldProps) {
        reconcileClassAttribute(htmlElement, newProps);
    }
    if ("style" in newProps || "_style" in newProps || "style" in oldProps || "_style" in oldProps) {
        reconcileStyleAttribute(htmlElement, newProps);
    }
    applyVisibility(htmlElement, newProps);
}

/**
 * Parses a lowercased event name and listener options from an event prop.
 *
 * @param propName - The full prop name, such as onClick$capture.
 * @param prefixLen - The length of the on or pe: prefix.
 * @returns The event name and any listener options.
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

/** Keeps a stable DOM listener that calls the latest handler. */
interface ManagedListenerEntry {
    handler: EventHandler | EventHandler[];
    options: AddEventListenerOptions;
    dispatch: (e: Event) => void;
}

const MANAGED_LISTENERS = Symbol("pikoManagedListeners");

type ManagedHost = HTMLElement & { [MANAGED_LISTENERS]?: Map<string, ManagedListenerEntry> };

/**
 * Finds the listener namespace for an event prop.
 *
 * @param propName - The event prop name.
 * @returns pe for a custom event prop or on otherwise.
 */
function eventPropNamespace(propName: string): string {
    return propName.startsWith("pe:") ? "pe" : "on";
}

/**
 * Resolves the effective listener options for an event, applying the
 * focus/blur capture default.
 *
 * @param eventName - The lowercased DOM event name.
 * @param listenerOptions - The options parsed from the prop name, if any.
 * @returns A fresh options object owned by the caller.
 */
function effectiveListenerOptions(eventName: string, listenerOptions?: AddEventListenerOptions): AddEventListenerOptions {
    const options: AddEventListenerOptions = {...listenerOptions};
    if ((eventName === "focus" || eventName === "blur") && options.capture === undefined) {
        options.capture = true;
    }
    return options;
}

/**
 * Invokes one handler with the element as `this`, isolating a throw.
 *
 * @param handler - The handler to invoke.
 * @param htmlElement - The element the listener is bound to (`this`).
 * @param event - The event being dispatched.
 */
function invokeManagedHandler(handler: EventHandler, htmlElement: HTMLElement, event: Event): void {
    if (typeof handler !== "function") {
        return;
    }
    try {
        handler.call(htmlElement, event);
    } catch (error) {
        queueMicrotask(() => {
            throw error;
        });
    }
}

/**
 * Registers, updates, or removes the managed listener for one event binding.
 *
 * @param htmlElement - The element to bind on.
 * @param namespace - The event prop namespace ("on" or "pe").
 * @param eventName - The lowercased DOM event name.
 * @param handler - The handler (or handler array) to install, or undefined to remove.
 * @param listenerOptions - The options parsed from the prop name, if any.
 */
function setManagedListener(
    htmlElement: HTMLElement,
    namespace: string,
    eventName: string,
    handler: EventHandler | EventHandler[] | undefined,
    listenerOptions?: AddEventListenerOptions
): void {
    const options = effectiveListenerOptions(eventName, listenerOptions);
    const flags = `${options.capture ? "c" : ""}${options.once ? "o" : ""}${options.passive ? "p" : ""}`;
    const key = `${namespace}|${eventName}|${flags}`;
    const host = htmlElement as ManagedHost;
    let map = host[MANAGED_LISTENERS];
    if (!handler) {
        const entry = map ? map.get(key) : undefined;
        if (entry && map) {
            htmlElement.removeEventListener(eventName, entry.dispatch, entry.options);
            map.delete(key);
        }
        return;
    }
    if (!map) {
        map = new Map();
        host[MANAGED_LISTENERS] = map;
    }
    const existing = map.get(key);
    if (existing) {
        existing.handler = handler;
        return;
    }
    const entries = map;
    const entry: ManagedListenerEntry = {
        handler,
        options,
        dispatch(e: Event) {
            if (entry.options.once) {
                entries.delete(key);
            }
            const current = entry.handler;
            if (Array.isArray(current)) {
                for (const fn of current) {
                    invokeManagedHandler(fn, htmlElement, e);
                }
                return;
            }
            invokeManagedHandler(current, htmlElement, e);
        },
    };
    map.set(key, entry);
    htmlElement.addEventListener(eventName, entry.dispatch, options);
}

/**
 * Checks whether children inherit the SVG namespace from their parent. Children of
 * foreignObject elements use the HTML namespace.
 *
 * @param parent - The DOM node that will receive the children.
 * @returns True when the children use the SVG namespace.
 */
function isSvgParent(parent: Node): boolean {
    return parent instanceof Element
        && parent.namespaceURI === SVG_NS
        && parent.localName.toLowerCase() !== FOREIGN_OBJECT_TAG;
}
