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

/** Node type constant for element nodes. */
const ELEMENT_NODE = 1;

/** Node type constant for text nodes. */
const TEXT_NODE = 3;

/** Node type constant for comment nodes. */
const COMMENT_NODE = 8;

/** Node type constant for document fragment nodes. */
const DOCUMENT_FRAGMENT_NODE = 11;

/** Reference to the document object, undefined in non-browser environments. */
const doc = typeof document === 'undefined' ? undefined : document;

/**
 * Key used to identify and match nodes during morphing.
 */
export type NodeKey = string | number | null;

/**
 * Configuration options for the fragment morpher.
 */
export interface MorphOptions {
    /** Custom function to extract a key from a node for matching. */
    getNodeKey?: (node: Node) => NodeKey;
    /** Called before a node is added. Return false to prevent addition. */
    onBeforeNodeAdded?: (node: Node) => Node | boolean;
    /** Called after a node has been added to the DOM. */
    onNodeAdded?: (node: Node) => void;
    /** Called before an element is updated. Return false to prevent update. */
    onBeforeElUpdated?: (fromEl: HTMLElement, toEl: HTMLElement) => boolean;
    /** Called after an element has been updated. */
    onElUpdated?: (el: HTMLElement) => void;
    /** Called before children of an element are updated. Return false to skip children. */
    onBeforeElChildrenUpdated?: (fromEl: HTMLElement, toEl: HTMLElement) => boolean;
    /** Called before a node is removed. Return false to prevent removal. */
    onBeforeNodeDiscarded?: (node: Node) => boolean;
    /** Called after a node has been removed from the DOM. */
    onNodeDiscarded?: (node: Node) => void;
    /** Initial preservation state: 'pk-refresh' to update, 'pk-no-refresh' to preserve. */
    initialState?: 'pk-refresh' | 'pk-no-refresh';
    /** Whether to morph only children, not the root element itself. */
    childrenOnly?: boolean;
    /** Preserve parent CSS scopes in partial attribute during morph (Level 1+). */
    preservePartialScopes?: boolean;
    /** For Level 2: only update these attributes, preserve all others. */
    ownedAttributes?: string[];
    /** Parent scopes to inherit for new nodes (computed from container). */
    _parentScopesToInherit?: string[];
}

/**
 * Synchronises a boolean attribute and its corresponding DOM property between two elements.
 *
 * @param fromEl - The live DOM element to update.
 * @param toEl - The target element containing the desired state.
 * @param name - The attribute/property name to synchronise.
 */
function syncBooleanAttrProp(fromEl: HTMLElement, toEl: HTMLElement, name: string) {
    const fromValue = (fromEl as unknown as Record<string, unknown>)[name];
    const toValue = (toEl as unknown as Record<string, unknown>)[name];
    if (fromValue !== toValue) {
        (fromEl as unknown as Record<string, unknown>)[name] = toValue;
        if (toValue) {
            fromEl.setAttribute(name, '');
        } else {
            fromEl.removeAttribute(name);
        }
    }
}

/** Handler function for synchronising element-specific state during morphing. */
type ElementHandler = (fromEl: HTMLElement, toEl: HTMLElement) => void;

/**
 * Synchronises input element state including checked, disabled, and value.
 *
 * @param fromEl - The live input element.
 * @param toEl - The target input element.
 */
function handleInput(fromEl: HTMLElement, toEl: HTMLElement): void {
    const from = fromEl as HTMLInputElement;
    const to = toEl as HTMLInputElement;
    syncBooleanAttrProp(from, to, 'checked');
    syncBooleanAttrProp(from, to, 'disabled');
    if (from !== doc?.activeElement && from.value !== to.value) {
        from.value = to.value;
    }
    if (from.type !== 'file' && !to.hasAttribute('value')) {
        from.removeAttribute('value');
    }
}

/**
 * Synchronises textarea element value.
 *
 * @param fromEl - The live textarea element.
 * @param toEl - The target textarea element.
 */
