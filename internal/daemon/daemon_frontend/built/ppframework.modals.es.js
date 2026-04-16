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
const HookEvent = {
  MODAL_OPEN: "modal:open",
  MODAL_CLOSE: "modal:close"
};
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
function createPikoHookManager(pk) {
  return {
    emit(event, payload) {
      pk._emitHook(event, payload);
    }
  };
}
function registerPkOpenModalListener(modalManager) {
  document.addEventListener("pk-open-modal", (event) => {
    const detail = event.detail;
    const triggerElement = event.target;
    void modalManager.openIfAvailable({
      selector: detail.selector,
      params: detail.params,
      title: detail.title,
      message: detail.message,
      cancelLabel: detail.cancelLabel,
      confirmLabel: detail.confirmLabel,
      confirmAction: detail.confirmAction,
      triggerElement
    });
  });
}
function showModalHelper(element, modalManager) {
  const selector = element.dataset.modalSelector;
  if (!selector) {
    console.warn('helpers.showModal() requires a "data-modal-selector" attribute.', element);
    return;
  }
  void modalManager.openIfAvailable({
    selector,
    params: /* @__PURE__ */ new Map(),
    title: element.dataset.modalTitle ?? "",
    message: element.dataset.modalMessage ?? "",
    cancelLabel: element.dataset.modalCancelMessage ?? "",
    triggerElement: element
  });
}
function resolveTargetModal(triggerElement, modalName, helperName) {
  if (typeof modalName === "string" && modalName) {
    const selector = `[modal="${modalName}"]`;
    const found = document.querySelector(selector);
    if (!found) {
      console.warn(`${helperName}: Could not find any modal with selector: ${selector}`);
    }
    return found;
  }
  const ancestor = triggerElement.closest("[modal]");
  if (!ancestor) {
    console.warn(`${helperName}: The triggering element is not inside a [modal].`, { triggerElement });
  }
  return ancestor;
}
function closeModalHelper(triggerElement, modalName) {
  const modalToClose = resolveTargetModal(triggerElement, modalName, "closeModal");
  if (!modalToClose) {
    return;
  }
  if (typeof modalToClose.close === "function") {
    modalToClose.close();
  } else {
    console.error(`The found modal does not have a public 'close()' method.`, { modalToClose });
  }
}
function updateModalHelper(triggerElement, modalName) {
  const modalToUpdate = resolveTargetModal(triggerElement, modalName, "updateModal");
  if (!modalToUpdate) {
    return;
  }
  if (typeof modalToUpdate.update === "function") {
    modalToUpdate.update();
  } else {
    console.error(`The found modal does not have a public 'update()' method.`, { modalToUpdate });
  }
}
function reloadPartialHelper(selector) {
  if (!selector || typeof selector !== "string") {
    console.error("reloadPartial helper requires a CSS selector string as its first argument.");
    return;
  }
  const partialToReload = document.querySelector(selector);
  if (!partialToReload) {
    console.warn(`reloadPartial: Could not find an element with the selector "${selector}".`);
    return;
  }
  if (typeof partialToReload.reload === "function") {
    partialToReload.reload();
  } else if (partialToReload.hasAttribute("partial") && partialToReload.hasAttribute("src")) {
    partialToReload.dispatchEvent(new CustomEvent("pk-reload-partial", { bubbles: true }));
  } else {
    console.error(`The element matching "${selector}" does not have a public 'reload()' method.`);
  }
}
function registerHelpers(pk) {
  const hookManager = createPikoHookManager(pk);
  const modalManager = createModalManager({ hookManager });
  registerPkOpenModalListener(modalManager);
  pk.registerHelper("showModal", (element) => {
    showModalHelper(element, modalManager);
  });
  pk.registerHelper("closeModal", (triggerElement, _event, ...args) => {
    closeModalHelper(triggerElement, args[0]);
  });
  pk.registerHelper("updateModal", (triggerElement, _event, ...args) => {
    updateModalHelper(triggerElement, args[0]);
  });
  pk.registerHelper("reloadPartial", (_triggerElement, _event, ...args) => {
    reloadPartialHelper(args[0]);
  });
}
waitForPiko("modals").then(registerHelpers).catch((err) => console.error(err.message));
//# sourceMappingURL=ppframework.modals.es.js.map
