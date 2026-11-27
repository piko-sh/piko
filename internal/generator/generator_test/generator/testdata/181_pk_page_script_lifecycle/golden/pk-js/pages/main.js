import { _createPagePKContext, _currentPageVisit } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
async function __getInstance__() {
  return __instances__.get(_currentPageVisit());
}
async function __createInstance__(__scope__, __visit__) {
  const pk = _createPagePKContext(__scope__, __visit__);
  function pageLog() {
    const host = window;
    host.__pageScriptLog ??= [];
    return host.__pageScriptLog;
  }
  let clicks = 0;
  function recordClick() {
    clicks++;
    pageLog().push("click " + clicks);
  }
  pk.onConnected(() => {
    pageLog().push("connected");
    const counter = pk.refs.counter;
    counter?.addEventListener("click", recordClick);
    pk.onCleanup(() => counter?.removeEventListener("click", recordClick));
  });
  pk.onDisconnected(() => {
    pageLog().push("disconnected");
  });
  return { pageLog, recordClick };
}
export async function pageLog(event, ...args) {
  return (await __getInstance__())?.pageLog(...args);
}
export async function recordClick(event, ...args) {
  return (await __getInstance__())?.recordClick(...args);
}
export function __reinit__(__visit__ = _currentPageVisit()) {
  const __s__ = document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));
}