function handleTextarea(fromEl: HTMLElement, toEl: HTMLElement): void {
    const from = fromEl as HTMLTextAreaElement;
    const to = toEl as HTMLTextAreaElement;
    if (from.value !== to.value) {
        from.value = to.value;
    }
}

/**
 * Synchronises option element selected state.
 *
 * @param fromEl - The live option element.
 * @param toEl - The target option element.
 */
function handleOption(fromEl: HTMLElement, toEl: HTMLElement): void {
    syncBooleanAttrProp(fromEl, toEl, 'selected');
}

/**
 * Synchronises select element state including multiple mode and selected options.
 *
 * @param fromEl - The live select element.
 * @param toEl - The target select element.
 */
function handleSelect(fromEl: HTMLElement, toEl: HTMLElement): void {
    const from = fromEl as HTMLSelectElement;
    const to = toEl as HTMLSelectElement;
    if (from.multiple !== to.multiple) {
        from.multiple = to.multiple;
    }
    if (!from.multiple) {
        const selectedValue = (Array.from(to.options).find(o => o.selected))?.value;
        if (from.value !== selectedValue) {
            from.value = selectedValue ?? '';
        }
    } else {
        const selectedValues = new Set(Array.from(to.options).filter(o => o.selected).map(o => o.value));
        Array.from(from.options).forEach(o => {
            o.selected = selectedValues.has(o.value);
        });
    }
}

/**
 * Synchronises details element open/closed state.
 *
 * @param fromEl - The live details element.
 * @param toEl - The target details element.
 */
function handleDetails(fromEl: HTMLElement, toEl: HTMLElement): void {
    const from = fromEl as HTMLDetailsElement;
    const to = toEl as HTMLDetailsElement;
    if (from.open !== to.open) {
        from.open = to.open;
    }
}

/**
 * Synchronises media element state including src, paused, muted, loop, and volume.
 *
 * @param fromEl - The live audio or video element.
 * @param toEl - The target audio or video element.
 */
function handleMedia(fromEl: HTMLElement, toEl: HTMLElement): void {
    const from = fromEl as HTMLMediaElement;
    const to = toEl as HTMLMediaElement;
    if (from.src !== to.src) {
        from.src = to.src;
    }
    if (to.hasAttribute('paused') && !from.paused) {
        from.pause();
    } else if (!to.hasAttribute('paused') && from.paused) {
        if (to.hasAttribute('autoplay')) {
            from.play().catch((err: unknown) => {
                console.warn('[fragmentMorpher] Media play failed:', err);
            });
        }
    }
    if (from.muted !== to.hasAttribute('muted')) {
        from.muted = to.hasAttribute('muted');
    }
    if (from.loop !== to.hasAttribute('loop')) {
        from.loop = to.hasAttribute('loop');
    }
    const toVolume = to.getAttribute('volume');
    if (toVolume !== null && from.volume !== parseFloat(toVolume)) {
        from.volume = parseFloat(toVolume);
    }
}

/** Lookup table of element-specific state handlers keyed by tag name. */
const nativeSpecialElHandlers: Record<string, ElementHandler> = {
    INPUT: handleInput,
    TEXTAREA: handleTextarea,
    OPTION: handleOption,
    SELECT: handleSelect,
    DETAILS: handleDetails,
    AUDIO: handleMedia,
    VIDEO: handleMedia,
};

/**
 * Constructor shape for form-associated custom elements.
 */
interface FormAssociatedConstructor {
    /** Whether the custom element participates in form submission. */
    formAssociated?: boolean;
}

/**
 * Custom element with form-associated state properties.
 */
interface CustomElementWithState extends HTMLElement {
    /** The element's current value. */
    value?: unknown;
    /** Whether the element is checked. */
    checked?: boolean;
    /** Callback to re-synchronise form state after external mutation. */
    _updateFormState?: () => void;
}

/**
 * Checks whether an element is a form-associated custom element.
 *
 * @param el - The element to check.
 * @returns True if the element's constructor declares formAssociated.
 */
function isFormAssociated(el: HTMLElement): boolean {
    return (el.constructor as FormAssociatedConstructor).formAssociated ?? false;
}

