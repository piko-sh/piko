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
function registerAnalyticsHooks(pk, config) {
  if (!config.disablePageView) {
    pk.hooks.on("page:view", (payload) => {
      const p = payload;
      gtag("event", "page_view", {
        page_path: new URL(p.url).pathname,
        page_title: p.title,
        page_referrer: p.referrer
      });
    });
  }
  pk.hooks.on("navigation:complete", (payload) => {
    const p = payload;
    gtag("event", "navigation", {
      event_category: "SPA",
      event_label: new URL(p.url).pathname,
      value: Math.round(p.duration),
      navigation_trigger: p.trigger
    });
  });
  pk.hooks.on("navigation:error", (payload) => {
    const p = payload;
    gtag("event", "exception", {
      description: `Navigation error: ${p.error}`,
      fatal: false,
      page_path: new URL(p.url).pathname
    });
  });
  pk.hooks.on("action:complete", (payload) => {
    const p = payload;
    gtag("event", "server_action", {
      event_category: "Actions",
      event_label: p.actionName,
      value: Math.round(p.duration),
      action_success: p.success
    });
  });
  pk.hooks.on("modal:open", (payload) => {
    const p = payload;
    gtag("event", "modal_view", {
      event_category: "Modals",
      event_label: p.modalId
    });
  });
  pk.hooks.on("error", (payload) => {
    const p = payload;
    gtag("event", "exception", {
      description: `${p.type}: ${p.message}`,
      fatal: false
    });
  });
}
waitForPiko("analytics").then((pk) => {
  const config = pk.getModuleConfig("analytics");
  if (!config?.trackingIds.length) {
    console.warn("[piko/analytics] No tracking IDs configured. Analytics will be disabled.");
    console.warn("[piko/analytics] Configure via piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{...})");
    return;
  }
  window.dataLayer = window.dataLayer ?? [];
  function gtagInit(...args) {
    window.dataLayer.push(args);
  }
  window.gtag = gtagInit;
  const primaryId = config.trackingIds[0];
  const script = document.createElement("script");
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(primaryId)}`;
  document.head.appendChild(script);
  gtagInit("js", /* @__PURE__ */ new Date());
  for (const trackingId of config.trackingIds) {
    const configOptions = {};
    if (config.anonymizeIp) {
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
  registerAnalyticsHooks(pk, config);
  const trackingCount = config.trackingIds.length;
  const idList = config.trackingIds.join(", ");
  console.warn(`[piko/analytics] Extension loaded - ${trackingCount} tracking ID(s): ${idList}`);
  console.warn("[piko/analytics] Tracking: page views, navigation, actions, modals, errors");
}).catch((err) => console.error(err.message));
//# sourceMappingURL=ppframework.analytics.es.js.map
