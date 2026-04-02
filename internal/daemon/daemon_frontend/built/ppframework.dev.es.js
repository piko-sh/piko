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
const SSE_URL = "/_piko/dev/events";
const MAX_RECONNECTS = 30;
const BASE_DELAY_MS = 500;
waitForPiko("dev").then((pk) => {
  let eventSource = null;
  let reconnectCount = 0;
  let reconnectTimer = null;
  let hasDirtyForm = false;
  pk.hooks.on("form:dirty", () => {
    hasDirtyForm = true;
  });
  pk.hooks.on("form:clean", () => {
    hasDirtyForm = false;
  });
  function connect() {
    eventSource = new EventSource(SSE_URL);
    eventSource.onopen = () => {
      reconnectCount = 0;
      console.debug("[piko/dev] Connected to dev event stream");
    };
    eventSource.addEventListener("rebuild-complete", (e) => {
      const data = JSON.parse(e.data);
      handleRebuild(pk, data.affectedRoutes);
    });
    eventSource.onerror = () => {
      eventSource?.close();
      eventSource = null;
      if (reconnectCount < MAX_RECONNECTS) {
        reconnectCount++;
        const delay = BASE_DELAY_MS * Math.min(reconnectCount, 5);
        reconnectTimer = setTimeout(connect, delay);
      }
    };
  }
  function handleRebuild(piko, affectedRoutes) {
    const currentPath = window.location.pathname;
    if (affectedRoutes.length > 0 && !affectedRoutes.includes("*")) {
      const affected = affectedRoutes.some(
        (route) => currentPath === route || currentPath.startsWith(route + "/")
      );
      if (!affected) {
        console.debug(
          "[piko/dev] Rebuild did not affect current page"
        );
        return;
      }
    }
    if (hasDirtyForm) {
      console.debug("[piko/dev] Skipping refresh: dirty form");
      return;
    }
    console.debug("[piko/dev] Soft-refreshing page");
    piko.nav.navigate(window.location.href, { replace: true, scroll: false });
  }
  connect();
  window.addEventListener("beforeunload", () => {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer);
    }
    eventSource?.close();
  });
  console.debug("[piko/dev] Extension loaded - auto-refresh enabled");
}).catch((err) => console.error("[piko/dev]", err.message));
//# sourceMappingURL=ppframework.dev.es.js.map