/**
 * Synchronises value and checked state for form-associated custom elements.
 *
 * @param fromEl - The live custom element.
 * @param toEl - The target custom element.
 */
function syncCustomElementState(fromEl: HTMLElement, toEl: HTMLElement) {
    const fromElWithState = fromEl as CustomElementWithState;
    const toElWithState = toEl as CustomElementWithState;

    if ('value' in toElWithState && fromElWithState.value !== toElWithState.value) {
        fromElWithState.value = toElWithState.value;
    }
    if ('checked' in toElWithState && fromElWithState.checked !== toElWithState.checked) {
        fromElWithState.checked = toElWithState.checked;
    }
    if (typeof fromElWithState._updateFormState === 'function') {
        fromElWithState._updateFormState();
    }
}

/**
 * Parses an HTML string into a DOM node or document fragment.
 *
 * @param html - The HTML string to parse.
 * @returns A single child node or a document fragment containing all children.
 */
function toElement(html: string): Node {
    if (!doc) {
        throw new Error("PartialMorpher requires a document environment.");
    }
    const template = doc.createElement('template');
    template.innerHTML = html.trim();

    const firstChild = template.content.firstChild;
    if (template.content.childNodes.length === 1 && firstChild) {
        return firstChild;
    }

    return template.content;
}

/**
 * Checks whether two nodes have the same node name.
 *
 * @param fromEl - The first node.
 * @param toEl - The second node.
 * @returns True if both nodes share the same nodeName.
 */
function compareNodeNames(fromEl: Node, toEl: Node): boolean {
    return fromEl.nodeName === toEl.nodeName;
}

/**
 * Default node key extractor that uses the element's id attribute.
 *
 * @param node - The node to extract a key from.
 * @returns The element id, or null for non-element nodes or elements without an id.
 */
function defaultGetNodeKey(node: Node): NodeKey {
    if (node.nodeType === ELEMENT_NODE) {
        const id = (node as HTMLElement).id;
        return id || null;
    }
    return null;
}

/**
 * Extracts parent scopes from a partial attribute value.
 * The partial attribute contains space-separated scope IDs where
 * the first value is self and the rest are parent scopes.
 *
 * @param partialAttr - The partial attribute value.
 * @returns The parent scope IDs (everything after the first value).
 */
function extractParentScopes(partialAttr: string): string[] {
    return partialAttr.trim().split(/\s+/).slice(1);
}

/**
 * Merges a self scope with parent scopes into a partial attribute value.
 *
 * @param selfScope - The element's own scope ID.
 * @param parentScopes - The parent scope IDs to append.
 * @returns The combined partial attribute value.
 */
function mergePartialScopes(selfScope: string, parentScopes: string[]): string {
    if (parentScopes.length === 0) {
        return selfScope;
    }
    return [selfScope, ...parentScopes].join(' ');
}

/**
 * Applies parent scopes to all elements with partial attributes in a node tree.
 * Ensures new nodes inherit the parent context when added to the DOM.
 *
 * @param node - The root node of the tree to process.
 * @param parentScopes - The parent scope IDs to apply.
 */
function applyParentScopesToTree(node: Node, parentScopes: string[]): void {
    if (parentScopes.length === 0) {
        return;
    }

    if (node.nodeType === ELEMENT_NODE) {
        const el = node as HTMLElement;
        const currentPartial = el.getAttribute('partial');
        if (currentPartial) {
            const selfScope = currentPartial.split(/\s+/)[0];
            el.setAttribute('partial', mergePartialScopes(selfScope, parentScopes));
        }
    }

    for (const child of Array.from(node.childNodes)) {
        applyParentScopesToTree(child, parentScopes);
    }
}

/**
 * Extracts parent scopes from existing children of a container.
 * Finds the first child with a partial attribute and extracts its parent scopes,
 * which represent the full context chain that new children should inherit.
 *
 * Excludes slotted content (elements with [slot] attribute) because they belong
 * to the parent page's scope context and should not influence scope inheritance
 * for dynamically loaded content.
 *
 * @param container - The container element to inspect.
 * @returns The parent scope IDs from the first matching child.
 */
