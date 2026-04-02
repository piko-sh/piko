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

import {patch, type VirtualNode} from '@/vdom';
import type {LifecycleManager} from '@/element/services/LifecycleManager';
import type {StateManager} from '@/element/services/StateManager';
import type {AttributeSyncService} from '@/element/services/AttributeSyncService';

/**
 * Coordinates VDOM rendering and animation frame scheduling.
 */
export interface RenderScheduler {
    /** Schedules a render for the next animation frame. */
    scheduleRender(): void;

    /** Performs an immediate render. */
    render(): void;

    /** Gets the current old VDOM tree for diffing. */
    getOldVDOM(): VirtualNode | null;

    /** Returns whether a render is currently scheduled. */
    isScheduled(): boolean;

    /** Returns whether the element has rendered at least once. */
    hasRendered(): boolean;

    /** Sets the pending render flag for initialisation phase. */
    setPendingAfterInit(value: boolean): void;

    /** Returns whether a render is pending after initialisation. */
    hasPendingAfterInit(): boolean;
}

/**
 * Options for creating a RenderScheduler.
 */
export interface RenderSchedulerOptions {
    /** Function to get the shadow root container. */
    getShadowRoot: () => ShadowRoot | null;

    /** Function to check if the element is connected to the DOM. */
    isConnected: () => boolean;

    /** Function to check if in initialisation phase. */
    isInitialising: () => boolean;

    /** Function that renders and returns the new VDOM tree. */
    renderVDOM: () => VirtualNode;

    /** Object for storing element references. */
    refs: Record<string, Node>;

    /** Lifecycle manager for callback execution. */
    lifecycleManager: LifecycleManager;

    /** State manager for tracking changed properties. */
    stateManager: StateManager;

    /** Attribute sync service for reflecting state to attributes. */
    attributeSyncService: AttributeSyncService;

    /** Optional callback invoked after each render cycle. */
    onRenderComplete?: () => void;
}

/**
 * Holds the current VDOM tree and configuration for a render cycle.
 */
interface RenderContext {
    /** The previous virtual DOM tree used for diffing. */
    oldVDOM: VirtualNode | null;
    /** The render scheduler configuration options. */
    options: RenderSchedulerOptions;
}

/**
 * Reflects changed properties to HTML attributes after a render cycle.
 *
 * @param ctx - The current render context.
 */
function reflectChangedProps(ctx: RenderContext): void {
    const {stateManager, attributeSyncService} = ctx.options;
    const changedProps = stateManager.getChangedProps();

    if (changedProps.size === 0) {
        return;
    }

    const changedPropsCopy = stateManager.clearChangedProps();
    const state = stateManager.getState();

    if (state) {
        changedPropsCopy.forEach((propName) => {
            if (Object.prototype.hasOwnProperty.call(state, propName)) {
                attributeSyncService.reflectStateToAttribute(propName, state[propName]);
            }
        });
    }

    ctx.options.lifecycleManager.executeUpdated(changedPropsCopy);
}

/**
 * Executes the core render logic: invokes lifecycle hooks, patches the VDOM, and reflects changed properties.
 *
 * @param ctx - The current render context.
 * @returns The new virtual DOM tree, or the existing one if rendering was skipped.
 */
function executeRender(ctx: RenderContext): VirtualNode | null {
    const {getShadowRoot, isConnected, renderVDOM, refs, lifecycleManager, onRenderComplete} =
        ctx.options;

    const shadowRoot = getShadowRoot();
    if (!shadowRoot || !isConnected()) {
        return ctx.oldVDOM;
    }

    lifecycleManager.executeBeforeRender();

    const newVirtualTree = renderVDOM();
    patch(ctx.oldVDOM, newVirtualTree, refs, shadowRoot, null);

    reflectChangedProps(ctx);

    lifecycleManager.executeAfterRender();

    onRenderComplete?.();

    return newVirtualTree;
}

/**
 * Creates a RenderScheduler for coordinating VDOM rendering and scheduling.
 *
 * @param options - Configuration options including callbacks and services.
 * @returns A new RenderScheduler instance.
 */
export function createRenderScheduler(options: RenderSchedulerOptions): RenderScheduler {
    let oldVDOM: VirtualNode | null = null;
    let renderScheduled = false;
    let pendingAfterInit = false;

    const ctx: RenderContext = {
        get oldVDOM() {
            return oldVDOM;
        },
        options,
    };

    return {
        scheduleRender(): void {
            if (options.isInitialising()) {
                pendingAfterInit = true;
                return;
            }

            if (!options.isConnected()) {
                return;
            }

            if (!renderScheduled) {
                renderScheduled = true;
                queueMicrotask(() => {
                    renderScheduled = false;
                    if (options.isConnected()) {
                        oldVDOM = executeRender(ctx);
                        options.lifecycleManager.executeConnected();
                    }
                });
            }
        },

        render(): void {
            oldVDOM = executeRender(ctx);
        },

        getOldVDOM(): VirtualNode | null {
            return oldVDOM;
        },

        isScheduled(): boolean {
            return renderScheduled;
        },

        hasRendered(): boolean {
            return oldVDOM !== null;
        },

        setPendingAfterInit(value: boolean): void {
            pendingAfterInit = value;
        },

        hasPendingAfterInit(): boolean {
            return pendingAfterInit;
        },
    };
}
