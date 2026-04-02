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
  function openLightbox(e, slideIndex) {
    console.log("Opening lightbox at slide", slideIndex);
  }
  function handleClick(e) {
    console.log("Clicked:", e);
  }
  function handleCounter(count) {
    console.log("Counter:", count);
  }
  function handleMultiple(e, counter, doubled) {
    console.log("Event:", e, "Counter:", counter, "Doubled:", doubled);
  }
  function handleNested(categoryIndex, itemIndex, itemName) {
    console.log("Category:", categoryIndex, "Item:", itemIndex, "Name:", itemName);
  }
  return { openLightbox, handleClick, handleCounter, handleMultiple, handleNested };
}
export async function openLightbox(event, ...args) {
  return (await __getInstance__(event)).openLightbox(...args);
}
export async function handleClick(event, ...args) {
  return (await __getInstance__(event)).handleClick(...args);
}
export async function handleCounter(event, ...args) {
  return (await __getInstance__(event)).handleCounter(...args);
}
export async function handleMultiple(event, ...args) {
  return (await __getInstance__(event)).handleMultiple(...args);
}
export async function handleNested(event, ...args) {
  return (await __getInstance__(event)).handleNested(...args);
}
getGlobalPageContext().setExports({ openLightbox, handleClick, handleCounter, handleMultiple, handleNested });
{
  const __s__ = document.querySelector("[partial_name]") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
export function __reinit__() {
  const __s__ = document.querySelector("[partial_name]") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