function extractParentScopesFromChildren(container: HTMLElement): string[] {
    const childWithPartial = container.querySelector('[partial]:not([slot])');
    const partialAttr = childWithPartial?.getAttribute('partial');
    if (partialAttr) {
        return extractParentScopes(partialAttr);
    }
    return [];
}

/** Attribute name used to mark attributes that should be preserved during morphing. */
const PRESERVE_ATTR_NAME = 'pk-no-refresh-attrs';

/**
 * Context for attribute morphing operations.
 */
interface MorphAttrsContext {
    /** Attribute names that should not be modified during morphing. */
    preservedAttrs: string[];
    /** Attribute names owned by this morph level (only these are updated). */
    ownedAttrs: string[] | undefined;
    /** Whether owned-attributes mode is active. */
    isOwnedMode: boolean;
    /** Whether partial scope preservation has already been handled. */
    partialScopeHandled: boolean;
}

/**
 * Builds the list of attribute names preserved via pk-no-refresh-attrs.
 *
 * @param fromEl - The element to read the preservation list from.
 * @returns The list of attribute names to preserve.
 */
function buildPreservedAttrsList(fromEl: HTMLElement): string[] {
    const preserved = fromEl.getAttribute(PRESERVE_ATTR_NAME)?.split(',').map(s => s.trim()) ?? [];
    if (fromEl.hasAttribute(PRESERVE_ATTR_NAME)) {
        preserved.push(PRESERVE_ATTR_NAME);
    }
    return preserved;
}

/**
 * Handles partial scope preservation by merging parent scopes from the live element
 * with the self scope from the target element.
 *
 * @param fromEl - The live DOM element.
 * @param toEl - The target element with the new self scope.
 * @param options - Morph options controlling scope preservation.
 * @returns True if partial scope was handled, false otherwise.
 */
function handlePartialScopePreservation(
    fromEl: HTMLElement,
    toEl: HTMLElement,
    options: MorphOptions | undefined
): boolean {
    if (!options?.preservePartialScopes) {
        return false;
    }
    const existingPartial = fromEl.getAttribute('partial');
    const newPartial = toEl.getAttribute('partial');
    if (!existingPartial || !newPartial) {
        return false;
    }
    const parentScopes = extractParentScopes(existingPartial);
    const selfScope = newPartial.split(/\s+/)[0];
    fromEl.setAttribute('partial', mergePartialScopes(selfScope, parentScopes));
    return true;
}

/**
 * Determines whether an attribute update should be skipped.
 *
 * @param attrName - The attribute name to check.
 * @param ctx - The morph attributes context.
 * @returns True if the attribute should not be updated.
 */
function shouldSkipAttrUpdate(attrName: string, ctx: MorphAttrsContext): boolean {
    if (attrName === 'pk-ev-bound' || attrName === 'pk-sync-bound') {
        return true;
    }
    if (ctx.partialScopeHandled && attrName === 'partial') {
        return true;
    }
    if (ctx.preservedAttrs.includes(attrName)) {
        return true;
    }
    if (ctx.isOwnedMode && ctx.ownedAttrs && !ctx.ownedAttrs.includes(attrName)) {
        return true;
    }
    return false;
}

/**
 * Copies attributes from the target element to the live element, skipping preserved ones.
 *
 * @param fromEl - The live DOM element to update.
 * @param toEl - The target element containing desired attributes.
 * @param ctx - The morph attributes context.
 */
function syncToAttrs(fromEl: HTMLElement, toEl: HTMLElement, ctx: MorphAttrsContext): void {
    for (const toAttr of Array.from(toEl.attributes)) {
        if (shouldSkipAttrUpdate(toAttr.name, ctx)) {
            continue;
        }
        if (fromEl.getAttributeNS(toAttr.namespaceURI, toAttr.localName) !== toAttr.value) {
            fromEl.setAttributeNS(toAttr.namespaceURI, toAttr.name, toAttr.value);
        }
    }
}

