import { _createPagePKContext, _currentPageVisit } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
async function __getInstance__() {
  return __instances__.get(_currentPageVisit());
}
async function __createInstance__(__scope__, __visit__) {
  const pk = _createPagePKContext(__scope__, __visit__);
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
  return (await __getInstance__())?.openLightbox(...args);
}
export async function handleClick(event, ...args) {
  return (await __getInstance__())?.handleClick(...args);
}
export async function handleCounter(event, ...args) {
  return (await __getInstance__())?.handleCounter(...args);
}
export async function handleMultiple(event, ...args) {
  return (await __getInstance__())?.handleMultiple(...args);
}
export async function handleNested(event, ...args) {
  return (await __getInstance__())?.handleNested(...args);
}
export function __reinit__(__visit__ = _currentPageVisit()) {
  const __s__ = document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));
}
