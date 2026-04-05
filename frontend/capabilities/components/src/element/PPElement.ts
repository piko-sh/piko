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

import {dom, VirtualNode} from '@/vdom';
import {getBehaviour} from '@/behaviours';
import type {CustomValidityResult, PropTypeDefinition, SlotChangeCallback, StateContext} from '@/element/types';
import {
    type AttributeSyncService,
    type BehaviourApplicator,
    createAttributeSyncService,
    createBehaviourApplicator,
    createFormAssociation,
    createLifecycleManager,
    createPropTypeRegistry,
    createRenderScheduler,
    createShadowDOMService,
    createSlotManager,
    createStateManager,
    type FormAssociation,
    type LifecycleManager,
    propertyToAttributeName,
    type PropTypeRegistry,
    type RenderScheduler,
    type ShadowDOMService,
    type SlotManager,
    type StateManager,
} from '@/element/services';

export {PropTypeDefinition};

/**
 * Base class for all Piko web components.
 */
export class PPElement extends HTMLElement {
    /** Prop type definitions for this component class. */
    static propTypes?: Record<string, PropTypeDefinition>;
    /** Names of behaviours to apply during construction. */
    static enabledBehaviours: string[] = [];
    /** Whether this component participates in form submission. */
    static formAssociated = false;

    /** Registry for prop type definitions and attribute mapping. */
    private _propTypeRegistry!: PropTypeRegistry;
    /** Manages component state and change tracking. */
    private _stateManager!: StateManager;
    /** Handles form-associated custom element internals. */
    private _formAssociation!: FormAssociation;
    /** Applies registered behaviours to the component. */
    private _behaviourApplicator!: BehaviourApplicator;
    /** Manages shadow DOM creation and CSS injection. */
    private _shadowDOMService!: ShadowDOMService;
    /** Manages slot content queries and change listeners. */
    private _slotManager!: SlotManager;
    /** Manages lifecycle callback registration and execution. */
    private _lifecycleManager!: LifecycleManager;
    /** Synchronises HTML attributes with component state. */
    private _attributeSyncService!: AttributeSyncService;
    /** Schedules and performs VDOM renders. */
    private _renderScheduler!: RenderScheduler;

    /** Cached default values derived from prop type definitions. */
    private _defaults?: Record<string, unknown>;

    /** Called when the component is associated with a form. */
    formAssociatedCallback?: (form: HTMLFormElement | null) => void;
    /** Called when the form owner sets the component's disabled state. */
    formDisabledCallback?: (disabled: boolean) => void;
    /** Called when the associated form is reset. */
    formResetCallback?: () => void;
    /** Called when the browser restores form state (e.g. back/forward navigation). */
    formStateRestoreCallback?: (state: string | File | FormData | null, mode: 'restore' | 'autocomplete') => void;
    /** Checks form validity without showing the browser's validation UI. */
    checkValidity?: () => boolean;
    /** Checks form validity and shows the browser's validation UI if invalid. */
    reportValidity?: () => boolean;
    /** Custom validity check method. Override in form-associated subclasses. */
    customValidity?: () => CustomValidityResult | null;
    /** Internal method that synchronises form value and validity state. */
    _updateFormState?: () => void;

    /** Map of named element references populated during rendering. */
    public refs: Record<string, Node> = {};

    /** Returns the previous virtual node from the last render. */
    protected get oldVDOM(): VirtualNode | null {
        return this._renderScheduler.getOldVDOM();
    }

    /** Returns whether a render is currently scheduled. */
    protected get renderScheduled(): boolean {
        return this._renderScheduler.isScheduled();
    }

    /** No-op setter kept for backward compatibility with tests. */
    protected set renderScheduled(_value: boolean) {
    }

    /** Returns the computed default values from prop type definitions. */
    protected get defaults(): Record<string, unknown> {
        return this._getDefaults();
    }

    /** Returns the set of property names that have changed since the last render. */
    protected get changedPropsSet(): Set<string> {
        return this._stateManager.getChangedProps();
    }

    /** Creates a new PPElement instance and initialises all composed services. */
    constructor() {
        super();
        this._initServices();
    }

    /**
     * Creates and wires all composed services in dependency order.
     */
    private _initServices(): void {
        const constructor = this.constructor as typeof PPElement;

        this._propTypeRegistry = createPropTypeRegistry({
            propTypes: constructor.propTypes,
        });

        this._stateManager = createStateManager({
            tagName: this.tagName,
        });

        this._formAssociation = createFormAssociation({
            host: this,
            formAssociated: constructor.formAssociated,
        });

        this._behaviourApplicator = createBehaviourApplicator({
            host: this,
            enabledBehaviours: constructor.enabledBehaviours,
            getBehaviour,
        });

        this._shadowDOMService = createShadowDOMService({
            host: this,
            componentCSS: constructor.css,
            delegatesFocus: constructor.formAssociated,
        });

        this._slotManager = createSlotManager({
            getShadowRoot: () => this.shadowRoot,
            tagName: this.tagName,
        });

        this._lifecycleManager = createLifecycleManager();

        this._attributeSyncService = createAttributeSyncService({
            host: this,
            propTypeRegistry: this._propTypeRegistry,
            getState: () => this._stateManager.getState(),
            getDefaults: () => this._getDefaults(),
        });

        this._renderScheduler = createRenderScheduler({
            getShadowRoot: () => this.shadowRoot,
            isConnected: () => this.isConnected,
            isInitialising: () => this._attributeSyncService.isInitialising(),
            renderVDOM: () => this.renderVDOM(),
            refs: this.refs,
            lifecycleManager: this._lifecycleManager,
            stateManager: this._stateManager,
            attributeSyncService: this._attributeSyncService,
            onRenderComplete: () => this._slotManager.flushPendingListeners(),
        });

        this._behaviourApplicator.applyBehaviours();
    }