/**
 * Removes attributes from the live element that are not present on the target element.
 *
 * @param fromEl - The live DOM element.
 * @param toEl - The target element.
 * @param ctx - The morph attributes context.
 */
function removeStaleAttrs(fromEl: HTMLElement, toEl: HTMLElement, ctx: MorphAttrsContext): void {
    for (const fromAttr of Array.from(fromEl.attributes)) {
        if (fromAttr.name === 'pk-ev-bound' || fromAttr.name === 'pk-sync-bound') {
            continue;
        }
        if (ctx.partialScopeHandled && fromAttr.name === 'partial') {
            continue;
        }
        if (ctx.preservedAttrs.includes(fromAttr.name)) {
            continue;
        }
        if (!toEl.hasAttributeNS(fromAttr.namespaceURI, fromAttr.localName)) {
            fromEl.removeAttributeNS(fromAttr.namespaceURI, fromAttr.localName);
        }
    }
}

/**
 * Synchronises all attributes from the target element onto the live element.
 *
 * @param fromEl - The live DOM element.
 * @param toEl - The target element.
 * @param options - Morph options controlling scope preservation and owned attributes.
 */
function morphAttrs(fromEl: HTMLElement, toEl: HTMLElement, options?: MorphOptions): void {
    const ownedAttrs = options?.ownedAttributes;
    const ctx: MorphAttrsContext = {
        preservedAttrs: buildPreservedAttrsList(fromEl),
        ownedAttrs,
        isOwnedMode: Boolean(ownedAttrs && ownedAttrs.length > 0),
        partialScopeHandled: handlePartialScopePreservation(fromEl, toEl, options)
    };

    syncToAttrs(fromEl, toEl, ctx);

    if (!ctx.isOwnedMode) {
        removeStaleAttrs(fromEl, toEl, ctx);
    }
}

/**
 * Determine whether a node is significant for morphing purposes.
 * Whitespace-only text nodes and non-standard node types are insignificant.
 *
 * @param node - The node to check.
 * @returns True if the node should participate in morphing.
 */
function isSignificantNode(node: Node): boolean {
    if (node.nodeType !== ELEMENT_NODE && node.nodeType !== TEXT_NODE && node.nodeType !== COMMENT_NODE) {
        return false;
    }
    if (node.nodeType === TEXT_NODE && !node.nodeValue?.trim()) {
        return false;
    }
    return true;
}

/**
 * Build keyed and unkeyed maps from the existing children of an element,
 * skipping whitespace-only text nodes and non-standard node types.
 *
 * @param fromEl - The parent element whose children are indexed.
 * @param getNodeKey - The key extractor function.
 * @returns An object containing the keyed map and the unkeyed array.
 */
function buildFromNodeMaps(
    fromEl: HTMLElement,
    getNodeKey: (node: Node) => NodeKey
): { fromNodesByKey: Map<NodeKey, Node>; unkeyedFromNodes: Node[] } {
    const fromNodesByKey = new Map<NodeKey, Node>();
    const unkeyedFromNodes: Node[] = [];

    for (const child of Array.from(fromEl.childNodes)) {
        if (!isSignificantNode(child)) {
            continue;
        }
        const key = getNodeKey(child) as NodeKey;
        if (key !== null) {
            fromNodesByKey.set(key, child);
        } else {
            unkeyedFromNodes.push(child);
        }
    }

    return {fromNodesByKey, unkeyedFromNodes};
}

/**
 * Discard from-nodes that were not matched during the child morph pass,
 * invoking lifecycle hooks before and after removal.
 *
 * @param fromNodesByKey - The keyed nodes that remain unmatched.
 * @param unkeyedFromNodes - The unkeyed nodes array.
 * @param unkeyedFromIndex - The current index into the unkeyed array.
 * @param options - The morph configuration options.
 */
