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
const FormsHookEvent = {
  FORM_DIRTY: "form:dirty",
  FORM_CLEAN: "form:clean"
};
const NO_TRACK_ATTR = "pk-no-track";
const TRACKED_ATTR = "data-pk-tracked";
const DEFERRED_SNAPSHOT_TIMEOUT_MS = 5e3;
function getFormSnapshot(form) {
  const data = new FormData(form);
  const entries = [];
  for (const [key, value] of data.entries()) {
    if (typeof value === "string") {
      entries.push([key, value]);
    }
  }
  const checkboxes = form.querySelectorAll('input[type="checkbox"], input[type="radio"]');
  for (const checkbox of checkboxes) {
    if (checkbox.name) {
      entries.push([`__checked_${checkbox.name}_${checkbox.value}`, String(checkbox.checked)]);
    }
  }
  entries.sort((a, b) => a[0].localeCompare(b[0]));
  return JSON.stringify(entries);
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
    hookManager.emit(FormsHookEvent.FORM_DIRTY, { formId: getFormId(form), timestamp: Date.now() });
  } else if (!tracked.isDirty && wasDirty) {
    hookManager.emit(FormsHookEvent.FORM_CLEAN, { formId: getFormId(form), timestamp: Date.now() });
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
    const timeout = new Promise(
      (resolve) => setTimeout(() => resolve("timeout"), DEFERRED_SNAPSHOT_TIMEOUT_MS)
    );
    const result = await Promise.race([
      Promise.all(
        Array.from(tagNames).map((name) => customElements.whenDefined(name))
      ).then(() => "defined"),
      timeout
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
      hookManager.emit(FormsHookEvent.FORM_CLEAN, { formId: getFormId(form), timestamp: Date.now() });
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
        hookManager.emit(FormsHookEvent.FORM_CLEAN, { formId: getFormId(form), timestamp: Date.now() });
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
function createFormsCapability(_services) {
  return {
    createFormStateManager(deps) {
      return createFormStateManager(deps);
    }
  };
}
waitForPiko("forms").then((pk) => {
  pk._registerCapability("forms", createFormsCapability);
}).catch((err) => {
  console.error("[piko/forms] Failed to initialise:", err);
});
//# sourceMappingURL=ppframework.forms.es.js.map