    /**
     * Computes and caches default values from prop type definitions.
     *
     * @returns The cached defaults record.
     */
    private _getDefaults(): Record<string, unknown> {
        if (this._defaults) {
            return this._defaults;
        }
        this._defaults = {};
        for (const propName of this._propTypeRegistry.getPropertyNames()) {
            const defaultValue = this._propTypeRegistry.getDefaultValue(propName);
            if (defaultValue !== undefined) {
                this._defaults[propName] = defaultValue;
            }
        }
        return this._defaults;
    }

    /**
     * Returns the list of attribute names to observe for changes.
     *
     * @returns The attribute names derived from prop type definitions.
     */
    static get observedAttributes(): string[] {
        const registry = createPropTypeRegistry({propTypes: this.propTypes});
        return registry.deriveObservedAttributes();
    }

    /**
     * Returns the component CSS to inject into the shadow root.
     *
     * @returns The CSS string, or undefined if none.
     */
    static get css(): string | undefined {
        return undefined;
    }

    /** Returns the component state context for backward compatibility. */
    get $$ctx(): StateContext | undefined {
        return this._stateManager.getContext();
    }

    /** Returns the ElementInternals instance for form-associated components. */
    get internals(): ElementInternals | undefined {
        return this._formAssociation.getInternals();
    }

    /** Returns the current component state object. */
    get state(): Record<string, unknown> | undefined {
        return this._stateManager.getState();
    }

    /**
     * Merges partial state into the component state and schedules a render.
     *
     * @param partialState - The state properties to update.
     */
    setState(partialState: Record<string, unknown>): void {
        this._stateManager.setState(partialState);
    }

    /**
     * Registers a callback to run when the component connects to the DOM.
     *
     * @param callback - The function to call on connection.
     */
    onConnected(callback: () => void): void {
        this._lifecycleManager.onConnected(callback);
    }

    /**
     * Registers a callback to run when the component disconnects from the DOM.
     *
     * @param callback - The function to call on disconnection.
     */
    onDisconnected(callback: () => void): void {
        this._lifecycleManager.onDisconnected(callback);
    }

    /**
     * Registers a callback to run before each render.
     *
     * @param callback - The function to call before rendering.
     */
    onBeforeRender(callback: () => void): void {
        this._lifecycleManager.onBeforeRender(callback);
    }

    /**
     * Registers a callback to run after each render.
     *
     * @param callback - The function to call after rendering.
     */
    onAfterRender(callback: () => void): void {
        this._lifecycleManager.onAfterRender(callback);
    }

    /**
     * Registers a callback to run when component properties change.
     *
     * @param callback - The function to call with the set of changed property names.
     */
    onUpdated(callback: (changedProperties: Set<string>) => void): void {
        this._lifecycleManager.onUpdated(callback);
    }

    /**
     * Registers a cleanup function to run when the component disconnects.
     *
     * Cleanups run after onDisconnected callbacks, then the array is cleared.
     * This allows co-locating setup and teardown logic.
     *
     * @param callback - The cleanup function to call on disconnection.
     */
    onCleanup(callback: () => void): void {
        this._lifecycleManager.onCleanup(callback);
    }

    /**
     * Returns the elements assigned to a named slot.
     *
     * @param slotName - The slot name, or empty string for the default slot.
     * @returns The array of slotted elements.
     */
    getSlottedElements(slotName = ""): Element[] {
        return this._slotManager.getSlottedElements(slotName);
    }

    /**
     * Attaches a listener for slot content changes.
     *
     * @param slotName - The slot name to watch.
     * @param callback - The function to call when slot content changes.
     */
    attachSlotListener(slotName: string, callback: SlotChangeCallback): void {
        this._slotManager.attachSlotListener(slotName, callback);
    }

    /**
     * Checks whether a slot has any assigned content.
     *
     * @param slotName - The slot name, or empty string for the default slot.
     * @returns True if the slot has content.
     */
    hasSlotContent(slotName = ""): boolean {
        return this._slotManager.hasSlotContent(slotName);
    }

    /**
     * Returns the virtual DOM tree for this component.
     *
     * Subclasses must override this method with their template logic.
     *
     * @returns The root virtual node.
     */
    renderVDOM(): VirtualNode {
        console.warn(`PPElement: renderVDOM() called on base class for ${this.tagName}.`);
        return dom.cmt(`No VDOM for ${this.tagName}`, `${this.tagName}-default-vdom`);
    }

