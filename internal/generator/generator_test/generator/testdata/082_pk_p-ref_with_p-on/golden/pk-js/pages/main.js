import { _createPagePKContext, _currentPageVisit } from "/_piko/dist/ppframework.core.es.js";
const __instances__ = /* @__PURE__ */ new WeakMap();
async function __getInstance__() {
  return __instances__.get(_currentPageVisit());
}
async function __createInstance__(__scope__, __visit__) {
  const pk = _createPagePKContext(__scope__, __visit__);
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
  return (await __getInstance__())?.clearSearch(...args);
}
export async function incrementCounter(event, ...args) {
  return (await __getInstance__())?.incrementCounter(...args);
}
export async function decrementCounter(event, ...args) {
  return (await __getInstance__())?.decrementCounter(...args);
}
export async function handleLogin(event, ...args) {
  return (await __getInstance__())?.handleLogin(...args);
}
export function __reinit__(__visit__ = _currentPageVisit()) {
  const __s__ = document.querySelector("[data-pageid]") ?? document.body;
  if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));
}
