const MAX_WAIT_MS = 5e3;
const POLL_INTERVAL_MS = 10;
function waitForPiko(extensionName) {
  return new Promise((resolve, reject) => {
    const startTime = Date.now();
    const check = () => {
      if (typeof window !== "undefined" && window.piko) {
        window.piko.ready(() => resolve(window.piko));
        return;
      }
      if (Date.now() - startTime > MAX_WAIT_MS) {
        reject(new Error(`[piko/${extensionName}] Timed out waiting for piko core.`));
        return;
      }
      setTimeout(check, POLL_INTERVAL_MS);
    };
    check();
  });
}
function getPageData() {
  const el = document.getElementById("pk-page-data");
  if (!el?.textContent) {
    return {};
  }
  try {
    return JSON.parse(el.textContent);
  } catch {
    console.warn("[piko/analytics] Failed to parse pk-page-data JSON");
    return {};
  }
}
function initGTM(containerId) {
  window.dataLayer.push({
    "gtm.start": (/* @__PURE__ */ new Date()).getTime(),
    event: "gtm.js"
  });
  const script = document.createElement("script");
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(containerId)}`;
  document.head.appendChild(script);
}
function initGA4(config) {
  const trackingIds = config.trackingIds ?? [];
  function gtagInit(...args) {
    window.dataLayer.push(args);
  }
  window.gtag = gtagInit;
  const primaryId = trackingIds[0];
  const script = document.createElement("script");
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(primaryId)}`;
  document.head.appendChild(script);
  gtagInit("js", /* @__PURE__ */ new Date());
  for (const trackingId of trackingIds) {
    const configOptions = {};
    if (config.anonymiseIp) {
      configOptions.anonymize_ip = true;
    }
    if (config.debugMode) {
      configOptions.debug_mode = true;
    }
    gtagInit("config", trackingId, configOptions);
    if (config.debugMode) {
      console.warn(`[piko/analytics] Configured tracking ID: ${trackingId}`);
    }
  }
}
function dispatchEvent(mode, ga4EventName, ga4Params, gtmEventName, gtmParams) {
  if (mode.hasGA4) {
    gtag("event", ga4EventName, ga4Params);
  }
  if (mode.hasGTM) {
    if (mode.debugMode) {
      console.warn(`[piko/analytics] GTM push: ${gtmEventName}`, gtmParams);
    }
    window.dataLayer.push({ event: gtmEventName, ...gtmParams });
  }
}
function registerAnalyticsHooks(pk, config, mode) {
  if (!config.disablePageView) {
    pk.hooks.on("page:view", (payload) => {
      const p = payload;
      const pagePath = new URL(p.url, window.location.origin).pathname;
      const pageData = getPageData();
      dispatchEvent(
        mode,
        "page_view",
        { page_path: pagePath, page_title: p.title, page_referrer: p.referrer, ...pageData },
        "piko_page_view",
        { page_path: pagePath, page_title: p.title, page_referrer: p.referrer, ...pageData }
      );
    });
  }
  pk.hooks.on("navigation:complete", (payload) => {
    const p = payload;
    const pagePath = new URL(p.url, window.location.origin).pathname;
    const duration = Math.round(p.duration);
    const pageData = getPageData();
    dispatchEvent(
      mode,
      "navigation",
      { event_category: "SPA", event_label: pagePath, value: duration, navigation_trigger: p.trigger, ...pageData },
      "piko_navigation",
      { page_path: pagePath, navigation_trigger: p.trigger, navigation_duration: duration, ...pageData }
    );
  });
  pk.hooks.on("navigation:error", (payload) => {
    const p = payload;
    const pagePath = new URL(p.url, window.location.origin).pathname;
    const description = `Navigation error: ${p.error}`;
    dispatchEvent(
      mode,
      "exception",
      { description, fatal: false, page_path: pagePath },
      "piko_error",
      { error_description: description, error_page: pagePath }
    );
  });
  pk.hooks.on("action:complete", (payload) => {
    const p = payload;
    const duration = Math.round(p.duration);
    dispatchEvent(
      mode,
      "server_action",
      { event_category: "Actions", event_label: p.actionName, value: duration, action_success: p.success },
      "piko_action",
      { action_name: p.actionName, action_success: p.success, action_duration: duration }
    );
  });
  pk.hooks.on("modal:open", (payload) => {
    const p = payload;
    dispatchEvent(
      mode,
      "modal_view",
      { event_category: "Modals", event_label: p.modalId },
      "piko_modal_view",
      { modal_id: p.modalId }
    );
  });
  pk.hooks.on("error", (payload) => {
    const p = payload;
    const description = `${p.type}: ${p.message}`;
    dispatchEvent(
      mode,
      "exception",
      { description, fatal: false },
      "piko_error",
      { error_description: description }
    );
  });
  pk.hooks.on("analytics:track", (payload) => {
    handleAnalyticsTrack(payload, mode);
  });
}
function handleAnalyticsTrack(payload, mode) {
  const p = payload;
  if (!p.eventName) {
    return;
  }
  dispatchEvent(
    mode,
    p.eventName,
    { ...p.params },
    `piko_${p.eventName}`,
    { ...p.params }
  );
  if (mode.debugMode) {
    console.warn(`[piko/analytics] Custom track: ${p.eventName}`, p.params);
  }
}
waitForPiko("analytics").then((pk) => {
  const config = pk.getModuleConfig("analytics");
  if (!config) {
    console.warn("[piko/analytics] No tracking IDs or GTM container configured. Analytics will be disabled.");
    console.warn("[piko/analytics] Configure via piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{...})");
    return;
  }
  const hasGA4 = (config.trackingIds?.length ?? 0) > 0;
  const hasGTM = !!config.gtmContainerId;
  if (!hasGA4 && !hasGTM) {
    console.warn("[piko/analytics] No tracking IDs or GTM container configured. Analytics will be disabled.");
    console.warn("[piko/analytics] Configure via piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{...})");
    return;
  }
  window.dataLayer = window.dataLayer ?? [];
  if (hasGTM && config.gtmContainerId) {
    initGTM(config.gtmContainerId);
  }
  if (hasGA4) {
    initGA4(config);
  }
  const mode = { hasGTM, hasGA4, debugMode: !!config.debugMode };
  registerAnalyticsHooks(pk, config, mode);
  pk.analytics.track = (eventName, params) => {
    pk._emitHook("analytics:track", {
      eventName,
      params: params ?? {},
      timestamp: Date.now()
    });
  };
  if (!config.disablePageView) {
    const pageData = getPageData();
    dispatchEvent(
      mode,
      "page_view",
      { page_path: window.location.pathname, page_title: document.title, page_referrer: document.referrer, ...pageData },
      "piko_page_view",
      { page_path: window.location.pathname, page_title: document.title, page_referrer: document.referrer, ...pageData }
    );
  }
  if (config.debugMode) {
    const parts = [];
    if (hasGTM) {
      parts.push(`GTM: ${config.gtmContainerId}`);
    }
    if (hasGA4) {
      parts.push(`GA4: ${config.trackingIds?.join(", ")}`);
    }
    console.warn(`[piko/analytics] Extension loaded - ${parts.join("; ")}`);
    console.warn("[piko/analytics] Tracking: page views, navigation, actions, modals, errors");
  }
}).catch((err) => console.error(err.message));
//# sourceMappingURL=ppframework.analytics.es.js.map
