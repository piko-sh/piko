import { _createPKContext, getGlobalPageContext } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
function __getScope__(e) {
  const t = e?.currentTarget ?? e?.target;
  if (!(t instanceof Element)) return document.body;
  return t.closest("[partial_name]") ?? t.closest("[data-pageid]") ?? document.body;
}
async function __getInstance__(e) {
  const s = __getScope__(e);
  if (!__instances__.has(s)) __instances__.set(s, __createInstance__(s));
  return __instances__.get(s);
}
async function __createInstance__(__scope__) {
  const pk = _createPKContext(__scope__);
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
  return (await __getInstance__(event)).focusEmail(...args);
}
export async function disableSubmit(event, ...args) {
  return (await __getInstance__(event)).disableSubmit(...args);
}
export async function getCanvas(event, ...args) {
  return (await __getInstance__(event)).getCanvas(...args);
}
getGlobalPageContext().setExports({ focusEmail, disableSubmit, getCanvas });
{
  const __s__ = document.querySelector("[partial_name]") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
export function __reinit__() {
  const __s__ = document.querySelector("[partial_name]") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