function discardUnmatchedNodes(
    fromNodesByKey: Map<NodeKey, Node>,
    unkeyedFromNodes: Node[],
    unkeyedFromIndex: number,
    options: MorphOptions
): void {
    fromNodesByKey.forEach(nodeToDiscard => {
        if (options.onBeforeNodeDiscarded?.(nodeToDiscard) !== false) {
            nodeToDiscard.parentNode?.removeChild(nodeToDiscard);
            options.onNodeDiscarded?.(nodeToDiscard);
        }
    });

    let unkeyedIndex = unkeyedFromIndex;
    while (unkeyedIndex < unkeyedFromNodes.length) {
        const nodeToDiscard = unkeyedFromNodes[unkeyedIndex];
        if (options.onBeforeNodeDiscarded?.(nodeToDiscard) !== false) {
            nodeToDiscard.parentNode?.removeChild(nodeToDiscard);
            options.onNodeDiscarded?.(nodeToDiscard);
        }
        unkeyedIndex++;
    }
}

/**
 * State tracked across the child-matching loop.
 */
interface ChildMatchState {
    /** The keyed from-node map. */
    fromNodesByKey: Map<NodeKey, Node>;
    /** The unkeyed from-nodes array. */
    unkeyedFromNodes: Node[];
    /** Current index into the unkeyed array. */
    unkeyedFromIndex: number;
}

/**
 * Find the best matching from-node for a given to-child, consuming it from
 * the keyed map or the unkeyed array as appropriate.
 *
 * @param toChild - The target child node to match.
 * @param state - The mutable match state.
 * @param getNodeKey - The key extractor function.
 * @returns The matched from-node, or null if no match was found.
 */
function findFromMatch(
    toChild: Node,
    state: ChildMatchState,
    getNodeKey: (node: Node) => NodeKey
): Node | null {
    const toKey = getNodeKey(toChild) as NodeKey;

    if (toKey !== null) {
        const match = state.fromNodesByKey.get(toKey);
        if (match) {
            state.fromNodesByKey.delete(toKey);
            return match;
        }
        return null;
    }

    if (state.unkeyedFromIndex < state.unkeyedFromNodes.length) {
        const potentialMatch = state.unkeyedFromNodes[state.unkeyedFromIndex];
        if (compareNodeNames(potentialMatch, toChild)) {
            state.unkeyedFromIndex++;
            return potentialMatch;
        }
    }

    return null;
}

/**
 * Recursively morphs children of the source element to match the target element.
 *
 * @param fromEl - The live parent element whose children are being morphed.
 * @param toEl - The target parent element with the desired child structure.
 * @param isParentPreserved - Whether the parent element is in a preserved (pk-no-refresh) state.
 * @param options - Morph configuration options.
 */
function morphChildren(fromEl: HTMLElement, toEl: HTMLElement, isParentPreserved: boolean, options: MorphOptions) {
    const getNodeKey = options.getNodeKey ?? defaultGetNodeKey;
    const {fromNodesByKey, unkeyedFromNodes} = buildFromNodeMaps(fromEl, getNodeKey);
    const state: ChildMatchState = {fromNodesByKey, unkeyedFromNodes, unkeyedFromIndex: 0};

    let fromChild = fromEl.firstChild;

    /** Advances the fromChild pointer past insignificant nodes. */
    const advanceFromPointer = () => {
        while (fromChild && !isSignificantNode(fromChild)) {
            fromChild = fromChild.nextSibling;
        }
    };

    for (const toChild of Array.from(toEl.childNodes)) {
        if (!isSignificantNode(toChild)) {
            continue;
        }

        const fromMatch = findFromMatch(toChild, state, getNodeKey);

        if (fromMatch) {
            const morphedNode = morphNode(fromMatch, toChild, isParentPreserved, options);

            advanceFromPointer();
            if (fromChild !== morphedNode) {
                fromEl.insertBefore(morphedNode, fromChild);
            } else {
                fromChild = fromChild.nextSibling;
            }
        } else {
            const newNode = toChild.cloneNode(true);
            if (options._parentScopesToInherit && options._parentScopesToInherit.length > 0) {
                applyParentScopesToTree(newNode, options._parentScopesToInherit);
            }
            if (options.onBeforeNodeAdded?.(newNode) !== false) {
                fromEl.insertBefore(newNode, fromChild);
                options.onNodeAdded?.(newNode);
            }
        }
    }

    discardUnmatchedNodes(fromNodesByKey, unkeyedFromNodes, state.unkeyedFromIndex, options);
}

