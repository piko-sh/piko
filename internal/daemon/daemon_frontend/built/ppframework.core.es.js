const capabilities = /* @__PURE__ */ new Map();
const capabilityPending = /* @__PURE__ */ new Map();
function _registerCapability(name, impl) {
  capabilities.set(name, impl);
  const cbs = capabilityPending.get(name);
  if (cbs) {
    capabilityPending.delete(name);
    for (const cb of cbs) {
      cb(impl);
    }
  }
}
function _getCapability(name) {
  return capabilities.get(name);
}
function _hasCapability(name) {
  return capabilities.has(name);
}
function _onCapabilityReady(name, cb) {
  const existing = capabilities.get(name);
  if (existing !== void 0) {
    cb(existing);
    return;
  }
  const queue = capabilityPending.get(name);
  if (queue) {
    queue.push(cb);
  } else {
    capabilityPending.set(name, [cb]);
  }
}
function _clearCapabilities() {
  capabilities.clear();
  capabilityPending.clear();
}
const helpers = /* @__PURE__ */ new Map();
function registerHelper(name, fn) {
  helpers.set(name, fn);
}
const hookListeners = /* @__PURE__ */ new Map();
function hooksOn(event, cb) {
  let set = hookListeners.get(event);
  if (!set) {
    set = /* @__PURE__ */ new Set();
    hookListeners.set(event, set);
  }
  set.add(cb);
  return () => {
    set.delete(cb);
  };
}
function hooksOff(event, cb) {
  hookListeners.get(event)?.delete(cb);
}
function hooksClear(event) {
  if (event) {
    hookListeners.delete(event);
  } else {
    hookListeners.clear();
  }
}
function emitHook(event, payload) {
  hookListeners.get(event)?.forEach((cb) => {
    try {
      cb(payload);
    } catch (e) {
      console.error("[piko] Hook error:", e);
    }
  });
}
const globalExports = /* @__PURE__ */ new Map();
const scopedExports = /* @__PURE__ */ new Map();
const partialInstances = /* @__PURE__ */ new Map();
function setExports(fns, scopeId) {
  for (const [name, fn] of Object.entries(fns)) {
    globalExports.set(name, fn);
    if (scopeId) {
      let scoped = scopedExports.get(scopeId);
      if (!scoped) {
        scoped = /* @__PURE__ */ new Map();
        scopedExports.set(scopeId, scoped);
      }
      scoped.set(name, fn);
    }
  }
}
function getFunction(name) {
  return globalExports.get(name);
}
function hasFunction(name) {
  return globalExports.has(name);
}
function getScopedFunction(name, scopeId) {
  const firstScope = scopeId.split(/\s+/)[0];
  return scopedExports.get(firstScope)?.get(name);
}
function getExportedFunctions() {
  return Array.from(globalExports.keys());
}
function clearPageContext() {
  globalExports.clear();
  scopedExports.clear();
}
function registerPartialInstance(partialName, partialId) {
  let ids = partialInstances.get(partialName);
  if (!ids) {
    ids = [];
    partialInstances.set(partialName, ids);
  }
  if (!ids.includes(partialId)) {
    ids.push(partialId);
  }
}
async function loadModule(url) {
  await import(
    /* @vite-ignore */
    url
  );
}
function getGlobalPageContext() {
  return {
    setExports,
    getFunction,
    hasFunction,
    getScopedFunction,
    getExportedFunctions,
    clear: clearPageContext,
    registerPartialInstance,
    loadModule
  };
}
function createRefs(scope = document.body) {
  const partialId = scope.getAttribute("partial") ?? scope.closest("[partial]")?.getAttribute("partial");
  return new Proxy({}, {
    get(_, name) {
      if (typeof name !== "string" || name === "then") {
        return void 0;
      }
      let el = null;
      if (partialId) {
        el = document.querySelector(`[partial~="${partialId}"][p-ref="${name}"]`);
      }
      el ??= scope.querySelector(`[p-ref="${name}"]`);
      return el;
    }
  });
}
const lifecycleCallbacks = /* @__PURE__ */ new WeakMap();
function _addLifecycleCallback(scope, hookName, cb) {
  let state = lifecycleCallbacks.get(scope);
  if (!state) {
    state = {};
    lifecycleCallbacks.set(scope, state);
  }
  const bucket = state[hookName] ?? [];
  bucket.push(cb);
  state[hookName] = bucket;
}
const elementCleanups = /* @__PURE__ */ new WeakMap();
const pageCleanups = [];
function onCleanup(fn, scope) {
  if (scope) {
    let arr = elementCleanups.get(scope);
    if (!arr) {
      arr = [];
      elementCleanups.set(scope, arr);
    }
    arr.push(fn);
  } else {
    pageCleanups.push(fn);
  }
}
function _createPKContext(scope) {
  return {
    refs: createRefs(scope),
    createRefs: (s) => createRefs(s ?? scope),
    onConnected: (cb) => _addLifecycleCallback(scope, "onConnected", cb),
    onDisconnected: (cb) => _addLifecycleCallback(scope, "onDisconnected", cb),
    onBeforeRender: (cb) => _addLifecycleCallback(scope, "onBeforeRender", cb),
    onAfterRender: (cb) => _addLifecycleCallback(scope, "onAfterRender", cb),
    onUpdated: (cb) => _addLifecycleCallback(scope, "onUpdated", cb),
    onCleanup: (fn) => onCleanup(fn, scope)
  };
}
const actionRegistry = /* @__PURE__ */ new Map();
class ActionBuilder {
  /**
   * Creates a new ActionBuilder.
   *
   * @param actionName - Server action name.
   * @param actionArgs - Arguments for the action.
   */
  constructor(actionName, actionArgs) {
    this.action = actionName;
    this.args = actionArgs;
  }
}
function createActionBuilder(name, args) {
  if (typeof args.toObject === "function") {
    args = args.toObject();
  }
  return new ActionBuilder(name, [args]);
}
function action(name, ...args) {
  return new ActionBuilder(name, args);
}
function registerActionFunction(name, factory) {
  actionRegistry.set(name, factory);
}
function getActionFunction(name) {
  return actionRegistry.get(name);
}
function isActionDescriptor(value) {
  return value !== null && typeof value === "object" && typeof value.action === "string";
}
registerHelper("submitForm", (el) => {
  const form = el.closest("form");
  if (form) {
    form.requestSubmit();
  }
});
registerHelper("submitModalForm", (el) => {
  const form = el.closest("form");
  if (form) {
    form.requestSubmit();
  }
});
registerHelper("resetForm", (el) => {
  const form = el.closest("form");
  if (form) {
    form.reset();
  }
});
registerHelper("redirect", (_el, _event, ...args) => {
  const url = args[0];
  if (url) {
    window.location.href = url;
  }
});
registerHelper("emitEvent", (el, _event, ...args) => {
  const eventName = args[0];
  if (eventName) {
    el.dispatchEvent(new CustomEvent(eventName, { bubbles: true, composed: true, detail: args.slice(1) }));
  }
});
registerHelper("dispatchEvent", (_el, _event, ...args) => {
  const eventName = args[0];
  if (eventName) {
    window.dispatchEvent(new CustomEvent(eventName, { detail: args.slice(1) }));
  }
});
const BOUND_MARKER = "pk-ev-bound";
const EVENT_ATTR_PREFIX = "p-on:";
const CUSTOM_EVENT_ATTR_PREFIX = "p-event:";
const BLOCKED_SCHEMES = ["javascript:", "data:", "blob:", "file:"];
const NATIVE_SCHEMES = ["tel:", "mailto:", "sms:", "geo:"];
function bindLinks(root, onNavigate) {
  root.querySelectorAll("a[piko\\:a]").forEach((link) => {
    const existing = link.__pkNav;
    if (existing) {
      link.removeEventListener("click", existing);
    }
    const handler = (event) => {
      const href = link.getAttribute("href");
      if (!href) {
        return;
      }
      const lower = href.toLowerCase();
      if (BLOCKED_SCHEMES.some((s) => lower.startsWith(s))) {
        event.preventDefault();
        return;
      }
      if (NATIVE_SCHEMES.some((s) => lower.startsWith(s))) {
        return;
      }
      event.preventDefault();
      onNavigate(href, event);
    };
    link.__pkNav = handler;
    link.addEventListener("click", handler);
  });
}
function b64Decode(s) {
  const BASE64_BLOCK = 4;
  let std = s.replace(/-/g, "+").replace(/_/g, "/");
  const pad = (BASE64_BLOCK - std.length % BASE64_BLOCK) % BASE64_BLOCK;
  std += "=".repeat(pad);
  return atob(std);
}
function unwrapArgWithInjection(arg, el, event) {
  if (arg && typeof arg === "object") {
    const encoded = arg;
    if (encoded.t === "e") {
      return event;
    }
    if (encoded.t === "f") {
      const form = el.closest("form");
      if (!form) {
        return {};
      }
      const fd = new FormData(form);
      const obj = {};
      for (const [k, v] of fd.entries()) {
        obj[k] = v;
      }
      return obj;
    }
    return encoded.v;
  }
  return void 0;
}
function dispatchActionDescriptor(descriptor, el, event) {
  if (!isActionDescriptor(descriptor)) {
    return;
  }
  const api = _getCapability("actions");
  if (api) {
    void api.handleAction(descriptor, el, event);
  }
}
function tryInvokeActionFn(fnName, encodedArgs, el, event) {
  const actionFn = actionRegistry.get(fnName);
  if (!actionFn) {
    return false;
  }
  const args = encodedArgs?.map((a) => unwrapArgWithInjection(a, el, event)) ?? [];
  dispatchActionDescriptor(actionFn(...args), el, event);
  return true;
}
function tryInvokePageFn(fnName, encodedArgs, el, event) {
  const scopeId = el.closest("[partial]")?.getAttribute("partial") ?? "";
  const pageFn = getFunction(fnName) ?? getScopedFunction(fnName, scopeId);
  if (!pageFn) {
    return false;
  }
  const args = encodedArgs?.map((a) => a.v) ?? [];
  const result = pageFn(event, ...args);
  if (result instanceof Promise) {
    void result.then((resolved) => {
      dispatchActionDescriptor(resolved, el, event);
    });
  } else {
    dispatchActionDescriptor(result, el, event);
  }
  return true;
}
function tryInvokeHelper(fnName, encodedArgs, el, event) {
  const helper = helpers.get(fnName);
  if (!helper) {
    return false;
  }
  const args = encodedArgs?.map((a) => String(a.v)) ?? [];
  void Promise.resolve(helper(el, event, ...args));
  return true;
}
function resolveAndDispatch(payload, el, event) {
  try {
    const decoded = JSON.parse(b64Decode(payload));
    const fnName = decoded.f;
    if (tryInvokeActionFn(fnName, decoded.a, el, event)) {
      return;
    }
    if (tryInvokePageFn(fnName, decoded.a, el, event)) {
      return;
    }
    if (tryInvokeHelper(fnName, decoded.a, el, event)) {
      return;
    }
    console.warn(`[piko] Handler "${fnName}" not found.`);
  } catch (e) {
    console.error("[piko] Failed to resolve action payload:", e);
  }
}
function bindActions(root) {
  root.querySelectorAll("*").forEach((el) => {
    if (el.hasAttribute(BOUND_MARKER)) {
      return;
    }
    let hasBound = false;
    for (const { name: attrName, value: attrValue } of Array.from(el.attributes)) {
      if (attrName.startsWith(EVENT_ATTR_PREFIX) || attrName.startsWith(CUSTOM_EVENT_ATTR_PREFIX)) {
        const isCustom = attrName.startsWith(CUSTOM_EVENT_ATTR_PREFIX);
        const prefixLength = isCustom ? CUSTOM_EVENT_ATTR_PREFIX.length : EVENT_ATTR_PREFIX.length;
        const key = attrName.slice(prefixLength);
        const parts = key.split(".");
        const eventName = parts[0].trim();
        if (!eventName) {
          continue;
        }
        const modifiers = new Set(parts.slice(1));
        const boundEl = el;
        const payload = attrValue;
        el.addEventListener(eventName, (event) => {
          if (modifiers.has("self") && event.target !== event.currentTarget) {
            return;
          }
          if (modifiers.has("prevent")) {
            event.preventDefault();
          }
          if (modifiers.has("stop")) {
            event.stopPropagation();
          }
          resolveAndDispatch(payload, boundEl, event);
        });
        hasBound = true;
      }
    }
    if (el.hasAttribute("p-modal:selector")) {
      el.addEventListener("click", () => {
        el.dispatchEvent(new CustomEvent("pk-open-modal", {
          bubbles: true,
          detail: {
            selector: el.getAttribute("p-modal:selector")?.trim() ?? "",
            title: el.getAttribute("p-modal:title")?.trim() ?? "",
            message: el.getAttribute("p-modal:message")?.trim() ?? "",
            cancelLabel: el.getAttribute("p-modal:cancel_label")?.trim() ?? "",
            confirmLabel: el.getAttribute("p-modal:confirm_label")?.trim() ?? "",
            confirmAction: el.getAttribute("p-modal:confirm_action")?.trim() ?? ""
          }
        }));
      });
      hasBound = true;
    }
    if (hasBound) {
      el.setAttribute(BOUND_MARKER, "true");
    }
  });
}
let moduleConfigCache = null;
function getModuleConfig(moduleName) {
  if (moduleConfigCache === null) {
    const configEl = document.getElementById("pk-module-config");
    if (configEl?.textContent) {
      try {
        moduleConfigCache = JSON.parse(configEl.textContent);
      } catch {
        moduleConfigCache = {};
      }
    } else {
      moduleConfigCache = {};
    }
  }
  return moduleConfigCache[moduleName] ?? null;
}
let readyCallbacks = [];
let isReady = false;
const pikoNamespace = {
  ready(cb) {
    if (isReady) {
      cb();
      return;
    }
    readyCallbacks?.push(cb);
  },
  _markReady() {
    isReady = true;
    const cbs = readyCallbacks;
    readyCallbacks = null;
    if (cbs) {
      for (const cb of cbs) {
        cb();
      }
    }
  },
  registerHelper,
  getModuleConfig,
  _registerCapability,
  _emitHook: emitHook,
  hooks: {
    on: hooksOn,
    off: hooksOff,
    clear: hooksClear,
    events: {
      FRAMEWORK_READY: "framework:ready",
      PAGE_VIEW: "page:view",
      NAVIGATION_START: "navigation:start",
      NAVIGATION_COMPLETE: "navigation:complete",
      NAVIGATION_ERROR: "navigation:error",
      ACTION_START: "action:start",
      ACTION_COMPLETE: "action:complete",
      MODAL_OPEN: "modal:open",
      MODAL_CLOSE: "modal:close",
      PARTIAL_RENDER: "partial:render",
      FORM_DIRTY: "form:dirty",
      FORM_CLEAN: "form:clean",
      NETWORK_ONLINE: "network:online",
      NETWORK_OFFLINE: "network:offline",
      ERROR: "error"
    }
  },
  bus: {
    _listeners: /* @__PURE__ */ new Map(),
    on(event, cb) {
      let set = this._listeners.get(event);
      if (!set) {
        set = /* @__PURE__ */ new Set();
        this._listeners.set(event, set);
      }
      set.add(cb);
      return () => {
        set.delete(cb);
      };
    },
    off(event, cb) {
      this._listeners.get(event)?.delete(cb);
    },
    emit(event, data) {
      this._listeners.get(event)?.forEach((cb) => {
        try {
          cb(data);
        } catch (e) {
          console.error(`[pk] Bus error for "${event}":`, e);
        }
      });
    }
  },
  nav: {
    navigate(url) {
      const nav = _getCapability("navigation");
      if (nav) {
        void nav.navigateTo(url);
      } else {
        window.location.href = url;
      }
    },
    back() {
      window.history.back();
    },
    forward() {
      window.history.forward();
    }
  },
  context: {
    get: getGlobalPageContext
  }
};
if (typeof window !== "undefined") {
  window.piko = pikoNamespace;
}
if (typeof window !== "undefined") {
  window.__pikoShimData__ = {
    hookListeners,
    helpers,
    capabilities,
    capabilityPending,
    globalExports,
    scopedExports,
    partialInstances,
    actionRegistry,
    lifecycleCallbacks,
    elementCleanups,
    pageCleanups,
    readyCallbacks: () => readyCallbacks,
    moduleConfigCache: () => moduleConfigCache
  };
}
const appRoot = document.querySelector("#app");
if (appRoot) {
  bindLinks(appRoot, (url) => {
    pikoNamespace.nav.navigate(url);
  });
  bindActions(appRoot);
}
pikoNamespace._markReady();
const bus = pikoNamespace.bus;
export {
  ActionBuilder,
  _clearCapabilities,
  _createPKContext,
  createRefs as _createRefs,
  _getCapability,
  _hasCapability,
  _onCapabilityReady,
  _registerCapability,
  action,
  bus,
  createActionBuilder,
  getActionFunction,
  getGlobalPageContext,
  isActionDescriptor,
  registerActionFunction
};
//# sourceMappingURL=ppframework.core.es.js.map
