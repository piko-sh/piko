(() => {
  const RECOVERY_RELOAD_STORAGE_KEY = "piko:runtime-reload-at";
  const RECOVERY_RELOAD_INTERVAL_MS = 6e4;
  const RECOVERY_WATCHED_PATH_PREFIXES = ["/_piko/dist/", "/_piko/assets/pk-js/"];
  const MODULE_PRELOAD_RELATION = "modulepreload";
  const ENFORCED_DISPOSITION = "enforce";
  const SCRIPT_DIRECTIVE_PREFIX = "script-src";
  const BLOCKED_BY_POLICY_MESSAGE = "[piko] The Content Security Policy blocked a script on this page, so the page is not reloaded.";
  const OFFLINE_MESSAGE = "[piko] A framework script failed to load while the browser is offline, so the page is not reloaded.";
  const ALREADY_RELOADING_MESSAGE = "[piko] A framework script failed to load while the page is already reloading.";
  const RECENTLY_RELOADED_MESSAGE = "[piko] A framework script failed to load again soon after a reload, so the page is not reloaded again. Reload the page to try again.";
  function watchedFailureURL(element, location) {
    const attribute = failedResourceAttribute(element);
    if (attribute === null) {
      return null;
    }
    let url;
    try {
      url = new URL(attribute, location.href);
    } catch (e) {
      return null;
    }
    if (url.origin !== location.origin) {
      return null;
    }
    if (!RECOVERY_WATCHED_PATH_PREFIXES.some((prefix) => url.pathname.startsWith(prefix))) {
      return null;
    }
    return url.href;
  }
  function claimRecoveryReload(reloadStore, now) {
    if (!reloadStore) {
      return false;
    }
    try {
      if (isWithinReloadInterval(readLastReloadAt(reloadStore), now)) {
        return false;
      }
      reloadStore.setItem(RECOVERY_RELOAD_STORAGE_KEY, String(now));
      return true;
    } catch (e) {
      return false;
    }
  }
  function installRuntimeRecovery(deps) {
    const installedAt = deps.now();
    let scriptBlockedByPolicy = false;
    let reloading = false;
    deps.target.addEventListener("securitypolicyviolation", (event) => {
      if (isEnforcedScriptViolation(event)) {
        scriptBlockedByPolicy = true;
      }
    }, true);
    deps.target.addEventListener("error", (event) => {
      const url = watchedFailureURL(event.target, deps.location);
      if (url === null) {
        return;
      }
      deps.defer(() => {
        if (reloading) {
          return;
        }
        const refusal = reloadRefusal(deps, scriptBlockedByPolicy, installedAt);
        if (refusal !== null) {
          deps.reportError(refusal, url);
          return;
        }
        reloading = true;
        deps.location.reload();
      });
    }, true);
  }
  function reloadRefusal(deps, scriptBlockedByPolicy, installedAt) {
    if (scriptBlockedByPolicy) {
      return BLOCKED_BY_POLICY_MESSAGE;
    }
    if (deps.isOffline()) {
      return OFFLINE_MESSAGE;
    }
    if (claimRecoveryReload(deps.reloadStore, deps.now())) {
      return null;
    }
    if (reloadClaimedSince(deps.reloadStore, installedAt)) {
      return ALREADY_RELOADING_MESSAGE;
    }
    return RECENTLY_RELOADED_MESSAGE;
  }
  function isEnforcedScriptViolation(violation) {
    if (violation.disposition !== ENFORCED_DISPOSITION) {
      return false;
    }
    const directive = violation.effectiveDirective || violation.violatedDirective;
    return typeof directive === "string" && directive.startsWith(SCRIPT_DIRECTIVE_PREFIX);
  }
  function readLastReloadAt(reloadStore) {
    const recorded = reloadStore.getItem(RECOVERY_RELOAD_STORAGE_KEY);
    return recorded === null ? Number.NaN : Number(recorded);
  }
  function isWithinReloadInterval(lastReloadAt, now) {
    if (!Number.isFinite(lastReloadAt) || lastReloadAt <= 0) {
      return false;
    }
    const elapsed = now - lastReloadAt;
    return elapsed >= 0 && elapsed < RECOVERY_RELOAD_INTERVAL_MS;
  }
  function reloadClaimedSince(reloadStore, since) {
    if (!reloadStore) {
      return false;
    }
    try {
      return readLastReloadAt(reloadStore) >= since;
    } catch (e) {
      return false;
    }
  }
  function failedResourceAttribute(element) {
    if (typeof element !== "object" || element === null || !("tagName" in element) || !("getAttribute" in element)) {
      return null;
    }
    const candidate = element;
    if (candidate.tagName === "SCRIPT") {
      return candidate.getAttribute("src");
    }
    if (candidate.tagName === "LINK" && isModulePreload(candidate)) {
      return candidate.getAttribute("href");
    }
    return null;
  }
  function isModulePreload(link) {
    var _a;
    const relation = (_a = link.getAttribute("rel")) != null ? _a : "";
    return relation.toLowerCase().split(/\s+/).includes(MODULE_PRELOAD_RELATION);
  }
  function browserRecoveryStore() {
    try {
      return typeof sessionStorage === "undefined" ? null : sessionStorage;
    } catch (e) {
      return null;
    }
  }
  if (typeof document !== "undefined" && document.currentScript !== null) {
    installRuntimeRecovery({
      target: window,
      location: window.location,
      reloadStore: browserRecoveryStore(),
      now: () => Date.now(),
      isOffline: () => navigator.onLine === false,
      defer: (task) => {
        setTimeout(task, 0);
      },
      reportError: (message, url) => {
        console.error(message, url);
      }
    });
  }
})();
