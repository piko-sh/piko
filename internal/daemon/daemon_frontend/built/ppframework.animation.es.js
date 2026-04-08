const DEFAULT_TYPE_SPEED = 50;
const DEFAULT_TYPEHTML_SPEED = 25;
const MILLISECONDS_PER_SECOND = 1e3;
const MAX_HTML_ENTITY_LENGTH = 10;
const DEFAULT_ANCHOR_WIDTH = 200;
const DEFAULT_ANCHOR_HEIGHT = 40;
const ANCHOR_PADDING = 6;
const customHandlers = /* @__PURE__ */ new Map();
function registerTimelineAction(name, handler) {
  customHandlers.set(name, handler);
}
function captureTypewriterTexts(component, actions, textMap) {
  for (const action of actions) {
    if (action.action !== "type") {
      continue;
    }
    const el = component.refs?.[action.ref];
    if (!el) {
      continue;
    }
    textMap.set(action.ref, el.textContent);
    el.textContent = "";
  }
}
function captureTypehtmlContents(component, actions, htmlMap) {
  for (const action of actions) {
    if (action.action !== "typehtml") {
      continue;
    }
    const el = component.refs?.[action.ref];
    if (!el) {
      continue;
    }
    htmlMap.set(action.ref, el.innerHTML);
    el.innerHTML = "";
  }
}
function evaluateTimeline(component, actions, currentTime, typewriterTexts, typehtmlContents) {
  evaluateVisibility(component, actions, currentTime);
  evaluateClasses(component, actions, currentTime);
  dispatchActions(component, actions, currentTime, typewriterTexts, typehtmlContents);
}
function evaluateVisibility(component, actions, currentTime) {
  const visibilityState = /* @__PURE__ */ new Map();
  const visibilityInitial = /* @__PURE__ */ new Map();
  for (const action of actions) {
    if (action.action !== "show" && action.action !== "hide") {
      continue;
    }
    if (!visibilityInitial.has(action.ref)) {
      visibilityInitial.set(action.ref, action.action === "hide");
    }
    if (currentTime >= action.time) {
      visibilityState.set(action.ref, action.action === "show");
    }
  }
  for (const [ref, initialVisible] of visibilityInitial) {
    const el = component.refs?.[ref];
    if (!el) {
      continue;
    }
    const visible = visibilityState.get(ref) ?? initialVisible;
    if (visible) {
      el.removeAttribute("p-timeline-hidden");
    } else {
      el.setAttribute("p-timeline-hidden", "");
    }
  }
}
function evaluateClasses(component, actions, currentTime) {
  const classState = /* @__PURE__ */ new Map();
  const classInitial = /* @__PURE__ */ new Map();
  for (const action of actions) {
    if (action.action !== "addclass" && action.action !== "removeclass") {
      continue;
    }
    const key = `${action.ref}\0${action.class}`;
    if (!classInitial.has(key)) {
      classInitial.set(key, action.action === "removeclass");
    }
    if (currentTime >= action.time) {
      classState.set(key, action.action === "addclass");
    }
  }
  for (const [key, initialPresent] of classInitial) {
    const sep = key.indexOf("\0");
    const ref = key.substring(0, sep);
    const className = key.substring(sep + 1);
    const el = component.refs?.[ref];
    if (!el) {
      continue;
    }
    const present = classState.get(key) ?? initialPresent;
    if (present) {
      el.classList.add(className);
    } else {
      el.classList.remove(className);
    }
  }
}
function clearTooltips(component, actions) {
  for (const action of actions) {
    if (action.action !== "tooltip") {
      continue;
    }
    const el = component.refs?.[action.ref];
    if (el) {
      el.removeAttribute("title");
    }
  }
}
function dispatchActions(component, actions, currentTime, typewriterTexts, typehtmlContents) {
  clearTooltips(component, actions);
  for (const action of actions) {
    const el = component.refs?.[action.ref];
    switch (action.action) {
      case "type":
        if (el) {
          evaluateTypewriter(el, action, currentTime, typewriterTexts);
        }
        break;
      case "typehtml":
        if (el) {
          evaluateTypehtmlWriter(el, action, currentTime, typehtmlContents);
        }
        break;
      case "tooltip":
        if (el && currentTime >= action.time && action.value) {
          el.setAttribute("title", action.value);
        }
        break;
      default: {
        const handler = customHandlers.get(action.action);
        if (handler) {
          handler(el ?? null, action, currentTime, component);
        }
        break;
      }
    }
  }
}
function evaluateTypewriter(element, action, currentTime, typewriterTexts) {
  const fullText = typewriterTexts.get(action.ref) ?? "";
  const speed = action.speed ?? DEFAULT_TYPE_SPEED;
  const elapsed = (currentTime - action.time) * MILLISECONDS_PER_SECOND;
  if (elapsed <= 0) {
    element.textContent = "";
    return;
  }
  const charsToShow = Math.min(
    Math.floor(elapsed / speed),
    fullText.length
  );
  element.textContent = fullText.substring(0, charsToShow);
}
function evaluateTypehtmlWriter(element, action, currentTime, htmlMap) {
  const fullHtml = htmlMap.get(action.ref) ?? "";
  const speed = action.speed ?? DEFAULT_TYPEHTML_SPEED;
  const elapsed = (currentTime - action.time) * MILLISECONDS_PER_SECOND;
  if (elapsed <= 0) {
    element.innerHTML = "";
    return;
  }
  const charsToShow = Math.floor(elapsed / speed);
  element.innerHTML = sliceHtml(fullHtml, charsToShow);
}
function trackHtmlTag(tag, openTags) {
  if (tag.startsWith("</")) {
    openTags.pop();
  } else if (!tag.endsWith("/>")) {
    const match = tag.match(/^<(\w+)/);
    if (match) {
      openTags.push(match[1]);
    }
  }
}
function sliceHtml(html, visibleCount) {
  let visible = 0;
  let i = 0;
  const openTags = [];
  while (i < html.length && visible < visibleCount) {
    if (html[i] === "<") {
      const tagEnd = html.indexOf(">", i);
      if (tagEnd === -1) {
        break;
      }
      const tag = html.substring(i, tagEnd + 1);
      trackHtmlTag(tag, openTags);
      i = tagEnd + 1;
    } else if (html[i] === "&") {
      const semiPos = html.indexOf(";", i);
      if (semiPos !== -1 && semiPos - i <= MAX_HTML_ENTITY_LENGTH) {
        i = semiPos + 1;
      } else {
        i++;
      }
      visible++;
    } else {
      visible++;
      i++;
    }
  }
  while (i < html.length && html[i] === "<") {
    const tagEnd = html.indexOf(">", i);
    if (tagEnd === -1) {
      break;
    }
    const tag = html.substring(i, tagEnd + 1);
    trackHtmlTag(tag, openTags);
    i = tagEnd + 1;
  }
  let result = html.substring(0, i);
  for (let t = openTags.length - 1; t >= 0; t--) {
    result += `</${openTags[t]}>`;
  }
  return result;
}
function findAnchorContainer(shadowRoot) {
  for (const child of Array.from(shadowRoot.children)) {
    if (child instanceof HTMLElement && child.tagName !== "STYLE") {
      return child;
    }
  }
  return null;
}
function computeAnchorPosition(targetRect, containerRect, elementWidth, elementHeight, wantBottom, wantRight) {
  let top;
  if (wantBottom) {
    top = targetRect.bottom - containerRect.top + ANCHOR_PADDING;
    if (top + elementHeight > containerRect.height - ANCHOR_PADDING) {
      top = targetRect.top - containerRect.top - elementHeight - ANCHOR_PADDING;
    }
  } else {
    top = targetRect.top - containerRect.top - elementHeight - ANCHOR_PADDING;
    if (top < ANCHOR_PADDING) {
      top = targetRect.bottom - containerRect.top + ANCHOR_PADDING;
    }
  }
  let left;
  if (wantRight) {
    left = targetRect.right - containerRect.left - elementWidth;
  } else {
    left = targetRect.left - containerRect.left;
  }
  if (left + elementWidth > containerRect.width - ANCHOR_PADDING) {
    left = containerRect.width - elementWidth - ANCHOR_PADDING;
  }
  if (left < ANCHOR_PADDING) {
    left = ANCHOR_PADDING;
  }
  if (top < ANCHOR_PADDING) {
    top = ANCHOR_PADDING;
  }
  if (top + elementHeight > containerRect.height - ANCHOR_PADDING) {
    top = containerRect.height - elementHeight - ANCHOR_PADDING;
  }
  return { top, left };
}
function evaluateAnchors(component) {
  const sr = component.shadowRoot;
  if (!sr) {
    return;
  }
  const anchored = sr.querySelectorAll("[p-timeline-anchor]");
  if (anchored.length === 0) {
    return;
  }
  const container = findAnchorContainer(sr);
  if (!container) {
    return;
  }
  const containerRect = container.getBoundingClientRect();
  for (const el of Array.from(anchored)) {
    const htmlEl = el;
    if (htmlEl.hasAttribute("p-timeline-hidden")) {
      continue;
    }
    const raw = htmlEl.getAttribute("p-timeline-anchor");
    if (!raw) {
      continue;
    }
    const parts = raw.split(" ");
    const targetRef = parts[0];
    const position = parts[1] || "bottom-left";
    const wantBottom = position.startsWith("bottom");
    const wantRight = position.endsWith("right");
    const target = component.refs?.[targetRef];
    if (!target || target.hasAttribute("p-timeline-hidden")) {
      htmlEl.style.top = "";
      htmlEl.style.left = "";
      htmlEl.style.right = "";
      htmlEl.style.bottom = "";
      continue;
    }
    const targetRect = target.getBoundingClientRect();
    const elementWidth = htmlEl.offsetWidth || DEFAULT_ANCHOR_WIDTH;
    const elementHeight = htmlEl.offsetHeight || DEFAULT_ANCHOR_HEIGHT;
    const pos = computeAnchorPosition(
      targetRect,
      containerRect,
      elementWidth,
      elementHeight,
      wantBottom,
      wantRight
    );
    htmlEl.style.top = `${pos.top}px`;
    htmlEl.style.left = `${pos.left}px`;
    htmlEl.style.right = "auto";
    htmlEl.style.bottom = "auto";
  }
}
function resolveTimeline(raw) {
  if (!Array.isArray(raw) || raw.length === 0) {
    return null;
  }
  if ("time" in raw[0]) {
    return raw;
  }
  const entries = raw;
  let fallback = null;
  for (const entry of entries) {
    if (entry.media == null || entry.media === "") {
      fallback = entry.actions;
      continue;
    }
    if (window.matchMedia(entry.media).matches) {
      return entry.actions;
    }
  }
  return fallback;
}
function setupAnimation(component) {
  const constructor = component.constructor;
  const rawTimeline = constructor.$$timeline;
  if (!rawTimeline || !Array.isArray(rawTimeline)) {
    console.warn(`Animation: no $$timeline data found on ${component.tagName}.`);
    return;
  }
  const validatedTimeline = rawTimeline;
  let sortedActions = [];
  let typewriterTexts = /* @__PURE__ */ new Map();
  let typehtmlContents = /* @__PURE__ */ new Map();
  let initialised = false;
  let captured = false;
  const ppEl = component;
  function activateTimeline() {
    const actions = resolveTimeline(validatedTimeline);
    if (!actions) {
      return;
    }
    sortedActions = [...actions].sort((a, b) => a.time - b.time);
    typewriterTexts = /* @__PURE__ */ new Map();
    typehtmlContents = /* @__PURE__ */ new Map();
    captured = false;
  }
  activateTimeline();
  const isResponsive = rawTimeline.length > 0 && "media" in rawTimeline[0];
  const mediaListeners = [];
  if (isResponsive) {
    const entries = rawTimeline;
    for (const entry of entries) {
      if (entry.media == null || entry.media === "") {
        continue;
      }
      const mql = window.matchMedia(entry.media);
      const handler = () => {
        activateTimeline();
        if (initialised) {
          captureTypewriterTexts(ppEl, sortedActions, typewriterTexts);
          captureTypehtmlContents(ppEl, sortedActions, typehtmlContents);
          captured = true;
          const currentTime = parseFloat(ppEl.getAttribute("time") ?? "0");
          evaluateTimeline(ppEl, sortedActions, currentTime, typewriterTexts, typehtmlContents);
          evaluateAnchors(ppEl);
        }
      };
      mql.addEventListener("change", handler);
      mediaListeners.push({ mql, handler });
    }
  }
  const observer = new MutationObserver(() => {
    if (!initialised) {
      return;
    }
    if (!captured) {
      captured = true;
      captureTypewriterTexts(ppEl, sortedActions, typewriterTexts);
      captureTypehtmlContents(ppEl, sortedActions, typehtmlContents);
    }
    const currentTime = parseFloat(ppEl.getAttribute("time") ?? "0");
    evaluateTimeline(ppEl, sortedActions, currentTime, typewriterTexts, typehtmlContents);
    evaluateAnchors(ppEl);
  });
  ppEl.onAfterRender(() => {
    if (initialised) {
      return;
    }
    initialised = true;
    observer.observe(ppEl, { attributes: true, attributeFilter: ["time"] });
  });
  ppEl.onDisconnected(() => {
    observer.disconnect();
    for (const { mql, handler } of mediaListeners) {
      mql.removeEventListener("change", handler);
    }
  });
}
window.__piko_animation = setupAnimation;
window.__piko_registerTimelineAction = registerTimelineAction;
//# sourceMappingURL=ppframework.animation.es.js.map
