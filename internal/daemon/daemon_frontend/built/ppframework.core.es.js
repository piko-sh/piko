const ELEMENT_NODE = 1;
const TEXT_NODE = 3;
const COMMENT_NODE = 8;
const DOCUMENT_FRAGMENT_NODE = 11;
const doc = typeof document === "undefined" ? void 0 : document;
function syncBooleanAttrProp(fromEl, toEl, name) {
  const fromValue = fromEl[name];
  const toValue = toEl[name];
  if (fromValue !== toValue) {
    fromEl[name] = toValue;
    if (toValue) {
      fromEl.setAttribute(name, "");
    } else {
      fromEl.removeAttribute(name);
    }
  }
}
function handleInput(fromEl, toEl) {
  const from = fromEl;
  const to = toEl;
  syncBooleanAttrProp(from, to, "checked");
  syncBooleanAttrProp(from, to, "disabled");
  if (from !== doc?.activeElement && from.value !== to.value) {
    from.value = to.value;
  }
  if (from.type !== "file" && !to.hasAttribute("value")) {
    from.removeAttribute("value");
  }
}
function handleTextarea(fromEl, toEl) {
  const from = fromEl;
  const to = toEl;
  if (from.value !== to.value) {
    from.value = to.value;
  }
}
function handleOption(fromEl, toEl) {
  syncBooleanAttrProp(fromEl, toEl, "selected");
}
function handleSelect(fromEl, toEl) {
  const from = fromEl;
  const to = toEl;
  if (from.multiple !== to.multiple) {
    from.multiple = to.multiple;
  }
  if (!from.multiple) {
    const selectedValue = Array.from(to.options).find((o) => o.selected)?.value;
    if (from.value !== selectedValue) {
      from.value = selectedValue ?? "";
    }
  } else {
    const selectedValues = new Set(Array.from(to.options).filter((o) => o.selected).map((o) => o.value));
    Array.from(from.options).forEach((o) => {
      o.selected = selectedValues.has(o.value);
    });
  }
}
function handleDetails(fromEl, toEl) {
  const from = fromEl;
  const to = toEl;
  if (from.open !== to.open) {
    from.open = to.open;
  }
}
function handleMedia(fromEl, toEl) {
  const from = fromEl;
  const to = toEl;
  if (from.src !== to.src) {
    from.src = to.src;
  }
  if (to.hasAttribute("paused") && !from.paused) {
    from.pause();
  } else if (!to.hasAttribute("paused") && from.paused) {
    if (to.hasAttribute("autoplay")) {
      from.play().catch((err) => {
        console.warn("[fragmentMorpher] Media play failed:", err);
      });
    }
  }
  if (from.muted !== to.hasAttribute("muted")) {
    from.muted = to.hasAttribute("muted");
  }
  if (from.loop !== to.hasAttribute("loop")) {
    from.loop = to.hasAttribute("loop");
  }
  const toVolume = to.getAttribute("volume");
  if (toVolume !== null && from.volume !== parseFloat(toVolume)) {
    from.volume = parseFloat(toVolume);
  }
}
const nativeSpecialElHandlers = {
  INPUT: handleInput,
  TEXTAREA: handleTextarea,
  OPTION: handleOption,
  SELECT: handleSelect,
  DETAILS: handleDetails,
  AUDIO: handleMedia,
  VIDEO: handleMedia
};
function isFormAssociated(el) {
  return el.constructor.formAssociated ?? false;
}
function syncCustomElementState(fromEl, toEl) {
  const fromElWithState = fromEl;
  const toElWithState = toEl;
  if ("value" in toElWithState && fromElWithState.value !== toElWithState.value) {
    fromElWithState.value = toElWithState.value;
  }
  if ("checked" in toElWithState && fromElWithState.checked !== toElWithState.checked) {
    fromElWithState.checked = toElWithState.checked;
  }
  if (typeof fromElWithState._updateFormState === "function") {
    fromElWithState._updateFormState();
  }
}
function toElement(html) {
  if (!doc) {
    throw new Error("PartialMorpher requires a document environment.");
  }
  const template = doc.createElement("template");
  template.innerHTML = html.trim();
  const firstChild = template.content.firstChild;
  if (template.content.childNodes.length === 1 && firstChild) {
    return firstChild;
  }
  return template.content;
}
function compareNodeNames(fromEl, toEl) {
  return fromEl.nodeName === toEl.nodeName;
}
function defaultGetNodeKey(node) {
  if (node.nodeType === ELEMENT_NODE) {
    const id = node.id;
    return id || null;
  }
  return null;
}
function extractParentScopes(partialAttr) {
  return partialAttr.trim().split(/\s+/).slice(1);
}
function mergePartialScopes(selfScope, parentScopes) {
  if (parentScopes.length === 0) {
    return selfScope;
  }
  return [selfScope, ...parentScopes].join(" ");
}
function applyParentScopesToTree(node, parentScopes) {
  if (parentScopes.length === 0) {
    return;
  }
  if (node.nodeType === ELEMENT_NODE) {
    const el = node;
    const currentPartial = el.getAttribute("partial");
    if (currentPartial) {
      const selfScope = currentPartial.split(/\s+/)[0];
      el.setAttribute("partial", mergePartialScopes(selfScope, parentScopes));
    }
  }
  for (const child of Array.from(node.childNodes)) {
    applyParentScopesToTree(child, parentScopes);
  }
}
function extractParentScopesFromChildren(container) {
  const childWithPartial = container.querySelector("[partial]:not([slot])");
  const partialAttr = childWithPartial?.getAttribute("partial");
  if (partialAttr) {
    return extractParentScopes(partialAttr);
  }
  return [];
}
const PRESERVE_ATTR_NAME = "pk-no-refresh-attrs";
function buildPreservedAttrsList(fromEl) {
  const preserved = fromEl.getAttribute(PRESERVE_ATTR_NAME)?.split(",").map((s) => s.trim()) ?? [];
  if (fromEl.hasAttribute(PRESERVE_ATTR_NAME)) {
    preserved.push(PRESERVE_ATTR_NAME);
  }
  return preserved;
}
function handlePartialScopePreservation(fromEl, toEl, options) {
  if (!options?.preservePartialScopes) {
    return false;
  }
  const existingPartial = fromEl.getAttribute("partial");
  const newPartial = toEl.getAttribute("partial");
  if (!existingPartial || !newPartial) {
    return false;
  }
  const parentScopes = extractParentScopes(existingPartial);
  const selfScope = newPartial.split(/\s+/)[0];
  fromEl.setAttribute("partial", mergePartialScopes(selfScope, parentScopes));
  return true;
}
function shouldSkipAttrUpdate(attrName, ctx) {
  if (attrName === "pk-ev-bound" || attrName === "pk-sync-bound") {
    return true;
  }
  if (ctx.partialScopeHandled && attrName === "partial") {
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
function syncToAttrs(fromEl, toEl, ctx) {
  for (const toAttr of Array.from(toEl.attributes)) {
    if (shouldSkipAttrUpdate(toAttr.name, ctx)) {
      continue;
    }
    if (fromEl.getAttributeNS(toAttr.namespaceURI, toAttr.localName) !== toAttr.value) {
      fromEl.setAttributeNS(toAttr.namespaceURI, toAttr.name, toAttr.value);
    }
  }
}
function removeStaleAttrs(fromEl, toEl, ctx) {
  for (const fromAttr of Array.from(fromEl.attributes)) {
    if (fromAttr.name === "pk-ev-bound" || fromAttr.name === "pk-sync-bound") {
      continue;
    }
    if (ctx.partialScopeHandled && fromAttr.name === "partial") {
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
function morphAttrs(fromEl, toEl, options) {
  const ownedAttrs = options?.ownedAttributes;
  const ctx = {
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
function isSignificantNode(node) {
  if (node.nodeType !== ELEMENT_NODE && node.nodeType !== TEXT_NODE && node.nodeType !== COMMENT_NODE) {
    return false;
  }
  if (node.nodeType === TEXT_NODE && !node.nodeValue?.trim()) {
    return false;
  }
  return true;
}
function buildFromNodeMaps(fromEl, getNodeKey2) {
  const fromNodesByKey = /* @__PURE__ */ new Map();
  const unkeyedFromNodes = [];
  for (const child of Array.from(fromEl.childNodes)) {
    if (!isSignificantNode(child)) {
      continue;
    }
    const key = getNodeKey2(child);
    if (key !== null) {
      fromNodesByKey.set(key, child);
    } else {
      unkeyedFromNodes.push(child);
    }
  }
  return { fromNodesByKey, unkeyedFromNodes };
}
function discardUnmatchedNodes(fromNodesByKey, unkeyedFromNodes, unkeyedFromIndex, options) {
  fromNodesByKey.forEach((nodeToDiscard) => {
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
function findFromMatch(toChild, state, getNodeKey2) {
  const toKey = getNodeKey2(toChild);
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
function morphChildren(fromEl, toEl, isParentPreserved, options) {
  const getNodeKey2 = options.getNodeKey ?? defaultGetNodeKey;
  const { fromNodesByKey, unkeyedFromNodes } = buildFromNodeMaps(fromEl, getNodeKey2);
  const state = { fromNodesByKey, unkeyedFromNodes, unkeyedFromIndex: 0 };
  let fromChild = fromEl.firstChild;
  const advanceFromPointer = () => {
    while (fromChild && !isSignificantNode(fromChild)) {
      fromChild = fromChild.nextSibling;
    }
  };
  for (const toChild of Array.from(toEl.childNodes)) {
    if (!isSignificantNode(toChild)) {
      continue;
    }
    const fromMatch = findFromMatch(toChild, state, getNodeKey2);
    if (fromMatch) {
      const cursorWasMatch = fromChild === fromMatch;
      const morphedNode = morphNode(fromMatch, toChild, isParentPreserved, options);
      if (cursorWasMatch) {
        fromChild = morphedNode.nextSibling;
        advanceFromPointer();
        continue;
      }
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
function morphElementNode(fromEl, toEl, isParentPreserved, currentOptions) {
  let preserveEl;
  if (fromEl.hasAttribute("pk-refresh")) {
    preserveEl = false;
  } else if (fromEl.hasAttribute("pk-no-refresh")) {
    preserveEl = true;
  } else {
    preserveEl = isParentPreserved;
  }
  if (!preserveEl) {
    if (currentOptions.onBeforeElUpdated?.(fromEl, toEl) !== false) {
      morphAttrs(fromEl, toEl, currentOptions);
      const nativeHandler = nativeSpecialElHandlers[fromEl.nodeName.toUpperCase()];
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
function replaceIncompatibleNode(from, to, currentOptions) {
  const replacement = to.cloneNode(true);
  if (currentOptions.onBeforeNodeAdded?.(replacement) === false) {
    return from;
  }
  from.parentNode?.replaceChild(replacement, from);
  currentOptions.onNodeDiscarded?.(from);
  currentOptions.onNodeAdded?.(replacement);
  return replacement;
}
function morphNode(from, to, isParentPreserved, currentOptions) {
  if (currentOptions.onBeforeNodeDiscarded?.(from) === false) {
    return from;
  }
  if (from.nodeType === ELEMENT_NODE && to.nodeType === DOCUMENT_FRAGMENT_NODE) {
    morphChildren(from, to, isParentPreserved, currentOptions);
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
  return morphElementNode(from, to, isParentPreserved, currentOptions);
}
function captureActiveElementKey(fromNode, getNodeKey2) {
  const activeEl = doc?.activeElement;
  const key = activeEl && fromNode.contains(activeEl) ? getNodeKey2(activeEl) : null;
  return { activeEl, key };
}
function buildEffectiveOptions(fromNode, options) {
  if (!options.preservePartialScopes || options._parentScopesToInherit) {
    return options;
  }
  const parentScopes = extractParentScopesFromChildren(fromNode);
  if (parentScopes.length === 0) {
    return options;
  }
  return { ...options, _parentScopesToInherit: parentScopes };
}
function performMorph(fromNode, toNode, options) {
  const initialPreserve = options.initialState === "pk-no-refresh";
  if (options.childrenOnly) {
    const containerPreserved = fromNode.hasAttribute("pk-no-refresh") || initialPreserve && !fromNode.hasAttribute("pk-refresh");
    morphChildren(fromNode, toNode, containerPreserved, options);
  } else {
    morphNode(fromNode, toNode, initialPreserve, options);
  }
}
function fragmentMorpher(fromNode, toNodeOrHTML, options = {}) {
  if (!fromNode || !toNodeOrHTML) {
    return;
  }
  const toNode = typeof toNodeOrHTML === "string" ? toElement(toNodeOrHTML) : toNodeOrHTML;
  const getNodeKey2 = options.getNodeKey ?? defaultGetNodeKey;
  const { activeEl, key: activeElKey } = captureActiveElementKey(fromNode, getNodeKey2);
  const effectiveOptions = buildEffectiveOptions(fromNode, options);
  performMorph(fromNode, toNode, effectiveOptions);
  if (activeElKey !== null && doc?.activeElement !== activeEl) {
    findAndFocusNodeByKey(fromNode, activeElKey, getNodeKey2);
  }
}
function findAndFocusNodeByKey(container, key, getNodeKey2) {
  if (getNodeKey2(container) === key) {
    container.focus();
    return;
  }
  const walker = doc?.createTreeWalker(container, Node.ELEMENT_NODE, { acceptNode: () => NodeFilter.FILTER_ACCEPT });
  if (!walker) {
    return;
  }
  while (walker.nextNode()) {
    if (getNodeKey2(walker.currentNode) === key) {
      walker.currentNode.focus();
      return;
    }
  }
}
let _onDOMUpdated = null;
function registerDOMUpdater(callback) {
  _onDOMUpdated = callback;
}
function notifyDOMUpdated(root) {
  if (_onDOMUpdated) {
    _onDOMUpdated(root);
  }
}
const LOADING_CLASS = "pk-loading";
const ARIA_BUSY_ATTR = "aria-busy";
function applyLoadingIndicator(el) {
  el.classList.add(LOADING_CLASS);
  el.setAttribute(ARIA_BUSY_ATTR, "true");
}
function removeLoadingIndicator(el) {
  el.classList.remove(LOADING_CLASS);
  el.removeAttribute(ARIA_BUSY_ATTR);
}
const REFRESH_LEVEL_NO_REFRESH_ATTRS$1 = 3;
const REFRESH_LEVEL_OWN_ATTRS = 2;
function detectRefreshLevel$1(el) {
  if (el.hasAttribute("pk-no-refresh-attrs")) {
    return REFRESH_LEVEL_NO_REFRESH_ATTRS$1;
  }
  if (el.hasAttribute("pk-own-attrs")) {
    return REFRESH_LEVEL_OWN_ATTRS;
  }
  if (el.hasAttribute("pk-refresh-root")) {
    return 1;
  }
  return 0;
}
function getOwnedAttributes(el) {
  const attr = el.getAttribute("pk-own-attrs");
  if (!attr) {
    return void 0;
  }
  return attr.split(",").map((s) => s.trim()).filter((s) => s.length > 0);
}
function parseHTML(html) {
  const parser = new DOMParser();
  const doc2 = parser.parseFromString(html, "text/html");
  return doc2.body.firstElementChild;
}
function applyRefresh(el, newContent, level, ownedAttrs) {
  switch (level) {
    case 0:
      fragmentMorpher(el, newContent, { childrenOnly: true });
      break;
    case 1:
      fragmentMorpher(el, newContent, {
        childrenOnly: false,
        preservePartialScopes: true
      });
      break;
    case REFRESH_LEVEL_OWN_ATTRS:
      fragmentMorpher(el, newContent, {
        childrenOnly: false,
        preservePartialScopes: true,
        ownedAttributes: ownedAttrs
      });
      break;
    case REFRESH_LEVEL_NO_REFRESH_ATTRS$1:
      fragmentMorpher(el, newContent, {
        childrenOnly: false,
        preservePartialScopes: true
      });
      break;
  }
}
async function performReload(el, name, options) {
  const baseSrc = el.getAttribute("partial_src");
  if (!baseSrc) {
    throw new Error(`Partial "${name}" has no partial_src attribute. Is the partial's template marked as public?`);
  }
  let effectiveData = options.data;
  if (!effectiveData) {
    const partialProps = el.getAttribute("partial_props");
    if (partialProps) {
      effectiveData = Object.fromEntries(new URLSearchParams(partialProps));
    }
  }
  const params = new URLSearchParams(effectiveData);
  params.set("_f", "true");
  const url = `${baseSrc}?${params.toString()}`;
  const level = options.level ?? detectRefreshLevel$1(el);
  applyLoadingIndicator(el);
  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Failed to reload partial: ${response.status}`);
    }
    const html = await response.text();
    const newContent = parseHTML(html);
    if (!newContent) {
      console.warn(`[pk] partial "${name}" received empty or invalid response`);
      return;
    }
    const ownedAttrs = options.ownedAttrs ?? getOwnedAttributes(el);
    applyRefresh(el, newContent, level, ownedAttrs);
    if (effectiveData) {
      el.setAttribute(
        "partial_props",
        new URLSearchParams(effectiveData).toString()
      );
    }
    notifyDOMUpdated(el);
  } catch (error) {
    console.error(`[pk] Failed to reload partial "${name}":`, {
      url,
      args: options.data,
      level,
      error
    });
    throw error;
  } finally {
    removeLoadingIndicator(el);
  }
}
function partial(nameOrElement) {
  let el;
  let name;
  if (typeof nameOrElement === "string") {
    name = nameOrElement;
    el = document.querySelector(`[partial_name="${name}"]`);
  } else {
    el = nameOrElement;
    name = el.getAttribute("partial_name") ?? el.getAttribute("partial") ?? "unknown";
  }
  return {
    element: el,
    async reload(data) {
      if (!el) {
        console.warn(`[pk] partial "${name}" not found`);
        return;
      }
      return performReload(el, name, { data });
    },
    async reloadWithOptions(options) {
      if (!el) {
        console.warn(`[pk] partial "${name}" not found`);
        return;
      }
      return performReload(el, name, options);
    }
  };
}
const listeners = /* @__PURE__ */ new Map();
const bus = {
  /**
   * Emits an event to all listeners.
   *
   * @param event - Event name.
   * @param data - Optional data to pass to listeners.
   */
  emit(event, data) {
    const handlers = listeners.get(event);
    if (handlers) {
      handlers.forEach((fn) => {
        try {
          fn(data);
        } catch (error) {
          console.error(`[pk] Error in bus handler for "${event}":`, error);
        }
      });
    }
  },
  /**
   * Subscribes to an event.
   *
   * @param event - Event name.
   * @param handler - Handler function.
   * @returns Unsubscribe function.
   */
  on(event, handler) {
    let eventListeners = listeners.get(event);
    if (!eventListeners) {
      eventListeners = /* @__PURE__ */ new Set();
      listeners.set(event, eventListeners);
    }
    eventListeners.add(handler);
    return () => {
      listeners.get(event)?.delete(handler);
    };
  },
  /**
   * Subscribes to an event once (auto-unsubscribes after first call).
   *
   * @param event - Event name.
   * @param handler - Handler function.
   * @returns Unsubscribe function (in case you want to cancel before it fires).
   */
  once(event, handler) {
    const wrappedHandler = (data) => {
      listeners.get(event)?.delete(wrappedHandler);
      handler(data);
    };
    return this.on(event, wrappedHandler);
  },
  /**
   * Removes all listeners for an event, or all listeners if no event specified.
   *
   * @param event - Optional event name.
   */
  off(event) {
    if (event) {
      listeners.delete(event);
    } else {
      listeners.clear();
    }
  }
};
const navigationGuards = [];
function parseQuery(search) {
  const params = new URLSearchParams(search);
  const result = {};
  params.forEach((value, key) => {
    if (!(key in result)) {
      result[key] = value;
    }
  });
  return result;
}
function getFramework() {
  if (typeof PPFramework.navigateTo === "function") {
    return {
      navigate: (url, options) => {
        const routerOptions = {
          replaceHistory: options?.replace
        };
        return PPFramework.navigateTo(url, void 0, routerOptions);
      }
    };
  }
  return null;
}
async function runBeforeNavigateGuards(url, currentUrl) {
  for (const guard of navigationGuards) {
    if (!guard.beforeNavigate) {
      continue;
    }
    const allowed = await guard.beforeNavigate(url, currentUrl);
    if (!allowed) {
      return false;
    }
  }
  return true;
}
function performFullReload(url, replace) {
  if (replace) {
    window.location.replace(url);
  } else {
    window.location.href = url;
  }
}
function performHistoryNavigation(url, replace, scroll, state) {
  const method = replace ? "replaceState" : "pushState";
  window.history[method](state, "", url);
  if (scroll) {
    window.scrollTo(0, 0);
  }
  window.dispatchEvent(new PopStateEvent("popstate", { state }));
}
function runAfterNavigateGuards(url, currentUrl) {
  for (const guard of navigationGuards) {
    guard.afterNavigate?.(url, currentUrl);
  }
}
async function navigate(url, options = {}) {
  const { replace = false, scroll = true, state = null, fullReload = false } = options;
  const currentUrl = window.location.href;
  const allowed = await runBeforeNavigateGuards(url, currentUrl);
  if (!allowed) {
    return;
  }
  if (fullReload) {
    performFullReload(url, replace);
    return;
  }
  const framework = getFramework();
  if (framework) {
    await framework.navigate(url, { replace, scroll });
  } else {
    performHistoryNavigation(url, replace, scroll, state);
  }
  runAfterNavigateGuards(url, currentUrl);
}
function goBack() {
  window.history.back();
}
function goForward() {
  window.history.forward();
}
function go(delta) {
  window.history.go(delta);
}
function currentRoute() {
  const location = window.location;
  const searchParams = new URLSearchParams(location.search);
  return {
    path: location.pathname,
    query: parseQuery(location.search),
    hash: location.hash.replace(/^#/, ""),
    href: location.href,
    origin: location.origin,
    getParam(name) {
      return searchParams.get(name);
    },
    hasParam(name) {
      return searchParams.has(name);
    },
    getParams(name) {
      return searchParams.getAll(name);
    }
  };
}
function buildUrl(path, params, hash) {
  const url = new URL(path, window.location.origin);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== null && value !== void 0) {
        url.searchParams.set(key, String(value));
      }
    }
  }
  if (hash) {
    url.hash = hash;
  }
  return url.pathname + url.search + url.hash;
}
async function updateQuery(params, options = {}) {
  const url = new URL(window.location.href);
  for (const [key, value] of Object.entries(params)) {
    if (value === null || value === void 0) {
      url.searchParams.delete(key);
    } else {
      url.searchParams.set(key, value);
    }
  }
  await navigate(url.pathname + url.search + url.hash, {
    ...options,
    scroll: options.scroll ?? false
  });
}
function registerNavigationGuard(guard) {
  navigationGuards.push(guard);
  return () => {
    const index = navigationGuards.indexOf(guard);
    if (index > -1) {
      navigationGuards.splice(index, 1);
    }
  };
}
function matchPath(pattern) {
  const currentPath = window.location.pathname;
  const regexPattern = pattern.replace(/[.+?^${}()|[\]\\]/g, "\\$&").replace(/:([^/]+)/g, "([^/]+)").replace(/\*/g, ".*");
  const regex = new RegExp(`^${regexPattern}$`);
  return regex.test(currentPath);
}
function extractParams(pattern) {
  const currentPath = window.location.pathname;
  const paramNames = [];
  const regexPattern = pattern.replace(/[.+?^${}()|[\]\\]/g, "\\$&").replace(/:([^/]+)/g, (_, name) => {
    paramNames.push(name);
    return "([^/]+)";
  }).replace(/\*/g, ".*");
  const regex = new RegExp(`^${regexPattern}$`);
  const match = currentPath.match(regex);
  if (!match) {
    return null;
  }
  const params = {};
  paramNames.forEach((name, index) => {
    params[name] = match[index + 1];
  });
  return params;
}
const PATTERNS = {
  email: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
  url: /^https?:\/\/[^\s/$.?#].[^\s]*$/i,
  phone: /^[+]?[(]?[0-9]{1,4}[)]?[-\s./0-9]*$/,
  date: /^\d{4}-\d{2}-\d{2}$/
};
const DEFAULT_MESSAGES = {
  required: "This field is required",
  minLength: "Value is too short",
  maxLength: "Value is too long",
  min: "Value is too small",
  max: "Value is too large",
  pattern: "Invalid format",
  email: "Invalid email address",
  url: "Invalid URL",
  phone: "Invalid phone number",
  date: "Invalid date format (YYYY-MM-DD)"
};
function resolveForm(selector) {
  if (selector instanceof HTMLFormElement) {
    return selector;
  }
  const element = document.querySelector(selector);
  if (element instanceof HTMLFormElement) {
    return element;
  }
  console.warn(`[pk] formData: "${selector}" is not a form element`);
  return null;
}
function formDataToObject(fd) {
  const result = {};
  fd.forEach((value, key) => {
    const arrayMatch = key.match(/^(.+)\[\]$/);
    const actualKey = arrayMatch ? arrayMatch[1] : key;
    if (arrayMatch || result[actualKey] !== void 0) {
      const existing = result[actualKey];
      if (Array.isArray(existing)) {
        existing.push(value);
      } else if (existing !== void 0) {
        result[actualKey] = [existing, value];
      } else {
        result[actualKey] = [value];
      }
    } else {
      result[actualKey] = value;
    }
  });
  return result;
}
function getStringValue(value) {
  if (value === null || value === void 0) {
    return "";
  }
  return String(value);
}
function validateLength(strValue, rule) {
  if (rule.minLength !== void 0 && strValue.length < rule.minLength) {
    return rule.message ?? `${DEFAULT_MESSAGES.minLength} (minimum ${rule.minLength} characters)`;
  }
  if (rule.maxLength !== void 0 && strValue.length > rule.maxLength) {
    return rule.message ?? `${DEFAULT_MESSAGES.maxLength} (maximum ${rule.maxLength} characters)`;
  }
  return null;
}
function validateNumericRange(value, rule) {
  const numValue = Number(value);
  if (isNaN(numValue)) {
    return null;
  }
  if (rule.min !== void 0 && numValue < rule.min) {
    return rule.message ?? `${DEFAULT_MESSAGES.min} (minimum ${rule.min})`;
  }
  if (rule.max !== void 0 && numValue > rule.max) {
    return rule.message ?? `${DEFAULT_MESSAGES.max} (maximum ${rule.max})`;
  }
  return null;
}
function validatePattern(strValue, rule) {
  if (rule.pattern === void 0) {
    return null;
  }
  const pattern = typeof rule.pattern === "string" ? new RegExp(rule.pattern) : rule.pattern;
  if (!pattern.test(strValue)) {
    return rule.message ?? DEFAULT_MESSAGES.pattern;
  }
  return null;
}
function validateFormat(strValue, rule) {
  if (rule.format === void 0) {
    return null;
  }
  const formatPattern = PATTERNS[rule.format];
  if (!formatPattern.test(strValue)) {
    return rule.message ?? DEFAULT_MESSAGES[rule.format];
  }
  return null;
}
function validateCustom(value, rule) {
  if (rule.custom === void 0) {
    return null;
  }
  const customResult = rule.custom(value);
  if (customResult === false) {
    return rule.message ?? "Validation failed";
  }
  if (typeof customResult === "string") {
    return customResult;
  }
  return null;
}
function validateField(value, rule) {
  const errors = [];
  const strValue = getStringValue(value);
  const isEmpty = strValue.trim() === "";
  if (rule.required && isEmpty) {
    errors.push(rule.message ?? DEFAULT_MESSAGES.required);
    return errors;
  }
  if (isEmpty) {
    return errors;
  }
  const validationResults = [
    validateLength(strValue, rule),
    validateNumericRange(value, rule),
    validatePattern(strValue, rule),
    validateFormat(strValue, rule),
    validateCustom(value, rule)
  ];
  for (const error of validationResults) {
    if (error !== null) {
      errors.push(error);
    }
  }
  return errors;
}
function formData(selector) {
  const form = resolveForm(selector);
  const fd = form ? new FormData(form) : new FormData();
  const formObject = formDataToObject(fd);
  return {
    toObject() {
      return { ...formObject };
    },
    toFormData() {
      return form ? new FormData(form) : new FormData();
    },
    toJSON() {
      return JSON.stringify(formObject);
    },
    get(key) {
      return formObject[key];
    },
    has(key) {
      return key in formObject;
    },
    getAll(key) {
      const value = formObject[key];
      if (Array.isArray(value)) {
        return value;
      }
      if (value !== void 0) {
        return [value];
      }
      return [];
    }
  };
}
function validate(selector, rules = {}) {
  const form = resolveForm(selector);
  const data = formData(selector);
  const formObject = data.toObject();
  const errors = {};
  let firstInvalidField = null;
  for (const [field, rule] of Object.entries(rules)) {
    const fieldErrors = validateField(formObject[field], rule);
    if (fieldErrors.length > 0) {
      errors[field] = fieldErrors;
      firstInvalidField ??= field;
    }
  }
  const isValid = Object.keys(errors).length === 0;
  return {
    isValid,
    errors,
    focus() {
      if (!form || !firstInvalidField) {
        return;
      }
      const field = form.elements.namedItem(firstInvalidField);
      if (field instanceof HTMLElement && "focus" in field) {
        field.focus();
      }
    },
    getErrors(field) {
      return errors[field] ?? [];
    },
    hasError(field) {
      return field in errors && errors[field].length > 0;
    }
  };
}
function resetForm(selector) {
  const form = resolveForm(selector);
  if (form) {
    form.reset();
  }
}
function setInputValue(input, value) {
  if (input.type === "checkbox") {
    input.checked = Boolean(value);
  } else if (input.type !== "file") {
    input.value = String(value ?? "");
  }
}
function setRadioNodeListValue(nodeList, value) {
  for (const element of Array.from(nodeList)) {
    if (!(element instanceof HTMLInputElement)) {
      continue;
    }
    if (element.type === "checkbox") {
      element.checked = Array.isArray(value) ? value.includes(element.value) : element.value === String(value);
    } else if (element.type === "radio") {
      element.checked = element.value === String(value);
    }
  }
}
function setSelectValue(select, value) {
  if (select.multiple && Array.isArray(value)) {
    for (const option of Array.from(select.options)) {
      option.selected = value.includes(option.value);
    }
  } else {
    select.value = String(value ?? "");
  }
}
function setFormValues(selector, values) {
  const form = resolveForm(selector);
  if (!form) {
    return;
  }
  for (const [name, value] of Object.entries(values)) {
    const elements = form.elements.namedItem(name);
    if (!elements) {
      continue;
    }
    if (elements instanceof RadioNodeList) {
      setRadioNodeListValue(elements, value);
      continue;
    }
    if (elements instanceof HTMLInputElement) {
      setInputValue(elements, value);
    } else if (elements instanceof HTMLSelectElement) {
      setSelectValue(elements, value);
    } else if (elements instanceof HTMLTextAreaElement) {
      elements.value = String(value ?? "");
    }
  }
}
function resolveElement(target) {
  if (target instanceof Element) {
    return target;
  }
  return document.querySelector(`[p-ref="${target}"]`) ?? document.querySelector(target);
}
function resolveHTMLElement(target) {
  if (target instanceof HTMLElement) {
    return target;
  }
  const byRef = document.querySelector(`[p-ref="${target}"]`);
  if (byRef instanceof HTMLElement) {
    return byRef;
  }
  const bySelector = document.querySelector(target);
  if (bySelector instanceof HTMLElement) {
    return bySelector;
  }
  return null;
}
function calculateDelay(attempt, options) {
  let calculatedDelay;
  if (options.backoff === "linear") {
    calculatedDelay = options.delay * attempt;
  } else {
    calculatedDelay = options.delay * Math.pow(2, attempt - 1);
  }
  return Math.min(calculatedDelay, options.maxDelay);
}
function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
function isDisableableElement(el) {
  return "disabled" in el;
}
function captureOriginalState(element) {
  return {
    text: element.innerText,
    disabled: isDisableableElement(element) ? element.disabled : false
  };
}
function applyLoadingState(element, className, text, disabled) {
  element.classList.add(className);
  if (text !== void 0) {
    element.innerText = text;
  }
  if (disabled && isDisableableElement(element)) {
    element.disabled = true;
  }
}
function restoreOriginalState(element, className, original, textWasSet, disabled) {
  element.classList.remove(className);
  if (textWasSet) {
    element.innerText = original.text;
  }
  if (disabled && isDisableableElement(element)) {
    element.disabled = original.disabled;
  }
}
async function loading(target, promise, options = {}) {
  const { className = "loading", text, disabled = true, minDuration = 0, onStart, onEnd } = options;
  const element = resolveHTMLElement(target);
  if (!element) {
    console.warn(`[pk] loading: target "${target}" not found`);
    return promise;
  }
  const originalState = captureOriginalState(element);
  applyLoadingState(element, className, text, disabled);
  onStart?.();
  const startTime = Date.now();
  try {
    const result = await promise;
    const elapsed = Date.now() - startTime;
    if (elapsed < minDuration) {
      await sleep(minDuration - elapsed);
    }
    return result;
  } finally {
    restoreOriginalState(element, className, originalState, text !== void 0, disabled);
    onEnd?.();
  }
}
async function withRetry(operation, options = {}) {
  const {
    attempts = 3,
    backoff = "exponential",
    delay: delay2 = 1e3,
    maxDelay = 3e4,
    onRetry,
    shouldRetry = () => true
  } = options;
  let lastError = null;
  for (let attempt = 1; attempt <= attempts; attempt++) {
    try {
      return await operation();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));
      if (attempt === attempts || !shouldRetry(lastError)) {
        throw lastError;
      }
      onRetry?.(attempt, lastError);
      const waitTime = calculateDelay(attempt, { backoff, delay: delay2, maxDelay });
      await sleep(waitTime);
    }
  }
  throw lastError ?? new Error("Retry failed");
}
async function withLoading(target, operation, options = {}) {
  return loading(target, operation(), options);
}
function debounceAsync(handler, delay2) {
  let timeoutId = null;
  let pendingResolve = null;
  let pendingReject = null;
  const debounced = (...args) => {
    if (timeoutId !== null) {
      clearTimeout(timeoutId);
    }
    return new Promise((resolve, reject) => {
      pendingResolve = resolve;
      pendingReject = reject;
      timeoutId = setTimeout(() => {
        void (async () => {
          timeoutId = null;
          try {
            const result = await handler(...args);
            pendingResolve?.(result);
          } catch (error) {
            pendingReject?.(error instanceof Error ? error : new Error(String(error)));
          }
        })();
      }, delay2);
    });
  };
  debounced.cancel = () => {
    if (timeoutId !== null) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  };
  return debounced;
}
function throttleAsync(handler, delay2) {
  let lastCall = 0;
  let pendingPromise = null;
  return async (...args) => {
    const now = Date.now();
    if (now - lastCall >= delay2) {
      lastCall = now;
      pendingPromise = handler(...args);
      return pendingPromise;
    }
    if (pendingPromise) {
      return pendingPromise;
    }
    return void 0;
  };
}
function resolveTarget(target) {
  if (target === "*") {
    return document;
  }
  return resolveElement(target);
}
function dispatch(target, eventName, detail, options) {
  const el = resolveTarget(target);
  if (!el) {
    console.warn(`[pk] dispatch: target "${target}" not found`);
    return;
  }
  const event = new CustomEvent(eventName, {
    detail,
    bubbles: options?.bubbles ?? true,
    composed: options?.composed ?? true
  });
  el.dispatchEvent(event);
}
function listen(target, eventName, callback) {
  const el = resolveTarget(target);
  if (!el) {
    console.warn(`[pk] listen: target "${target}" not found`);
    return () => {
    };
  }
  const handler = (e) => {
    callback(e);
  };
  el.addEventListener(eventName, handler);
  return () => {
    el.removeEventListener(eventName, handler);
  };
}
function listenOnce(target, eventName, callback) {
  const unsubscribe = listen(target, eventName, (event) => {
    unsubscribe();
    callback(event);
  });
  return () => {
    unsubscribe();
  };
}
function waitForEvent(target, eventName, timeout2) {
  return new Promise((resolve, reject) => {
    let timeoutId = null;
    const unsubscribe = listenOnce(target, eventName, (event) => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
      resolve(event.detail);
    });
    if (timeout2 !== void 0 && timeout2 > 0) {
      timeoutId = setTimeout(() => {
        unsubscribe();
        reject(new Error(`Timeout waiting for event "${eventName}"`));
      }, timeout2);
    }
  });
}
function debounce(handler, ms) {
  let timeoutId;
  return (...args) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => handler(...args), ms);
  };
}
function throttle(handler, ms) {
  let lastCall = 0;
  return (...args) => {
    const now = Date.now();
    if (now - lastCall >= ms) {
      lastCall = now;
      handler(...args);
    }
  };
}
const RETRY_BACKOFF_BASE_MS = 1e3;
const debounceRegistry = /* @__PURE__ */ new Map();
function delay$1(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
function toStringRecord(args) {
  if (!args) {
    return void 0;
  }
  const result = {};
  for (const [key, value] of Object.entries(args)) {
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      result[key] = value;
    } else if (value !== null && value !== void 0) {
      result[key] = String(value);
    }
  }
  return result;
}
function sanitiseHTML(html) {
  const template = document.createElement("template");
  template.innerHTML = html;
  template.content.querySelectorAll("script").forEach((s) => s.remove());
  template.content.querySelectorAll("*").forEach((el) => {
    for (const attr of Array.from(el.attributes)) {
      if (attr.name.startsWith("on")) {
        el.removeAttribute(attr.name);
      }
    }
  });
  const container = document.createElement("div");
  container.appendChild(template.content.cloneNode(true));
  return container.innerHTML;
}
function applyOptimisticUpdate(element, optimistic) {
  if (typeof optimistic === "string") {
    element.innerHTML = sanitiseHTML(optimistic);
  } else if (typeof optimistic === "object" && optimistic !== null) {
    const opts = optimistic;
    if (typeof opts.innerHTML === "string") {
      element.innerHTML = sanitiseHTML(opts.innerHTML);
    }
    if (typeof opts.className === "string") {
      element.className = opts.className;
    }
    if (typeof opts.addClass === "string") {
      element.classList.add(opts.addClass);
    }
    if (typeof opts.removeClass === "string") {
      element.classList.remove(opts.removeClass);
    }
  }
}
async function executeReload(handle, options, retriesLeft, maxRetries) {
  const { args, loading: loading2 = true, optimistic, onSuccess, onError } = options;
  try {
    if (optimistic !== void 0 && handle.element) {
      applyOptimisticUpdate(handle.element, optimistic);
    }
    if (loading2 && handle.element) {
      applyLoadingIndicator(handle.element);
    }
    await handle.reload(toStringRecord(args));
    onSuccess?.(handle.element?.innerHTML ?? "");
  } catch (error) {
    if (retriesLeft > 0) {
      const backoffDelay = Math.pow(2, maxRetries - retriesLeft) * RETRY_BACKOFF_BASE_MS;
      await delay$1(backoffDelay);
      return executeReload(handle, options, retriesLeft - 1, maxRetries);
    }
    console.error(`[pk] Failed to reload partial after ${maxRetries} retries:`, {
      name: handle.element?.getAttribute("data-partial"),
      error
    });
    onError?.(error);
    throw error;
  } finally {
    if (loading2 && handle.element) {
      removeLoadingIndicator(handle.element);
    }
  }
}
async function reloadPartial(nameOrElement, options = {}) {
  const handle = partial(nameOrElement);
  const name = typeof nameOrElement === "string" ? nameOrElement : nameOrElement.getAttribute("partial_name") ?? "unknown";
  if (!handle.element) {
    console.warn(`[pk] reloadPartial: partial "${name}" not found`);
    return;
  }
  const { retry = 0, debounce: debounceMs, ...reloadOpts } = options;
  if (debounceMs && debounceMs > 0) {
    let debouncedFn = debounceRegistry.get(name);
    if (!debouncedFn) {
      debouncedFn = debounce(async () => {
        await executeReload(handle, reloadOpts, retry, retry);
      }, debounceMs);
      debounceRegistry.set(name, debouncedFn);
    }
    debouncedFn();
    return;
  }
  return executeReload(handle, reloadOpts, retry, retry);
}
async function reloadGroup(names, options = {}) {
  const { mode = "parallel", args, loading: loading2, onProgress } = options;
  if (mode === "parallel") {
    const promises = names.map(
      (name, index) => reloadPartial(name, { args, loading: loading2 }).then(() => {
        onProgress?.(index + 1, names.length);
      })
    );
    await Promise.all(promises);
  } else {
    for (let i = 0; i < names.length; i++) {
      await reloadPartial(names[i], { args, loading: loading2 });
      onProgress?.(i + 1, names.length);
    }
  }
}
function autoRefresh(name, options) {
  const { interval, when, onError = "retry", maxRetries = 3 } = options;
  let intervalId = null;
  let retryCount = 0;
  let stopped = false;
  const refresh = async () => {
    if (stopped) {
      return;
    }
    if (when && !when()) {
      return;
    }
    try {
      await reloadPartial(name);
      retryCount = 0;
    } catch (error) {
      retryCount++;
      console.warn(`[pk] autoRefresh "${name}" failed (attempt ${retryCount}/${maxRetries}):`, error);
      if (onError === "stop" || retryCount >= maxRetries) {
        if (intervalId !== null) {
          clearInterval(intervalId);
          intervalId = null;
        }
        stopped = true;
        console.warn(`[pk] autoRefresh "${name}" stopped after ${retryCount} failures`);
      }
    }
  };
  intervalId = setInterval(() => void refresh(), interval);
  return () => {
    stopped = true;
    if (intervalId !== null) {
      clearInterval(intervalId);
      intervalId = null;
    }
  };
}
async function reloadCascade(tree, options = {}) {
  const { args, onNodeComplete } = options;
  await reloadPartial(tree.name, { args });
  onNodeComplete?.(tree.name);
  if (tree.children && tree.children.length > 0) {
    await Promise.all(
      tree.children.map((child) => reloadCascade(child, options))
    );
  }
}
const MAX_RECONNECT_BACKOFF_MULTIPLIER = 5;
function parseSSEData(data) {
  if (typeof data === "string") {
    try {
      return JSON.parse(data);
    } catch {
    }
  }
  return data;
}
function createMessageHandler(state, name, onMessage) {
  return (event) => {
    if (state.stopped) {
      return;
    }
    const data = parseSSEData(event.data);
    if (onMessage) {
      onMessage(data);
    } else {
      void reloadPartial(name);
    }
  };
}
function createErrorHandler(state, url, onError, reconnectDelay, maxReconnects, onClose, connect) {
  return () => {
    if (state.stopped) {
      return;
    }
    state.eventSource?.close();
    state.eventSource = null;
    if (onError === "stop") {
      state.stopped = true;
      console.warn(`[pk] SSE connection to "${url}" failed, stopping`);
      onClose?.();
      return;
    }
    if (state.reconnectCount < maxReconnects) {
      state.reconnectCount++;
      const delay2 = reconnectDelay * Math.min(state.reconnectCount, MAX_RECONNECT_BACKOFF_MULTIPLIER);
      console.warn(`[pk] SSE connection to "${url}" lost, reconnecting in ${delay2}ms (attempt ${state.reconnectCount}/${maxReconnects})`);
      state.reconnectTimeout = setTimeout(() => {
        if (!state.stopped) {
          connect();
        }
      }, delay2);
    } else {
      state.stopped = true;
      console.error(`[pk] SSE connection to "${url}" failed after ${maxReconnects} attempts`);
      onClose?.();
    }
  };
}
function subscribeToUpdates(name, options) {
  const {
    url,
    onMessage,
    onError = "reconnect",
    reconnectDelay = 3e3,
    maxReconnects = 10,
    eventTypes = ["message"],
    onOpen,
    onClose
  } = options;
  const state = {
    eventSource: null,
    reconnectCount: 0,
    reconnectTimeout: null,
    stopped: false
  };
  const handleMessage = createMessageHandler(state, name, onMessage);
  const connect = () => {
    if (state.stopped) {
      return;
    }
    try {
      state.eventSource = new EventSource(url);
      state.eventSource.onopen = () => {
        state.reconnectCount = 0;
        onOpen?.();
      };
      state.eventSource.onerror = createErrorHandler(
        state,
        url,
        onError,
        reconnectDelay,
        maxReconnects,
        onClose,
        connect
      );
      for (const eventType of eventTypes) {
        state.eventSource.addEventListener(eventType, handleMessage);
      }
    } catch (error) {
      console.error(`[pk] Failed to create SSE connection to "${url}":`, { error });
      state.eventSource = null;
    }
  };
  connect();
  return () => {
    state.stopped = true;
    if (state.reconnectTimeout) {
      clearTimeout(state.reconnectTimeout);
    }
    state.eventSource?.close();
    onClose?.();
  };
}
function createSSESubscription(name, options) {
  let state = "connecting";
  let reconnectCount = 0;
  const unsubscribe = subscribeToUpdates(name, {
    ...options,
    onOpen: () => {
      state = "open";
      reconnectCount = 0;
      options.onOpen?.();
    },
    onClose: () => {
      state = "closed";
      options.onClose?.();
    },
    onError: options.onError
  });
  return {
    unsubscribe,
    get state() {
      return state;
    },
    get reconnectCount() {
      return reconnectCount;
    }
  };
}
const DEFAULT_IDLE_CALLBACK_TIMEOUT_MS = 50;
function whenVisible(target, callback, options = {}) {
  const {
    threshold = 0,
    root = null,
    rootMargin = "0px",
    once: triggerOnce = true
  } = options;
  const element = resolveElement(target);
  if (!element) {
    console.warn(`[pk] whenVisible: target "${target}" not found`);
    return () => {
    };
  }
  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          callback(entry);
          if (triggerOnce) {
            observer.disconnect();
          }
        } else if (!triggerOnce) {
          callback(entry);
        }
      }
    },
    { threshold, root, rootMargin }
  );
  observer.observe(element);
  return () => {
    observer.disconnect();
  };
}
function withAbortSignal(operation) {
  const controller = new AbortController();
  return {
    promise: operation(controller.signal),
    abort: () => controller.abort(),
    signal: controller.signal
  };
}
function timeout(ms) {
  let timeoutId = null;
  let rejectFn = null;
  const promise = new Promise((resolve, reject) => {
    rejectFn = reject;
    timeoutId = setTimeout(resolve, ms);
  });
  return {
    promise,
    cancel: () => {
      if (timeoutId !== null) {
        clearTimeout(timeoutId);
        timeoutId = null;
        rejectFn?.(new Error("Timeout cancelled"));
      }
    }
  };
}
function poll(operation, options) {
  const {
    interval,
    until,
    maxAttempts,
    onPoll,
    onStop
  } = options;
  let stopped = false;
  let attempt = 0;
  let timeoutId = null;
  const runPoll = async () => {
    if (stopped) {
      return;
    }
    attempt++;
    try {
      const result = await operation();
      onPoll?.(result, attempt);
      if (until) {
        const shouldStop = await until();
        if (shouldStop) {
          stopped = true;
          onStop?.("condition");
          return;
        }
      }
      if (maxAttempts !== void 0 && attempt >= maxAttempts) {
        stopped = true;
        onStop?.("maxAttempts");
        return;
      }
      timeoutId = setTimeout(() => {
        void runPoll();
      }, interval);
    } catch (error) {
      console.error("[pk] poll error:", {
        fn: operation.name || "anonymous",
        interval,
        stopped,
        error
      });
      if (!stopped) {
        timeoutId = setTimeout(() => {
          void runPoll();
        }, interval);
      }
    }
  };
  void runPoll();
  return () => {
    stopped = true;
    if (timeoutId !== null) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
    onStop?.("manual");
  };
}
function watchMutations(target, callback, options = {}) {
  const {
    childList = true,
    attributes = false,
    characterData = false,
    subtree = false,
    attributeFilter
  } = options;
  const element = resolveElement(target);
  if (!element) {
    console.warn(`[pk] watchMutations: target "${target}" not found`);
    return () => {
    };
  }
  const observer = new MutationObserver(callback);
  observer.observe(element, {
    childList,
    attributes,
    characterData,
    subtree,
    attributeFilter
  });
  return () => {
    observer.disconnect();
  };
}
function whenIdle(task, options) {
  if ("requestIdleCallback" in window) {
    const id2 = window.requestIdleCallback(task, options);
    return () => window.cancelIdleCallback(id2);
  }
  const timeoutMs = options?.timeout ?? DEFAULT_IDLE_CALLBACK_TIMEOUT_MS;
  const id = setTimeout(() => task(), timeoutMs);
  return () => clearTimeout(id);
}
function nextFrame() {
  return new Promise((resolve) => {
    requestAnimationFrame(resolve);
  });
}
async function waitFrames(count) {
  for (let i = 0; i < count; i++) {
    await nextFrame();
  }
}
function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}
function once(factory) {
  let called = false;
  let result;
  return () => {
    if (!called) {
      called = true;
      result = factory();
    }
    return result;
  };
}
const MAX_TRACE_ENTRIES = 1e3;
class PKTracer {
  constructor() {
    this.enabled = false;
    this.config = {
      partialReloads: true,
      events: true,
      handlers: true,
      sse: true
    };
    this.entries = [];
    this.maxEntries = MAX_TRACE_ENTRIES;
  }
  /**
   * Enables tracing with optional configuration.
   *
   * @param config - Partial configuration to merge with defaults.
   */
  enable(config) {
    this.enabled = true;
    if (config) {
      this.config = { ...this.config, ...config };
    }
    console.log("%c[Piko] Tracing enabled", "color: #29e; font-weight: bold", this.config);
  }
  /** Disables tracing. */
  disable() {
    this.enabled = false;
    console.log("%c[Piko] Tracing disabled", "color: #999");
  }
  /**
   * Checks if tracing is enabled.
   *
   * @returns True if tracing is currently enabled.
   */
  isEnabled() {
    return this.enabled;
  }
  /** Clears all trace entries. */
  clear() {
    this.entries = [];
  }
  /**
   * Returns all trace entries.
   *
   * @returns A shallow copy of the entries array.
   */
  getEntries() {
    return [...this.entries];
  }
  /**
   * Returns aggregated metrics grouped by name.
   *
   * @returns Record mapping names to their aggregated metrics.
   */
  getMetrics() {
    const metrics = /* @__PURE__ */ new Map();
    for (const entry of this.entries) {
      let m = metrics.get(entry.name);
      if (!m) {
        m = { count: 0, totalDuration: 0, maxDuration: 0, minDuration: Infinity };
        metrics.set(entry.name, m);
      }
      m.count++;
      if (entry.duration !== void 0) {
        m.totalDuration += entry.duration;
        m.maxDuration = Math.max(m.maxDuration, entry.duration);
        m.minDuration = Math.min(m.minDuration, entry.duration);
      }
    }
    return Object.fromEntries(
      Array.from(metrics.entries()).map(([name, m]) => [
        name,
        {
          count: m.count,
          avgDuration: m.count > 0 ? m.totalDuration / m.count : 0,
          maxDuration: m.maxDuration,
          minDuration: m.minDuration === Infinity ? 0 : m.minDuration
        }
      ])
    );
  }
  /**
   * Adds a trace entry, trimming old entries if the buffer exceeds capacity.
   *
   * @param entry - Trace entry to add.
   */
  addEntry(entry) {
    this.entries.push(entry);
    if (this.entries.length > this.maxEntries) {
      this.entries = this.entries.slice(-this.maxEntries);
    }
  }
  /**
   * Traces a partial reload.
   *
   * @param name - Partial name.
   * @param duration - Duration in milliseconds.
   * @param args - Optional metadata.
   */
  tracePartialReload(name, duration, args) {
    if (!this.enabled || !this.config.partialReloads) {
      return;
    }
    this.addEntry({
      type: "partial",
      name,
      duration,
      timestamp: Date.now(),
      metadata: args
    });
    console.log(
      `%c[Piko] Partial Reload: "${name}" (${duration.toFixed(1)}ms)`,
      "color: #29e; font-weight: bold",
      args ? `
  Args: ${JSON.stringify(args)}` : ""
    );
  }
  /**
   * Traces an event emission.
   *
   * @param eventName - Event name.
   * @param source - Source of the event.
   * @param payload - Optional event payload.
   */
  traceEvent(eventName, source, payload) {
    if (!this.enabled || !this.config.events) {
      return;
    }
    this.addEntry({
      type: "event",
      name: eventName,
      timestamp: Date.now(),
      metadata: { source, payload }
    });
    console.log(
      `%c[Piko] Event: "${eventName}"`,
      "color: #4a4; font-weight: bold",
      `
  Source: ${source}`,
      payload !== void 0 ? `
  Payload: ${JSON.stringify(payload)}` : ""
    );
  }
  /**
   * Traces a handler execution.
   *
   * @param handlerName - Handler name.
   * @param duration - Duration in milliseconds.
   * @param result - Optional result value.
   */
  traceHandler(handlerName, duration, result) {
    if (!this.enabled || !this.config.handlers) {
      return;
    }
    this.addEntry({
      type: "handler",
      name: handlerName,
      duration,
      timestamp: Date.now(),
      metadata: result !== void 0 ? { result } : void 0
    });
    console.log(
      `%c[Piko] Handler: "${handlerName}" (${duration.toFixed(1)}ms)`,
      "color: #a4a; font-weight: bold"
    );
  }
  /**
   * Traces an SSE connection event.
   *
   * @param url - SSE endpoint URL.
   * @param event - Connection event type.
   * @param data - Optional event data.
   */
  traceSSE(url, event, data) {
    if (!this.enabled || !this.config.sse) {
      return;
    }
    this.addEntry({
      type: "sse",
      name: url,
      timestamp: Date.now(),
      metadata: { event, data }
    });
    const colours = {
      connect: "color: #4a4",
      disconnect: "color: #a44",
      message: "color: #29e",
      error: "color: #f44"
    };
    console.log(
      `%c[Piko] SSE ${event}: "${url}"`,
      `${colours[event]}; font-weight: bold`,
      data !== void 0 ? `
  Data: ${JSON.stringify(data)}` : ""
    );
  }
}
const trace = new PKTracer();
function traceLog(name, data) {
  if (!trace.isEnabled()) {
    return;
  }
  trace.traceEvent(name, "manual", data);
}
function traceAsync(name, operation) {
  return async () => {
    const start = performance.now();
    try {
      return await operation();
    } finally {
      const duration = performance.now() - start;
      trace.traceHandler(name, duration);
    }
  };
}
const activeRefreshers = /* @__PURE__ */ new Map();
function processElement(el) {
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
  let when;
  if (whenCondition === "visible") {
    when = () => document.visibilityState === "visible";
  } else if (whenCondition === "focus") {
    when = () => document.hasFocus();
  }
  const onErrorStr = el.dataset.autoRefreshOnError;
  const onError = onErrorStr === "stop" ? "stop" : "retry";
  const cleanup = autoRefresh(partialName, {
    interval,
    when,
    onError
  });
  activeRefreshers.set(el, cleanup);
}
function cleanupElement(el) {
  const cleanup = activeRefreshers.get(el);
  if (cleanup) {
    cleanup();
    activeRefreshers.delete(el);
  }
}
function cleanupElementAndDescendants(el) {
  cleanupElement(el);
  const descendants = el.querySelectorAll("[data-auto-refresh]");
  descendants.forEach(cleanupElement);
}
function initAutoRefreshObserver() {
  const existingElements = document.querySelectorAll("[data-auto-refresh]");
  existingElements.forEach(processElement);
  function processAddedNode(node) {
    if (!(node instanceof HTMLElement)) {
      return;
    }
    if (node.hasAttribute("data-auto-refresh")) {
      processElement(node);
    }
    node.querySelectorAll("[data-auto-refresh]").forEach(processElement);
  }
  function processRemovedNode(node) {
    if (node instanceof HTMLElement) {
      cleanupElementAndDescendants(node);
    }
  }
  const observer = new MutationObserver((mutations) => {
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
function stopAllAutoRefreshers() {
  for (const cleanup of activeRefreshers.values()) {
    cleanup();
  }
  activeRefreshers.clear();
}
function getActiveRefresherCount() {
  return activeRefreshers.size;
}
function findFunctionInGlobal(name, exports$1) {
  const exportedFunction = exports$1[name];
  return typeof exportedFunction === "function" ? exportedFunction : void 0;
}
function findFunctionInScopes(name, scopes) {
  for (const scope of Object.values(scopes)) {
    const scopedFunction = scope[name];
    if (typeof scopedFunction === "function") {
      return scopedFunction;
    }
  }
  return void 0;
}
function collectExportedFunctionNames(global, scopes) {
  const fns = /* @__PURE__ */ new Set();
  for (const key of Object.keys(global)) {
    if (typeof global[key] === "function") {
      fns.add(key);
    }
  }
  for (const scope of Object.values(scopes)) {
    for (const key of Object.keys(scope)) {
      if (typeof scope[key] === "function") {
        fns.add(key);
      }
    }
  }
  return Array.from(fns);
}
function createPageContext(options = {}) {
  let globalExports = {};
  let scopedExports = {};
  let nameToIds = {};
  const ctx = {
    getFunction(name) {
      return findFunctionInGlobal(name, globalExports) ?? findFunctionInScopes(name, scopedExports);
    },
    hasFunction(name) {
      return typeof globalExports[name] === "function" || Object.values(scopedExports).some((scope) => typeof scope[name] === "function");
    },
    getExportedFunctions() {
      return collectExportedFunctionNames(globalExports, scopedExports);
    },
    clear() {
      globalExports = {};
      scopedExports = {};
      nameToIds = {};
    },
    async loadModule(url, partialName) {
      try {
        const module = await import(
          /* @vite-ignore */
          url
        );
        if (partialName) {
          const partialId = module.__PARTIAL_ID__ ?? partialName;
          scopedExports[partialId] = { ...scopedExports[partialId] ?? {}, ...module };
          ctx.registerPartialInstance(partialName, partialId);
        } else {
          globalExports = { ...globalExports, ...module };
        }
        if (typeof module.__reinit__ === "function") {
          module.__reinit__();
        }
        options.onModuleLoaded?.(url, ctx.getExportedFunctions());
      } catch (error) {
        if (options.onError) {
          options.onError(error, `loadModule(${url})`);
        } else {
          console.error("[PageContext] Failed to load module:", { url, error });
        }
      }
    },
    setExports(newExports) {
      globalExports = { ...globalExports, ...newExports };
    },
    getScopedFunction(name, partialId) {
      const scope = scopedExports[partialId];
      return scope ? findFunctionInGlobal(name, scope) : void 0;
    },
    getFunctionsByPartialName(partialName, fnName) {
      return (nameToIds[partialName] ?? []).map((id) => scopedExports[id]).filter((scope) => Boolean(scope)).map((scope) => scope[fnName]).filter((value) => typeof value === "function");
    },
    registerPartialInstance(partialName, partialId) {
      nameToIds[partialName] ??= [];
      if (!nameToIds[partialName].includes(partialId)) {
        nameToIds[partialName].push(partialId);
      }
    },
    getRegisteredPartialNames() {
      return Object.keys(nameToIds);
    }
  };
  return ctx;
}
let globalPageContext = null;
function getGlobalPageContext() {
  globalPageContext ??= createPageContext();
  return globalPageContext;
}
function findClosestMatch(target, candidates, threshold = 3) {
  if (candidates.length === 0) {
    return void 0;
  }
  let bestMatch;
  let bestDistance = Infinity;
  for (const candidate of candidates) {
    const distance = levenshteinDistance(target.toLowerCase(), candidate.toLowerCase());
    if (distance < bestDistance && distance <= threshold) {
      bestDistance = distance;
      bestMatch = candidate;
    }
  }
  return bestMatch;
}
function levenshteinDistance(a, b) {
  if (a.length === 0) {
    return b.length;
  }
  if (b.length === 0) {
    return a.length;
  }
  const matrix = [];
  for (let i = 0; i <= b.length; i++) {
    matrix[i] = [i];
  }
  for (let j = 0; j <= a.length; j++) {
    matrix[0][j] = j;
  }
  for (let i = 1; i <= b.length; i++) {
    for (let j = 1; j <= a.length; j++) {
      if (b.charAt(i - 1) === a.charAt(j - 1)) {
        matrix[i][j] = matrix[i - 1][j - 1];
      } else {
        matrix[i][j] = Math.min(
          matrix[i - 1][j - 1] + 1,
          matrix[i][j - 1] + 1,
          matrix[i - 1][j] + 1
        );
      }
    }
  }
  return matrix[b.length][a.length];
}
if (typeof window !== "undefined") {
  window.__pikoPageContext = getGlobalPageContext();
}
const HookEvent = {
  /** Fired when the framework is fully initialised. */
  FRAMEWORK_READY: "framework:ready",
  /** Fired on page view (initial load and each navigation). */
  PAGE_VIEW: "page:view",
  /** Fired before SPA navigation begins. */
  NAVIGATION_START: "navigation:start",
  /** Fired after SPA navigation completes successfully. */
  NAVIGATION_COMPLETE: "navigation:complete",
  /** Fired when navigation fails. */
  NAVIGATION_ERROR: "navigation:error",
  /** Fired when a server action is triggered. */
  ACTION_START: "action:start",
  /** Fired when a server action completes. */
  ACTION_COMPLETE: "action:complete",
  /** Fired when a modal opens. */
  MODAL_OPEN: "modal:open",
  /** Fired when a modal closes. */
  MODAL_CLOSE: "modal:close",
  /** Fired after a partial render completes. */
  PARTIAL_RENDER: "partial:render",
  /** Fired when a form becomes dirty (has unsaved changes). */
  FORM_DIRTY: "form:dirty",
  /** Fired when a form becomes clean (changes saved or reset). */
  FORM_CLEAN: "form:clean",
  /** Fired when network connection is restored. */
  NETWORK_ONLINE: "network:online",
  /** Fired when network connection is lost. */
  NETWORK_OFFLINE: "network:offline",
  /** Fired when an error occurs. */
  ERROR: "error",
  /** Fired when user code requests a custom analytics event via piko.analytics.track(). */
  ANALYTICS_TRACK: "analytics:track"
};
function buildHooksAPI(registerHook, removeHook, clearHooks, getIsReady, hookEventRef) {
  return {
    on(event, callback, options) {
      return registerHook(event, callback, options);
    },
    once(event, callback, options) {
      return registerHook(event, callback, { ...options, once: true });
    },
    off(event, id) {
      removeHook(event, id);
    },
    clear(event) {
      clearHooks(event);
    },
    get ready() {
      return getIsReady();
    },
    events: hookEventRef
  };
}
function processHookQueue(registerHook) {
  const windowWithQueue = window;
  const queue = windowWithQueue.__PP_HOOKS_QUEUE__;
  if (!Array.isArray(queue)) {
    return;
  }
  for (const queued of queue) {
    registerHook(queued.event, queued.callback, queued.options);
  }
  windowWithQueue.__PP_HOOKS_QUEUE__ = [];
}
function createHookManager() {
  const hooks = /* @__PURE__ */ new Map();
  let isReady = false, hookIdCounter = 0;
  const generateId = () => `hook_${++hookIdCounter}`;
  const sortByPriority = (a, b) => b.priority - a.priority;
  const registerHook = (event, callback, options = {}) => {
    const id = options.id ?? generateId();
    const hook = {
      id,
      callback,
      once: options.once ?? false,
      priority: options.priority ?? 0
    };
    const eventHooks = hooks.get(event) ?? [];
    if (!hooks.has(event)) {
      hooks.set(event, eventHooks);
    }
    eventHooks.push(hook);
    eventHooks.sort(sortByPriority);
    return () => removeHook(event, id);
  };
  const removeHook = (event, id) => {
    const eventHooks = hooks.get(event);
    if (!eventHooks) {
      return;
    }
    const index = eventHooks.findIndex((h) => h.id === id);
    if (index !== -1) {
      eventHooks.splice(index, 1);
    }
  };
  const emit = (event, payload) => {
    const eventHooks = hooks.get(event);
    if (!eventHooks) {
      return;
    }
    const toRemove = [];
    for (const hook of eventHooks) {
      try {
        hook.callback(payload);
        if (hook.once) {
          toRemove.push(hook.id);
        }
      } catch (error) {
        console.error(
          `HookManager: Error in hook "${hook.id}" for event "${event}":`,
          error
        );
      }
    }
    for (const id of toRemove) {
      removeHook(event, id);
    }
  };
  const api = buildHooksAPI(
    registerHook,
    removeHook,
    (event) => {
      if (event) {
        hooks.delete(event);
      } else {
        hooks.clear();
      }
    },
    () => isReady,
    HookEvent
  );
  return {
    api,
    emit,
    processQueue() {
      processHookQueue(registerHook);
    },
    setReady() {
      isReady = true;
    }
  };
}
let _readyCallbacks = [];
let _isReady = false;
var piko;
((piko2) => {
  piko2.partial = partial;
  piko2.bus = bus;
  ((nav2) => {
    nav2.navigate = navigate;
    nav2.back = goBack;
    nav2.forward = goForward;
    nav2.go = go;
    nav2.current = currentRoute;
    nav2.buildUrl = buildUrl;
    nav2.updateQuery = updateQuery;
    nav2.guard = registerNavigationGuard;
    nav2.matchPath = matchPath;
    nav2.extractParams = extractParams;
    function navigateTo(url, event2) {
      void PPFramework.navigateTo(url, event2);
    }
    nav2.navigateTo = navigateTo;
  })(piko2.nav || (piko2.nav = {}));
  ((form2) => {
    form2.data = formData;
    form2.validate = validate;
    form2.reset = resetForm;
    form2.setValues = setFormValues;
  })(piko2.form || (piko2.form = {}));
  ((ui2) => {
    ui2.loading = loading;
    ui2.withLoading = withLoading;
    ui2.withRetry = withRetry;
  })(piko2.ui || (piko2.ui = {}));
  ((event2) => {
    event2.dispatch = dispatch;
    event2.listen = listen;
    event2.listenOnce = listenOnce;
    event2.waitFor = waitForEvent;
  })(piko2.event || (piko2.event = {}));
  ((partials2) => {
    partials2.reload = reloadPartial;
    partials2.reloadGroup = reloadGroup;
    partials2.reloadCascade = reloadCascade;
    partials2.autoRefresh = autoRefresh;
    function render(options) {
      return PPFramework.remoteRender(options);
    }
    partials2.render = render;
  })(piko2.partials || (piko2.partials = {}));
  ((sse2) => {
    sse2.subscribe = subscribeToUpdates;
    sse2.create = createSSESubscription;
  })(piko2.sse || (piko2.sse = {}));
  ((timing2) => {
    timing2.debounce = debounce;
    timing2.throttle = throttle;
    timing2.debounceAsync = debounceAsync;
    timing2.throttleAsync = throttleAsync;
    timing2.timeout = timeout;
    timing2.poll = poll;
    timing2.nextFrame = nextFrame;
    timing2.waitFrames = waitFrames;
  })(piko2.timing || (piko2.timing = {}));
  ((util2) => {
    util2.whenVisible = whenVisible;
    util2.withAbortSignal = withAbortSignal;
    util2.watchMutations = watchMutations;
    util2.whenIdle = whenIdle;
    util2.deferred = deferred;
    util2.once = once;
  })(piko2.util || (piko2.util = {}));
  ((trace2) => {
    trace2.enable = trace.enable;
    trace2.disable = trace.disable;
    trace2.isEnabled = trace.isEnabled;
    trace2.clear = trace.clear;
    trace2.getEntries = trace.getEntries;
    trace2.getMetrics = trace.getMetrics;
    trace2.log = traceLog;
    function async(name, operation) {
      return traceAsync(name, operation)();
    }
    trace2.async = async;
  })(piko2.trace || (piko2.trace = {}));
  ((autoRefreshObserver2) => {
    autoRefreshObserver2.init = initAutoRefreshObserver;
    autoRefreshObserver2.stopAll = stopAllAutoRefreshers;
    autoRefreshObserver2.getActiveCount = getActiveRefresherCount;
  })(piko2.autoRefreshObserver || (piko2.autoRefreshObserver = {}));
  ((context2) => {
    context2.get = getGlobalPageContext;
  })(piko2.context || (piko2.context = {}));
  ((hooks2) => {
    function on(hookEvent, callback, options) {
      return PPFramework.hooks.on(hookEvent, callback, options);
    }
    hooks2.on = on;
    function once2(hookEvent, callback, options) {
      return PPFramework.hooks.once(hookEvent, callback, options);
    }
    hooks2.once = once2;
    function off(hookEvent, id) {
      PPFramework.hooks.off(hookEvent, id);
    }
    hooks2.off = off;
    function clear(hookEvent) {
      PPFramework.hooks.clear(hookEvent);
    }
    hooks2.clear = clear;
    hooks2.events = HookEvent;
  })(piko2.hooks || (piko2.hooks = {}));
  ((analytics2) => {
    function track(eventName, params) {
      PPFramework.emitHook(HookEvent.ANALYTICS_TRACK, {
        eventName,
        params: params ?? {},
        timestamp: Date.now()
      });
    }
    analytics2.track = track;
  })(piko2.analytics || (piko2.analytics = {}));
  function registerHelper(name, helper) {
    PPFramework.registerHelper(name, helper);
  }
  piko2.registerHelper = registerHelper;
  function getModuleConfig2(moduleName) {
    return PPFramework.getModuleConfig(moduleName);
  }
  piko2.getModuleConfig = getModuleConfig2;
  ((loader2) => {
    function toggle(visible) {
      PPFramework.toggleLoader(visible);
    }
    loader2.toggle = toggle;
    function progress(percent) {
      PPFramework.updateProgressBar(percent);
    }
    loader2.progress = progress;
    function error(message) {
      PPFramework.displayError(message);
    }
    loader2.error = error;
    function create(color) {
      PPFramework.createLoaderIndicator(color);
    }
    loader2.create = create;
  })(piko2.loader || (piko2.loader = {}));
  ((modal2) => {
    function open(options) {
      return PPFramework.openModalIfAvailable(options);
    }
    modal2.open = open;
  })(piko2.modal || (piko2.modal = {}));
  ((network2) => {
    function isOnline() {
      return PPFramework.isOnline;
    }
    network2.isOnline = isOnline;
  })(piko2.network || (piko2.network = {}));
  function patchPartial(html, selector) {
    PPFramework.patchPartial(html, selector);
  }
  piko2.patchPartial = patchPartial;
  function ready(callback) {
    if (_isReady) {
      callback();
      return;
    }
    _readyCallbacks?.push(callback);
  }
  piko2.ready = ready;
  function _markReady() {
    _isReady = true;
    const callbacks = _readyCallbacks;
    _readyCallbacks = null;
    if (callbacks) {
      for (const callback of callbacks) {
        callback();
      }
    }
  }
  piko2._markReady = _markReady;
  ((actions2) => {
    function dispatch2(actionName, element, domEvent) {
      PPFramework.dispatchAction(actionName, element, domEvent);
    }
    actions2.dispatch = dispatch2;
  })(piko2.actions || (piko2.actions = {}));
  ((helpers2) => {
    function execute(domEvent, actionString, element) {
      PPFramework.executeHelper(domEvent, actionString, element);
    }
    helpers2.execute = execute;
  })(piko2.helpers || (piko2.helpers = {}));
  ((assets2) => {
    function resolve(src, moduleName) {
      return PPFramework.assetSrc(src, moduleName);
    }
    assets2.resolve = resolve;
  })(piko2.assets || (piko2.assets = {}));
})(piko || (piko = {}));
if (typeof window !== "undefined") {
  window.piko = piko;
}
const pageCleanups = [];
const elementCleanups = /* @__PURE__ */ new WeakMap();
const partialLifecycleState = /* @__PURE__ */ new WeakMap();
const connectedPartials = /* @__PURE__ */ new WeakSet();
function _getOrCreateState(scope) {
  let state = partialLifecycleState.get(scope);
  if (!state) {
    state = {
      callbacks: {
        onConnected: [],
        onDisconnected: [],
        onBeforeRender: [],
        onAfterRender: [],
        onUpdated: []
      },
      connectedOnce: false,
      cleanups: []
    };
    partialLifecycleState.set(scope, state);
  }
  return state;
}
function _registerLifecycle(scope, callbacks) {
  const state = _getOrCreateState(scope);
  if (callbacks.onConnected) {
    state.callbacks.onConnected.push(callbacks.onConnected);
  }
  if (callbacks.onDisconnected) {
    state.callbacks.onDisconnected.push(callbacks.onDisconnected);
  }
  if (callbacks.onBeforeRender) {
    state.callbacks.onBeforeRender.push(callbacks.onBeforeRender);
  }
  if (callbacks.onAfterRender) {
    state.callbacks.onAfterRender.push(callbacks.onAfterRender);
  }
  if (callbacks.onUpdated) {
    state.callbacks.onUpdated.push(callbacks.onUpdated);
  }
  if (scope.isConnected && !state.connectedOnce) {
    _executeConnected(scope);
  }
}
function _addLifecycleCallback(scope, hookName, callback) {
  const state = _getOrCreateState(scope);
  state.callbacks[hookName].push(callback);
  if (hookName === "onConnected" && scope.isConnected && !state.connectedOnce) {
    _executeConnected(scope);
  }
}
function _executeConnected(scope) {
  const state = partialLifecycleState.get(scope);
  if (!state || state.connectedOnce) {
    return;
  }
  state.connectedOnce = true;
  connectedPartials.add(scope);
  for (const callback of state.callbacks.onConnected) {
    try {
      callback();
    } catch (error) {
      console.error("[pk] Error in onConnected:", error);
    }
  }
}
function _executeDisconnected(scope) {
  const state = partialLifecycleState.get(scope);
  if (!state) {
    return;
  }
  for (const callback of state.callbacks.onDisconnected) {
    try {
      callback();
    } catch (error) {
      console.error("[pk] Error in onDisconnected:", error);
    }
  }
  state.connectedOnce = false;
  connectedPartials.delete(scope);
  for (const cleanup of state.cleanups) {
    try {
      cleanup();
    } catch (error) {
      console.error("[pk] Error in partial cleanup:", error);
    }
  }
  state.cleanups.length = 0;
}
function _executeBeforeRender(scope) {
  const state = partialLifecycleState.get(scope);
  if (!state || state.callbacks.onBeforeRender.length === 0) {
    return;
  }
  for (const callback of state.callbacks.onBeforeRender) {
    try {
      callback();
    } catch (error) {
      console.error("[pk] Error in onBeforeRender:", error);
    }
  }
}
function _executeAfterRender(scope) {
  const state = partialLifecycleState.get(scope);
  if (!state || state.callbacks.onAfterRender.length === 0) {
    return;
  }
  for (const callback of state.callbacks.onAfterRender) {
    try {
      callback();
    } catch (error) {
      console.error("[pk] Error in onAfterRender:", error);
    }
  }
}
function _executeUpdated(scope, context) {
  const state = partialLifecycleState.get(scope);
  if (!state || state.callbacks.onUpdated.length === 0) {
    return;
  }
  for (const callback of state.callbacks.onUpdated) {
    try {
      callback(context);
    } catch (error) {
      console.error("[pk] Error in onUpdated:", error);
    }
  }
}
function _executeConnectedForPartials(container) {
  const partials = container.querySelectorAll("[partial]");
  for (const partial2 of partials) {
    if (partialLifecycleState.has(partial2) && !connectedPartials.has(partial2)) {
      _executeConnected(partial2);
    }
  }
}
function onCleanup(cleanupFunction, scope) {
  if (scope) {
    let scopeCleanups = elementCleanups.get(scope);
    if (!scopeCleanups) {
      scopeCleanups = [];
      elementCleanups.set(scope, scopeCleanups);
    }
    scopeCleanups.push(cleanupFunction);
  } else {
    pageCleanups.push(cleanupFunction);
  }
}
function _runPageCleanup() {
  for (const cleanupFunction of pageCleanups) {
    try {
      cleanupFunction();
    } catch (error) {
      console.error("[pk] Error in page cleanup:", error);
    }
  }
  pageCleanups.length = 0;
}
let cleanupObserver = null;
function _initCleanupObserver() {
  if (cleanupObserver) {
    return;
  }
  cleanupObserver = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      for (const node of mutation.removedNodes) {
        if (typeof Element !== "undefined" && node instanceof Element) {
          runElementCleanups(node);
        }
      }
    }
  });
  cleanupObserver.observe(document.body, {
    childList: true,
    subtree: true
  });
}
function runElementCleanups(element) {
  executeDisconnectedForRemovedPartials(element);
  const cleanups = elementCleanups.get(element);
  if (cleanups) {
    for (const cleanupFunction of cleanups) {
      try {
        cleanupFunction();
      } catch (error) {
        console.error("[pk] Error in element cleanup:", error);
      }
    }
    elementCleanups.delete(element);
  }
  for (const child of element.querySelectorAll("*")) {
    const childCleanups = elementCleanups.get(child);
    if (!childCleanups) {
      continue;
    }
    for (const cleanupFunction of childCleanups) {
      try {
        cleanupFunction();
      } catch (error) {
        console.error("[pk] Error in element cleanup:", error);
      }
    }
    elementCleanups.delete(child);
  }
}
function executeDisconnectedForRemovedPartials(element) {
  if (element.hasAttribute("partial")) {
    _executeDisconnected(element);
  }
  const partials = element.querySelectorAll("[partial]");
  for (const partial2 of partials) {
    _executeDisconnected(partial2);
  }
}
function getCallbackNames(callbacks) {
  const names = [];
  if (callbacks.onConnected.length > 0) {
    names.push("onConnected");
  }
  if (callbacks.onDisconnected.length > 0) {
    names.push("onDisconnected");
  }
  if (callbacks.onBeforeRender.length > 0) {
    names.push("onBeforeRender");
  }
  if (callbacks.onAfterRender.length > 0) {
    names.push("onAfterRender");
  }
  if (callbacks.onUpdated.length > 0) {
    names.push("onUpdated");
  }
  return names;
}
function createDebugAPI() {
  return {
    getPartialInfo(element) {
      const state = partialLifecycleState.get(element);
      const cleanups = elementCleanups.get(element);
      return {
        exists: state !== void 0,
        partialName: element.getAttribute("partial_name") ?? element.getAttribute("data-partial-name"),
        partialId: element.getAttribute("partial") ?? element.getAttribute("data-partial"),
        isConnected: connectedPartials.has(element),
        connectedOnce: state?.connectedOnce ?? false,
        registeredCallbacks: state ? getCallbackNames(state.callbacks) : [],
        cleanupCount: (state?.cleanups.length ?? 0) + (cleanups?.length ?? 0)
      };
    },
    isConnected(element) {
      return connectedPartials.has(element);
    },
    getCleanupCount(element) {
      const state = partialLifecycleState.get(element);
      const cleanups = elementCleanups.get(element);
      return (state?.cleanups.length ?? 0) + (cleanups?.length ?? 0);
    },
    getRegisteredCallbacks(element) {
      const state = partialLifecycleState.get(element);
      return state ? getCallbackNames(state.callbacks) : [];
    },
    getAllConnectedPartials() {
      const partials = document.querySelectorAll("[partial], [data-partial]");
      return Array.from(partials).filter((el) => connectedPartials.has(el));
    },
    isAvailable() {
      return true;
    }
  };
}
if (typeof window !== "undefined") {
  window.__pikoDebug = createDebugAPI();
}
function createRefs(scope = document.body) {
  const partialId = scope.getAttribute("partial") ?? scope.closest("[partial]")?.getAttribute("partial");
  return new Proxy({}, {
    get(_, name) {
      if (typeof name !== "string" || name === "then") {
        return void 0;
      }
      let el;
      if (partialId) {
        el = document.querySelector(`[partial~="${partialId}"][p-ref="${name}"]`);
      }
      el ??= scope.querySelector(`[p-ref="${name}"]`);
      if (!el) {
        console.warn(`[pk] ref "${name}" not found in scope`);
      }
      return el;
    }
  });
}
createRefs();
function _createPKContext(scope) {
  return {
    refs: createRefs(scope),
    createRefs: (s) => createRefs(s ?? scope),
    onConnected: (callback) => _addLifecycleCallback(scope, "onConnected", callback),
    onDisconnected: (callback) => _addLifecycleCallback(scope, "onDisconnected", callback),
    onBeforeRender: (callback) => _addLifecycleCallback(scope, "onBeforeRender", callback),
    onAfterRender: (callback) => _addLifecycleCallback(scope, "onAfterRender", callback),
    onUpdated: (callback) => _addLifecycleCallback(scope, "onUpdated", callback),
    onCleanup: (cleanupFunction) => onCleanup(cleanupFunction, scope)
  };
}
const BLOCK_DELIMITER = "\n\n";
const DEFAULT_EVENT_TYPE = "message";
const COMPLETE_EVENT_TYPE = "complete";
const ERROR_EVENT_TYPE = "error";
const EVENT_LINE_PREFIX = "event: ";
const DATA_LINE_PREFIX = "data: ";
const ID_LINE_PREFIX = "id: ";
function parseSSEBlock(block) {
  const trimmed = block.trim();
  if (!trimmed || trimmed.startsWith(":")) {
    return null;
  }
  let eventType = "";
  let rawData = "";
  let id;
  for (const line of trimmed.split("\n")) {
    if (line.startsWith(EVENT_LINE_PREFIX)) {
      eventType = line.substring(EVENT_LINE_PREFIX.length);
    } else if (line.startsWith(DATA_LINE_PREFIX)) {
      rawData = line.substring(DATA_LINE_PREFIX.length);
    } else if (line.startsWith(ID_LINE_PREFIX)) {
      id = line.substring(ID_LINE_PREFIX.length);
    }
  }
  if (!eventType && !rawData) {
    return null;
  }
  if (!eventType) {
    eventType = DEFAULT_EVENT_TYPE;
  }
  let data = rawData;
  try {
    data = JSON.parse(rawData);
  } catch {
  }
  return { eventType, data, id };
}
function processSSEBlock(block, callbacks) {
  const parsed = parseSSEBlock(block);
  if (!parsed) {
    return null;
  }
  if (parsed.eventType === COMPLETE_EVENT_TYPE) {
    return { completeData: parsed.data };
  }
  if (parsed.eventType === ERROR_EVENT_TYPE) {
    const errorData = parsed.data;
    const message = typeof errorData?.message === "string" ? errorData.message : "SSE stream error";
    throw createActionError(0, message, void 0, parsed.data);
  }
  callbacks.onEvent(parsed.data, parsed.eventType);
  if (parsed.id && callbacks.onEventId) {
    callbacks.onEventId(parsed.id);
  }
  return null;
}
async function consumeSSEStream(reader, callbacks) {
  const decoder = new TextDecoder();
  let buffer = "";
  let completeData = void 0;
  let receivedComplete = false;
  for (; ; ) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split(BLOCK_DELIMITER);
    buffer = parts.pop() ?? "";
    for (const part of parts) {
      const result = processSSEBlock(part, callbacks);
      if (!result) {
        continue;
      }
      completeData = result.completeData;
      receivedComplete = true;
    }
  }
  return { completeData, receivedComplete };
}
async function readSSEStream(body, callbacks, _signal) {
  const reader = body.getReader();
  try {
    const { completeData, receivedComplete } = await consumeSSEStream(reader, callbacks);
    if (!receivedComplete) {
      throw createActionError(0, "SSE stream ended without completion");
    }
    return completeData;
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw createActionError(0, "Request cancelled");
    }
    if (error !== null && typeof error === "object" && "status" in error) {
      throw error;
    }
    const message = error instanceof Error ? error.message : "SSE connection lost";
    throw createActionError(0, message);
  } finally {
    try {
      reader.releaseLock();
    } catch {
    }
  }
}
const CSRF_TOKEN_META_NAME = "csrf-token";
const CSRF_EPHEMERAL_META_NAME = "csrf-ephemeral";
function getCSRFTokenFromMeta() {
  return document.querySelector(`meta[name="${CSRF_TOKEN_META_NAME}"]`)?.content ?? null;
}
function getCSRFEphemeralFromMeta() {
  return document.querySelector(`meta[name="${CSRF_EPHEMERAL_META_NAME}"]`)?.content ?? null;
}
const HTTP_STATUS_UNPROCESSABLE$1 = 422;
const HTTP_STATUS_FORBIDDEN$1 = 403;
const CSRF_ERROR_EXPIRED = "csrf_expired";
const CSRF_ERROR_INVALID = "csrf_invalid";
const DEFAULT_RETRY_BASE_DELAY = 1e3;
const MAX_RETRY_DELAY = 3e4;
const DEFAULT_SSE_RECONNECT_DELAY = 3e3;
const MAX_SSE_RECONNECT_DELAY = 3e4;
const HTTP_STATUS_TIMEOUT = 408;
const HTTP_STATUS_SERVER_ERROR = 500;
const HTTP_STATUS_OK = 200;
const RANDOM_STRING_RADIX = 36;
const RANDOM_STRING_SLICE_START = 2;
const RANDOM_STRING_SLICE_END = 9;
const debounceTimers = /* @__PURE__ */ new Map();
function getCSRFTokens(element) {
  const actionToken = element?.getAttribute("data-csrf-action-token") ?? getCSRFTokenFromMeta();
  const ephemeralToken = element?.getAttribute("data-csrf-ephemeral-token") ?? getCSRFEphemeralFromMeta();
  return { actionToken, ephemeralToken };
}
function isCSRFError(status, responseData) {
  return status === HTTP_STATUS_FORBIDDEN$1 && (responseData.error === CSRF_ERROR_EXPIRED || responseData.error === CSRF_ERROR_INVALID);
}
function attemptCSRFRecovery(responseData, element, retryAction) {
  if (responseData.error === CSRF_ERROR_INVALID) {
    window.location.reload();
    return true;
  }
  const partial2 = element.closest("[partial_src]");
  if (partial2) {
    partial2.dispatchEvent(new CustomEvent("refresh-partial", {
      bubbles: false,
      detail: {
        afterMorph: () => {
          const refreshedEl = partial2.querySelector("[data-csrf-action-token]");
          if (refreshedEl) {
            retryAction();
          } else {
            console.warn("[ActionExecutor] Could not find element with CSRF token after partial refresh");
          }
        }
      }
    }));
    return true;
  }
  window.location.reload();
  return true;
}
function validateForm(element, event) {
  const form = element.closest("form");
  if (!form) {
    return true;
  }
  if (form.noValidate) {
    return true;
  }
  const submitter = event?.submitter;
  if (submitter?.formNoValidate) {
    return true;
  }
  return form.reportValidity();
}
function clearPreviousErrors(form) {
  form.querySelectorAll("[error]").forEach((el) => {
    el.removeAttribute("error");
  });
}
function applyServerErrors(form, errors) {
  clearPreviousErrors(form);
  for (const [fieldName, messages] of Object.entries(errors)) {
    const errorMessage = messages.join(", ");
    const fields = form.querySelectorAll(`[name="${fieldName}"]`);
    if (fields.length > 0) {
      fields.forEach((field) => {
        field.setAttribute("error", errorMessage);
      });
    }
  }
}
function showLoading(target, element) {
  if (target === true) {
    applyLoadingIndicator(element);
  } else if (typeof target === "string") {
    const el = document.querySelector(target);
    if (el) {
      applyLoadingIndicator(el);
    }
  } else if (target instanceof HTMLElement) {
    applyLoadingIndicator(target);
  }
}
function hideLoading(target, element) {
  if (target === true) {
    removeLoadingIndicator(element);
  } else if (typeof target === "string") {
    const el = document.querySelector(target);
    if (el) {
      removeLoadingIndicator(el);
    }
  } else if (target instanceof HTMLElement) {
    removeLoadingIndicator(target);
  }
}
function calculateRetryDelay(attempt, config) {
  const backoff = config.backoff ?? "exponential";
  if (backoff === "linear") {
    return Math.min(DEFAULT_RETRY_BASE_DELAY * (attempt + 1), MAX_RETRY_DELAY);
  }
  return Math.min(DEFAULT_RETRY_BASE_DELAY * Math.pow(2, attempt), MAX_RETRY_DELAY);
}
function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
async function executeWithRetry(actionName, args, method, actionToken, ephemeralToken, retryConfig, options) {
  const maxAttempts = retryConfig?.attempts ?? 1;
  let lastError = null;
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      return await executeServerAction(actionName, args, method, actionToken, ephemeralToken, options);
    } catch (error) {
      if (error instanceof Error && !("status" in error)) {
        lastError = createActionError(0, error.message);
      } else {
        lastError = error;
      }
      const isTimeout = lastError.status === HTTP_STATUS_TIMEOUT;
      const isCancelled = lastError.status === 0 && lastError.message === "Request cancelled";
      const isRetryable = (lastError.status === 0 || lastError.status >= HTTP_STATUS_SERVER_ERROR || isTimeout) && !isCancelled;
      if (!isRetryable || attempt >= maxAttempts - 1) {
        throw lastError;
      }
      const retryDelay = calculateRetryDelay(attempt, retryConfig ?? {});
      await delay(retryDelay);
    }
  }
  throw lastError;
}
function buildActionBody(args, ephemeralToken) {
  const headers = {};
  const bodyData = {};
  if (args.length > 0) {
    if (args.length === 1 && typeof args[0] === "object" && args[0] !== null) {
      Object.assign(bodyData, args[0]);
    } else {
      bodyData["args"] = args.map((v, i) => ({ [i]: v })).reduce((acc, b) => ({ ...acc, ...b }), {});
    }
  }
  if (ephemeralToken) {
    bodyData["_csrf_ephemeral_token"] = ephemeralToken;
  }
  const hasFiles = Object.values(bodyData).some(
    (v) => v instanceof File || v instanceof Blob
  );
  if (hasFiles) {
    const formData2 = new FormData();
    for (const [key, value] of Object.entries(bodyData)) {
      if (value instanceof File) {
        formData2.append(key, value, value.name);
      } else if (value instanceof Blob) {
        formData2.append(key, value);
      } else if (value !== void 0 && value !== null) {
        formData2.append(key, String(value));
      }
    }
    return { body: formData2, headers };
  }
  headers["Content-Type"] = "application/json";
  return { body: JSON.stringify(bodyData), headers };
}
function createRequestAbortController(options) {
  const controller = new AbortController();
  let timeoutId;
  if (options?.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort());
    }
  }
  if (options?.timeout && options.timeout > 0) {
    timeoutId = setTimeout(() => controller.abort(), options.timeout);
  }
  return { controller, timeoutId };
}
async function parseActionResponse(response) {
  let responseData;
  try {
    responseData = await response.json();
  } catch {
    throw createActionError(
      response.status,
      "Failed to parse server response",
      void 0,
      void 0
    );
  }
  if (!response.ok) {
    const validationErrors = response.status === HTTP_STATUS_UNPROCESSABLE$1 ? responseData.errors : void 0;
    throw createActionError(
      response.status,
      responseData.message ?? responseData.error ?? `Action failed with status ${response.status}`,
      validationErrors,
      responseData.error ?? responseData.data,
      responseData._helpers
    );
  }
  return responseData;
}
async function executeServerAction(actionName, args, method, actionToken, ephemeralToken, options) {
  const { body, headers } = buildActionBody(args, ephemeralToken);
  if (actionToken) {
    headers["X-CSRF-Action-Token"] = actionToken;
  }
  const { controller, timeoutId } = createRequestAbortController(options);
  try {
    const response = await fetch(`/_piko/actions/${actionName}`, {
      method,
      headers,
      credentials: "same-origin",
      body,
      signal: controller.signal
    });
    return await parseActionResponse(response);
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      const isTimeout = options?.timeout && !options.signal?.aborted;
      throw createActionError(
        isTimeout ? HTTP_STATUS_TIMEOUT : 0,
        isTimeout ? "Request timeout" : "Request cancelled",
        void 0,
        void 0
      );
    }
    throw error;
  } finally {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
  }
}
function getDebounceKey(actionName, element) {
  element.dataset.ppActionId ??= `action-${Date.now()}-${Math.random().toString(RANDOM_STRING_RADIX).slice(RANDOM_STRING_SLICE_START, RANDOM_STRING_SLICE_END)}`;
  return `${actionName}:${element.dataset.ppActionId}`;
}
function clearDebounce(key) {
  const timer = debounceTimers.get(key);
  if (timer) {
    clearTimeout(timer);
    debounceTimers.delete(key);
  }
}
async function executeHelpers(helpers, element, event) {
  const helperRegistry = getGlobalHelperRegistry();
  const errors = [];
  for (const helper of helpers) {
    try {
      const args = (helper.args ?? []).map((a) => String(a));
      await helperRegistry.execute(helper.name, element, event, args);
    } catch (error) {
      console.error(`[ActionExecutor] Helper "${helper.name}" failed:`, error);
      errors.push(error);
    }
  }
  if (errors.length > 0) {
    throw new AggregateError(errors, `${errors.length} helper(s) failed`);
  }
}
async function invokeSuccessCallback(descriptor, response, element, event) {
  if (!descriptor.onSuccess) {
    return;
  }
  try {
    const data = response.data ?? response;
    const next = descriptor.onSuccess(data);
    if (isActionDescriptor(next)) {
      await handleAction(next, element, event);
    }
  } catch (error) {
    console.error("[ActionExecutor] onSuccess callback failed:", error);
    throw error;
  }
}
function validateAndEmitStart(descriptor, element, form, actionStartTime, event) {
  if (form) {
    clearPreviousErrors(form);
    if (!validateForm(element, event)) {
      return false;
    }
  }
  return true;
}
async function executeServerRequest(descriptor, element) {
  const { actionToken, ephemeralToken } = getCSRFTokens(element);
  if (descriptor.onProgress) {
    const sseOptions = { timeout: descriptor.timeout, signal: descriptor.signal };
    let data;
    if (descriptor.retryStream) {
      data = await executeServerActionSSEWithRetry({
        actionName: descriptor.action,
        args: descriptor.args ?? [],
        method: descriptor.method ?? "POST",
        actionToken,
        ephemeralToken,
        onProgress: descriptor.onProgress,
        retryConfig: descriptor.retryStream,
        options: sseOptions
      });
    } else {
      data = await executeServerActionSSE({
        actionName: descriptor.action,
        args: descriptor.args ?? [],
        method: descriptor.method ?? "POST",
        actionToken,
        ephemeralToken,
        onProgress: descriptor.onProgress,
        options: sseOptions
      });
    }
    return { data, status: HTTP_STATUS_OK };
  }
  return await executeWithRetry(
    descriptor.action,
    descriptor.args ?? [],
    descriptor.method ?? "POST",
    actionToken,
    ephemeralToken,
    descriptor.retry,
    { timeout: descriptor.timeout, signal: descriptor.signal }
  );
}
async function handleActionSuccess(descriptor, response, element, event, form, actionStartTime) {
  if (!descriptor.shouldSuppressHelpers && response._helpers && response._helpers.length > 0) {
    await executeHelpers(response._helpers, element, event);
  }
  await invokeSuccessCallback(descriptor, response, element, event);
}
function invokeErrorHandlers(actionError, descriptor, rawError) {
  if (descriptor.onError) {
    try {
      descriptor.onError(actionError);
    } catch (callbackError) {
      console.error("[ActionExecutor] onError callback failed:", callbackError);
    }
  } else {
    console.error("[ActionExecutor] Action failed:", rawError);
  }
}
async function handleActionFailure(descriptor, error, element, event, form, actionStartTime) {
  const actionError = error;
  const errorCode = typeof actionError.data === "string" ? actionError.data : void 0;
  if (isCSRFError(actionError.status, { error: errorCode })) {
    const recovered = attemptCSRFRecovery(
      { error: errorCode },
      element,
      () => {
        void executeAction(descriptor, element, event);
      }
    );
    if (recovered) {
      return true;
    }
  }
  if (actionError._helpers && actionError._helpers.length > 0) {
    await executeHelpers(actionError._helpers, element, event);
  }
  if (actionError.status === HTTP_STATUS_UNPROCESSABLE$1 && actionError.validationErrors && form) {
    applyServerErrors(form, actionError.validationErrors);
  }
  invokeErrorHandlers(actionError, descriptor, error);
  return false;
}
async function executeAction(descriptor, element, event) {
  const actionStartTime = Date.now();
  const form = element.closest("form");
  if (!validateAndEmitStart(descriptor, element, form, actionStartTime, event)) {
    return;
  }
  if (descriptor.optimistic) {
    try {
      descriptor.optimistic();
    } catch (error) {
      console.error("[ActionExecutor] Optimistic update failed:", error);
    }
  }
  if (descriptor.loading !== void 0) {
    showLoading(descriptor.loading, element);
  }
  try {
    const response = await executeServerRequest(descriptor, element);
    await handleActionSuccess(descriptor, response, element, event, form, actionStartTime);
  } catch (error) {
    const recovered = await handleActionFailure(descriptor, error, element, event, form);
    if (recovered) {
      return;
    }
    throw error;
  } finally {
    if (descriptor.onComplete) {
      try {
        descriptor.onComplete();
      } catch (error) {
        console.error("[ActionExecutor] onComplete callback failed:", error);
      }
    }
    if (descriptor.loading !== void 0) {
      hideLoading(descriptor.loading, element);
    }
  }
}
async function handleAction(descriptor, element, event) {
  if (descriptor.debounce && descriptor.debounce > 0) {
    const debounceKey = getDebounceKey(descriptor.action, element);
    clearDebounce(debounceKey);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        debounceTimers.delete(debounceKey);
        executeAction(descriptor, element, event).then(resolve).catch((err) => {
          console.error("[ActionExecutor] Debounced action failed:", err);
          reject(err instanceof Error ? err : new Error(String(err)));
        });
      }, descriptor.debounce);
      debounceTimers.set(debounceKey, timer);
    });
  }
  return executeAction(descriptor, element, event);
}
const SSE_CONTENT_TYPE = "text/event-stream";
const SSE_ACCEPT_HEADER = "text/event-stream";
function calculateSSEReconnectDelay(attempt, config) {
  const baseDelay = config.baseDelay ?? DEFAULT_SSE_RECONNECT_DELAY;
  const maxDelay = config.maxDelay ?? MAX_SSE_RECONNECT_DELAY;
  const backoff = config.backoff ?? "linear";
  if (backoff === "linear") {
    return Math.min(baseDelay * (attempt + 1), maxDelay);
  }
  return Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
}
function buildSSEHeaders(actionToken, ephemeralToken, lastEventId) {
  const headers = {
    "Content-Type": "application/json",
    "Accept": SSE_ACCEPT_HEADER
  };
  if (actionToken && ephemeralToken) {
    headers["X-CSRF-Action-Token"] = actionToken;
  }
  if (lastEventId) {
    headers["Last-Event-ID"] = lastEventId;
  }
  return headers;
}
function buildSSEUrl(actionName, actionToken, ephemeralToken) {
  let url = `/_piko/actions/${actionName}`;
  if (actionToken && ephemeralToken) {
    url += `?_csrf_ephemeral_token=${encodeURIComponent(ephemeralToken)}`;
  }
  return url;
}
function buildSSEBody(args) {
  const bodyData = {};
  if (args.length > 0) {
    if (args.length === 1 && typeof args[0] === "object" && args[0] !== null) {
      Object.assign(bodyData, args[0]);
    } else {
      bodyData["args"] = args.map((v, i) => ({ [i]: v })).reduce((acc, b) => ({ ...acc, ...b }), {});
    }
  }
  return bodyData;
}
function setupAbortControl(options) {
  const controller = new AbortController();
  let timeoutId;
  if (options?.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort());
    }
  }
  if (options?.timeout && options.timeout > 0) {
    timeoutId = setTimeout(() => controller.abort(), options.timeout);
  }
  return { controller, timeoutId };
}
async function throwSSEErrorResponse(response) {
  let responseData;
  try {
    responseData = await response.json();
  } catch {
    throw createActionError(response.status, `Action failed with status ${response.status}`);
  }
  const validationErrors = response.status === HTTP_STATUS_UNPROCESSABLE$1 ? responseData.errors : void 0;
  throw createActionError(
    response.status,
    responseData.message ?? responseData.error ?? `Action failed with status ${response.status}`,
    validationErrors,
    responseData.error ?? responseData.data,
    responseData._helpers
  );
}
function rethrowAsActionError(error, options) {
  if (error instanceof DOMException && error.name === "AbortError") {
    const isTimeout = options?.timeout && !options.signal?.aborted;
    throw createActionError(
      isTimeout ? HTTP_STATUS_TIMEOUT : 0,
      isTimeout ? "Request timeout" : "Request cancelled"
    );
  }
  throw error;
}
async function executeServerActionSSEInternal(params) {
  const { actionName, args, method, actionToken, ephemeralToken, onProgress, options, lastEventId, onEventId } = params;
  const headers = buildSSEHeaders(actionToken, ephemeralToken, lastEventId);
  const url = buildSSEUrl(actionName, actionToken, ephemeralToken);
  const bodyData = buildSSEBody(args);
  const { controller, timeoutId } = setupAbortControl(options);
  try {
    const response = await fetch(url, {
      method,
      headers,
      credentials: "same-origin",
      body: JSON.stringify(bodyData),
      signal: controller.signal
    });
    if (!response.ok) {
      await throwSSEErrorResponse(response);
    }
    const contentType = response.headers.get("Content-Type") ?? "";
    if (contentType.startsWith(SSE_CONTENT_TYPE) && response.body) {
      return await readSSEStream(response.body, { onEvent: onProgress, onEventId }, controller.signal);
    }
    let responseData;
    try {
      responseData = await response.json();
    } catch {
      throw createActionError(response.status, "Failed to parse server response");
    }
    return responseData;
  } catch (error) {
    return rethrowAsActionError(error, options);
  } finally {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
  }
}
async function executeServerActionSSE(params) {
  return executeServerActionSSEInternal(params);
}
async function executeServerActionSSEWithRetry(params) {
  const { actionName, args, method, actionToken, ephemeralToken, onProgress, retryConfig, options } = params;
  let reconnectCount = 0;
  let lastEventId;
  const maxReconnects = retryConfig.maxReconnects;
  for (; ; ) {
    try {
      let isReconnection = reconnectCount > 0;
      const wrappedOnProgress = (data, eventType) => {
        if (isReconnection) {
          reconnectCount = 0;
          isReconnection = false;
        }
        onProgress(data, eventType);
      };
      const result = await executeServerActionSSEInternal({
        actionName,
        args,
        method,
        actionToken,
        ephemeralToken,
        onProgress: wrappedOnProgress,
        options,
        lastEventId,
        onEventId: (id) => {
          lastEventId = id;
        }
      });
      return result;
    } catch (error) {
      const actionError = error;
      if (actionError.message === "Request cancelled") {
        throw error;
      }
      if (actionError.data !== void 0) {
        throw error;
      }
      if (actionError.status !== 0) {
        throw error;
      }
      if (reconnectCount >= maxReconnects) {
        throw createActionError(
          0,
          `SSE stream failed after ${reconnectCount} reconnection attempts`
        );
      }
      retryConfig.onDisconnect?.();
      const reconnectDelay = calculateSSEReconnectDelay(reconnectCount, retryConfig);
      await delay(reconnectDelay);
      if (options?.signal?.aborted) {
        throw createActionError(0, "Request cancelled");
      }
      reconnectCount++;
      retryConfig.onReconnect?.(reconnectCount);
    }
  }
}
async function callServerActionDirect(actionName, args, method = "POST", options) {
  const { actionToken, ephemeralToken } = getCSRFTokens();
  if (options?.onProgress) {
    let data;
    const sseOptions = { timeout: options.timeout, signal: options.signal };
    if (options.retryStream) {
      data = await executeServerActionSSEWithRetry({
        actionName,
        args,
        method,
        actionToken,
        ephemeralToken,
        onProgress: options.onProgress,
        retryConfig: options.retryStream,
        options: sseOptions
      });
    } else {
      data = await executeServerActionSSE({
        actionName,
        args,
        method,
        actionToken,
        ephemeralToken,
        onProgress: options.onProgress,
        options: sseOptions
      });
    }
    return {
      data,
      status: HTTP_STATUS_OK,
      message: void 0,
      helpers: void 0
    };
  }
  const response = await executeServerAction(
    actionName,
    args,
    method,
    actionToken,
    ephemeralToken,
    options
  );
  const helpers = response._helpers;
  if (!options?.suppressHelpers && helpers && helpers.length > 0) {
    await executeHelpers(helpers, document.body);
  }
  return {
    data: response.data ?? response,
    status: response.status ?? HTTP_STATUS_OK,
    message: response.message,
    helpers
  };
}
const HTTP_STATUS_UNPROCESSABLE = 422;
const HTTP_STATUS_UNAUTHORIZED = 401;
const HTTP_STATUS_FORBIDDEN = 403;
function createActionError(status, message, validationErrors, data, helpers) {
  return {
    status,
    message,
    validationErrors,
    data,
    _helpers: helpers,
    get isNetworkError() {
      return this.status === 0;
    },
    get isValidationError() {
      return this.status === HTTP_STATUS_UNPROCESSABLE && this.validationErrors !== void 0;
    },
    get isAuthError() {
      return this.status === HTTP_STATUS_UNAUTHORIZED || this.status === HTTP_STATUS_FORBIDDEN;
    }
  };
}
function isActionDescriptor(value) {
  return value !== null && typeof value === "object" && typeof value.action === "string";
}
class ActionBuilder {
  /**
   * Creates a new ActionBuilder.
   *
   * @param actionName - Server action name.
   * @param args - Arguments to pass to the action.
   */
  constructor(actionName, args) {
    this._suppressHelpers = false;
    this._action = actionName;
    this._args = args;
  }
  /** Returns the server action name. */
  get action() {
    return this._action;
  }
  /** Returns the arguments to pass to the action. */
  get args() {
    return this._args;
  }
  /** Returns the configured HTTP method. */
  get method() {
    return this._method;
  }
  /** Returns the optimistic update callback. */
  get optimistic() {
    return this._optimistic;
  }
  /** Returns the success callback. */
  get onSuccess() {
    return this._onSuccess;
  }
  /** Returns the error callback. */
  get onError() {
    return this._onError;
  }
  /** Returns the completion callback. */
  get onComplete() {
    return this._onComplete;
  }
  /** Returns the loading state target. */
  get loading() {
    return this._loading;
  }
  /** Returns the debounce delay in milliseconds. */
  get debounce() {
    return this._debounce;
  }
  /** Returns the retry configuration. */
  get retry() {
    return this._retry;
  }
  /** Returns the timeout in milliseconds. */
  get timeout() {
    return this._timeout;
  }
  /** Returns the external abort signal. */
  get signal() {
    return this._signal;
  }
  /** Returns whether automatic helper execution is suppressed. */
  get shouldSuppressHelpers() {
    return this._suppressHelpers;
  }
  /** Returns the SSE progress callback. */
  get onProgress() {
    return this._onProgress;
  }
  /** Returns the SSE stream retry configuration. */
  get retryStream() {
    return this._retryStream;
  }
  /**
   * Sets the HTTP method.
   *
   * @param method - The HTTP method to use.
   * @returns This builder for chaining.
   */
  setMethod(method) {
    this._method = method;
    return this;
  }
  /**
   * Sets the optimistic update callback.
   *
   * Runs immediately before the server request.
   *
   * @param optimisticUpdate - Callback to run before the request.
   * @returns This builder for chaining.
   */
  setOptimistic(optimisticUpdate) {
    this._optimistic = optimisticUpdate;
    return this;
  }
  /**
   * Sets the success callback.
   *
   * Can return another ActionDescriptor to chain actions.
   *
   * @param successHandler - Callback receiving the typed response.
   * @returns This builder for chaining.
   */
  setOnSuccess(successHandler) {
    this._onSuccess = successHandler;
    return this;
  }
  /**
   * Sets the error callback.
   *
   * Use this to roll back optimistic updates.
   *
   * @param errorHandler - Callback receiving the ActionError.
   * @returns This builder for chaining.
   */
  setOnError(errorHandler) {
    this._onError = errorHandler;
    return this;
  }
  /**
   * Sets the completion callback.
   *
   * Runs after success or error, always.
   *
   * @param completeHandler - Callback to run on completion.
   * @returns This builder for chaining.
   */
  setOnComplete(completeHandler) {
    this._onComplete = completeHandler;
    return this;
  }
  /**
   * Sets the loading state target.
   *
   * - `true`: Apply to trigger element.
   * - `string`: CSS selector.
   * - `HTMLElement`: Specific element.
   *
   * @param target - The loading state target.
   * @returns This builder for chaining.
   */
  setLoading(target) {
    this._loading = target;
    return this;
  }
  /**
   * Sets the debounce delay.
   *
   * @param ms - Delay in milliseconds.
   * @returns This builder for chaining.
   */
  setDebounce(ms) {
    this._debounce = ms;
    return this;
  }
  /**
   * Sets the retry configuration.
   *
   * @param attempts - Number of retry attempts.
   * @param backoff - Optional backoff strategy.
   * @returns This builder for chaining.
   */
  setRetry(attempts, backoff) {
    this._retry = { attempts, backoff };
    return this;
  }
  /**
   * Sets the timeout in milliseconds.
   *
   * The request is aborted after this duration.
   *
   * @param ms - Timeout duration.
   * @returns This builder for chaining.
   */
  setTimeout(ms) {
    this._timeout = ms;
    return this;
  }
  /**
   * Sets the external abort signal for cancellation.
   *
   * @param signal - The AbortSignal to use.
   * @returns This builder for chaining.
   */
  setSignal(signal) {
    this._signal = signal;
    return this;
  }
  /**
   * Sets the progress callback for SSE streaming.
   *
   * When set, the action framework automatically uses SSE transport
   * (POST with Accept: text/event-stream). The callback fires for each
   * intermediate event; terminal events route to onSuccess/onError.
   *
   * @param progressHandler - Callback receiving event data and event type.
   * @returns This builder for chaining.
   */
  setOnProgress(progressHandler) {
    this._onProgress = progressHandler;
    return this;
  }
  /**
   * Sets the retry configuration for SSE streams.
   *
   * When set alongside onProgress, the action builder automatically
   * reconnects the SSE stream on connection drops with configurable
   * backoff. Use maxReconnects: Infinity for long-lived streams.
   *
   * @param config - Retry stream configuration.
   * @returns This builder for chaining.
   */
  setRetryStream(config) {
    this._retryStream = config;
    return this;
  }
  /**
   * Sets the HTTP method (alias for setMethod).
   *
   * @param method - The HTTP method to use.
   * @returns This builder for chaining.
   */
  withMethod(method) {
    return this.setMethod(method);
  }
  /**
   * Sets the optimistic update callback (alias for setOptimistic).
   *
   * @param optimisticUpdate - Callback to run before the request.
   * @returns This builder for chaining.
   */
  withOptimistic(optimisticUpdate) {
    return this.setOptimistic(optimisticUpdate);
  }
  /**
   * Sets the success callback (alias for setOnSuccess).
   *
   * @param successHandler - Callback receiving the typed response.
   * @returns This builder for chaining.
   */
  withOnSuccess(successHandler) {
    return this.setOnSuccess(successHandler);
  }
  /**
   * Sets the error callback (alias for setOnError).
   *
   * @param errorHandler - Callback receiving the ActionError.
   * @returns This builder for chaining.
   */
  withOnError(errorHandler) {
    return this.setOnError(errorHandler);
  }
  /**
   * Sets the completion callback (alias for setOnComplete).
   *
   * @param completeHandler - Callback to run on completion.
   * @returns This builder for chaining.
   */
  withOnComplete(completeHandler) {
    return this.setOnComplete(completeHandler);
  }
  /**
   * Sets the loading state target (alias for setLoading).
   *
   * @param target - The loading state target.
   * @returns This builder for chaining.
   */
  withLoading(target) {
    return this.setLoading(target);
  }
  /**
   * Sets the debounce delay (alias for setDebounce).
   *
   * @param ms - Delay in milliseconds.
   * @returns This builder for chaining.
   */
  withDebounce(ms) {
    return this.setDebounce(ms);
  }
  /**
   * Sets the retry configuration (alias for setRetry).
   *
   * @param attempts - Number of retry attempts.
   * @param backoff - Optional backoff strategy.
   * @returns This builder for chaining.
   */
  withRetry(attempts, backoff) {
    return this.setRetry(attempts, backoff);
  }
  /**
   * Sets the timeout (alias for setTimeout).
   *
   * @param ms - Timeout duration in milliseconds.
   * @returns This builder for chaining.
   */
  withTimeout(ms) {
    return this.setTimeout(ms);
  }
  /**
   * Sets the external abort signal (alias for setSignal).
   *
   * @param signal - The AbortSignal to use.
   * @returns This builder for chaining.
   */
  withSignal(signal) {
    return this.setSignal(signal);
  }
  /**
   * Sets the SSE progress callback (alias for setOnProgress).
   *
   * @param progressHandler - Callback receiving event data and event type.
   * @returns This builder for chaining.
   */
  withOnProgress(progressHandler) {
    return this.setOnProgress(progressHandler);
  }
  /**
   * Sets the SSE stream retry configuration (alias for setRetryStream).
   *
   * @param config - Retry stream configuration.
   * @returns This builder for chaining.
   */
  withRetryStream(config) {
    return this.setRetryStream(config);
  }
  /**
   * Suppress automatic execution of server-response helpers.
   *
   * When called, helpers returned by the server (e.g. redirect, resetForm)
   * will NOT be executed automatically. This is useful when you need to
   * process the response programmatically and don't want side effects.
   *
   * The helpers are still available on the DirectCallResponse for manual
   * inspection when using `.call()`.
   *
   * @returns This builder for chaining.
   */
  suppressHelpers() {
    this._suppressHelpers = true;
    return this;
  }
  /**
   * Builds the final ActionDescriptor.
   *
   * Note: The builder itself implements ActionDescriptor, so calling build()
   * is optional. You can use the builder directly.
   *
   * @returns The constructed ActionDescriptor.
   */
  build() {
    return {
      action: this._action,
      args: this._args,
      method: this._method,
      optimistic: this._optimistic,
      onSuccess: this._onSuccess,
      onError: this._onError,
      onComplete: this._onComplete,
      loading: this._loading,
      debounce: this._debounce,
      retry: this._retry,
      timeout: this._timeout,
      signal: this._signal,
      shouldSuppressHelpers: this._suppressHelpers || void 0,
      onProgress: this._onProgress,
      retryStream: this._retryStream
    };
  }
  /**
   * Execute the action directly and return the typed response.
   *
   * This is an imperative alternative to the callback pattern, useful in
   * component scripts where you need the response data directly.
   *
   * Helpers from the server response are processed automatically before
   * the promise resolves.
   *
   * @returns Promise resolving to the typed response data.
   */
  async call() {
    const response = await callServerActionDirect(
      this._action,
      this._args,
      this._method ?? "POST",
      {
        timeout: this._timeout,
        signal: this._signal,
        suppressHelpers: this._suppressHelpers || void 0,
        onProgress: this._onProgress,
        retryStream: this._retryStream
      }
    );
    return response.data;
  }
}
function createActionBuilder(name, args) {
  if (typeof args.toObject === "function") {
    args = args.toObject();
  }
  return new ActionBuilder(name, [args]);
}
const actionFunctionRegistry = /* @__PURE__ */ new Map();
function registerActionFunction(name, actionFactory) {
  actionFunctionRegistry.set(name, actionFactory);
}
function getActionFunction(name) {
  return actionFunctionRegistry.get(name);
}
const LOADER_FADE_MS = 300;
const PROGRESS_MIN = 0;
const PROGRESS_MAX$1 = 100;
function createLoaderUI(options = {}) {
  const {
    colour = "#29e",
    fadeMs = LOADER_FADE_MS,
    container = document.body
  } = options;
  const el = document.createElement("div");
  el.id = "ppf-loader-bar";
  el.style.cssText = [
    "position:fixed",
    "top:0",
    "left:0",
    "width:0%",
    "height:2px",
    `background:${colour}`,
    "transition:width .2s",
    "z-index:9999",
    "pointer-events:none",
    "display:none"
  ].join(";");
  container.appendChild(el);
  return {
    show() {
      el.style.display = "block";
      el.style.width = "0%";
    },
    hide() {
      el.style.width = "100%";
      setTimeout(() => {
        el.style.width = "0%";
        el.style.display = "none";
      }, fadeMs);
    },
    setProgress(percent) {
      let pct = percent;
      if (pct < PROGRESS_MIN) {
        pct = PROGRESS_MIN;
      }
      if (pct > PROGRESS_MAX$1) {
        pct = PROGRESS_MAX$1;
      }
      el.style.display = "block";
      el.style.width = `${pct}%`;
    },
    destroy() {
      el.remove();
    }
  };
}
const ERROR_DISPLAY_MS = 5e3;
function createErrorDisplay(options = {}) {
  const {
    displayMs = ERROR_DISPLAY_MS,
    container = document.body
  } = options;
  let currentElement = null;
  let timeoutId = null;
  return {
    show(message) {
      const existingEl = document.getElementById("ppf-error-message");
      if (existingEl) {
        existingEl.textContent = message;
        return;
      }
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
      const errEl = document.createElement("div");
      errEl.id = "ppf-error-message";
      errEl.style.cssText = [
        "position:fixed",
        "top:60px",
        "left:50%",
        "transform:translateX(-50%)",
        "background:#f44336",
        "color:#fff",
        "padding:6px 12px",
        "border-radius:4px",
        "font-family:sans-serif",
        "z-index:99999"
      ].join(";");
      errEl.textContent = message;
      container.appendChild(errEl);
      currentElement = errEl;
      timeoutId = setTimeout(() => {
        if (errEl.parentNode) {
          errEl.parentNode.removeChild(errEl);
        }
        currentElement = null;
        timeoutId = null;
      }, displayMs);
    },
    clear() {
      if (timeoutId) {
        clearTimeout(timeoutId);
        timeoutId = null;
      }
      if (currentElement?.parentNode) {
        currentElement.parentNode.removeChild(currentElement);
        currentElement = null;
      }
    }
  };
}
function createHelperRegistry() {
  const registry = /* @__PURE__ */ new Map();
  return {
    register(name, helper) {
      if (registry.has(name)) {
        console.warn(`HelperRegistry: Overwriting already registered helper "${name}".`);
      }
      registry.set(name, helper);
    },
    get(name) {
      return registry.get(name);
    },
    async execute(name, element, event, args) {
      const helper = registry.get(name);
      if (helper) {
        await helper(element, event, ...args);
      } else {
        console.warn(`HelperRegistry: Unknown helper "${name}"`);
      }
    },
    has(name) {
      return registry.has(name);
    }
  };
}
function createSpriteSheetManager() {
  return {
    merge(newSheet) {
      if (!newSheet) {
        return;
      }
      const mainSheet = document.getElementById("sprite");
      if (!mainSheet) {
        console.warn("SpriteSheetManager: Main sprite sheet with id='sprite' not found. Cannot merge new sprites.");
        newSheet.id = "sprite";
        newSheet.style.display = "none";
        document.body.appendChild(newSheet);
        return;
      }
      const newSymbols = newSheet.querySelectorAll("symbol");
      newSymbols.forEach((newSymbol) => {
        const symbolId = newSymbol.id;
        if (!symbolId) {
          console.warn("SpriteSheetManager: Found a symbol without an ID, skipping.", newSymbol);
          return;
        }
        const existingSymbol = mainSheet.querySelector(`symbol[id="${symbolId}"]`);
        if (existingSymbol) {
          existingSymbol.replaceWith(newSymbol.cloneNode(true));
        } else {
          mainSheet.appendChild(newSymbol.cloneNode(true));
        }
      });
    },
    ensureExists() {
      if (!document.getElementById("sprite")) {
        const sheet = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        sheet.id = "sprite";
        sheet.style.display = "none";
        document.body.appendChild(sheet);
      }
    }
  };
}
function createModuleLoader() {
  const loadedModuleScripts = /* @__PURE__ */ new Set();
  function loadScript(src) {
    if (loadedModuleScripts.has(src)) {
      return;
    }
    loadedModuleScripts.add(src);
    const newScript = document.createElement("script");
    newScript.type = "module";
    newScript.src = src;
    document.body.appendChild(newScript);
  }
  return {
    loadFromDocument(doc2) {
      const moduleScripts = doc2.querySelectorAll('script[type="module"]');
      moduleScripts.forEach((scriptEl) => {
        const src = scriptEl.getAttribute("src");
        if (src) {
          loadScript(src);
        }
      });
    },
    async loadFromDocumentAsync(doc2) {
      const loadPromises = [];
      let failCount = 0;
      const moduleScripts = doc2.querySelectorAll('script[type="module"]');
      moduleScripts.forEach((scriptEl) => {
        const src = scriptEl.getAttribute("src");
        if (src && !loadedModuleScripts.has(src)) {
          loadedModuleScripts.add(src);
          loadPromises.push(
            import(
              /* webpackIgnore: true */
              src
            ).catch((err) => {
              failCount++;
              console.error(`ModuleLoader: Failed to load module ${src}:`, err);
            })
          );
        }
      });
      await Promise.all(loadPromises);
      if (failCount > 0) {
        console.error(`ModuleLoader: ${failCount}/${loadPromises.length} module(s) failed to load`);
      }
    },
    hasLoaded(src) {
      return loadedModuleScripts.has(src);
    },
    getLoadedModules() {
      return loadedModuleScripts;
    }
  };
}
function initModuleLoaderFromPage(loader) {
  const initialScripts = document.querySelectorAll('script[type="module"]');
  const loadedModules = loader.getLoadedModules();
  initialScripts.forEach((scriptEl) => {
    const src = scriptEl.getAttribute("src");
    if (src) {
      loadedModules.add(src);
    }
  });
}
function createLinkHeaderParser() {
  return {
    parseAndApply(linkHeader) {
      if (!linkHeader) {
        return;
      }
      const links = linkHeader.split(/,\s*/);
      links.forEach((link) => {
        const parts = link.split(";");
        const urlMatch = parts[0].trim().match(/<(.+)>/);
        if (!urlMatch) {
          return;
        }
        const url = urlMatch[1];
        const params = {};
        for (let i = 1; i < parts.length; i++) {
          const paramParts = parts[i].trim().split("=");
          const key = paramParts[0];
          params[key] = paramParts[1] ? paramParts[1].replace(/"/g, "") : "true";
        }
        if (document.querySelector(`link[href="${url}"]`)) {
          return;
        }
        const linkEl = document.createElement("link");
        linkEl.href = url;
        if (params.rel) {
          linkEl.rel = params.rel;
        }
        if (params.as) {
          linkEl.setAttribute("as", params.as);
        }
        if ("crossorigin" in params) {
          linkEl.crossOrigin = params.crossorigin === "true" ? "" : params.crossorigin;
        }
        if (params.type) {
          linkEl.type = params.type;
        }
        document.head.appendChild(linkEl);
      });
    }
  };
}
const BASE64_BLOCK_SIZE = 4;
function resolveArgsWithEvent(args, event, element) {
  return args.map((a) => {
    if (a.t === "e") {
      return event;
    }
    if (a.t === "f") {
      if (!element) {
        console.warn("[DOMBinder] $form used but no element context available");
        return null;
      }
      const form = element.closest("form");
      if (!form) {
        console.warn("[DOMBinder] $form used but no form ancestor found for element:", element);
        return null;
      }
      return formData(form);
    }
    return a.v;
  });
}
function resolveArgsForAction(args, event, element) {
  return args.map((a) => {
    if (a.t === "e") {
      return event;
    }
    if (a.t === "f") {
      if (!element) {
        console.warn("[DOMBinder] $form used but no element context available");
        return {};
      }
      const form = element.closest("form");
      if (!form) {
        console.warn("[DOMBinder] $form used but no form ancestor found for element:", element);
        return {};
      }
      return formData(form).toObject();
    }
    return a.v;
  });
}
const BOUND_MARKER = "pk-ev-bound";
function getAttrTrimmed(el, attr) {
  return el.getAttribute(attr)?.trim() ?? "";
}
const P_MODAL_PARAM_PREFIX = "p-modal-param:";
const P_ON_PREFIX_LEN = 5;
const P_EVENT_PREFIX_LEN = 8;
const NATIVE_URI_SCHEMES = [
  "tel:",
  "mailto:",
  "sms:",
  "geo:",
  "webcal:",
  "facetime:",
  "facetime-audio:",
  "skype:",
  "whatsapp:",
  "viber:",
  "maps:",
  "comgooglemaps:"
];
const BLOCKED_URI_SCHEMES = ["javascript:", "data:", "blob:", "file:"];
function isNativeScheme(href) {
  const lowerHref = href.toLowerCase();
  return NATIVE_URI_SCHEMES.some((scheme) => lowerHref.startsWith(scheme));
}
function isBlockedScheme(href) {
  const lowerHref = href.toLowerCase();
  return BLOCKED_URI_SCHEMES.some((scheme) => lowerHref.startsWith(scheme));
}
function findPartialScope(el) {
  let current = el;
  while (current) {
    const partialId = current.getAttribute("partial");
    if (partialId) {
      return partialId;
    }
    current = current.parentElement;
  }
  return void 0;
}
function parseFunctionReference(ref) {
  if (ref.startsWith("@")) {
    const dotIndex = ref.indexOf(".");
    if (dotIndex > 1) {
      return {
        partialName: ref.slice(1, dotIndex),
        fnName: ref.slice(dotIndex + 1)
      };
    }
  }
  return { partialName: null, fnName: ref };
}
function collectModalParams(el) {
  const params = /* @__PURE__ */ new Map();
  for (const { name, value } of Array.from(el.attributes)) {
    if (name.startsWith(P_MODAL_PARAM_PREFIX)) {
      const paramName = name.slice(P_MODAL_PARAM_PREFIX.length).trim();
      params.set(paramName, value.trim());
    }
  }
  return params;
}
function executeHelper(helper, ctx) {
  const args = ctx.resolvedArgs.map((a) => {
    if (a.t === "e") {
      return ctx.event.type;
    }
    if (a.t === "f") {
      const form = ctx.el.closest("form");
      if (!form) {
        return "";
      }
      return formData(form).toJSON();
    }
    return String(a.v);
  });
  try {
    const result = helper(ctx.el, ctx.event, ...args);
    if (result instanceof Promise) {
      result.catch((err) => {
        console.error("[DOMBinder] Async helper execution failed:", err);
      });
    }
  } catch (err) {
    console.error("[DOMBinder] Helper execution failed:", err);
  }
}
function dispatchIfActionDescriptor(result, el, event, errorPrefix, fnName) {
  if (isActionDescriptor(result)) {
    handleAction(result, el, event).catch((err) => {
      console.error(`${errorPrefix} Action execution failed for "${fnName}":`, err);
    });
    return;
  }
  if (result instanceof Promise) {
    result.then((resolved) => {
      if (isActionDescriptor(resolved)) {
        return handleAction(resolved, el, event);
      }
      return void 0;
    }).catch((err) => {
      console.error(`${errorPrefix} Async action execution failed for "${fnName}":`, err);
    });
  }
}
function executePageFunction(pageFunction, fnName, ctx, errorPrefix) {
  if (!pageFunction) {
    return;
  }
  try {
    const args = resolveArgsWithEvent(ctx.resolvedArgs, ctx.event, ctx.el);
    const result = pageFunction(ctx.event, ...args);
    dispatchIfActionDescriptor(result, ctx.el, ctx.event, errorPrefix, fnName);
  } catch (error) {
    console.error(`${errorPrefix} Error in page handler "${fnName}":`, error);
  }
}
function executeRegisteredAction(actionFunction, fnName, ctx, errorPrefix) {
  try {
    const args = resolveArgsForAction(ctx.resolvedArgs, ctx.event, ctx.el);
    const result = actionFunction(...args);
    dispatchIfActionDescriptor(result, ctx.el, ctx.event, errorPrefix, fnName);
  } catch (error) {
    console.error(`${errorPrefix} Error in action handler "${fnName}":`, error);
  }
}
function handleCustomEventNoModifier(ctx) {
  const pageContext = getGlobalPageContext();
  if (pageContext.hasFunction(ctx.payload.f)) {
    executePageFunction(
      pageContext.getFunction(ctx.payload.f),
      ctx.payload.f,
      ctx,
      "[DOMBinder]"
    );
    return;
  }
  const helper = ctx.helperRegistry.get(ctx.payload.f);
  if (helper) {
    executeHelper(helper, ctx);
    return;
  }
  const actionFn = getActionFunction(ctx.payload.f);
  if (actionFn) {
    executeRegisteredAction(actionFn, ctx.payload.f, ctx, "[DOMBinder]");
    return;
  }
  const available = pageContext.getExportedFunctions();
  const suggestion = findClosestMatch(ctx.payload.f, available);
  let message = `[DOMBinder] Function "${ctx.payload.f}" not found for p-event handler.`;
  message += ` Did you forget to export it?`;
  if (suggestion) {
    message += ` Did you mean "${suggestion}"?`;
  }
  if (available.length > 0) {
    message += ` Available functions: ${available.join(", ")}`;
  }
  console.warn(message);
}
function executeSinglePartialFunction(partialFunction, ctx, args, errorLabel) {
  try {
    const result = partialFunction(ctx.event, ...args);
    dispatchIfActionDescriptor(result, ctx.el, ctx.event, "[DOMBinder]", errorLabel);
  } catch (error) {
    console.error(`[DOMBinder] Error in ${errorLabel}:`, error);
  }
}
function handleExplicitPartialCall(ctx, explicitPartial, fnName) {
  const pageContext = getGlobalPageContext();
  const fns = pageContext.getFunctionsByPartialName(explicitPartial, fnName);
  if (fns.length > 0) {
    const args = resolveArgsWithEvent(ctx.resolvedArgs, ctx.event, ctx.el);
    const errorLabel = `@${explicitPartial}.${fnName}`;
    fns.forEach((partialFunction) => executeSinglePartialFunction(partialFunction, ctx, args, errorLabel));
    return;
  }
  const available = pageContext.getRegisteredPartialNames();
  const suggestion = findClosestMatch(explicitPartial, available);
  let message = `[DOMBinder] Partial "${explicitPartial}" not found or has no function "${fnName}".`;
  if (suggestion) {
    message += ` Did you mean "@${suggestion}"?`;
  }
  if (available.length > 0) {
    message += ` Registered partials: ${available.join(", ")}`;
  }
  console.warn(message);
}
function handleImplicitScopeCall(ctx, fnName) {
  const pageContext = getGlobalPageContext();
  const partialId = findPartialScope(ctx.el);
  let pageFunction;
  if (partialId) {
    pageFunction = pageContext.getScopedFunction(fnName, partialId);
  }
  pageFunction ??= pageContext.getFunction(fnName);
  if (pageFunction) {
    executePageFunction(pageFunction, fnName, ctx, "[DOMBinder]");
    return;
  }
  const helper = ctx.helperRegistry.get(fnName);
  if (helper) {
    executeHelper(helper, ctx);
    return;
  }
  const actionFn = getActionFunction(fnName);
  if (actionFn) {
    executeRegisteredAction(actionFn, fnName, ctx, "[DOMBinder]");
    return;
  }
  const available = pageContext.getExportedFunctions();
  const suggestion = findClosestMatch(fnName, available);
  let message = `[DOMBinder] Function "${fnName}" not found.`;
  if (partialId) {
    message += ` (searched partial scope and global)`;
  }
  message += ` Did you forget to export it?`;
  if (suggestion) {
    message += ` Did you mean "${suggestion}"?`;
  }
  if (available.length > 0) {
    message += ` Available functions: ${available.join(", ")}`;
  }
  console.warn(message);
}
function handleNoModifier(ctx) {
  const { partialName: explicitPartial, fnName } = parseFunctionReference(ctx.payload.f);
  if (explicitPartial) {
    handleExplicitPartialCall(ctx, explicitPartial, fnName);
  } else {
    handleImplicitScopeCall(ctx, fnName);
  }
}
function dispatchHandler(ctx) {
  if (ctx.isCustomEvent) {
    handleCustomEventNoModifier(ctx);
    return;
  }
  handleNoModifier(ctx);
}
function createModalHandler(el, callbacks) {
  return () => {
    callbacks.onOpenModal({
      selector: getAttrTrimmed(el, "p-modal:selector"),
      params: collectModalParams(el),
      title: getAttrTrimmed(el, "p-modal:title"),
      message: getAttrTrimmed(el, "p-modal:message"),
      cancelLabel: getAttrTrimmed(el, "p-modal:cancel_label"),
      confirmLabel: getAttrTrimmed(el, "p-modal:confirm_label"),
      confirmAction: getAttrTrimmed(el, "p-modal:confirm_action"),
      element: el
    });
  };
}
function urlSafeBase64ToStd(s) {
  let std = s.replace(/-/g, "+").replace(/_/g, "/");
  const pad = (BASE64_BLOCK_SIZE - std.length % BASE64_BLOCK_SIZE) % BASE64_BLOCK_SIZE;
  std += "=".repeat(pad);
  return std;
}
function parsePayload(encodedPayload, el) {
  try {
    return JSON.parse(atob(urlSafeBase64ToStd(encodedPayload)));
  } catch (e) {
    console.error("DOMBinder: Could not decode action payload.", { encodedPayload, element: el, error: e });
    return null;
  }
}
function createActionHandler(key, encodedPayload, isCustomEvent, el, helperRegistry, callbacks) {
  const parts = key.split(".");
  const eventName = parts[0].trim();
  const modifiers = new Set(parts.slice(1));
  if (!eventName) {
    return { eventName: "", handlerFunc: null };
  }
  const listenerOptions = {};
  if (modifiers.has("capture")) {
    listenerOptions.capture = true;
  }
  if (modifiers.has("passive")) {
    listenerOptions.passive = true;
  }
  let firedOnce = false;
  const handlerFunc = (event) => {
    if (modifiers.has("self") && event.target !== event.currentTarget) {
      return;
    }
    if (modifiers.has("prevent")) {
      event.preventDefault();
    }
    if (modifiers.has("stop")) {
      event.stopPropagation();
    }
    if (modifiers.has("once") && firedOnce) {
      return;
    }
    firedOnce = true;
    const payload = parsePayload(encodedPayload, el);
    if (!payload) {
      return;
    }
    const resolvedArgs = payload.a ?? [];
    const method = (el.getAttribute("data-pk-action-method") ?? "POST").toUpperCase();
    const ctx = {
      payload,
      resolvedArgs,
      el,
      event,
      method,
      helperRegistry,
      callbacks,
      isCustomEvent,
      eventName
    };
    dispatchHandler(ctx);
  };
  const hasOptions = listenerOptions.capture === true || listenerOptions.passive === true;
  return { eventName, handlerFunc, listenerOptions: hasOptions ? listenerOptions : void 0 };
}
function handlerGroupKey(eventName, listenerOptions) {
  if (!listenerOptions) {
    return eventName;
  }
  let key = eventName;
  if (listenerOptions.capture) {
    key += "$capture";
  }
  if (listenerOptions.passive) {
    key += "$passive";
  }
  return key;
}
function addHandler(handlers, eventName, eventHandler, listenerOptions) {
  const key = handlerGroupKey(eventName, listenerOptions);
  if (!handlers.has(key)) {
    handlers.set(key, { eventName, funcs: [], listenerOptions });
  }
  handlers.get(key)?.funcs.push(eventHandler);
}
function createDOMBinder(helperRegistry, callbacks) {
  const onNavigateLinkClick = (event) => {
    const linkEl = event.currentTarget;
    const href = linkEl?.getAttribute("href");
    if (!href) {
      return;
    }
    if (isBlockedScheme(href)) {
      event.preventDefault();
      console.warn("DOMBinder: Blocked navigation to dangerous URI scheme:", href);
      return;
    }
    if (isNativeScheme(href)) {
      return;
    }
    event.preventDefault();
    const morph = linkEl?.getAttribute("morph") ?? void 0;
    callbacks.onNavigate(href, event, { morph });
  };
  function bindLinks(rootElement) {
    rootElement.querySelectorAll("a").forEach((linkEl) => {
      if (!linkEl.hasAttribute("piko:a")) {
        return;
      }
      linkEl.removeEventListener("click", onNavigateLinkClick);
      linkEl.addEventListener("click", onNavigateLinkClick);
    });
  }
  function bindActions(rootElement) {
    rootElement.querySelectorAll("*").forEach((el) => {
      if (el.hasAttribute(BOUND_MARKER)) {
        return;
      }
      const handlers = /* @__PURE__ */ new Map();
      let hasBound = false;
      for (const { name: attrName, value: attrValue } of Array.from(el.attributes)) {
        let result = null;
        if (attrName.startsWith("p-on:")) {
          result = createActionHandler(attrName.slice(P_ON_PREFIX_LEN), attrValue, false, el, helperRegistry, callbacks);
        } else if (attrName.startsWith("p-event:")) {
          result = createActionHandler(attrName.slice(P_EVENT_PREFIX_LEN), attrValue, true, el, helperRegistry, callbacks);
        }
        if (result?.handlerFunc) {
          addHandler(handlers, result.eventName, result.handlerFunc, result.listenerOptions);
          hasBound = true;
        }
      }
      if (el.hasAttribute("p-modal:selector")) {
        addHandler(handlers, "click", createModalHandler(el, callbacks));
        hasBound = true;
      }
      handlers.forEach(({ eventName, funcs, listenerOptions }) => {
        el.addEventListener(eventName, (event) => {
          for (const handler of funcs) {
            try {
              handler(event);
            } catch (err) {
              console.error("[DOMBinder] Event handler failed:", err);
            }
          }
        }, listenerOptions);
      });
      if (hasBound) {
        el.setAttribute(BOUND_MARKER, "true");
      }
    });
  }
  return {
    bind: (root) => {
      bindLinks(root);
      bindActions(root);
    },
    bindLinks,
    bindActions
  };
}
const VISIBILITY_DEBOUNCE_MS = 150;
const INPUT_DEBOUNCE_MS = 400;
const SYNC_BOUND_MARKER = "pk-sync-bound";
const SYNC_TRIGGER_TAGS = ["SELECT", "INPUT", "PP-SELECT", "PP-CHECKBOX"];
function extractPrimaryValue(attrValue) {
  if (!attrValue) {
    return "";
  }
  const parts = attrValue.trim().split(/\s+/);
  return parts[parts.length - 1] || "";
}
function gatherFormData(form) {
  if (!form) {
    return void 0;
  }
  const formData2 = new FormData(form);
  const data = {};
  for (const key of new Set(formData2.keys())) {
    data[key] = formData2.getAll(key);
  }
  return data;
}
function isElementVisible(el) {
  const rect = el.getBoundingClientRect();
  return rect.top < window.innerHeight && rect.bottom > 0 && rect.left < window.innerWidth && rect.right > 0 && rect.width > 0 && rect.height > 0;
}
const REFRESH_LEVEL_NO_REFRESH_ATTRS = 3;
function detectRefreshLevel(el) {
  if (el.hasAttribute("pk-no-refresh-attrs")) {
    return REFRESH_LEVEL_NO_REFRESH_ATTRS;
  }
  if (el.hasAttribute("pk-own-attrs")) {
    return 2;
  }
  if (el.hasAttribute("pk-refresh-root")) {
    return 1;
  }
  return 0;
}
function createUpdateServer(binding) {
  const { containerEl, partialSrc, callbacks } = binding;
  return async (formData2 = null) => {
    const form = containerEl.closest("form");
    const gatheredData = formData2 ?? gatherFormData(form);
    const level = detectRefreshLevel(containerEl);
    const childrenOnly = level === 0;
    const preservePartialScopes = level >= 1;
    const ownedAttributes = level === 2 ? getOwnedAttributes(containerEl) : void 0;
    await callbacks.onRemoteRender({
      src: partialSrc,
      formData: gatheredData,
      patchMethod: "morph",
      childrenOnly,
      preservePartialScopes,
      ownedAttributes,
      querySelector: `[partial_src="${partialSrc}"]`,
      patchLocation: containerEl
    });
  };
}
function setupContainerEventListeners(binding) {
  const { containerEl, debounceTimers: debounceTimers2 } = binding;
  const updateServer = createUpdateServer(binding);
  containerEl.addEventListener("input", (_event) => {
    clearTimeout(debounceTimers2.get(containerEl));
    debounceTimers2.set(containerEl, setTimeout(() => void updateServer(), INPUT_DEBOUNCE_MS));
  });
  containerEl.addEventListener("change", (event) => {
    const target = event.target;
    if (!SYNC_TRIGGER_TAGS.includes(target.tagName)) {
      return;
    }
    if (target.tagName === "INPUT" && target.type === "text") {
      return;
    }
    clearTimeout(debounceTimers2.get(containerEl));
    void updateServer();
  });
  containerEl.addEventListener("refresh-partial", (event) => {
    event.stopPropagation();
    const customEvent = event;
    void updateServer(customEvent.detail?.formData ?? null).then(() => {
      customEvent.detail?.afterMorph?.();
    }).catch((err) => {
      console.error("[SyncPartialManager] refresh-partial failed:", err);
    });
  });
}
function createSyncPartialManager(callbacks) {
  const debounceVisibleElements = /* @__PURE__ */ new Set();
  let visibilityDebounceTimer;
  const visibilityState = /* @__PURE__ */ new WeakMap();
  const processVisibleBatch = () => {
    debounceVisibleElements.forEach((el) => el.dispatchEvent(new CustomEvent("refresh-partial", { bubbles: false })));
    debounceVisibleElements.clear();
  };
  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const containerEl = entry.target;
        const wasVisible = visibilityState.get(containerEl) === "visible";
        if (entry.isIntersecting && !wasVisible) {
          debounceVisibleElements.add(containerEl);
        }
        visibilityState.set(containerEl, entry.isIntersecting ? "visible" : "hidden");
      }
      if (debounceVisibleElements.size > 0) {
        clearTimeout(visibilityDebounceTimer);
        visibilityDebounceTimer = setTimeout(processVisibleBatch, VISIBILITY_DEBOUNCE_MS);
      }
    },
    { root: null, threshold: 0.1 }
  );
  return {
    bind(rootElement) {
      const containers = rootElement.querySelectorAll(`[partial_mode="sync"]:not([${SYNC_BOUND_MARKER}])`);
      containers.forEach((containerEl) => {
        const partialSrcAttr = containerEl.getAttribute("partial_src");
        const partialSrc = extractPrimaryValue(partialSrcAttr);
        if (!partialSrc) {
          console.warn('SyncPartialManager: A sync container is missing its "partial_src" attribute.', containerEl);
          return;
        }
        const binding = { containerEl, partialSrc, callbacks, debounceTimers: /* @__PURE__ */ new WeakMap() };
        setupContainerEventListeners(binding);
        requestAnimationFrame(() => requestAnimationFrame(() => {
          visibilityState.set(containerEl, isElementVisible(containerEl) ? "visible" : "hidden");
          observer.observe(containerEl);
        }));
        containerEl.setAttribute(SYNC_BOUND_MARKER, "true");
      });
    }
  };
}
const OFFLINE_BANNER_ID = "ppf-offline-banner";
function createOfflineBanner() {
  const banner = document.createElement("div");
  banner.id = OFFLINE_BANNER_ID;
  banner.setAttribute("role", "alert");
  banner.setAttribute("aria-live", "assertive");
  banner.textContent = "You are offline. Some features may be unavailable.";
  banner.style.cssText = [
    "position:fixed",
    "top:0",
    "left:0",
    "right:0",
    "padding:8px 16px",
    "background:#f44336",
    "color:white",
    "text-align:center",
    "font-size:14px",
    "z-index:10000",
    "display:none"
  ].join(";");
  return banner;
}
function createNetworkStatus(deps) {
  const { hookManager } = deps;
  let online = navigator.onLine;
  const listeners2 = /* @__PURE__ */ new Map();
  listeners2.set("online", /* @__PURE__ */ new Set());
  listeners2.set("offline", /* @__PURE__ */ new Set());
  let banner = document.getElementById(OFFLINE_BANNER_ID);
  if (!banner) {
    banner = createOfflineBanner();
    document.body.appendChild(banner);
  }
  if (!online) {
    banner.style.display = "block";
  }
  const notifyListeners = (event) => {
    const callbacks = listeners2.get(event);
    if (!callbacks) {
      return;
    }
    for (const callback of callbacks) {
      try {
        callback();
      } catch (error) {
        console.error(`NetworkStatus: Error in ${event} listener:`, error);
      }
    }
  };
  const handleOnline = () => {
    online = true;
    banner.style.display = "none";
    hookManager.emit(HookEvent.NETWORK_ONLINE, { timestamp: Date.now() });
    notifyListeners("online");
  };
  const handleOffline = () => {
    online = false;
    banner.style.display = "block";
    hookManager.emit(HookEvent.NETWORK_OFFLINE, { timestamp: Date.now() });
    notifyListeners("offline");
  };
  window.addEventListener("online", handleOnline);
  window.addEventListener("offline", handleOffline);
  return {
    get isOnline() {
      return online;
    },
    on(event, callback) {
      const callbacks = listeners2.get(event);
      if (callbacks) {
        callbacks.add(callback);
      }
      return () => {
        callbacks?.delete(callback);
      };
    },
    destroy() {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
      banner.remove();
      listeners2.clear();
    }
  };
}
const ANNOUNCER_ID = "ppf-a11y-announcer";
function createAnnouncerElement() {
  const element = document.createElement("div");
  element.id = ANNOUNCER_ID;
  element.setAttribute("role", "status");
  element.setAttribute("aria-live", "polite");
  element.setAttribute("aria-atomic", "true");
  element.style.cssText = [
    "position:absolute",
    "width:1px",
    "height:1px",
    "padding:0",
    "margin:-1px",
    "overflow:hidden",
    "clip:rect(0,0,0,0)",
    "white-space:nowrap",
    "border:0"
  ].join(";");
  return element;
}
function createA11yAnnouncer() {
  let element = document.getElementById(ANNOUNCER_ID);
  if (!element) {
    element = createAnnouncerElement();
    document.body.appendChild(element);
  }
  const announce = (message, priority = "polite") => {
    if (!element) {
      return;
    }
    element.setAttribute("aria-live", priority);
    element.textContent = "";
    requestAnimationFrame(() => {
      if (element) {
        element.textContent = message;
      }
    });
  };
  return {
    announce,
    announceNavigation(pageTitle) {
      announce(`Navigated to ${pageTitle}`);
    },
    announceLoading() {
      announce("Loading page");
    },
    announceError(errorMessage) {
      announce(errorMessage, "assertive");
    },
    destroy() {
      element?.remove();
      element = null;
    }
  };
}
const NO_TRACK_ATTR = "pk-no-track";
const TRACKED_ATTR = "data-pk-tracked";
const DEFERRED_SNAPSHOT_TIMEOUT_MS = 5e3;
function isInsideNoTrackContainer(el) {
  return el.closest(`[${NO_TRACK_ATTR}]`) !== null;
}
function getFormSnapshot(form) {
  const excludedNames = collectNoTrackFieldNames(form);
  const data = new FormData(form);
  const entries = [];
  for (const [key, value] of data.entries()) {
    if (typeof value === "string" && !excludedNames.has(key)) {
      entries.push([key, value]);
    }
  }
  const checkboxes = form.querySelectorAll('input[type="checkbox"], input[type="radio"]');
  for (const checkbox of checkboxes) {
    if (checkbox.name && !excludedNames.has(checkbox.name)) {
      entries.push([`__checked_${checkbox.name}_${checkbox.value}`, String(checkbox.checked)]);
    }
  }
  entries.sort((a, b) => a[0].localeCompare(b[0]));
  return JSON.stringify(entries);
}
function collectNoTrackFieldNames(form) {
  const excluded = /* @__PURE__ */ new Set();
  const elements = form.elements;
  for (let i = 0; i < elements.length; i++) {
    const el = elements[i];
    if (el.name && isInsideNoTrackContainer(el)) {
      excluded.add(el.name);
    }
  }
  return excluded;
}
function getFormId(form) {
  if (form.id) {
    return form.id;
  }
  const action = form.action || "no-action";
  const forms = document.querySelectorAll("form");
  const index = Array.from(forms).indexOf(form);
  return `form-${index}-${action.slice(-20)}`;
}
function diffSnapshots(initial, current) {
  const initialEntries = JSON.parse(initial);
  const currentEntries = JSON.parse(current);
  const initialMap = new Map(initialEntries);
  const currentMap = new Map(currentEntries);
  const diffs = [];
  for (const [key, value] of currentMap) {
    const initialValue = initialMap.get(key);
    if (initialValue === void 0) {
      diffs.push({ field: key, initial: "(absent)", current: value });
    } else if (initialValue !== value) {
      diffs.push({ field: key, initial: initialValue, current: value });
    }
  }
  for (const [key, value] of initialMap) {
    if (!currentMap.has(key)) {
      diffs.push({ field: key, initial: value, current: "(absent)" });
    }
  }
  return diffs;
}
function updateDirtyState(form, trackedForms, hookManager) {
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
      `[pk] Form "${getFormId(form)}" is now dirty. Changed fields:`,
      diffs,
      `

If this form should not trigger unsaved changes warnings, add the "pk-no-track" attribute to the <form> element.
If a custom element is incorrectly reporting a changed value, check that its setFormValue() call in connectedCallback matches the server-rendered initial state.`
    );
    hookManager.emit(HookEvent.FORM_DIRTY, { formId: getFormId(form), timestamp: Date.now() });
  } else if (!tracked.isDirty && wasDirty) {
    hookManager.emit(HookEvent.FORM_CLEAN, { formId: getFormId(form), timestamp: Date.now() });
  }
}
function createFormInputHandler(trackedForms, hookManager) {
  return (event) => {
    const target = event.target;
    if (!target) {
      return;
    }
    const form = target.closest("form");
    if (form instanceof HTMLFormElement && trackedForms.has(form)) {
      updateDirtyState(form, trackedForms, hookManager);
    }
  };
}
function checkHasDirtyForms(trackedForms, hookManager) {
  for (const tracked of trackedForms.values()) {
    updateDirtyState(tracked.form, trackedForms, hookManager);
    if (tracked.isDirty) {
      return true;
    }
  }
  return false;
}
function containsCustomElements(form) {
  const allElements = form.querySelectorAll("*");
  for (const el of allElements) {
    if (el.localName.includes("-")) {
      return true;
    }
  }
  return false;
}
async function deferFormSnapshot(form, trackedForms) {
  const undefinedElements = form.querySelectorAll(":not(:defined)");
  if (undefinedElements.length > 0) {
    const tagNames = /* @__PURE__ */ new Set();
    for (const el of undefinedElements) {
      tagNames.add(el.localName);
    }
    const timeout2 = new Promise(
      (resolve) => setTimeout(() => resolve("timeout"), DEFERRED_SNAPSHOT_TIMEOUT_MS)
    );
    const result = await Promise.race([
      Promise.all(
        Array.from(tagNames).map((name) => customElements.whenDefined(name))
      ).then(() => "defined"),
      timeout2
    ]);
    if (result === "timeout") {
      console.warn(
        `[pk] Timed out waiting for custom elements in form "${getFormId(form)}":`,
        Array.from(tagNames)
      );
    }
  }
  await new Promise((resolve) => requestAnimationFrame(() => {
    requestAnimationFrame(() => resolve());
  }));
  const tracked = trackedForms.get(form);
  if (!tracked) {
    return;
  }
  tracked.initialSnapshot = getFormSnapshot(form);
  tracked.snapshotPending = false;
}
function internalTrackForm(form, trackedForms) {
  if (trackedForms.has(form) || form.hasAttribute(NO_TRACK_ATTR)) {
    return;
  }
  if (containsCustomElements(form)) {
    trackedForms.set(form, { form, initialSnapshot: "", isDirty: false, snapshotPending: true });
    void deferFormSnapshot(form, trackedForms);
  } else {
    trackedForms.set(form, { form, initialSnapshot: getFormSnapshot(form), isDirty: false, snapshotPending: false });
  }
  form.setAttribute(TRACKED_ATTR, "true");
}
function collectForms(node) {
  const forms = [];
  if (node instanceof HTMLFormElement && !node.hasAttribute(NO_TRACK_ATTR)) {
    forms.push(node);
  }
  if (node instanceof HTMLElement) {
    const nested = node.querySelectorAll("form:not([pk-no-track])");
    for (const form of nested) {
      forms.push(form);
    }
  }
  return forms;
}
function createFormObserver(trackedForms) {
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
function createFormSubmitHandler(trackedForms, hookManager) {
  return (event) => {
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
      hookManager.emit(HookEvent.FORM_CLEAN, { formId: getFormId(form), timestamp: Date.now() });
    }
  };
}
function createBeforeUnloadHandler(hasDirtyForms) {
  return (event) => {
    if (hasDirtyForms()) {
      event.preventDefault();
      event.returnValue = "";
    }
  };
}
function setupFormListeners(trackedForms, handleFormInput, handleFormSubmit, handleBeforeUnload) {
  const formObserver = createFormObserver(trackedForms);
  formObserver.observe(document.body, { childList: true, subtree: true });
  document.addEventListener("input", handleFormInput);
  document.addEventListener("change", handleFormInput);
  document.addEventListener("submit", handleFormSubmit);
  window.addEventListener("beforeunload", handleBeforeUnload);
  return formObserver;
}
function createFormStateManager(deps) {
  const { hookManager } = deps;
  const confirmFn = deps.confirmFn ?? ((message) => window.confirm(message));
  const trackedForms = /* @__PURE__ */ new Map();
  const handleFormInput = createFormInputHandler(trackedForms, hookManager);
  const hasDirtyForms = () => checkHasDirtyForms(trackedForms, hookManager);
  const handleBeforeUnload = createBeforeUnloadHandler(hasDirtyForms);
  const handleFormSubmit = createFormSubmitHandler(trackedForms, hookManager);
  const formObserver = setupFormListeners(trackedForms, handleFormInput, handleFormSubmit, handleBeforeUnload);
  return {
    trackForm(form) {
      internalTrackForm(form, trackedForms);
    },
    untrackForm(form) {
      trackedForms.delete(form);
      form.removeAttribute(TRACKED_ATTR);
    },
    markFormClean(form) {
      const tracked = trackedForms.get(form);
      if (!tracked) {
        return;
      }
      tracked.initialSnapshot = getFormSnapshot(form);
      tracked.snapshotPending = false;
      const wasDirty = tracked.isDirty;
      tracked.isDirty = false;
      if (wasDirty) {
        hookManager.emit(HookEvent.FORM_CLEAN, { formId: getFormId(form), timestamp: Date.now() });
      }
    },
    hasDirtyForms,
    getDirtyFormIds() {
      const dirty = [];
      for (const tracked of trackedForms.values()) {
        updateDirtyState(tracked.form, trackedForms, hookManager);
        if (tracked.isDirty) {
          dirty.push(getFormId(tracked.form));
        }
      }
      return dirty;
    },
    confirmNavigation: () => !hasDirtyForms() || confirmFn("You have unsaved changes. Leave anyway?"),
    scanAndTrackForms(root = document.body) {
      for (const form of trackedForms.keys()) {
        if (!document.contains(form)) {
          trackedForms.delete(form);
        }
      }
      const forms = root.querySelectorAll("form:not([pk-no-track])");
      for (const form of forms) {
        internalTrackForm(form, trackedForms);
      }
    },
    untrackAll() {
      for (const form of trackedForms.keys()) {
        form.removeAttribute(TRACKED_ATTR);
      }
      trackedForms.clear();
    },
    destroy() {
      formObserver.disconnect();
      document.removeEventListener("input", handleFormInput);
      document.removeEventListener("change", handleFormInput);
      document.removeEventListener("submit", handleFormSubmit);
      window.removeEventListener("beforeunload", handleBeforeUnload);
      trackedForms.clear();
    }
  };
}
const browserDOMOperations = {
  /** Creates an HTML element by tag name. */
  createElement(tagName) {
    return document.createElement(tagName);
  },
  /** Creates a text node. */
  createTextNode(data) {
    return document.createTextNode(data);
  },
  /** Creates a comment node. */
  createComment(data) {
    return document.createComment(data);
  },
  /** Creates a document fragment. */
  createDocumentFragment() {
    return document.createDocumentFragment();
  },
  /** Queries the document for the first element matching a selector. */
  querySelector(selectors) {
    return document.querySelector(selectors);
  },
  /** Queries the document for all elements matching a selector. */
  querySelectorAll(selectors) {
    return document.querySelectorAll(selectors);
  },
  /** Returns the element with the given ID. */
  getElementById(elementId) {
    return document.getElementById(elementId);
  },
  /** Returns the document head element. */
  getHead() {
    return document.head;
  },
  /** Returns the currently focused element. */
  getActiveElement() {
    return document.activeElement;
  },
  /** Parses an HTML string into a Document. */
  parseHTML(html) {
    const parser = new DOMParser();
    return parser.parseFromString(html, "text/html");
  }
};
const browserWindowOperations = {
  /** Returns the current Location object. */
  getLocation() {
    return window.location;
  },
  /** Returns the location origin. */
  getLocationOrigin() {
    return window.location.origin;
  },
  /** Returns the location href. */
  getLocationHref() {
    return window.location.href;
  },
  /** Sets the location href, triggering a full page navigation. */
  setLocationHref(href) {
    window.location.href = href;
  },
  /** Reloads the current page. */
  locationReload() {
    window.location.reload();
  },
  /** Pushes a new entry onto the history stack. */
  historyPushState(data, unused, url) {
    window.history.pushState(data, unused, url);
  },
  /** Replaces the current history entry. */
  historyReplaceState(data, unused, url) {
    window.history.replaceState(data, unused, url);
  },
  /** Returns the current history state. */
  getHistoryState() {
    return window.history.state;
  },
  /** Adds an event listener to the window. */
  addEventListener(type, listener) {
    window.addEventListener(type, listener);
  },
  /** Removes an event listener from the window. */
  removeEventListener(type, listener) {
    window.removeEventListener(type, listener);
  },
  /** Returns the current vertical scroll position. */
  getScrollY() {
    return window.scrollY;
  },
  /** Scrolls the window to the specified position. */
  scrollTo(x, y) {
    window.scrollTo(x, y);
  },
  /** Sets the scroll restoration mode. */
  setScrollRestoration(mode) {
    if ("scrollRestoration" in history) {
      history.scrollRestoration = mode;
    }
  },
  /** Returns the current scroll restoration mode. */
  getScrollRestoration() {
    if ("scrollRestoration" in history) {
      return history.scrollRestoration;
    }
    return "auto";
  }
};
const browserHTTPOperations = {
  /** Fetches a resource from the network. */
  fetch(input, init) {
    return fetch(input, init);
  }
};
async function readWithProgress(response, onProgress) {
  const lenHeader = response.headers.get("Content-Length");
  if (!lenHeader) {
    return response.text();
  }
  const totalSize = parseInt(lenHeader, 10);
  if (isNaN(totalSize) || totalSize <= 0) {
    return response.text();
  }
  const reader = response.body?.getReader();
  if (!reader) {
    return response.text();
  }
  let loaded = 0;
  const chunks = [];
  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    if (!value) {
      continue;
    }
    chunks.push(value);
    loaded += value.length;
    onProgress(loaded, totalSize);
  }
  const combined = new Uint8Array(loaded);
  let position = 0;
  for (const chunk of chunks) {
    combined.set(chunk, position);
    position += chunk.length;
  }
  return new TextDecoder("utf-8").decode(combined);
}
function createFetchClient(deps = {}) {
  const http = deps.http ?? browserHTTPOperations;
  let controller = null;
  return {
    /**
     * Performs a GET request and returns the response as text.
     *
     * @param url - The URL to fetch.
     * @param options - The optional fetch client options.
     * @returns A tuple of [success, responseText].
     */
    async get(url, options = {}) {
      try {
        controller = new AbortController();
        const response = await http.fetch(url, {
          method: "GET",
          credentials: "same-origin",
          signal: controller.signal
        });
        if (!response.ok) {
          return [false, null];
        }
        let text;
        if (options.onProgress) {
          text = await readWithProgress(response, options.onProgress);
        } else {
          text = await response.text();
        }
        return [true, text];
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          throw error;
        }
        console.error("FetchClient.get error:", error);
        return [false, null];
      }
    },
    /**
     * Performs a POST request with optional body and headers.
     *
     * @param url - The URL to fetch.
     * @param body - The optional request body.
     * @param headers - The optional request headers.
     * @returns The fetch Response.
     */
    async post(url, body, headers = {}) {
      controller = new AbortController();
      return http.fetch(url, {
        method: "POST",
        headers,
        body,
        credentials: "same-origin",
        signal: controller.signal
      });
    },
    /** Aborts any in-progress request. */
    abort() {
      controller?.abort();
      controller = null;
    },
    /** Returns the current AbortController for external management. */
    getController() {
      return controller;
    }
  };
}
function addFragmentQuery(urlValue) {
  try {
    const parsedUrl = new URL(urlValue, window.location.origin);
    parsedUrl.searchParams.set("_f", "1");
    return parsedUrl.toString();
  } catch {
    if (urlValue.includes("?")) {
      return `${urlValue}&_f=1`;
    }
    return `${urlValue}?_f=1`;
  }
}
function buildRemoteUrl(base, args) {
  const urlObj = new URL(base, window.location.origin);
  urlObj.searchParams.set("_f", "1");
  for (const [paramName, paramValue] of Object.entries(args)) {
    urlObj.searchParams.set(paramName, String(paramValue));
  }
  return urlObj.toString();
}
function isSameDomain(loc) {
  return loc.hostname === window.location.hostname;
}
const PROGRESS_MAX = 100;
function safeInvokeCallback(callback, url) {
  if (!callback) {
    return;
  }
  try {
    callback(url);
  } catch (error) {
    console.warn("[Router] Error in navigation callback:", {
      url,
      callback: callback.name || "anonymous",
      error
    });
  }
}
function emitNavigationError(deps, targetUrl, errorMessage, displayMessage) {
  deps.errorDisplay.show(displayMessage ?? `Navigation to ${targetUrl} failed. Loading full page...`);
  deps.a11yAnnouncer?.announceError("Navigation failed");
  deps.hookManager?.emit(HookEvent.NAVIGATION_ERROR, {
    url: targetUrl,
    error: errorMessage,
    timestamp: Date.now()
  });
  deps.windowOps.setLocationHref(targetUrl);
}
function updateHistoryState(windowOps, targetUrl, replaceHistory) {
  const currentState = { scrollY: windowOps.getScrollY() };
  windowOps.historyReplaceState(currentState, "", windowOps.getLocationHref());
  const newState = { scrollY: 0 };
  if (replaceHistory) {
    windowOps.historyReplaceState(newState, "", targetUrl);
  } else {
    windowOps.historyPushState(newState, "", targetUrl);
  }
}
function emitNavigationSuccess(deps, ctx, pageTitle) {
  const duration = Date.now() - ctx.startTime;
  deps.a11yAnnouncer?.announceNavigation(pageTitle);
  deps.hookManager?.emit(HookEvent.NAVIGATION_COMPLETE, {
    url: ctx.targetUrl,
    previousUrl: ctx.previousUrl,
    timestamp: Date.now(),
    duration
  });
  deps.hookManager?.emit(HookEvent.PAGE_VIEW, {
    url: ctx.targetUrl,
    title: pageTitle,
    referrer: ctx.previousUrl,
    isInitialLoad: false,
    timestamp: Date.now()
  });
  deps.formStateManager?.scanAndTrackForms();
}
function handleNavigationError(deps, targetUrl, error) {
  if (error instanceof DOMException && error.name === "AbortError") {
    console.warn("Fetch aborted:", targetUrl);
    return;
  }
  console.error("navigateTo error:", error);
  const errorMessage = error instanceof Error ? error.message : "Unknown error";
  emitNavigationError(deps, targetUrl, errorMessage);
}
function shouldCancelNavigation(isPopNavigation, formStateManager) {
  if (isPopNavigation) {
    return false;
  }
  return Boolean(formStateManager?.hasDirtyForms() && !formStateManager.confirmNavigation());
}
function emitNavigationStart(deps, ctx, localBeforeNavigate) {
  safeInvokeCallback(localBeforeNavigate, ctx.targetUrl);
  deps.hookManager?.emit(HookEvent.NAVIGATION_START, {
    url: ctx.targetUrl,
    previousUrl: ctx.previousUrl,
    timestamp: ctx.startTime
  });
  deps.a11yAnnouncer?.announceLoading();
}
function buildScrollOptions(ctx, options, windowOps) {
  const hash = new URL(ctx.targetUrl, windowOps.getLocationOrigin()).hash;
  return {
    restoreScrollY: ctx.isPopNavigation ? options.restoreScrollY : void 0,
    hash: !ctx.isPopNavigation || options.restoreScrollY === void 0 ? hash : void 0,
    morph: options.morph
  };
}
async function performNavigation(state, deps, targetUrl, event, options) {
  const { fetchClient, loader, onPageLoad, windowOps, domOps, formStateManager } = deps;
  if (event) {
    event.preventDefault();
  }
  const isPopNavigation = Boolean(options.isPopState);
  if (shouldCancelNavigation(isPopNavigation, formStateManager)) {
    return;
  }
  formStateManager?.untrackAll();
  if (state.navigating) {
    fetchClient.abort();
  }
  state.navigating = true;
  loader.show();
  const ctx = {
    targetUrl,
    previousUrl: windowOps.getLocationHref(),
    startTime: Date.now(),
    isPopNavigation,
    options
  };
  const localBeforeNavigate = options.beforeNavigate ?? state.globalConfig.beforeNavigate;
  const localAfterNavigate = options.afterNavigate ?? state.globalConfig.afterNavigate;
  emitNavigationStart(deps, ctx, localBeforeNavigate);
  try {
    if (!isPopNavigation) {
      updateHistoryState(windowOps, targetUrl, options.replaceHistory);
    }
    const [success, htmlString] = await fetchClient.get(addFragmentQuery(targetUrl), {
      onProgress: (loaded, total) => loader.setProgress(loaded / total * PROGRESS_MAX)
    });
    if (!success || !htmlString) {
      emitNavigationError(deps, targetUrl, "Fetch failed");
      return;
    }
    const parsedDocument = domOps.parseHTML(htmlString);
    if (!parsedDocument.querySelector("#app")) {
      emitNavigationError(deps, targetUrl, "No #app in response", "No #app in fragment. Loading full page...");
      return;
    }
    const scrollOptions = buildScrollOptions(ctx, options, windowOps);
    await onPageLoad(parsedDocument, targetUrl, scrollOptions);
    focusMainContent(domOps);
    emitNavigationSuccess(deps, ctx, parsedDocument.title || document.title);
  } catch (error) {
    handleNavigationError(deps, targetUrl, error);
  } finally {
    state.navigating = false;
    loader.hide();
    safeInvokeCallback(localAfterNavigate, targetUrl);
  }
}
function focusMainContent(domOps) {
  const mainContent = domOps.querySelector('[role="main"], main, #app');
  if (mainContent) {
    const hadTabIndex = mainContent.hasAttribute("tabindex");
    const originalTabIndex = mainContent.getAttribute("tabindex");
    mainContent.setAttribute("tabindex", "-1");
    mainContent.focus({ preventScroll: true });
    if (hadTabIndex && originalTabIndex !== null) {
      mainContent.setAttribute("tabindex", originalTabIndex);
    } else if (!hadTabIndex) {
      mainContent.removeAttribute("tabindex");
    }
  }
}
function createRouter(deps) {
  const windowOps = deps.windowOps ?? browserWindowOperations;
  const domOps = deps.domOps ?? browserDOMOperations;
  const state = { navigating: false, globalConfig: {} };
  windowOps.setScrollRestoration("manual");
  const navDeps = {
    fetchClient: deps.fetchClient,
    loader: deps.loader,
    errorDisplay: deps.errorDisplay,
    onPageLoad: deps.onPageLoad,
    windowOps,
    domOps,
    hookManager: deps.hookManager,
    formStateManager: deps.formStateManager,
    a11yAnnouncer: deps.a11yAnnouncer
  };
  const navigateTo = (targetUrl, event, options = {}) => performNavigation(state, navDeps, targetUrl, event, options);
  const popstateHandler = () => {
    const location = windowOps.getLocation();
    if (!isSameDomain(location)) {
      windowOps.locationReload();
      return;
    }
    const historyState = windowOps.getHistoryState();
    const restoreScrollY = historyState?.scrollY;
    void navigateTo(windowOps.getLocationHref(), void 0, {
      replaceHistory: true,
      isPopState: true,
      restoreScrollY
    });
  };
  windowOps.addEventListener("popstate", popstateHandler);
  return {
    navigateTo,
    isNavigating: () => state.navigating,
    setConfig: (config) => {
      state.globalConfig = config;
    },
    destroy: () => windowOps.removeEventListener("popstate", popstateHandler)
  };
}
const HASH_SHIFT = 5;
const HASH_RADIX = 36;
function sha1(str) {
  let h = 0;
  const length = str.length;
  for (let i = 0; i < length; i++) {
    h = (h << HASH_SHIFT) - h + str.charCodeAt(i) | 0;
  }
  return h.toString(HASH_RADIX);
}
function buildFormData(input) {
  if (input instanceof URLSearchParams) {
    return input;
  }
  const params = new URLSearchParams();
  if (input instanceof FormData) {
    for (const [key, value] of input.entries()) {
      if (typeof value === "string") {
        params.append(key, value);
      }
    }
    return params;
  }
  if (input instanceof Map || typeof input === "object") {
    const entries = input instanceof Map ? input.entries() : Object.entries(input);
    for (const [key, value] of entries) {
      if (Array.isArray(value)) {
        value.forEach((v) => {
          if (v !== null && v !== void 0) {
            params.append(key, String(v));
          }
        });
      } else if (value !== null && value !== void 0) {
        params.append(key, String(value));
      }
    }
    return params;
  }
  console.warn("RemoteRenderer: `options.formData` was provided but is not a recognised type. Ignoring.");
  return void 0;
}
function processStyleBlocks(parsedDoc, domOps) {
  const styleBlocks = parsedDoc.querySelectorAll("style[pk-page]");
  styleBlocks.forEach((srcStyleEl) => {
    const cssText = srcStyleEl.textContent ?? "";
    if (!cssText.trim()) {
      return;
    }
    const pageId = parsedDoc.querySelector("#app")?.dataset.pageid;
    const styleKey = pageId ?? sha1(cssText);
    if (domOps.getHead().querySelector(`style[data-pk-style-key="${styleKey}"]`)) {
      return;
    }
    const newStyleEl = domOps.createElement("style");
    newStyleEl.setAttribute("pk-page", "");
    newStyleEl.setAttribute("data-pk-style-key", styleKey);
    newStyleEl.textContent = cssText;
    domOps.getHead().appendChild(newStyleEl);
  });
}
function getNodeKey(node) {
  if (node.nodeType !== 1) {
    return null;
  }
  const el = node;
  return el.dataset.stableId ?? el.getAttribute("p-key") ?? (el.id || null);
}
function transformRelativeKeys(sourceEl, targetEl) {
  const targetKey = targetEl.getAttribute("p-key");
  const sourceKey = sourceEl.getAttribute("p-key");
  if (!targetKey || !sourceKey) {
    return;
  }
  const elementsWithKeys = [sourceEl, ...Array.from(sourceEl.querySelectorAll("[p-key]"))];
  for (const el of elementsWithKeys) {
    const currentKey = el.getAttribute("p-key");
    if (!currentKey) {
      continue;
    }
    if (currentKey === sourceKey) {
      el.setAttribute("p-key", targetKey);
    } else if (currentKey.startsWith(`${sourceKey}:`)) {
      const suffix = currentKey.slice(sourceKey.length);
      el.setAttribute("p-key", targetKey + suffix);
    }
  }
}
function syncPatchAttributes(target, sourceEl) {
  if (!target.patchAttributes || !target.patchLocation) {
    return;
  }
  for (const attr of Array.from(sourceEl.attributes)) {
    const shouldSync = target.patchAttributes.includes(attr.name);
    const isDifferent = target.patchLocation.getAttribute(attr.name) !== attr.value;
    if (shouldSync && isDifferent) {
      target.patchLocation.setAttribute(attr.name, attr.value);
    }
  }
}
function applyPatch(ctx) {
  const { target, sourceEl, patchMethod, onDOMUpdated, domOps } = ctx;
  const { patchLocation } = target;
  if (patchLocation.hasAttribute("partial")) {
    _executeBeforeRender(patchLocation);
  }
  if (patchMethod === "morph") {
    fragmentMorpher(patchLocation, sourceEl, {
      childrenOnly: target.childrenOnly,
      preservePartialScopes: target.preservePartialScopes,
      ownedAttributes: target.ownedAttributes,
      getNodeKey,
      onNodeAdded(node) {
        if (node.nodeType === 1) {
          onDOMUpdated(node);
        }
        return node;
      },
      onBeforeElUpdated(fromEl, toEl) {
        return !fromEl.isEqualNode(toEl) && fromEl !== domOps.getActiveElement();
      }
    });
  } else {
    patchLocation.innerHTML = "";
    Array.from(sourceEl.children).forEach((child) => {
      patchLocation.appendChild(child.cloneNode(true));
    });
    onDOMUpdated(patchLocation);
  }
  syncPatchAttributes(target, sourceEl);
  if (patchLocation.hasAttribute("partial")) {
    _executeAfterRender(patchLocation);
    _executeUpdated(patchLocation, { patchMethod });
  }
}
function findSourceElement(rootEl, selector) {
  if (!selector) {
    return rootEl;
  }
  const matched = rootEl.querySelector(selector);
  if (!matched) {
    console.warn(`RemoteRenderer: selector "${selector}" not found`);
  }
  return matched ?? rootEl;
}
function buildTargetsList(options) {
  const targets = options.targets ?? [];
  if (options.querySelector ?? options.patchLocation) {
    targets.push({
      querySelector: options.querySelector ?? void 0,
      patchLocation: options.patchLocation ?? void 0,
      patchMethod: options.patchMethod ?? void 0,
      patchAttributes: options.patchAttributes ?? void 0,
      childrenOnly: options.childrenOnly ?? true,
      preservePartialScopes: options.preservePartialScopes ?? void 0,
      ownedAttributes: options.ownedAttributes ?? void 0
    });
  }
  return targets;
}
async function fetchFragment(fullUrl, options, ctx) {
  const fetchOptions = { method: "GET" };
  if (options.formData) {
    fetchOptions.method = "POST";
    const body = buildFormData(options.formData);
    if (body) {
      fetchOptions.body = body.toString();
      fetchOptions.headers = {
        "Content-Type": "application/x-www-form-urlencoded"
      };
    }
  }
  const response = await ctx.http.fetch(fullUrl, fetchOptions);
  if (!response.ok) {
    console.error("RemoteRenderer: fetch failed", response.status, response.statusText);
    return null;
  }
  const responseType = response.headers.get("X-PP-Response-Support");
  if (responseType !== "fragment-patch") {
    console.warn(`RemoteRenderer: expected 'fragment-patch' response but got '${responseType ?? "none"}'. Reloading page.`);
    ctx.windowOps.locationReload();
    return null;
  }
  const linkHeader = response.headers.get("Link");
  if (linkHeader) {
    ctx.linkHeaderParser.parseAndApply(linkHeader);
  }
  return response.text();
}
async function reloadCachedModules(parsedDoc, moduleLoader) {
  const moduleScripts = parsedDoc.querySelectorAll('script[type="module"]');
  const cachedScripts = [];
  for (const scriptEl of Array.from(moduleScripts)) {
    const src = scriptEl.getAttribute("src");
    if (src && moduleLoader.hasLoaded(src)) {
      cachedScripts.push(src);
    }
  }
  await moduleLoader.loadFromDocumentAsync(parsedDoc);
  if (cachedScripts.length > 0) {
    const pageContext = getGlobalPageContext();
    for (const src of cachedScripts) {
      void pageContext.loadModule(src).catch((err) => {
        console.error("[RemoteRenderer] Failed to reload cached module:", err);
      });
    }
  }
}
function applyRenderTargets(targets, rootEl, options, deps) {
  for (const target of targets) {
    if (!target.patchLocation) {
      console.warn("RemoteRenderer: target has no patchLocation");
      continue;
    }
    const sourceEl = findSourceElement(rootEl, target.querySelector);
    if (!sourceEl) {
      console.warn("RemoteRenderer: no valid element in fetched HTML");
      return;
    }
    transformRelativeKeys(sourceEl, target.patchLocation);
    applyPatch({
      target: { ...target, patchLocation: target.patchLocation },
      sourceEl,
      patchMethod: target.patchMethod ?? options.patchMethod ?? "replace",
      onDOMUpdated: deps.onDOMUpdated,
      domOps: deps.domOps
    });
    const patchLocationId = target.querySelector || target.patchLocation.id || "unknown";
    deps.hookManager?.emit(HookEvent.PARTIAL_RENDER, {
      src: options.src,
      patchLocation: patchLocationId,
      timestamp: Date.now()
    });
    _executeConnectedForPartials(target.patchLocation);
  }
}
function createRemoteRenderer(deps) {
  const { moduleLoader, spriteSheetManager, linkHeaderParser, onDOMUpdated, hookManager } = deps;
  const domOps = deps.domOps ?? browserDOMOperations;
  const windowOps = deps.windowOps ?? browserWindowOperations;
  const http = deps.http ?? browserHTTPOperations;
  const fetchCtx = { linkHeaderParser, http, windowOps };
  const renderTargetDeps = { onDOMUpdated, domOps, hookManager };
  async function render(options) {
    const fullUrl = buildRemoteUrl(options.src, options.args ?? {});
    let htmlContent = null;
    try {
      htmlContent = await fetchFragment(fullUrl, options, fetchCtx);
    } catch (error) {
      console.error("RemoteRenderer: network error:", error);
      return;
    }
    if (!htmlContent) {
      return;
    }
    const parsedDoc = domOps.parseHTML(htmlContent);
    spriteSheetManager.merge(parsedDoc.getElementById("sprite"));
    processStyleBlocks(parsedDoc, domOps);
    await reloadCachedModules(parsedDoc, moduleLoader);
    const rootEl = parsedDoc.querySelector("#app") ?? parsedDoc.documentElement;
    const targets = buildTargetsList(options);
    applyRenderTargets(targets, rootEl, options, renderTargetDeps);
  }
  function patchPartial(htmlString, cssSelector) {
    const doc2 = domOps.parseHTML(htmlString);
    const newPartialEl = doc2.querySelector(cssSelector);
    if (!newPartialEl) {
      return console.warn(`RemoteRenderer: patchPartial - no element found for selector ${cssSelector}`);
    }
    const currentEl = domOps.querySelector(cssSelector);
    if (!currentEl) {
      return console.warn(`RemoteRenderer: patchPartial - no existing element found for selector ${cssSelector}`);
    }
    currentEl.innerHTML = newPartialEl.innerHTML;
    onDOMUpdated(currentEl);
    hookManager?.emit(HookEvent.PARTIAL_RENDER, {
      src: "inline",
      patchLocation: cssSelector,
      timestamp: Date.now()
    });
  }
  return { render, patchPartial };
}
function createModalManager(deps = {}) {
  const { hookManager } = deps;
  return {
    /**
     * Opens a modal if available, dispatching a fallback event if not found.
     *
     * @param options - The modal request options.
     */
    async openIfAvailable(options) {
      const {
        selector: modalSelector,
        params = /* @__PURE__ */ new Map(),
        title: modalTitle = "",
        message: modalMessage = "",
        cancelLabel: modalCancelLabel = "",
        confirmLabel: modalConfirmLabel = "",
        confirmAction: modalConfirmAction = "",
        triggerElement,
        fallbackEventName = "modal-not-found"
      } = options;
      const modalElem = document.querySelector(modalSelector);
      if (!modalElem) {
        console.warn(`ModalManager: Could not find modal "${modalSelector}". Falling back to dispatch event.`);
        triggerElement.dispatchEvent(new CustomEvent(fallbackEventName, { bubbles: true, composed: true }));
        return;
      }
      const modalId = modalElem.id || modalSelector;
      hookManager?.emit(HookEvent.MODAL_OPEN, {
        modalId,
        url: window.location.href,
        timestamp: Date.now()
      });
      const requestFn = modalElem.request;
      if (typeof requestFn === "function") {
        const confirmed = await requestFn({
          modal_title: modalTitle,
          message: modalMessage,
          cancel_label: modalCancelLabel,
          confirm_label: modalConfirmLabel,
          confirm_action: modalConfirmAction,
          params: Object.fromEntries(params.entries())
        });
        hookManager?.emit(HookEvent.MODAL_CLOSE, {
          modalId,
          timestamp: Date.now()
        });
        if (confirmed) {
          triggerElement.dispatchEvent(
            new CustomEvent("modal-confirmed", {
              detail: {},
              bubbles: true,
              composed: true
            })
          );
        } else {
          triggerElement.dispatchEvent(
            new CustomEvent("modal-cancelled", {
              detail: {},
              bubbles: true,
              composed: true
            })
          );
        }
      } else {
        console.warn(`ModalManager: The modal "${modalSelector}" does not have a request() function. Trying open...`);
        modalElem.setAttribute("open", "true");
      }
    }
  };
}
function registerPartialInstancesFromDOM(doc2) {
  const partials = doc2.querySelectorAll("[partial][data-partial-name]");
  const pageContext = getGlobalPageContext();
  for (const el of partials) {
    const partialId = el.getAttribute("partial");
    const partialName = el.getAttribute("data-partial-name");
    if (partialId && partialName) {
      pageContext.registerPartialInstance(partialName, partialId);
    }
  }
}
const globalHelperRegistry = createHelperRegistry();
function RegisterHelper(name, setupFunction) {
  globalHelperRegistry.register(name, setupFunction);
}
function getGlobalHelperRegistry() {
  return globalHelperRegistry;
}
function loadPageScripts(doc2) {
  const pageScriptMetas = doc2.querySelectorAll('meta[name="pk-script"]');
  for (const meta of pageScriptMetas) {
    const scriptUrl = meta.getAttribute("content");
    const partialName = meta.getAttribute("data-partial-name");
    if (scriptUrl) {
      void getGlobalPageContext().loadModule(scriptUrl, partialName ?? void 0).catch((err) => {
        console.error("[PPFramework] Failed to load page script:", err);
      });
    }
  }
}
const loadedWidgetScripts = /* @__PURE__ */ new Set();
function loadWidgetScripts(doc2) {
  const metas = doc2.querySelectorAll('meta[name="pk-widget-script"]');
  if (metas.length === 0) {
    return;
  }
  let pendingCount = 0;
  function onScriptReady() {
    pendingCount--;
    if (pendingCount <= 0) {
      document.dispatchEvent(new Event("piko:widgetinit"));
    }
  }
  for (const meta of metas) {
    const src = meta.getAttribute("content");
    if (!src || loadedWidgetScripts.has(src)) {
      continue;
    }
    loadedWidgetScripts.add(src);
    pendingCount++;
    const script = document.createElement("script");
    script.src = src;
    script.async = true;
    script.defer = true;
    script.onload = onScriptReady;
    script.onerror = onScriptReady;
    document.head.appendChild(script);
  }
  if (pendingCount === 0) {
    document.dispatchEvent(new Event("piko:widgetinit"));
  }
}
function navigationNodeKey(node) {
  if (node.nodeType !== Node.ELEMENT_NODE) {
    return null;
  }
  const el = node;
  return el.dataset.stableId ?? el.getAttribute("p-key") ?? (el.id || null);
}
function scrollToAnchor(hash) {
  if (!hash || hash === "#") {
    return;
  }
  const elementId = hash.slice(1);
  const element = document.getElementById(elementId);
  if (element) {
    element.scrollIntoView({ behavior: "instant", block: "start" });
  }
}
function handleScrollPosition(scrollOptions) {
  if (scrollOptions.restoreScrollY !== void 0) {
    window.scrollTo({ top: scrollOptions.restoreScrollY, behavior: "instant" });
  } else if (scrollOptions.hash) {
    scrollToAnchor(scrollOptions.hash);
  } else {
    window.scrollTo({ top: 0, behavior: "instant" });
  }
}
function performDOMUpdate(deps, parsedDocument, oldAppRoot, newAppRoot, scrollOptions) {
  _runPageCleanup();
  getGlobalPageContext().clear();
  if (scrollOptions.morph === "none") {
    oldAppRoot.innerHTML = newAppRoot.innerHTML;
  } else {
    fragmentMorpher(oldAppRoot, newAppRoot, {
      childrenOnly: true,
      getNodeKey: navigationNodeKey
    });
  }
  handleScrollPosition(scrollOptions);
  deps.bindDOM(oldAppRoot);
  deps.moduleLoader.loadFromDocument(parsedDocument);
  loadPageScripts(parsedDocument);
  registerPartialInstancesFromDOM(parsedDocument);
  _executeConnectedForPartials(oldAppRoot);
  loadWidgetScripts(parsedDocument);
  const newPageStyle = parsedDocument.querySelector("style[pk-page]");
  const oldPageStyle = document.head.querySelector("style[pk-page]");
  if (oldPageStyle) {
    oldPageStyle.remove();
  }
  if (newPageStyle) {
    const clonedStyle = newPageStyle.cloneNode(true);
    document.head.appendChild(clonedStyle);
  }
  const newTitle = parsedDocument.querySelector("title");
  if (newTitle) {
    document.title = (newTitle.textContent ?? "").trim();
  }
}
async function handlePageLoad(deps, parsedDocument, _targetUrl, scrollOptions) {
  const newSpriteSheet = parsedDocument.getElementById("sprite");
  deps.spriteSheetManager.merge(newSpriteSheet);
  const newAppRoot = parsedDocument.querySelector("#app");
  const oldAppRoot = document.querySelector("#app");
  if (!oldAppRoot || !newAppRoot) {
    return;
  }
  const domUpdateDeps = { bindDOM: deps.bindDOM, moduleLoader: deps.moduleLoader };
  if ("startViewTransition" in document && typeof document.startViewTransition === "function") {
    const transition = document.startViewTransition(() => {
      performDOMUpdate(domUpdateDeps, parsedDocument, oldAppRoot, newAppRoot, scrollOptions);
    });
    await transition.updateCallbackDone;
  } else {
    performDOMUpdate(domUpdateDeps, parsedDocument, oldAppRoot, newAppRoot, scrollOptions);
  }
}
function initFrameworkServices(services, options, instance) {
  services.globalConfig = options;
  services.loader = createLoaderUI({ colour: options.loaderColour ?? "#29e" });
  services.errorDisplay = createErrorDisplay();
  services.networkStatus = createNetworkStatus({ hookManager: services.hookManager });
  services.a11yAnnouncer = createA11yAnnouncer();
  services.formStateManager = createFormStateManager({ hookManager: services.hookManager });
  services.fetchClient = createFetchClient();
  const bindDOM = (root) => {
    services.domBinder?.bind(root);
    services.syncPartialManager?.bind(root);
  };
  registerDOMUpdater(bindDOM);
  services.remoteRenderer = createRemoteRenderer({
    moduleLoader: services.moduleLoader,
    spriteSheetManager: services.spriteSheetManager,
    linkHeaderParser: services.linkHeaderParser,
    onDOMUpdated: bindDOM,
    hookManager: services.hookManager
  });
  const pageLoadDeps = {
    spriteSheetManager: services.spriteSheetManager,
    bindDOM,
    moduleLoader: services.moduleLoader
  };
  services.router = createRouter({
    fetchClient: services.fetchClient,
    loader: services.loader,
    errorDisplay: services.errorDisplay,
    onPageLoad: (doc2, url, scroll) => handlePageLoad(pageLoadDeps, doc2, url, scroll),
    hookManager: services.hookManager,
    formStateManager: services.formStateManager,
    a11yAnnouncer: services.a11yAnnouncer
  });
  services.router.setConfig({
    beforeNavigate: options.beforeNavigate,
    afterNavigate: options.afterNavigate
  });
  services.domBinder = createDOMBinder(services.helperRegistry, {
    onNavigate: (url, _event, linkOptions) => {
      void instance.navigateTo(url, void 0, { morph: linkOptions.morph });
    },
    onOpenModal: (opts) => {
      void services.modalManager.openIfAvailable({
        selector: opts.selector,
        params: opts.params,
        title: opts.title,
        message: opts.message,
        cancelLabel: opts.cancelLabel,
        confirmLabel: opts.confirmLabel,
        confirmAction: opts.confirmAction,
        triggerElement: opts.element
      });
    }
  });
  services.syncPartialManager = createSyncPartialManager({
    onRemoteRender: (renderOptions) => instance.remoteRender(renderOptions)
  });
}
function initFrameworkDOM(services) {
  services.spriteSheetManager.ensureExists();
  initModuleLoaderFromPage(services.moduleLoader);
  loadPageScripts(document);
  loadWidgetScripts(document);
  registerPartialInstancesFromDOM(document);
  _executeConnectedForPartials(document);
  const appRoot = document.querySelector("#app");
  if (appRoot) {
    services.domBinder?.bind(appRoot);
    services.syncPartialManager?.bind(appRoot);
  }
  services.formStateManager?.scanAndTrackForms();
  services.hookManager.processQueue();
  services.hookManager.emit(HookEvent.FRAMEWORK_READY, {
    version: "1.0.0",
    loadTime: performance.now(),
    timestamp: Date.now()
  });
  services.hookManager.emit(HookEvent.PAGE_VIEW, {
    url: window.location.href,
    title: document.title,
    referrer: document.referrer,
    isInitialLoad: true,
    timestamp: Date.now()
  });
}
function createInitialServices() {
  const hookManager = createHookManager();
  return {
    hookManager,
    spriteSheetManager: createSpriteSheetManager(),
    moduleLoader: createModuleLoader(),
    linkHeaderParser: createLinkHeaderParser(),
    modalManager: createModalManager({ hookManager }),
    helperRegistry: globalHelperRegistry,
    networkStatus: null,
    a11yAnnouncer: null,
    formStateManager: null,
    loader: null,
    errorDisplay: null,
    fetchClient: null,
    router: null,
    remoteRenderer: null,
    domBinder: null,
    syncPartialManager: null,
    globalConfig: {},
    moduleConfigCache: null
  };
}
function buildFrameworkInstance(services) {
  const instance = {
    /** Gets the set of loaded module script URLs. */
    get loadedModuleScripts() {
      return services.moduleLoader.getLoadedModules();
    },
    /** Gets whether a navigation is currently in progress. */
    get navigating() {
      return services.router?.isNavigating() ?? false;
    },
    /** No-op setter retained for backwards compatibility. */
    set navigating(_value) {
    },
    /** Gets the loader bar element from the DOM. */
    get loaderElement() {
      return document.getElementById("ppf-loader-bar");
    },
    /** No-op setter retained for backwards compatibility. */
    set loaderElement(_value) {
    },
    /** Gets the current AbortController from the fetch client. */
    get currentAbortController() {
      return services.fetchClient?.getController() ?? null;
    },
    /** No-op setter retained for backwards compatibility. */
    set currentAbortController(_value) {
    },
    /** Gets the global configuration options. */
    get globalConfig() {
      return services.globalConfig;
    },
    /** Sets the global configuration and updates the router. */
    set globalConfig(value) {
      services.globalConfig = value;
      services.router?.setConfig({ beforeNavigate: value.beforeNavigate, afterNavigate: value.afterNavigate });
    },
    hooks: services.hookManager.api,
    emitHook: services.hookManager.emit,
    registerHelper: services.helperRegistry.register.bind(services.helperRegistry),
    /** Gets whether the browser is currently online. */
    get isOnline() {
      return services.networkStatus?.isOnline ?? navigator.onLine;
    },
    getModuleConfig: (moduleName) => getModuleConfig(services, moduleName),
    init(options = {}) {
      initFrameworkServices(services, options, instance);
      initFrameworkDOM(services);
    },
    async navigateTo(targetUrl, evt, options = {}) {
      if (!services.router) {
        console.warn("PPFramework: navigateTo called before init()");
        return;
      }
      return services.router.navigateTo(targetUrl, evt, options);
    },
    async remoteRender(options) {
      if (!services.remoteRenderer) {
        console.warn("PPFramework: remoteRender called before init()");
        return;
      }
      return services.remoteRenderer.render(options);
    },
    dispatchAction: (actionName, element, event) => {
      dispatchActionImpl(actionName, element, event);
    },
    buildRemoteUrl,
    addFragmentQuery,
    isSameDomain,
    assetSrc: (src, moduleName) => resolveAssetSrc(src, moduleName),
    createLoaderIndicator(color) {
      if (services.loader) {
        services.loader.destroy();
      }
      services.loader = createLoaderUI({ colour: color });
    },
    toggleLoader(isVisible) {
      if (!services.loader) {
        return;
      }
      if (isVisible) {
        services.loader.show();
      } else {
        services.loader.hide();
      }
    },
    updateProgressBar(percentValue) {
      services.loader?.setProgress(percentValue);
    },
    displayError(message) {
      services.errorDisplay?.show(message);
    },
    loadModuleScripts(doc2) {
      services.moduleLoader.loadFromDocument(doc2);
    },
    patchPartial(htmlString, cssSelector) {
      services.remoteRenderer?.patchPartial(htmlString, cssSelector);
    },
    async openModalIfAvailable(options) {
      return services.modalManager.openIfAvailable(options);
    },
    executeHelper: (event, actionString, element) => {
      executeHelperImpl(event, actionString, element);
    },
    executeServerHelper(name, args, triggerElement, event) {
      const stringArgs = args.map((a) => String(a));
      void services.helperRegistry.execute(name, triggerElement, event, stringArgs);
    }
  };
  return instance;
}
function createPPFramework() {
  const services = createInitialServices();
  return buildFrameworkInstance(services);
}
function getModuleConfig(services, moduleName) {
  if (services.moduleConfigCache === null) {
    const configEl = document.getElementById("pk-module-config");
    if (configEl?.textContent) {
      try {
        services.moduleConfigCache = JSON.parse(configEl.textContent);
      } catch {
        console.warn("[PPFramework] Failed to parse module config JSON");
        services.moduleConfigCache = {};
      }
    } else {
      services.moduleConfigCache = {};
    }
  }
  return services.moduleConfigCache[moduleName] ?? null;
}
function dispatchActionImpl(actionName, element, event) {
  if (element.tagName === "FORM" || element.type === "submit") {
    event?.preventDefault();
  }
  const actionFn = getActionFunction(actionName);
  if (actionFn) {
    const form = element.closest("form");
    let formData2;
    if (form) {
      const fd = new FormData(form);
      formData2 = {};
      for (const [key, value] of fd.entries()) {
        formData2[key] = value;
      }
    }
    const args = formData2 ? [formData2] : [];
    const result = actionFn(...args);
    handleAction(result, element, event).catch((err) => {
      console.error(`[PPFramework] dispatchAction failed for "${actionName}":`, err);
    });
    return;
  }
  console.warn(`[PPFramework] Action function "${actionName}" not found in registry.`);
}
function executeHelperImpl(event, actionString, element) {
  event.preventDefault();
  const match = actionString.match(/(\w+)(?:\((.*)\))?/);
  if (!match) {
    console.warn(`PPFramework: Could not parse helper action string: "${actionString}"`);
    return;
  }
  const helperName = match[1];
  const paramsStr = match[2] ?? "";
  const args = paramsStr.split(",").map((p) => p.trim()).filter((p) => p);
  void globalHelperRegistry.execute(helperName, element, event, args);
}
function resolveAssetSrc(src, moduleName) {
  if (!src || src.startsWith("http://") || src.startsWith("https://") || src.startsWith("data:") || src.startsWith("/")) {
    return src;
  }
  let resolvedSrc = src;
  if (src.startsWith("@/") && moduleName) {
    resolvedSrc = `${moduleName}/${src.slice(2)}`;
  }
  return `/_piko/assets/${resolvedSrc}`;
}
const PPFramework = createPPFramework();
document.addEventListener("DOMContentLoaded", () => {
});
const findForm = (element, helperName, formSelector) => {
  if (formSelector) {
    const form = document.querySelector(formSelector);
    if (!form) {
      console.error(`PPFramework Helper '${helperName}' failed: Could not find any form matching the selector "${formSelector}".`, {
        triggeringElement: element
      });
      return null;
    }
    return form;
  }
  const parentForm = element.closest("form");
  if (!parentForm) {
    console.warn(`helpers.${helperName}() was used without a selector, but no parent form could be found.`, {
      triggeringElement: element
    });
    return null;
  }
  return parentForm;
};
const submitFormHelper = (element, _event, ...args) => {
  const formSelector = args[0];
  const form = findForm(element, "submitForm", formSelector);
  form?.requestSubmit();
};
const resetFormHelper = (element, _event, ...args) => {
  const formSelector = args[0];
  const form = findForm(element, "resetForm", formSelector);
  form?.reset();
};
RegisterHelper("submitForm", submitFormHelper);
RegisterHelper("submitModalForm", submitFormHelper);
RegisterHelper("resetForm", resetFormHelper);
const redirectHelper = (_element, _event, ...args) => {
  const url = args[0];
  const replace = args[1] === "true";
  if (!url) {
    console.error("The 'redirect' helper requires a URL string as its first argument.");
    return;
  }
  queueMicrotask(() => {
    if (replace) {
      window.location.replace(url);
    } else {
      window.location.assign(url);
    }
  });
};
RegisterHelper("redirect", redirectHelper);
const emitEventHelper = (element, _event, ...args) => {
  const eventName = args[0];
  if (!eventName || typeof eventName !== "string") {
    console.error("The 'emitEvent' helper requires a non-empty event name string as its first argument.", {
      triggeringElement: element
    });
    return;
  }
  const finalOptions = {
    bubbles: true,
    composed: true,
    detail: args[1]
  };
  element.dispatchEvent(new CustomEvent(eventName, finalOptions));
};
RegisterHelper("emitEvent", emitEventHelper);
const dispatchEventHelper = (_element, _event, ...args) => {
  const eventName = args[0];
  if (!eventName) {
    console.error("The 'dispatchEvent' helper requires an event name as its first argument.");
    return;
  }
  window.dispatchEvent(new CustomEvent(eventName));
};
RegisterHelper("dispatchEvent", dispatchEventHelper);
_initCleanupObserver();
PPFramework.init();
piko._markReady();
export {
  ActionBuilder,
  _createPKContext,
  createRefs as _createRefs,
  _initCleanupObserver,
  _registerLifecycle,
  _runPageCleanup,
  bus,
  createActionBuilder,
  getGlobalPageContext,
  piko,
  registerActionFunction
};
//# sourceMappingURL=ppframework.core.es.js.map
