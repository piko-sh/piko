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

import {HookEvent, type HookManager} from './HookManager';

/** Tracks the internal state of a form being monitored for changes. */
interface TrackedForm {
    /** The form element being tracked. */
    form: HTMLFormElement;
    /** Serialised snapshot of the form's initial state. */
    initialSnapshot: string;
    /** Whether the form has been modified from its initial state. */
    isDirty: boolean;
    /** Whether the initial snapshot is pending while custom elements initialise. */
    snapshotPending: boolean;
}

/** Manages form dirty state tracking for unsaved changes protection. */
export interface FormStateManager {
    /**
     * Starts tracking a form's state. Skips forms with the pk-no-track attribute.
     * @param form - The form element to track.
     */
    trackForm(form: HTMLFormElement): void;

    /**
     * Stops tracking a form and removes the tracking marker attribute.
     * @param form - The form element to untrack.
     */
    untrackForm(form: HTMLFormElement): void;

    /**
     * Marks a form as clean by resetting its initial snapshot to the current state.
     * @param form - The form element to mark clean.
     */
    markFormClean(form: HTMLFormElement): void;

    /**
     * Checks whether any tracked form has unsaved changes.
     * @returns True if at least one form is dirty.
     */
    hasDirtyForms(): boolean;

    /**
     * Gets the list of identifiers for all dirty forms.
     * @returns An array of form identifiers.
     */
    getDirtyFormIds(): string[];

    /**
     * Prompts the user if there are unsaved changes before navigation.
     * @returns True if navigation should proceed, false to cancel.
     */
    confirmNavigation(): boolean;

    /**
     * Scans the DOM for forms and tracks any new ones. Removes tracking for forms no longer in the DOM.
     * @param root - The root element to scan within.
     */
    scanAndTrackForms(root?: Element): void;

    /** Stops tracking all forms. Used when navigation is confirmed to prevent stale dirty state. */
    untrackAll(): void;

    /** Cleans up event listeners and clears all tracked forms. */
    destroy(): void;
}

/** Dependencies for creating a FormStateManager. */
export interface FormStateManagerDependencies {
    /** Hook manager for emitting form state events. */
    hookManager: HookManager;
    /** Optional confirm function for testing. Defaults to window.confirm. */
    confirmFn?: (message: string) => boolean;
}

/** Attribute to opt out of form tracking. */
const NO_TRACK_ATTR = 'pk-no-track';

/** Data attribute marking a form as tracked. */
const TRACKED_ATTR = 'data-pk-tracked';

/** Number of trailing characters to use from the form action for ID generation. */
const ACTION_SUFFIX_LENGTH = 20;

/** Maximum time in milliseconds to wait for custom elements before snapshotting anyway. */
const DEFERRED_SNAPSHOT_TIMEOUT_MS = 5000;

/**
 * Serialises form state into a comparable string, including checkbox and radio states.
 * Entries are sorted for consistent comparison.
 * @param form - The form element to snapshot.
 * @returns A JSON string representing the form's current state.
 */
function getFormSnapshot(form: HTMLFormElement): string {
    const data = new FormData(form);
    const entries: [string, string][] = [];

    for (const [key, value] of data.entries()) {
        if (typeof value === 'string') {
            entries.push([key, value]);
        }
    }

    const checkboxes = form.querySelectorAll<HTMLInputElement>('input[type="checkbox"], input[type="radio"]');
    for (const checkbox of checkboxes) {
        if (checkbox.name) {
            entries.push([`__checked_${checkbox.name}_${checkbox.value}`, String(checkbox.checked)]);
        }
    }

    entries.sort((a, b) => a[0].localeCompare(b[0]));
    return JSON.stringify(entries);
}

/**
 * Generates a stable identifier for a form, falling back to a position-based pseudo-ID.
 * @param form - The form element to identify.
 * @returns The form's identifier string.
 */
function getFormId(form: HTMLFormElement): string {
    if (form.id) {
        return form.id;
    }
    const action = form.action || 'no-action';
    const forms = document.querySelectorAll('form');
    const index = Array.from(forms).indexOf(form);
    return `form-${index}-${action.slice(-ACTION_SUFFIX_LENGTH)}`;
}

/**
 * Compares two snapshots and returns a list of fields that differ.
 * @param initial - The initial snapshot JSON string.
 * @param current - The current snapshot JSON string.
 * @returns An array of objects describing each changed field.
 */
