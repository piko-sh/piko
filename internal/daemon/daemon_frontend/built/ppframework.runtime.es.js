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
    delay = 1e3,
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
      const waitTime = calculateDelay(attempt, { backoff, delay, maxDelay });
      await sleep(waitTime);
    }
  }
  throw lastError ?? new Error("Retry failed");
}
async function withLoading(target, operation, options = {}) {
  return loading(target, operation(), options);
}
function debounceAsync(handler, delay) {
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
      }, delay);
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
function throttleAsync(handler, delay) {
  let lastCall = 0;
  let pendingPromise = null;
  return async (...args) => {
    const now = Date.now();
    if (now - lastCall >= delay) {
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
  function partial(_nameOrElement) {
    return null;
  }
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
    function data(_selector) {
      return null;
    }
    form2.data = data;
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
    function reload(_name, _options) {
      return Promise.resolve();
    }
    partials2.reload = reload;
    function render(options) {
      return PPFramework.remoteRender(options);
    }
    partials2.render = render;
  })(piko2.partials || (piko2.partials = {}));
  ((sse2) => {
    function subscribe(_url, _callback) {
      return null;
    }
    sse2.subscribe = subscribe;
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
    function init() {
    }
    autoRefreshObserver2.init = init;
    function stopAll() {
    }
    autoRefreshObserver2.stopAll = stopAll;
    function getActiveCount() {
      return 0;
    }
    autoRefreshObserver2.getActiveCount = getActiveCount;
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
    function track(_eventName, _params) {
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
    function create(colour) {
      PPFramework.createLoaderIndicator(colour);
    }
    loader2.create = create;
  })(piko2.loader || (piko2.loader = {}));
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
  function _emitHook(hookEvent, payload) {
    PPFramework._emitHook(hookEvent, payload);
  }
  piko2._emitHook = _emitHook;
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
function _executeConnectedForPartials(container) {
  const partials = container.querySelectorAll("[partial]");
  for (const partial of partials) {
    if (partialLifecycleState.has(partial) && !connectedPartials.has(partial)) {
      _executeConnected(partial);
    }
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
  for (const partial of partials) {
    _executeDisconnected(partial);
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
function isActionDescriptor(value) {
  return value !== null && typeof value === "object" && typeof value.action === "string";
}
const actionFunctionRegistry = /* @__PURE__ */ new Map();
function registerActionFunction(name, actionFactory) {
  actionFunctionRegistry.set(name, actionFactory);
}
function getActionFunction(name) {
  return actionFunctionRegistry.get(name);
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
    loadFromDocument(doc) {
      const moduleScripts = doc.querySelectorAll('script[type="module"]');
      moduleScripts.forEach((scriptEl) => {
        const src = scriptEl.getAttribute("src");
        if (src) {
          loadScript(src);
        }
      });
    },
    async loadFromDocumentAsync(doc) {
      const loadPromises = [];
      let failCount = 0;
      const moduleScripts = doc.querySelectorAll('script[type="module"]');
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
const capabilities = /* @__PURE__ */ new Map();
const pendingCallbacks = /* @__PURE__ */ new Map();
function _registerCapability(name, impl) {
  capabilities.set(name, impl);
  const callbacks = pendingCallbacks.get(name);
  if (callbacks) {
    pendingCallbacks.delete(name);
    for (const cb of callbacks) {
      cb(impl);
    }
  }
}
function _getCapability(name) {
  return capabilities.get(name);
}
function _onCapabilityReady(name, callback) {
  const existing = capabilities.get(name);
  if (existing !== void 0) {
    callback(existing);
    return;
  }
  const queue = pendingCallbacks.get(name);
  if (queue) {
    queue.push(callback);
  } else {
    pendingCallbacks.set(name, [callback]);
  }
}
function createFormData(form) {
  const fd = new FormData(form);
  const obj = {};
  for (const [key, value] of fd.entries()) {
    obj[key] = value;
  }
  return {
    toObject: () => ({ ...obj }),
    toJSON: () => JSON.stringify(obj)
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
      return createFormData(form);
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
      return createFormData(form).toObject();
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
      return createFormData(form).toJSON();
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
function executeViaCapability(descriptor, el, event) {
  const api = _getCapability("actions");
  if (api) {
    return api.handleAction(descriptor, el, event);
  }
  return void 0;
}
function dispatchIfActionDescriptor(result, el, event, errorPrefix, fnName) {
  if (isActionDescriptor(result)) {
    executeViaCapability(result, el, event)?.catch((err) => {
      console.error(`${errorPrefix} Action execution failed for "${fnName}":`, err);
    });
    return;
  }
  if (result instanceof Promise) {
    result.then((resolved) => {
      if (isActionDescriptor(resolved)) {
        return executeViaCapability(resolved, el, event);
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
    callbacks.onNavigate(href, event);
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
function registerPartialInstancesFromDOM(doc) {
  const partials = doc.querySelectorAll("[partial][data-partial-name]");
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
function loadPageScripts(doc) {
  const pageScriptMetas = doc.querySelectorAll('meta[name="pk-script"]');
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
  oldAppRoot.innerHTML = newAppRoot.innerHTML;
  handleScrollPosition(scrollOptions);
  deps.bindDOM(oldAppRoot);
  deps.moduleLoader.loadFromDocument(parsedDocument);
  loadPageScripts(parsedDocument);
  registerPartialInstancesFromDOM(parsedDocument);
  _executeConnectedForPartials(oldAppRoot);
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
function initFrameworkServices(services, options, _instance) {
  services.globalConfig = options;
  services.errorDisplay = createErrorDisplay();
  services.networkStatus = createNetworkStatus({ hookManager: services.hookManager });
  const bindDOM = (root) => {
    services.domBinder?.bind(root);
  };
  const pageLoadDeps = {
    spriteSheetManager: services.spriteSheetManager,
    bindDOM,
    moduleLoader: services.moduleLoader
  };
  _onCapabilityReady("navigation", (factory) => {
    const createNav = factory;
    services.navigation = createNav({
      onPageLoad: (doc, url, scroll) => handlePageLoad(pageLoadDeps, doc, url, scroll),
      hookManager: services.hookManager,
      errorDisplay: services.errorDisplay,
      formStateManager: null,
      moduleLoader: services.moduleLoader,
      spriteSheetManager: services.spriteSheetManager,
      linkHeaderParser: services.linkHeaderParser,
      getPageContext: () => getGlobalPageContext(),
      onDOMUpdated: bindDOM,
      runPageCleanup: _runPageCleanup,
      executeConnectedForPartials: _executeConnectedForPartials,
      executeBeforeRender: (id) => {
      },
      executeAfterRender: (id) => {
      },
      executeUpdated: (id) => {
      },
      addFragmentQuery,
      isSameDomain,
      buildRemoteUrl
    });
  });
  _onCapabilityReady("actions", (factory) => {
    const createActions = factory;
    services.actions = createActions({
      hookManager: services.hookManager,
      formStateManager: null,
      helperRegistry: services.helperRegistry
    });
  });
  services.domBinder = createDOMBinder(services.helperRegistry, {
    onNavigate: (url, _event) => {
      if (services.navigation) {
        void services.navigation.navigateTo(url);
      } else {
        window.location.href = url;
      }
    },
    onOpenModal: (opts) => {
      opts.element.dispatchEvent(new CustomEvent("pk-open-modal", {
        bubbles: true,
        detail: {
          selector: opts.selector,
          params: opts.params,
          title: opts.title,
          message: opts.message,
          cancelLabel: opts.cancelLabel,
          confirmLabel: opts.confirmLabel,
          confirmAction: opts.confirmAction
        }
      }));
    }
  });
}
function initFrameworkDOM(services) {
  services.spriteSheetManager.ensureExists();
  initModuleLoaderFromPage(services.moduleLoader);
  loadPageScripts(document);
  registerPartialInstancesFromDOM(document);
  _executeConnectedForPartials(document);
  const appRoot = document.querySelector("#app");
  if (appRoot) {
    services.domBinder?.bind(appRoot);
  }
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
    helperRegistry: globalHelperRegistry,
    networkStatus: null,
    errorDisplay: null,
    navigation: null,
    actions: null,
    domBinder: null,
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
      return services.navigation?.isNavigating() ?? false;
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
    /** Gets the current AbortController -- no longer directly accessible. */
    get currentAbortController() {
      return null;
    },
    /** No-op setter retained for backwards compatibility. */
    set currentAbortController(_value) {
    },
    /** Gets the global configuration options. */
    get globalConfig() {
      return services.globalConfig;
    },
    /** Sets the global configuration options. */
    set globalConfig(value) {
      services.globalConfig = value;
    },
    hooks: services.hookManager.api,
    _emitHook(event, payload) {
      services.hookManager.emit(event, payload);
    },
    registerHelper: services.helperRegistry.register.bind(services.helperRegistry),
    /** Gets whether the browser is currently online. */
    get isOnline() {
      return services.networkStatus?.isOnline ?? navigator.onLine;
    },
    getModuleConfig: (moduleName) => getModuleConfig(services, moduleName),
    init(options = {}) {
      initFrameworkServices(services, options);
      initFrameworkDOM(services);
    },
    async navigateTo(targetUrl, evt, options = {}) {
      if (!services.navigation) {
        window.location.href = targetUrl;
        return;
      }
      return services.navigation.navigateTo(targetUrl, evt, options);
    },
    async remoteRender(options) {
      if (!services.navigation) {
        console.warn("PPFramework: remoteRender requires navigation capability");
        return;
      }
      return services.navigation.remoteRender(options);
    },
    dispatchAction: (actionName, element, event) => {
      dispatchActionImpl(actionName, element, event);
    },
    buildRemoteUrl,
    addFragmentQuery,
    isSameDomain,
    assetSrc: (src, moduleName) => resolveAssetSrc(src, moduleName),
    createLoaderIndicator(colour) {
      services.navigation?.createLoaderIndicator(colour);
    },
    toggleLoader(isVisible) {
      services.navigation?.toggleLoader(isVisible);
    },
    updateProgressBar(percentValue) {
      services.navigation?.updateProgressBar(percentValue);
    },
    displayError(message) {
      services.errorDisplay?.show(message);
    },
    loadModuleScripts(doc) {
      services.moduleLoader.loadFromDocument(doc);
    },
    patchPartial(htmlString, cssSelector) {
      services.navigation?.patchPartial(htmlString, cssSelector);
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
    let formData;
    if (form) {
      const fd = new FormData(form);
      formData = {};
      for (const [key, value] of fd.entries()) {
        formData[key] = value;
      }
    }
    const args = formData ? [formData] : [];
    const result = actionFn(...args);
    const actionsApi = _getCapability("actions");
    if (actionsApi) {
      actionsApi.handleAction(result, element, event).catch((err) => {
        console.error(`[PPFramework] dispatchAction failed for "${actionName}":`, err);
      });
    }
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
function upgradeFromShim() {
  const shimData = window.__pikoShimData__;
  _initCleanupObserver();
  if (shimData) {
    shimData.helpers.forEach((fn, name) => {
      RegisterHelper(name, fn);
    });
    shimData.capabilities.forEach((impl, name) => {
      _registerCapability(name, impl);
    });
    shimData.actionRegistry.forEach((factory, name) => {
      registerActionFunction(name, factory);
    });
  }
  removeShimLinkHandlers();
  PPFramework.init();
  if (shimData) {
    shimData.hookListeners.forEach((listeners2, event) => {
      listeners2.forEach((cb) => {
        PPFramework.hooks.on(event, cb);
      });
    });
    const pageContext = getGlobalPageContext();
    shimData.globalExports.forEach((fn, name) => {
      pageContext.setExports({ [name]: fn });
    });
  }
  upgradePikoNamespace();
}
function removeShimLinkHandlers() {
  document.querySelectorAll("a[piko\\:a]").forEach((link) => {
    const nav = link.__pkNav;
    if (nav) {
      link.removeEventListener("click", nav);
      delete link.__pkNav;
    }
  });
}
function upgradePikoNamespace() {
  const piko2 = window.piko;
  piko2.bus = bus;
  piko2.nav = {
    navigate,
    back: goBack,
    forward: goForward,
    go,
    current: currentRoute,
    buildUrl,
    updateQuery,
    guard: registerNavigationGuard,
    matchPath,
    extractParams
  };
  piko2.ui = { loading, withLoading, withRetry };
  piko2.event = { dispatch, listen, listenOnce, waitFor: waitForEvent };
  piko2.timing = {
    debounce,
    throttle,
    debounceAsync,
    throttleAsync,
    timeout,
    poll,
    nextFrame,
    waitFrames
  };
  piko2.util = {
    whenVisible,
    withAbortSignal,
    watchMutations,
    whenIdle,
    deferred,
    once
  };
  piko2.trace = {
    enable: trace.enable,
    disable: trace.disable,
    isEnabled: trace.isEnabled,
    clear: trace.clear,
    getEntries: trace.getEntries,
    getMetrics: trace.getMetrics,
    log: traceLog
  };
  piko2.network = { isOnline: () => PPFramework.isOnline };
  piko2.loader = {
    toggle: (visible) => PPFramework.toggleLoader(visible),
    progress: (percent) => PPFramework.updateProgressBar(percent),
    error: (message) => PPFramework.displayError(message),
    create: (colour) => PPFramework.createLoaderIndicator(colour)
  };
  piko2.context = { get: getGlobalPageContext };
  piko2.hooks = PPFramework.hooks;
  piko2.registerHelper = (name, fn) => RegisterHelper(name, fn);
  piko2._registerCapability = _registerCapability;
  piko2.getModuleConfig = PPFramework.getModuleConfig;
  piko2._emitHook = (event, payload) => PPFramework._emitHook(event, payload);
}
upgradeFromShim();
//# sourceMappingURL=ppframework.runtime.es.js.map
