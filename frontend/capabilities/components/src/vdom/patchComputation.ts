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

import type {VirtualNode} from './types';

/**
 * Checks if a value is undefined or null.
 *
 * @param x - The value to check.
 * @returns True if the value is undefined or null.
 */
export function isUndef(x: unknown): x is undefined | null {
    return x === undefined || x === null;
}

/**
 * Checks if a VirtualNode is hidden (has _c prop set to false).
 *
 * @param vnode - The virtual node to check.
 * @returns True if the node is hidden.
 */
export function isVNodeHidden(vnode: VirtualNode): boolean {
    if (vnode.props && Object.prototype.hasOwnProperty.call(vnode.props, '_c')) {
        return !vnode.props._c;
    }
    return false;
}

/**
 * Compares two VirtualNodes to determine if they represent the same element.
 *
 * Uses key, type, and tag for comparison.
 *
 * @param vnode1 - The first virtual node.
 * @param vnode2 - The second virtual node.
 * @returns True if the nodes represent the same element.
 */
export function areSameVNode(vnode1: VirtualNode, vnode2: VirtualNode): boolean {
    return vnode1.key === vnode2.key &&
        vnode1._type === vnode2._type &&
        vnode1.tag === vnode2.tag;
}

/**
 * Creates a map from child keys to their indices for efficient lookup.
 *
 * @param children - Array of child virtual nodes.
 * @param beginIndex - Start index for the range (inclusive).
 * @param endIndex - End index for the range (inclusive).
 * @returns Map of keys to indices.
 */
export function createKeyToOldIdxMap(
    children: (VirtualNode | undefined)[],
    beginIndex: number,
    endIndex: number
): Map<string, number> {
    const map = new Map<string, number>();
    for (let i = beginIndex; i <= endIndex; i++) {
        const childVNode = children[i];
        if (childVNode?.key != null) {
            map.set(childVNode.key, i);
        }
    }
    return map;
}

/**
 * Determines the type descriptor for a hidden node's comment placeholder.
 *
 * @param vnode - The virtual node to check.
 * @returns The type: 'fragment' or 'node'.
 */
export function getHiddenNodeType(vnode: VirtualNode): 'fragment' | 'node' {
    return vnode._type === 'fragment' ? 'fragment' : 'node';
}

/**
 * Generates a placeholder message for a hidden node comment.
 *
 * @param nodeType - The type of node: 'fragment' or 'node'.
 * @param key - The node's key, if any.
 * @param fallbackPrefix - Prefix for fallback key generation.
 * @returns The placeholder message string.
 */
export function createHiddenPlaceholderMessage(
    nodeType: 'fragment' | 'node',
    key: string | undefined,
    fallbackPrefix: string
): string {
    const keyPart = key ?? `err-${fallbackPrefix}`;
    return `hidden ${nodeType} _k=${keyPart}`;
}

/**
 * Decision type for patch operations.
 */
export type PatchDecision =
    | { action: 'noop' }
    | { action: 'remove' }
    | { action: 'create' }
    | { action: 'replace' }
    | { action: 'update' }
    | { action: 'update-hidden' }
    | { action: 'show-hidden' };

/**
 * Determines what patch operation should be performed on a vnode.
 *
 * @param oldVNode - The existing virtual node, or null.
 * @param newVNode - The new virtual node, or null.
 * @returns The patch decision indicating what action to take.
 */
export function computePatchDecision(
    oldVNode: VirtualNode | null,
    newVNode: VirtualNode | null
): PatchDecision {
    if (newVNode == null) {
        if (oldVNode != null) {
            return {action: 'remove'};
        }
        return {action: 'noop'};
    }

    if (oldVNode === newVNode) {
        return {action: 'noop'};
    }

    if (!oldVNode) {
        return {action: 'create'};
    }

    const oldIsHidden = isVNodeHidden(oldVNode);
    const newIsHidden = isVNodeHidden(newVNode);

    if (!areSameVNode(oldVNode, newVNode)) {
        return {action: 'replace'};
    }

    if (newIsHidden && oldIsHidden) {
        return {action: 'update-hidden'};
    }

    if (newIsHidden && !oldIsHidden) {
        return {action: 'update-hidden'};
    }

    if (!newIsHidden && oldIsHidden) {
        return {action: 'show-hidden'};
    }

    return {action: 'update'};
}

/**
 * Represents the result of comparing children for the updateChildren algorithm.
 */
export interface ChildComparisonResult {
    /** The type of match found between old and new children. */
    type: 'start-start' | 'end-end' | 'start-end' | 'end-start' | 'keyed' | 'new';
    /** Index in the old children array. */
    oldIndex?: number;
    /** Index in the new children array. */
    newIndex?: number;
}

/**
 * Compares children vnodes at the current pointer positions.
 *
 * @param oldStartVNode - The old start virtual node.
 * @param oldEndVNode - The old end virtual node.
 * @param newStartVNode - The new start virtual node.
 * @param newEndVNode - The new end virtual node.
 * @returns The type of match found, or 'none' if no match.
 */
export function compareChildrenPointers(
    oldStartVNode: VirtualNode | undefined,
    oldEndVNode: VirtualNode | undefined,
    newStartVNode: VirtualNode | undefined,
    newEndVNode: VirtualNode | undefined
): 'start-start' | 'end-end' | 'start-end' | 'end-start' | 'none' {
    if (oldStartVNode && newStartVNode && areSameVNode(oldStartVNode, newStartVNode)) {
        return 'start-start';
    }
    if (oldEndVNode && newEndVNode && areSameVNode(oldEndVNode, newEndVNode)) {
        return 'end-end';
    }
    if (oldStartVNode && newEndVNode && areSameVNode(oldStartVNode, newEndVNode)) {
        return 'start-end';
    }
    if (oldEndVNode && newStartVNode && areSameVNode(oldEndVNode, newStartVNode)) {
        return 'end-start';
    }
    return 'none';
}
