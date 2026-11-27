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
    /**
     * Custom elements that were not defined when tracking began. Their fields are
     * left out of the snapshot until each one is defined and has set its value.
     */
    pendingElements: Set<Element>;
}

/** A single serialised snapshot entry, as a key and value pair. */
type SnapshotEntry = [string, string];

/** Decides whether a field, given by name, belongs in a snapshot. */
type FieldNamePredicate = (name: string) => boolean;

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

/** Key prefix for the checked state of checkboxes and radio buttons in a snapshot. */
const CHECKED_KEY_PREFIX = '__checked_';

/**
 * Checks whether a form field is inside a pk-no-track container. Fields nested
 * inside an element with the pk-no-track attribute are excluded from dirty
 * tracking. This allows captcha widgets, CSRF tokens, and other programmatic
 * fields to opt out without affecting the rest of the form.
 * @param el - The form field element to check.
 * @returns True if the field should be excluded from tracking.
 */
function isInsideNoTrackContainer(el: Element): boolean {
    return el.closest(`[${NO_TRACK_ATTR}]`) !== null;
}

/**
 * Collects the snapshot entries of the fields that the predicate accepts,
 * including checkbox and radio states.
 * @param form - The form element to read.
 * @param includeField - Decides, by field name, which fields to read.
 * @returns The unsorted snapshot entries.
 */
function collectSnapshotEntries(form: HTMLFormElement, includeField: FieldNamePredicate): SnapshotEntry[] {
    const entries: SnapshotEntry[] = [];

    for (const [key, value] of new FormData(form).entries()) {
        if (typeof value === 'string' && includeField(key)) {
            entries.push([key, value]);
        }
    }

    const checkboxes = form.querySelectorAll<HTMLInputElement>('input[type="checkbox"], input[type="radio"]');
    for (const checkbox of checkboxes) {
        if (checkbox.name && includeField(checkbox.name)) {
            entries.push([`${CHECKED_KEY_PREFIX}${checkbox.name}_${checkbox.value}`, String(checkbox.checked)]);
        }
    }

    return entries;
}

/**
 * Sorts snapshot entries and serialises them into a comparable string.
 * @param entries - The snapshot entries.
 * @returns A JSON string of the sorted entries.
 */
function serialiseEntries(entries: SnapshotEntry[]): string {
    entries.sort((a, b) => a[0].localeCompare(b[0]));
    return JSON.stringify(entries);
}

/**
 * Serialises form state into a comparable string, including checkbox and radio
 * states. Fields inside pk-no-track containers and fields of custom elements
 * that are not ready yet are left out. Entries are sorted for consistent
 * comparison.
 * @param tracked - The tracked form to snapshot.
 * @returns A JSON string representing the form's current state.
 */
function getFormSnapshot(tracked: TrackedForm): string {
    const excludedNames = collectNoTrackFieldNames(tracked.form);
    for (const name of collectPendingFieldNames(tracked)) {
        excludedNames.add(name);
    }
    return serialiseEntries(collectSnapshotEntries(tracked.form, name => !excludedNames.has(name)));
}

/**
 * Collects the field names that belong to a custom element, which are its own name
 * and the names of form controls inside it.
 * @param form - The form that holds the element.
 * @param element - The custom element.
 * @returns The set of field names.
 */
function collectElementFieldNames(form: HTMLFormElement, element: Element): Set<string> {
    const names = new Set<string>();
    const ownName = element.getAttribute('name');
    if (ownName) {
        names.add(ownName);
    }
    const controls = form.elements;
    for (let i = 0; i < controls.length; i++) {
        const control = controls[i] as HTMLElement & { name?: string };
        if (control.name && element.contains(control)) {
            names.add(control.name);
        }
    }
    return names;
}

/**
 * Collects the field names of every custom element in the form that is not
 * ready yet.
 * @param tracked - The tracked form.
 * @returns The set of field names to leave out of snapshots.
 */
function collectPendingFieldNames(tracked: TrackedForm): Set<string> {
    const names = new Set<string>();
    for (const element of tracked.pendingElements) {
        for (const name of collectElementFieldNames(tracked.form, element)) {
            names.add(name);
        }
    }
    return names;
}

/**
 * Checks whether a snapshot key belongs to one of the given fields.
 * @param key - The snapshot key.
 * @param names - The field names.
 * @returns True if the key is for one of the fields.
 */
function isKeyForFields(key: string, names: ReadonlySet<string>): boolean {
    if (names.has(key)) {
        return true;
    }
    if (!key.startsWith(CHECKED_KEY_PREFIX)) {
        return false;
    }
    const rest = key.slice(CHECKED_KEY_PREFIX.length);
    for (const name of names) {
        if (rest.startsWith(`${name}_`)) {
            return true;
        }
    }
    return false;
}

/**
 * Adds the current values of the given fields to the form's initial snapshot,
 * replacing any values already stored for them.
 * @param tracked - The tracked form.
 * @param names - The field names to add.
 */
