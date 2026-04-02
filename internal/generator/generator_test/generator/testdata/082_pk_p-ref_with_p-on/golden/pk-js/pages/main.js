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
  function clearSearch() {
    const input = pk.refs.searchInput;
    if (input) {
      input.value = "";
      input.focus();
    }
  }
  function incrementCounter() {
    const display = pk.refs.counterDisplay;
    if (display) {
      const current = parseInt(display.textContent || "0", 10);
      display.textContent = String(current + 1);
    }
  }
  function decrementCounter() {
    const display = pk.refs.counterDisplay;
    if (display) {
      const current = parseInt(display.textContent || "0", 10);
      display.textContent = String(current - 1);
    }
  }
  function handleLogin(event) {
    event.preventDefault();
    const email = pk.refs.emailField;
    const password = pk.refs.passwordField;
    const form = pk.refs.loginForm;
    const submit = pk.refs.submitButton;
    if (!email?.value || !password?.value) {
      console.error("Email and password are required");
      return;
    }
    submit.disabled = true;
    console.log("Login attempt:", email.value);
  }
  return { clearSearch, incrementCounter, decrementCounter, handleLogin };
}
export async function clearSearch(event, ...args) {
  return (await __getInstance__(event)).clearSearch(...args);
}
export async function incrementCounter(event, ...args) {
  return (await __getInstance__(event)).incrementCounter(...args);
}
export async function decrementCounter(event, ...args) {
  return (await __getInstance__(event)).decrementCounter(...args);
}
export async function handleLogin(event, ...args) {
  return (await __getInstance__(event)).handleLogin(...args);
}
getGlobalPageContext().setExports({ clearSearch, incrementCounter, decrementCounter, handleLogin });
{
  const __s__ = document.querySelector("[partial_name]") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
export function __reinit__() {
  const __s__ = document.querySelector("[partial_name]") ?? document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}