function diffSnapshots(initial: string, current: string): Array<{field: string; initial: string; current: string}> {
    const initialEntries = JSON.parse(initial) as [string, string][];
    const currentEntries = JSON.parse(current) as [string, string][];

    const initialMap = new Map(initialEntries);
    const currentMap = new Map(currentEntries);
    const diffs: Array<{field: string; initial: string; current: string}> = [];

    for (const [key, value] of currentMap) {
        const initialValue = initialMap.get(key);
        if (initialValue === undefined) {
            diffs.push({field: key, initial: '(absent)', current: value});
        } else if (initialValue !== value) {
            diffs.push({field: key, initial: initialValue, current: value});
        }
    }

    for (const [key, value] of initialMap) {
        if (!currentMap.has(key)) {
            diffs.push({field: key, initial: value, current: '(absent)'});
        }
    }

    return diffs;
}

/**
 * Updates the dirty state of a tracked form and emits hook events on transitions.
 * Logs changed fields on clean-to-dirty transitions to aid debugging of form-associated
 * custom elements that incorrectly report changed values.
 * @param form - The form element to check.
 * @param trackedForms - The map of currently tracked forms.
 * @param hookManager - The hook manager for emitting events.
 */
function updateDirtyState(
    form: HTMLFormElement,
    trackedForms: Map<HTMLFormElement, TrackedForm>,
    hookManager: HookManager
): void {
    const tracked = trackedForms.get(form);
    if (!tracked || tracked.snapshotPending) {
        return;
    }

    const currentSnapshot = getFormSnapshot(form);
    const wasDirty = tracked.isDirty;
    tracked.isDirty = currentSnapshot !== tracked.initialSnapshot;

    if (tracked.isDirty && !wasDirty) {
        const diffs = diffSnapshots(tracked.initialSnapshot, currentSnapshot);
        console.warn(
            `[pk] Form "${getFormId(form)}" is now dirty. Changed fields:`, diffs,
            `\n\nIf this form should not trigger unsaved changes warnings, add the "pk-no-track" attribute to the <form> element.`
            + `\nIf a custom element is incorrectly reporting a changed value, check that its setFormValue() call in connectedCallback matches the server-rendered initial state.`
        );
        hookManager.emit(HookEvent.FORM_DIRTY, {formId: getFormId(form), timestamp: Date.now()});
    } else if (!tracked.isDirty && wasDirty) {
        hookManager.emit(HookEvent.FORM_CLEAN, {formId: getFormId(form), timestamp: Date.now()});
    }
}

/**
 * Creates a delegated input handler that updates dirty state for tracked forms.
 * @param trackedForms - The map of currently tracked forms.
 * @param hookManager - The hook manager for emitting events.
 * @returns An event handler function.
 */
function createFormInputHandler(
    trackedForms: Map<HTMLFormElement, TrackedForm>,
    hookManager: HookManager
): (event: Event) => void {
    return (event: Event): void => {
        const target = event.target as Element | null;
        if (!target) {
            return;
        }
        const form = target.closest('form');
        if (form instanceof HTMLFormElement && trackedForms.has(form)) {
            updateDirtyState(form, trackedForms, hookManager);
        }
    };
}

/**
 * Checks whether any tracked form has unsaved changes, refreshing dirty state first.
 * @param trackedForms - The map of currently tracked forms.
 * @param hookManager - The hook manager for emitting events.
 * @returns True if at least one form is dirty.
 */
function checkHasDirtyForms(
    trackedForms: Map<HTMLFormElement, TrackedForm>,
    hookManager: HookManager
): boolean {
    for (const tracked of trackedForms.values()) {
        updateDirtyState(tracked.form, trackedForms, hookManager);
        if (tracked.isDirty) {
            return true;
        }
    }
    return false;
}

/**
 * Checks whether a form contains any custom elements (elements with a hyphen in their tag name).
 * Custom elements always contain a hyphen per the HTML spec, regardless of whether they
 * have been registered via customElements.define() yet.
 * @param form - The form element to check.
 * @returns True if the form contains at least one custom element.
 */
function containsCustomElements(form: HTMLFormElement): boolean {
    const allElements = form.querySelectorAll('*');
    for (const el of allElements) {
        if (el.localName.includes('-')) {
            return true;
        }
    }
    return false;
}

/**
 * Defers the initial snapshot of a form until custom elements have had time to initialise.
 * For undefined elements, waits for registration via customElements.whenDefined().
 * Then waits two animation frames for connectedCallback and post-render setFormValue()
 * calls to complete before snapshotting.
 * Falls back to snapshotting after a timeout if elements are never defined.
 * @param form - The form element awaiting its initial snapshot.
 * @param trackedForms - The map of currently tracked forms.
 */
