import { _createPagePKContext, _currentPageVisit } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
async function __getInstance__() {
  return __instances__.get(_currentPageVisit());
}
async function __createInstance__(__scope__, __visit__) {
  const pk = _createPagePKContext(__scope__, __visit__);
  function handleClick() {
    console.log("Button clicked");
  }
  function handleSubmit(event) {
    event.preventDefault();
    console.log("Form submitted");
  }
  return { handleClick, handleSubmit };
}
export async function handleClick(event, ...args) {
  return (await __getInstance__())?.handleClick(...args);
}
export async function handleSubmit(event, ...args) {
  return (await __getInstance__())?.handleSubmit(...args);
}
export function __reinit__(__visit__ = _currentPageVisit()) {
  const __s__ = document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));
}