/**
 * Morph an element node by resolving its preservation state, syncing attributes
 * and special-element state, then recursively morphing children.
 *
 * @param fromEl - The live DOM element.
 * @param toEl - The target element.
 * @param isParentPreserved - Whether the parent is in a preserved state.
 * @param currentOptions - The morph configuration options.
 * @returns The live element after morphing.
 */
function morphElementNode(
    fromEl: HTMLElement,
    toEl: HTMLElement,
    isParentPreserved: boolean,
    currentOptions: MorphOptions
): HTMLElement {
    let preserveEl;
    if (fromEl.hasAttribute('pk-refresh')) {
        preserveEl = false;
    } else if (fromEl.hasAttribute('pk-no-refresh')) {
        preserveEl = true;
    } else {
        preserveEl = isParentPreserved;
    }

    if (!preserveEl) {
        if (currentOptions.onBeforeElUpdated?.(fromEl, toEl) !== false) {
            morphAttrs(fromEl, toEl, currentOptions);

            const nativeHandler = nativeSpecialElHandlers[fromEl.nodeName.toUpperCase()] as ElementHandler | undefined;
            if (nativeHandler) {
                nativeHandler(fromEl, toEl);
            } else if (isFormAssociated(fromEl)) {
                syncCustomElementState(fromEl, toEl);
            }
        }
    }

    if (currentOptions.onBeforeElChildrenUpdated?.(fromEl, toEl) !== false) {
        morphChildren(fromEl, toEl, preserveEl, currentOptions);
    }

    if (!preserveEl && currentOptions.onElUpdated) {
        currentOptions.onElUpdated(fromEl);
    }

    return fromEl;
}

/**
 * Replace a node whose type or name differs from the target, invoking
 * the appropriate lifecycle hooks.
 *
 * @param from - The live DOM node to replace.
 * @param to - The target node providing the replacement content.
 * @param currentOptions - The morph configuration options.
 * @returns The replacement node, or the original if the hook prevented it.
 */
function replaceIncompatibleNode(from: Node, to: Node, currentOptions: MorphOptions): Node {
    const replacement = to.cloneNode(true);
    if (currentOptions.onBeforeNodeAdded?.(replacement) === false) {
        return from;
    }
    from.parentNode?.replaceChild(replacement, from);
    currentOptions.onNodeDiscarded?.(from);
    currentOptions.onNodeAdded?.(replacement);
    return replacement;
}

/**
 * Morphs a single node to match a target node, handling type changes, text updates,
 * attribute syncs, and recursive child morphing.
 *
 * @param from - The live DOM node.
 * @param to - The target node.
 * @param isParentPreserved - Whether the parent is in a preserved state.
 * @param currentOptions - Morph configuration options.
 * @returns The resulting node (may be replaced if types differ).
 */
function morphNode(from: Node, to: Node, isParentPreserved: boolean, currentOptions: MorphOptions): Node {
    if (currentOptions.onBeforeNodeDiscarded?.(from) === false) {
        return from;
    }

    if (from.nodeType === ELEMENT_NODE && to.nodeType === DOCUMENT_FRAGMENT_NODE) {
        morphChildren(from as HTMLElement, to as HTMLElement, isParentPreserved, currentOptions);
        return from;
    }

    if (from.nodeType !== to.nodeType || !compareNodeNames(from, to)) {
        return replaceIncompatibleNode(from, to, currentOptions);
    }

    if (from.nodeType === TEXT_NODE || from.nodeType === COMMENT_NODE) {
        if (!isParentPreserved && from.nodeValue !== to.nodeValue) {
            from.nodeValue = to.nodeValue;
        }
        return from;
    }

    return morphElementNode(from as HTMLElement, to as HTMLElement, isParentPreserved, currentOptions);
}