async function deferFormSnapshot(
    form: HTMLFormElement,
    trackedForms: Map<HTMLFormElement, TrackedForm>
): Promise<void> {
    const undefinedElements = form.querySelectorAll(':not(:defined)');

    if (undefinedElements.length > 0) {
        const tagNames = new Set<string>();
        for (const el of undefinedElements) {
            tagNames.add(el.localName);
        }

        const timeout = new Promise<'timeout'>(resolve =>
            setTimeout(() => resolve('timeout'), DEFERRED_SNAPSHOT_TIMEOUT_MS)
        );

        const result = await Promise.race([
            Promise.all(
                Array.from(tagNames).map(name => customElements.whenDefined(name))
            ).then(() => 'defined' as const),
            timeout
        ]);

        if (result === 'timeout') {
            console.warn(`[pk] Timed out waiting for custom elements in form "${getFormId(form)}":`,
                Array.from(tagNames));
        }
    }

    await new Promise<void>(resolve => requestAnimationFrame(() => {
        requestAnimationFrame(() => resolve());
    }));

    const tracked = trackedForms.get(form);
    if (!tracked) {
        return;
    }

    tracked.initialSnapshot = getFormSnapshot(form);
    tracked.snapshotPending = false;
}

/**
 * Tracks a form if it is eligible (not already tracked, no opt-out attribute).
 * When the form contains custom elements, the initial snapshot is deferred to allow
 * connectedCallback and post-render setFormValue() calls to complete. This handles both
 * first-visit (elements undefined) and return-visit (elements already defined but freshly
 * inserted) scenarios.
 * @param form - The form element to track.
 * @param trackedForms - The map of currently tracked forms.
 */
function internalTrackForm(form: HTMLFormElement, trackedForms: Map<HTMLFormElement, TrackedForm>): void {
    if (trackedForms.has(form) || form.hasAttribute(NO_TRACK_ATTR)) {
        return;
    }

    if (containsCustomElements(form)) {
        trackedForms.set(form, {form, initialSnapshot: '', isDirty: false, snapshotPending: true});
        void deferFormSnapshot(form, trackedForms);
    } else {
        trackedForms.set(form, {form, initialSnapshot: getFormSnapshot(form), isDirty: false, snapshotPending: false});
    }

    form.setAttribute(TRACKED_ATTR, 'true');
}

/**
 * Collects all form elements from a node, including the node itself if it is a form.
 * @param node - The DOM node to search within.
 * @returns An array of eligible form elements.
 */
function collectForms(node: Node): HTMLFormElement[] {
    const forms: HTMLFormElement[] = [];
    if (node instanceof HTMLFormElement && !node.hasAttribute(NO_TRACK_ATTR)) {
        forms.push(node);
    }
    if (node instanceof HTMLElement) {
        const nested = node.querySelectorAll<HTMLFormElement>('form:not([pk-no-track])');
        for (const form of nested) {
            forms.push(form);
        }
    }
    return forms;
}

/**
 * Creates a MutationObserver that automatically tracks forms when they are added
 * to the DOM and untracks them when removed. This handles forms rendered by
 * partials, custom elements, and other asynchronous DOM insertions.
 * @param trackedForms - The map of currently tracked forms.
 * @returns The MutationObserver instance.
 */
function createFormObserver(trackedForms: Map<HTMLFormElement, TrackedForm>): MutationObserver {
    return new MutationObserver((mutations) => {
        for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
                for (const form of collectForms(node)) {
                    internalTrackForm(form, trackedForms);
                }
            }
            for (const node of mutation.removedNodes) {
                for (const form of collectForms(node)) {
                    trackedForms.delete(form);
                    form.removeAttribute(TRACKED_ATTR);
                }
            }
        }
    });
}

/**
 * Creates a submit handler that resets the dirty state of the submitted form.
 * @param trackedForms - The map of tracked forms.
 * @param hookManager - The hook manager for emitting form events.
 * @returns An event handler for form submit events.
 */
function createFormSubmitHandler(
    trackedForms: Map<HTMLFormElement, TrackedForm>,
    hookManager: HookManager
): (event: Event) => void {
    return (event: Event): void => {
        const form = event.target;
        if (!(form instanceof HTMLFormElement)) {
            return;
        }
        const tracked = trackedForms.get(form);
        if (!tracked) {
            return;
        }
        tracked.initialSnapshot = getFormSnapshot(form);
        tracked.snapshotPending = false;
        const wasDirty = tracked.isDirty;
        tracked.isDirty = false;
        if (wasDirty) {
            hookManager.emit(HookEvent.FORM_CLEAN, {formId: getFormId(form), timestamp: Date.now()});
        }
    };
}

