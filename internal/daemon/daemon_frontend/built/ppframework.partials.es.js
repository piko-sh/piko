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
function getOwnedAttributes(el) {
  const attr = el.getAttribute("pk-own-attrs");
  if (!attr) {
    return void 0;
  }
  return attr.split(",").map((s) => s.trim()).filter((s) => s.length > 0);
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
  const formData = new FormData(form);
  const data = {};
  for (const key of new Set(formData.keys())) {
    data[key] = formData.getAll(key);
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
  return async (formData = null) => {
    const form = containerEl.closest("form");
    const gatheredData = formData ?? gatherFormData(form);
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
  const { containerEl, debounceTimers } = binding;
  const updateServer = createUpdateServer(binding);
  containerEl.addEventListener("input", (_event) => {
    clearTimeout(debounceTimers.get(containerEl));
    debounceTimers.set(containerEl, setTimeout(() => void updateServer(), INPUT_DEBOUNCE_MS));
  });
  containerEl.addEventListener("change", (event) => {
    const target = event.target;
    if (!SYNC_TRIGGER_TAGS.includes(target.tagName)) {
      return;
    }
    if (target.tagName === "INPUT" && target.type === "text") {
      return;
    }
    clearTimeout(debounceTimers.get(containerEl));
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
function createPartialsCapability() {
  return {
    createSyncPartialManager(callbacks) {
      return createSyncPartialManager(callbacks);
    }
  };
}
waitForPiko("partials").then((pk) => {
  pk._registerCapability("partials", createPartialsCapability);
}).catch((err) => {
  console.error("[piko/partials] Failed to initialise:", err);
});
//# sourceMappingURL=ppframework.partials.es.js.map
