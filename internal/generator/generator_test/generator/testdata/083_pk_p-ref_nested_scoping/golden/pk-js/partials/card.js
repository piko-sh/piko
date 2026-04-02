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
  function increment() {
    const counter = pk.refs.counterValue;
    if (counter) {
      const value = parseInt(counter.textContent || "0", 10);
      counter.textContent = String(value + 1);
    }
  }
  function decrement() {
    const counter = pk.refs.counterValue;
    if (counter) {
      const value = parseInt(counter.textContent || "0", 10);
      counter.textContent = String(value - 1);
    }
  }
  function focusCardInput() {
    const input = pk.refs.inputField;
    if (input) {
      input.focus();
      input.select();
    }
  }
  return { increment, decrement, focusCardInput };
}
export async function increment(event, ...args) {
  return (await __getInstance__(event)).increment(...args);
}
export async function decrement(event, ...args) {
  return (await __getInstance__(event)).decrement(...args);
}
export async function focusCardInput(event, ...args) {
  return (await __getInstance__(event)).focusCardInput(...args);
}
getGlobalPageContext().setExports({ increment, decrement, focusCardInput });
{
  const __s__ = document.querySelector("[partial_name='card']") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
export function __reinit__() {
  const __s__ = document.querySelector("[partial_name='card']") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
