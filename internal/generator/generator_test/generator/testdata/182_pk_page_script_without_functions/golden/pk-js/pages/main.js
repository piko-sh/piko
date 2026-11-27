import { _createPagePKContext, _currentPageVisit } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
async function __createInstance__(__scope__, __visit__) {
  const pk = _createPagePKContext(__scope__, __visit__);
  const host = window;
  host.__pageScriptLog ??= [];
  const pageLog = host.__pageScriptLog;
  let clicks = 0;
  pk.onConnected(() => {
    pageLog.push("connected");
    const counter = pk.refs.counter;
    const countClick = () => {
      clicks++;
      pageLog.push("click " + clicks);
    };
    counter?.addEventListener("click", countClick);
    pk.onCleanup(() => counter?.removeEventListener("click", countClick));
  });
  pk.onDisconnected(() => {
    pageLog.push("disconnected");
  });
  return {};
}
export function __reinit__(__visit__ = _currentPageVisit()) {
  const __s__ = document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));
}
