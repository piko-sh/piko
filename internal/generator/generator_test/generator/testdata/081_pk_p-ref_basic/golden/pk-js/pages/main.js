import { _createPagePKContext, _currentPageVisit } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
async function __getInstance__() {
  return __instances__.get(_currentPageVisit());
}
async function __createInstance__(__scope__, __visit__) {
  const pk = _createPagePKContext(__scope__, __visit__);
  function focusEmail() {
    pk.refs.emailInput?.focus();
  }
  function disableSubmit() {
    if (pk.refs.submitBtn) {
      pk.refs.submitBtn.disabled = true;
    }
  }
  function getCanvas() {
    return pk.refs.chartCanvas;
  }
  return { focusEmail, disableSubmit, getCanvas };
}
export async function focusEmail(event, ...args) {
  return (await __getInstance__())?.focusEmail(...args);
}
export async function disableSubmit(event, ...args) {
  return (await __getInstance__())?.disableSubmit(...args);
}
export async function getCanvas(event, ...args) {
  return (await __getInstance__())?.getCanvas(...args);
}
export function __reinit__(__visit__ = _currentPageVisit()) {
  const __s__ = document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));
}
