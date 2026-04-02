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

import {autoRefresh} from './coordination';

/** Tracks active refreshers for cleanup, keyed by element. */
const activeRefreshers = new Map<Element, () => void>();

/**
 * Processes a single element with data-auto-refresh.
 *
 * Reads the interval, partial name, conditional refresh strategy, and error
 * handling configuration from data attributes and starts auto-refresh.
 *
 * @param el - The element to process.
 */
function processElement(el: HTMLElement): void {
    if (activeRefreshers.has(el)) {
        return;
    }

    const intervalStr = el.dataset.autoRefresh;
    const partialName = el.dataset.partial;

    if (!intervalStr || !partialName) {
        return;
    }

    const interval = parseInt(intervalStr, 10);
    if (isNaN(interval) || interval <= 0) {
        console.warn(`[pk] Invalid auto-refresh interval: "${intervalStr}" on element`, el);
        return;
    }

    const whenCondition = el.dataset.autoRefreshWhen;
    let when: (() => boolean) | undefined;

    if (whenCondition === 'visible') {
        when = () => document.visibilityState === 'visible';
    } else if (whenCondition === 'focus') {
        when = () => document.hasFocus();
    }

    const onErrorStr = el.dataset.autoRefreshOnError;
    const onError: 'retry' | 'stop' = onErrorStr === 'stop' ? 'stop' : 'retry';

    const cleanup = autoRefresh(partialName, {
        interval,
        when,
        onError
    });

    activeRefreshers.set(el, cleanup);
}

/**
 * Cleans up auto-refresh for a single element.
 *
 * @param el - The element to clean up.
 */
function cleanupElement(el: Element): void {
    const cleanup = activeRefreshers.get(el);
    if (cleanup) {
        cleanup();
        activeRefreshers.delete(el);
    }
}

/**
 * Cleans up auto-refresh for an element and all its descendants.
 *
 * @param el - The root element to clean up recursively.
 */
function cleanupElementAndDescendants(el: Element): void {
    cleanupElement(el);

    const descendants = el.querySelectorAll('[data-auto-refresh]');
    descendants.forEach(cleanupElement);
}

/**
 * Initialises the auto-refresh observer.
 *
 * Call this once during application initialisation.
 * It will automatically manage data-auto-refresh elements.
 */
export function initAutoRefreshObserver(): void {
    const existingElements = document.querySelectorAll<HTMLElement>('[data-auto-refresh]');
    existingElements.forEach(processElement);

    /**
     * Processes a newly added DOM node, starting auto-refresh if applicable.
     *
     * @param node - The added node.
     */
    function processAddedNode(node: Node): void {
        if (!(node instanceof HTMLElement)) {
            return;
        }
        if (node.hasAttribute('data-auto-refresh')) {
            processElement(node);
        }
        node.querySelectorAll<HTMLElement>('[data-auto-refresh]').forEach(processElement);
    }

    /**
     * Processes a removed DOM node, stopping any active auto-refresh.
     *
     * @param node - The removed node.
     */
    function processRemovedNode(node: Node): void {
        if (node instanceof HTMLElement) {
            cleanupElementAndDescendants(node);
        }
    }

    const observer = new MutationObserver(mutations => {
        for (const mutation of mutations) {
            mutation.addedNodes.forEach(processAddedNode);
            mutation.removedNodes.forEach(processRemovedNode);
        }
    });

    observer.observe(document.body, {
        childList: true,
        subtree: true
    });
}

/**
 * Stops all active auto-refreshers.
 *
 * Useful for cleanup or testing.
 */
export function stopAllAutoRefreshers(): void {
    for (const cleanup of activeRefreshers.values()) {
        cleanup();
    }
    activeRefreshers.clear();
}

/**
 * Returns the count of active auto-refreshers.
 *
 * Useful for debugging or testing.
 *
 * @returns The number of currently active auto-refreshers.
 */
export function getActiveRefresherCount(): number {
    return activeRefreshers.size;
}