/**
 * Captures the currently focused element and its key for later restoration.
 *
 * @param fromNode - The container to check for focused descendants.
 * @param getNodeKey - Key extractor function.
 * @returns The active element and its key, or null key if not focused within the container.
 */
function captureActiveElementKey(
    fromNode: HTMLElement,
    getNodeKey: (node: Node) => NodeKey
): { activeEl: HTMLElement | null; key: NodeKey } {
    const activeEl = doc?.activeElement as HTMLElement | null;
    const key = (activeEl && fromNode.contains(activeEl)) ? getNodeKey(activeEl) : null;
    return {activeEl, key};
}

/**
 * Builds effective morph options by computing parent scopes from existing children.
 *
 * @param fromNode - The container element.
 * @param options - The original morph options.
 * @returns The options with _parentScopesToInherit populated if applicable.
 */
function buildEffectiveOptions(fromNode: HTMLElement, options: MorphOptions): MorphOptions {
    if (!options.preservePartialScopes || options._parentScopesToInherit) {
        return options;
    }
    const parentScopes = extractParentScopesFromChildren(fromNode);
    if (parentScopes.length === 0) {
        return options;
    }
    return {...options, _parentScopesToInherit: parentScopes};
}

/**
 * Performs the morph operation, dispatching to childrenOnly or full node morphing.
 *
 * @param fromNode - The live DOM element to morph.
 * @param toNode - The target node structure.
 * @param options - Morph configuration options.
 */
function performMorph(
    fromNode: HTMLElement,
    toNode: Node,
    options: MorphOptions
): void {
    const initialPreserve = options.initialState === 'pk-no-refresh';

    if (options.childrenOnly) {
        const containerPreserved = fromNode.hasAttribute('pk-no-refresh') ||
            (initialPreserve && !fromNode.hasAttribute('pk-refresh'));
        morphChildren(fromNode, toNode as HTMLElement, containerPreserved, options);
    } else {
        morphNode(fromNode, toNode, initialPreserve, options);
    }
}

/**
 * Morphs a DOM element to match a target structure with optional preservation.
 *
 * Performs efficient DOM diffing and patching, preserving elements marked with
 * pk-no-refresh and updating elements marked with pk-refresh.
 *
 * @param fromNode - The existing DOM element to morph.
 * @param toNodeOrHTML - The target structure as a Node or HTML string.
 * @param options - Configuration options for the morph operation.
 */
export default function fragmentMorpher(
    fromNode: HTMLElement | null,
    toNodeOrHTML: Node | string | null,
    options: MorphOptions = {}
): void {
    if (!fromNode || !toNodeOrHTML) {
        return;
    }

    const toNode = typeof toNodeOrHTML === 'string' ? toElement(toNodeOrHTML) : toNodeOrHTML;
    const getNodeKey = options.getNodeKey ?? defaultGetNodeKey;
    const {activeEl, key: activeElKey} = captureActiveElementKey(fromNode, getNodeKey);
    const effectiveOptions = buildEffectiveOptions(fromNode, options);

    performMorph(fromNode, toNode, effectiveOptions);

    if (activeElKey !== null && doc?.activeElement !== activeEl) {
        findAndFocusNodeByKey(fromNode, activeElKey, getNodeKey);
    }
}

/**
 * Walks a container tree to find and focus the element matching the given key.
 *
 * @param container - The container to search within.
 * @param key - The node key to match.
 * @param getNodeKey - Key extractor function.
 */
function findAndFocusNodeByKey(container: HTMLElement, key: NodeKey, getNodeKey: (node: Node) => NodeKey) {
    if (getNodeKey(container) === key) {
        container.focus();
        return;
    }
    const walker = doc?.createTreeWalker(container, Node.ELEMENT_NODE, {acceptNode: () => NodeFilter.FILTER_ACCEPT});
    if (!walker) {
        return;
    }
    while (walker.nextNode()) {
        if (getNodeKey(walker.currentNode) === key) {
            (walker.currentNode as HTMLElement).focus();
            return;
        }
    }
}