function addFieldsToSnapshot(tracked: TrackedForm, names: ReadonlySet<string>): void {
    const noTrackNames = collectNoTrackFieldNames(tracked.form);
    const kept = (JSON.parse(tracked.initialSnapshot) as SnapshotEntry[])
        .filter(([key]) => !isKeyForFields(key, names));
    const added = collectSnapshotEntries(tracked.form, name => names.has(name) && !noTrackNames.has(name));
    tracked.initialSnapshot = serialiseEntries([...kept, ...added]);
}

/**
 * Collects the names of form fields that are inside pk-no-track containers.
 * @param form - The form element to scan.
 * @returns A set of field names to exclude from the snapshot.
 */
function collectNoTrackFieldNames(form: HTMLFormElement): Set<string> {
    const excluded = new Set<string>();
    const elements = form.elements;
    for (let i = 0; i < elements.length; i++) {
        const el = elements[i] as HTMLElement & { name?: string };
        if (el.name && isInsideNoTrackContainer(el)) {
            excluded.add(el.name);
        }
    }
    return excluded;
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

    const currentSnapshot = getFormSnapshot(tracked);
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
 * Waits two animation frames so that connectedCallback and post-render
 * setFormValue() calls can complete.
 * @returns A promise that resolves after two animation frames.
 */
function waitForSettle(): Promise<void> {
    return new Promise<void>(resolve => requestAnimationFrame(() => {
        requestAnimationFrame(() => resolve());
    }));
}

/**
 * Checks whether the given tracked entry is still the one tracking its form.
 * @param tracked - The tracked form entry.
 * @param trackedForms - The map of currently tracked forms.
 * @returns True if the entry is still current.
 */
function isStillTracked(tracked: TrackedForm, trackedForms: Map<HTMLFormElement, TrackedForm>): boolean {
    return trackedForms.get(tracked.form) === tracked;
}

/**
 * Records the initial snapshot of the fields that are ready, after the settle
 * frames. Fields of custom elements that are not defined yet are left out.
 * @param tracked - The tracked form awaiting its initial snapshot.
 * @param trackedForms - The map of currently tracked forms.
 */
async function recordReadySnapshot(
    tracked: TrackedForm,
    trackedForms: Map<HTMLFormElement, TrackedForm>
): Promise<void> {
    await waitForSettle();
    if (!isStillTracked(tracked, trackedForms)) {
        return;
    }
    tracked.initialSnapshot = getFormSnapshot(tracked);
    tracked.snapshotPending = false;
}

/**
 * Adds a custom element's fields to the initial snapshot once the element is
 * defined and has settled. If the element is never defined, its fields are
 * never part of the snapshot.
 * @param tracked - The tracked form that holds the element.
 * @param element - The custom element that is not defined yet.
 * @param readySnapshot - Resolves once the ready part has been recorded.
 * @param trackedForms - The map of currently tracked forms.
 */
async function addElementWhenDefined(
    tracked: TrackedForm,
    element: Element,
    readySnapshot: Promise<void>,
    trackedForms: Map<HTMLFormElement, TrackedForm>
): Promise<void> {
    await Promise.all([customElements.whenDefined(element.localName), readySnapshot]);
    await waitForSettle();
    if (!isStillTracked(tracked, trackedForms)) {
        return;
    }
    const names = collectElementFieldNames(tracked.form, element);
    tracked.pendingElements.delete(element);
    addFieldsToSnapshot(tracked, names);
}

/**
 * Defers the initial snapshot of a form until custom elements have had time to initialise.
 * The fields that are ready are recorded after two animation frames. Each custom element
 * that is not defined yet adds its fields to the snapshot once it is defined and has
 * settled. Until then its fields are left out of every dirty check.
 * @param tracked - The tracked form awaiting its initial snapshot.
 * @param trackedForms - The map of currently tracked forms.
 */
function deferFormSnapshot(
    tracked: TrackedForm,
    trackedForms: Map<HTMLFormElement, TrackedForm>
): void {
    for (const element of tracked.form.querySelectorAll(':not(:defined)')) {
        tracked.pendingElements.add(element);
    }

    const readySnapshot = recordReadySnapshot(tracked, trackedForms);

    for (const element of tracked.pendingElements) {
        void addElementWhenDefined(tracked, element, readySnapshot, trackedForms);
    }
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

    const tracked: TrackedForm = {
        form,
        initialSnapshot: '',
        isDirty: false,
        snapshotPending: false,
        pendingElements: new Set()
    };
    trackedForms.set(form, tracked);

    if (containsCustomElements(form)) {
        tracked.snapshotPending = true;
        deferFormSnapshot(tracked, trackedForms);
    } else {
        tracked.initialSnapshot = getFormSnapshot(tracked);
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
        tracked.initialSnapshot = getFormSnapshot(tracked);
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
            tracked.initialSnapshot = getFormSnapshot(tracked);
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
