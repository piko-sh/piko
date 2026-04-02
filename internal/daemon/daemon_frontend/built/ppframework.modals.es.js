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
function registerHelpers(pk) {
  pk.registerHelper("showModal", (element, _event) => {
    const selector = element.dataset.modalSelector;
    if (!selector) {
      console.warn('helpers.showModal() requires a "data-modal-selector" attribute.', element);
      return;
    }
    void pk.modal.open({
      selector,
      params: /* @__PURE__ */ new Map(),
      title: element.dataset.modalTitle ?? "",
      message: element.dataset.modalMessage ?? "",
      cancelLabel: element.dataset.modalCancelMessage ?? "",
      triggerElement: element
    });
  });
  pk.registerHelper("closeModal", (triggerElement, _event, ...args) => {
    const modalName = args[0];
    let modalToClose = null;
    if (typeof modalName === "string" && modalName) {
      const selector = `[modal="${modalName}"]`;
      modalToClose = document.querySelector(selector);
      if (!modalToClose) {
        console.warn(`closeModal: Could not find any modal with selector: ${selector}`);
        return;
      }
    } else {
      modalToClose = triggerElement.closest("[modal]");
      if (!modalToClose) {
        console.warn(`closeModal: The triggering element is not inside a [modal].`, { triggerElement });
        return;
      }
    }
    if (typeof modalToClose.close === "function") {
      modalToClose.close();
    } else {
      console.error(`The found modal does not have a public 'close()' method.`, { modalToClose });
    }
  });
  pk.registerHelper("updateModal", (triggerElement, _event, ...args) => {
    const modalName = args[0];
    let modalToUpdate = null;
    if (typeof modalName === "string" && modalName) {
      const selector = `[modal="${modalName}"]`;
      modalToUpdate = document.querySelector(selector);
      if (!modalToUpdate) {
        console.warn(`updateModal: Could not find any modal with selector: ${selector}`);
        return;
      }
    } else {
      modalToUpdate = triggerElement.closest("[modal]");
      if (!modalToUpdate) {
        console.warn(`updateModal: The triggering element is not inside a [modal].`, { triggerElement });
        return;
      }
    }
    if (typeof modalToUpdate.update === "function") {
      modalToUpdate.update();
    } else {
      console.error(`The found modal does not have a public 'update()' method.`, { modalToUpdate });
    }
  });
  pk.registerHelper("reloadPartial", (_triggerElement, _event, ...args) => {
    const selector = args[0];
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
  });
  console.debug("[piko/modals] Extension loaded - helpers: showModal, closeModal, updateModal, reloadPartial");
}
waitForPiko("modals").then(registerHelpers).catch((err) => console.error(err.message));
//# sourceMappingURL=ppframework.modals.es.js.map
