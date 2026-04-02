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
  function closeModal() {
    const wrapper = document.querySelector(".modal-wrapper");
    if (wrapper) {
      wrapper.classList.remove("open");
    }
  }
  function openModal() {
    const wrapper = document.querySelector(".modal-wrapper");
    if (wrapper) {
      wrapper.classList.add("open");
    }
  }
  return { closeModal, openModal };
}
export async function closeModal(event, ...args) {
  return (await __getInstance__(event)).closeModal(...args);
}
export async function openModal(event, ...args) {
  return (await __getInstance__(event)).openModal(...args);
}
getGlobalPageContext().setExports({ closeModal, openModal });
{
  const __s__ = document.querySelector("[partial_name='modal-wrapper']") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
export function __reinit__() {
  const __s__ = document.querySelector("[partial_name='modal-wrapper']") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
