const MAX_WAIT_MS = 5e3;
const POLL_INTERVAL_MS = 10;
function waitForPiko(extensionName) {
  return new Promise((resolve, reject) => {
    const startTime = Date.now();
    const check = () => {
      if (typeof window !== "undefined" && window.piko) {
        window.piko.ready(() => resolve(window.piko));
        return;
      }
      if (Date.now() - startTime > MAX_WAIT_MS) {
        reject(new Error(`[piko/${extensionName}] Timed out waiting for piko core.`));
        return;
      }
      setTimeout(check, POLL_INTERVAL_MS);
    };
    check();
  });
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
const NavigationHookEvent = {
  NAVIGATION_START: "navigation:start",
  NAVIGATION_COMPLETE: "navigation:complete",
  NAVIGATION_ERROR: "navigation:error",
  PAGE_VIEW: "page:view",
  PARTIAL_RENDER: "partial:render"
};
const PROGRESS_MAX$1 = 100;
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
  deps.hookManager?.emit(NavigationHookEvent.NAVIGATION_ERROR, {
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
  deps.hookManager?.emit(NavigationHookEvent.NAVIGATION_COMPLETE, {
    url: ctx.targetUrl,
    previousUrl: ctx.previousUrl,
    timestamp: Date.now(),
    duration
  });
  deps.hookManager?.emit(NavigationHookEvent.PAGE_VIEW, {
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
  deps.hookManager?.emit(NavigationHookEvent.NAVIGATION_START, {
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
    hash: !ctx.isPopNavigation || options.restoreScrollY === void 0 ? hash : void 0
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
      onProgress: (loaded, total) => loader.setProgress(loaded / total * PROGRESS_MAX$1)
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
const LOADER_FADE_MS = 300;
const PROGRESS_MIN = 0;
const PROGRESS_MAX = 100;
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
      if (pct > PROGRESS_MAX) {
        pct = PROGRESS_MAX;
      }
      el.style.display = "block";
      el.style.width = `${pct}%`;
    },
    destroy() {
      el.remove();
    }
  };
}
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
  const { target, sourceEl, patchMethod, onDOMUpdated, domOps, lifecycle } = ctx;
  const { patchLocation } = target;
  if (patchLocation.hasAttribute("partial")) {
    lifecycle.executeBeforeRender?.(patchLocation);
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
    lifecycle.executeAfterRender?.(patchLocation);
    lifecycle.executeUpdated?.(patchLocation, { patchMethod });
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
async function reloadCachedModules(parsedDoc, moduleLoader, getPageCtx) {
  const moduleScripts = parsedDoc.querySelectorAll('script[type="module"]');
  const cachedScripts = [];
  for (const scriptEl of Array.from(moduleScripts)) {
    const src = scriptEl.getAttribute("src");
    if (src && moduleLoader.hasLoaded(src)) {
      cachedScripts.push(src);
    }
  }
  await moduleLoader.loadFromDocumentAsync(parsedDoc);
  if (cachedScripts.length > 0 && getPageCtx) {
    const pageContext = getPageCtx();
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
      domOps: deps.domOps,
      lifecycle: deps.lifecycle
    });
    const patchLocationId = target.querySelector || target.patchLocation.id || "unknown";
    deps.hookManager?.emit(NavigationHookEvent.PARTIAL_RENDER, {
      src: options.src,
      patchLocation: patchLocationId,
      timestamp: Date.now()
    });
    deps.lifecycle.executeConnectedForPartials?.(target.patchLocation);
  }
}
function createRemoteRenderer(deps) {
  const { moduleLoader, spriteSheetManager, linkHeaderParser, onDOMUpdated, hookManager, getPageContext, executeBeforeRender, executeAfterRender, executeUpdated, executeConnectedForPartials } = deps;
  const domOps = deps.domOps ?? browserDOMOperations;
  const windowOps = deps.windowOps ?? browserWindowOperations;
  const http = deps.http ?? browserHTTPOperations;
  const fetchCtx = { linkHeaderParser, http, windowOps };
  const lifecycle = { executeBeforeRender, executeAfterRender, executeUpdated, executeConnectedForPartials };
  const renderTargetDeps = { onDOMUpdated, domOps, hookManager, lifecycle };
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
    await reloadCachedModules(parsedDoc, moduleLoader, getPageContext);
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
    hookManager?.emit(NavigationHookEvent.PARTIAL_RENDER, {
      src: "inline",
      patchLocation: cssSelector,
      timestamp: Date.now()
    });
  }
  return { render, patchPartial };
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
function _setNavigateAdapter(navigateFn) {
}
function createNavigationCapability(services) {
  const fetchClient = createFetchClient();
  let loader = createLoaderUI({ colour: "#29e" });
  const a11yAnnouncer = createA11yAnnouncer();
  const remoteRenderer = createRemoteRenderer({
    moduleLoader: services.moduleLoader,
    spriteSheetManager: services.spriteSheetManager,
    linkHeaderParser: services.linkHeaderParser,
    onDOMUpdated: services.onDOMUpdated,
    hookManager: services.hookManager,
    getPageContext: services.getPageContext,
    executeBeforeRender: services.executeBeforeRender,
    executeAfterRender: services.executeAfterRender,
    executeUpdated: services.executeUpdated,
    executeConnectedForPartials: services.executeConnectedForPartials
  });
  const router = createRouter({
    fetchClient,
    loader,
    errorDisplay: services.errorDisplay,
    onPageLoad: services.onPageLoad,
    hookManager: services.hookManager,
    formStateManager: services.formStateManager ?? void 0,
    a11yAnnouncer
  });
  _setNavigateAdapter(router.navigateTo.bind(router));
  return {
    navigateTo: router.navigateTo.bind(router),
    remoteRender: remoteRenderer.render.bind(remoteRenderer),
    patchPartial: remoteRenderer.patchPartial.bind(remoteRenderer),
    isNavigating: router.isNavigating.bind(router),
    toggleLoader(visible) {
      if (visible) {
        loader.show();
      } else {
        loader.hide();
      }
    },
    updateProgressBar(percent) {
      loader.setProgress(percent);
    },
    createLoaderIndicator(colour) {
      loader.destroy();
      loader = createLoaderUI({ colour });
    },
    destroy: router.destroy.bind(router)
  };
}
waitForPiko("navigation").then((pk) => {
  pk._registerCapability("navigation", createNavigationCapability);
}).catch((err) => {
  console.error("[piko/navigation] Failed to initialise:", err);
});
//# sourceMappingURL=ppframework.navigation.es.js.map
