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
  pk.registerHelper("showToast", (_triggerElement, _event, ...args) => {
    const message = args[0];
    const variant = args[1] ?? "info";
    const durationStr = args[2] ?? "5000";
    const duration = parseInt(durationStr, 10);
    if (!message) {
      console.warn("showToast helper called without a message.");
      return;
    }
    document.dispatchEvent(new CustomEvent("pk-show-toast", {
      detail: {
        message,
        variant,
        duration
      }
    }));
  });
  console.debug("[piko/toasts] Extension loaded - helpers: showToast");
}
waitForPiko("toasts").then(registerHelpers).catch((err) => console.error(err.message));
//# sourceMappingURL=ppframework.toasts.es.js.map