/**
 * Creates a beforeunload handler that prevents navigation when dirty forms exist.
 * @param hasDirtyForms - A function that returns whether any tracked forms are dirty.
 * @returns A beforeunload event handler.
 */
function createBeforeUnloadHandler(hasDirtyForms: () => boolean): (event: BeforeUnloadEvent) => void {
    return (event: BeforeUnloadEvent): void => {
        if (hasDirtyForms()) {
            event.preventDefault();
            event.returnValue = '';
        }
    };
}

/**
 * Sets up the MutationObserver and global event listeners for form tracking.
 * @param trackedForms - The map of tracked forms.
 * @param handleFormInput - The input/change event handler.
 * @param handleFormSubmit - The submit event handler.
 * @param handleBeforeUnload - The beforeunload event handler.
 * @returns The MutationObserver instance for later disconnection.
 */
function setupFormListeners(
    trackedForms: Map<HTMLFormElement, TrackedForm>,
    handleFormInput: (event: Event) => void,
    handleFormSubmit: (event: Event) => void,
    handleBeforeUnload: (event: BeforeUnloadEvent) => void
): MutationObserver {
    const formObserver = createFormObserver(trackedForms);
    formObserver.observe(document.body, {childList: true, subtree: true});

    document.addEventListener('input', handleFormInput);
    document.addEventListener('change', handleFormInput);
    document.addEventListener('submit', handleFormSubmit);
    window.addEventListener('beforeunload', handleBeforeUnload);

    return formObserver;
}

/**
 * Creates a FormStateManager for tracking form dirty state with beforeunload protection.
 * Uses a MutationObserver to automatically track forms as they appear in the DOM,
 * handling forms rendered by partials, custom elements, and other deferred insertions.
 * @param deps - The dependencies including the hook manager.
 * @returns A new FormStateManager instance.
 */
export function createFormStateManager(deps: FormStateManagerDependencies): FormStateManager {
    const {hookManager} = deps;
    const confirmFn = deps.confirmFn ?? ((message: string) => window.confirm(message));
    const trackedForms = new Map<HTMLFormElement, TrackedForm>();

    const handleFormInput = createFormInputHandler(trackedForms, hookManager);
    const hasDirtyForms = (): boolean => checkHasDirtyForms(trackedForms, hookManager);
    const handleBeforeUnload = createBeforeUnloadHandler(hasDirtyForms);
    const handleFormSubmit = createFormSubmitHandler(trackedForms, hookManager);
    const formObserver = setupFormListeners(trackedForms, handleFormInput, handleFormSubmit, handleBeforeUnload);

    return {
        trackForm(form: HTMLFormElement): void {
            internalTrackForm(form, trackedForms);
        },
        untrackForm(form: HTMLFormElement): void {
            trackedForms.delete(form);
            form.removeAttribute(TRACKED_ATTR);
        },

        markFormClean(form: HTMLFormElement): void {
            const tracked = trackedForms.get(form);
            if (!tracked) {
                return;
            }
            tracked.initialSnapshot = getFormSnapshot(form);
            tracked.snapshotPending = false;
            const wasDirty = tracked.isDirty;
            tracked.isDirty = false;
            if (wasDirty) {
                hookManager.emit(HookEvent.FORM_CLEAN, {formId: getFormId(form), timestamp: Date.now()});
            }
        },

        hasDirtyForms,

        getDirtyFormIds(): string[] {
            const dirty: string[] = [];
            for (const tracked of trackedForms.values()) {
                updateDirtyState(tracked.form, trackedForms, hookManager);
                if (tracked.isDirty) {
                    dirty.push(getFormId(tracked.form));
                }
            }
            return dirty;
        },

        confirmNavigation: () => !hasDirtyForms() || confirmFn('You have unsaved changes. Leave anyway?'),

        scanAndTrackForms(root: Element = document.body): void {
            for (const form of trackedForms.keys()) {
                if (!document.contains(form)) {
                    trackedForms.delete(form);
                }
            }
            const forms = root.querySelectorAll<HTMLFormElement>('form:not([pk-no-track])');
            for (const form of forms) {
                internalTrackForm(form, trackedForms);
            }
        },

        untrackAll(): void {
            for (const form of trackedForms.keys()) {
                form.removeAttribute(TRACKED_ATTR);
            }
            trackedForms.clear();
        },

        destroy(): void {
            formObserver.disconnect();
            document.removeEventListener('input', handleFormInput);
            document.removeEventListener('change', handleFormInput);
            document.removeEventListener('submit', handleFormSubmit);
            window.removeEventListener('beforeunload', handleBeforeUnload);
            trackedForms.clear();
        }
    };
}