    /** Performs an immediate synchronous render. */
    render(): void {
        this._renderScheduler.render();
    }

    /** Schedules an asynchronous render on the next microtask. */
    scheduleRender(): void {
        this._renderScheduler.scheduleRender();
    }

    /**
     * Initialises the component with a state context.
     *
     * Sets up state defaults, synchronises HTML attributes to state,
     * reflects state back to attributes, and triggers the first render
     * if the element is already connected.
     *
     * @param optsFromInstance - The state context provided by the compiled component.
     */
    init(optsFromInstance: StateContext): void {
        this._attributeSyncService.setInitialising(true);
        this._renderScheduler.setPendingAfterInit(false);

        this._stateManager.setContext(optsFromInstance);
        this._shadowDOMService.ensureShadowRoot();

        const state = this._stateManager.getState();
        if (state) {
            const defaults = this._getDefaults();
            for (const propName of this._propTypeRegistry.getPropertyNames()) {
                if (!(propName in state) && defaults[propName] !== undefined) {
                    state[propName] = defaults[propName];
                }
            }
        }

        this._attributeSyncService.syncAllAttributesToState();
        this._attributeSyncService.reflectAllStateToAttributes();

        this._attributeSyncService.setInitialising(false);
        this._stateManager.clearChangedProps();

        if (this._renderScheduler.hasPendingAfterInit()) {
            this.scheduleRender();
        } else if (this.isConnected && !this._renderScheduler.hasRendered()) {
            this.render();
            this._lifecycleManager.executeConnected();
        }
    }

    /** Called when the element is inserted into the DOM. */
    connectedCallback(): void {
        if (!this._stateManager.hasState()) {
            console.warn(
                `PPElement ${this.tagName}: connectedCallback - init() incomplete or $$ctx not set.`
            );
        }
        if (
            this._stateManager.hasState() &&
            !this._attributeSyncService.isInitialising() &&
            this.isConnected
        ) {
            if (!this._renderScheduler.hasRendered()) {
                this.scheduleRender();
            } else {
                this._lifecycleManager.executeConnected();
            }
        }
    }

    /** Called when the element is removed from the DOM. */
    disconnectedCallback(): void {
        this._lifecycleManager.executeDisconnected();
        this._lifecycleManager.executeCleanups();
        this._lifecycleManager.resetConnectedState();
    }

    /**
     * Called when an observed attribute changes.
     *
     * Skips processing while reflecting to attributes or during initialisation.
     *
     * @param attributeName - The name of the changed attribute.
     * @param _oldValue - The previous attribute value.
     * @param newValue - The new attribute value.
     */
    attributeChangedCallback(
        attributeName: string,
        _oldValue: string | null,
        newValue: string | null
    ): void {
        if (
            this._attributeSyncService.isReflectingToAttribute() ||
            !this._stateManager.getState() ||
            this._attributeSyncService.isInitialising()
        ) {
            return;
        }

        const propertyName = this._propTypeRegistry.attributeToPropertyName(attributeName);
        const propDef = this._propTypeRegistry.get(propertyName);

        if (propDef) {
            this.applyHtmlAttributeToState(propertyName, newValue);
        }
    }

    /**
     * Applies an HTML attribute value to the component state.
     *
     * @param propertyName - The property name mapped from the attribute.
     * @param attributeValue - The new attribute value.
     */
    protected applyHtmlAttributeToState(propertyName: string, attributeValue: string | null): void {
        this._attributeSyncService.applyAttributeToState(propertyName, attributeValue);
    }

    /**
     * Reflects a state property value back to an HTML attribute.
     *
     * @param propertyName - The property name to reflect.
     * @param propertyValue - The property value to set as an attribute.
     */
    protected reflectStatePropertyToAttribute(propertyName: string, propertyValue: unknown): void {
        this._attributeSyncService.reflectStateToAttribute(propertyName, propertyValue);
    }

    /**
     * Translates a raw attribute string into a typed property value.
     *
     * @param typeHint - The expected type hint for conversion.
     * @param attributeValue - The raw attribute value.
     * @param propertyName - The target property name.
     * @param isNullable - Whether the property accepts null values.
     * @returns The converted property value.
     */
    protected translateAttributeValue(
        typeHint: string,
        attributeValue: string | null,
        propertyName: string,
        isNullable = false
    ): unknown {
        return this._attributeSyncService.translateAttributeValue(
            typeHint as import("./types").PropType,
            attributeValue,
            propertyName,
            isNullable
        );
    }

    /**
     * Converts an HTML attribute name to a component property name.
     *
     * @param attributeName - The attribute name to convert.
     * @returns The corresponding property name.
     */
    protected attributeNameToPropertyName(attributeName: string): string {
        return this._propTypeRegistry.attributeToPropertyName(attributeName);
    }

    /**
     * Converts a property name to an HTML attribute name.
     *
     * @param propertyName - The property name to convert.
     * @returns The corresponding attribute name.
     */
    static propertyNameToAttributeName(propertyName: string): string {
        return propertyToAttributeName(propertyName);
    }
}
