const behaviourRegistry = {};
function registerBehaviour(name, setupFunction) {
  if (behaviourRegistry[name]) {
    console.warn(`PPBehaviour: Overwriting already registered behaviour "${name}".`);
  }
  behaviourRegistry[name] = setupFunction;
}
function getBehaviour(name) {
  return behaviourRegistry[name];
}
function createUpdateFormState(component, internals) {
  let lastReportedValue;
  return () => {
    const value = component.state?.value ?? null;
    internals.setFormValue(value);
    const customValidityFn = component.customValidity;
    if (typeof customValidityFn === "function") {
      const result = customValidityFn();
      if (result) {
        internals.setValidity(result.validity, result.message, result.anchor);
      } else {
        internals.setValidity({});
      }
    } else {
      const nativeInputRef = component.refs.nativeInput;
      const nativeEl = nativeInputRef ?? component.shadowRoot?.querySelector("input, select, textarea");
      if (nativeEl) {
        internals.setValidity(nativeEl.validity, nativeEl.validationMessage || "Please validate", nativeEl);
      } else {
        internals.setValidity({});
      }
    }
    if (value !== lastReportedValue) {
      lastReportedValue = value;
      component.dispatchEvent(new Event("input", { bubbles: true, composed: true }));
    }
  };
}
function attachFormLifecycleCallbacks(component, updateFormState) {
  component.formAssociatedCallback = (_form) => {
  };
  component.formDisabledCallback = (disabled) => {
    if (component.state) {
      component.state.disabled = disabled;
    }
  };
  component.formResetCallback = () => {
    if (component.state && component.$$ctx?.$$initialState) {
      const initialValue = component.$$ctx.state.value;
      component.state.value = initialValue ?? "";
      queueMicrotask(updateFormState);
    }
  };
  component.formStateRestoreCallback = (state, _mode) => {
    if (component.state) {
      component.state.value = state;
      queueMicrotask(updateFormState);
    }
  };
}
function defineFormProperties(component, internals) {
  Object.defineProperties(component, {
    form: { get: () => internals.form, enumerable: true },
    validity: { get: () => internals.validity, enumerable: true },
    validationMessage: { get: () => internals.validationMessage, enumerable: true },
    willValidate: { get: () => internals.willValidate, enumerable: true },
    name: { get: () => component.getAttribute("name"), enumerable: true },
    type: { get: () => component.localName, enumerable: true },
    labels: { get: () => internals.labels, enumerable: true }
  });
  component.checkValidity = () => internals.checkValidity();
  component.reportValidity = () => internals.reportValidity();
}
const formAssociatedBehaviour = (component) => {
  const internals = component.internals;
  if (!internals) {
    console.error(
      `Form behaviour enabled on ${component.tagName}, but 'internals' was not attached. Ensure the compiler is setting 'static formAssociated = true' on the component class.`
    );
    return;
  }
  const updateFormState = createUpdateFormState(component, internals);
  component._updateFormState = updateFormState;
  component.onUpdated((changedProps) => {
    if (changedProps.has("value") || changedProps.has("required") || changedProps.has("pattern") || changedProps.has("min") || changedProps.has("max")) {
      updateFormState();
    }
  });
  component.onConnected(updateFormState);
  component.onAfterRender(updateFormState);
  attachFormLifecycleCallbacks(component, updateFormState);
  defineFormProperties(component, internals);
};
registerBehaviour("form", formAssociatedBehaviour);
const animationBehaviour = (component) => {
  const setup = window.__piko_animation;
  if (!setup) {
    console.warn(
      `Animation behaviour enabled on ${component.tagName}, but the animation extension is not loaded. Ensure the compiler is adding the extension import.`
    );
    return;
  }
  setup(component);
};
registerBehaviour("animation", animationBehaviour);
function getPiko() {
  return window.piko;
}
const dom = {
  ws(id) {
    return dom.txt(" ", id, { _isWhitespace: true });
  },
  txt(content, id, props = null) {
    const finalProps = { ...props ?? {} };
    return {
      _type: "text",
      text: String(content ?? ""),
      props: finalProps,
      children: null,
      key: id
    };
  },
  html(content, id) {
    return {
      _type: "element",
      tag: "div",
      props: {},
      children: null,
      html: content,
      key: id
    };
  },
  cmt(content, id, props = null) {
    const finalProps = { ...props ?? {} };
    return {
      _type: "comment",
      text: String(content ?? ""),
      props: finalProps,
      children: null,
      key: id
    };
  },
  el(tag, id, props = {}, children = []) {
    const finalProps = { ...props };
    const childArray = normaliseChildren(children);
    return {
      _type: "element",
      tag,
      props: finalProps,
      children: childArray,
      key: id
    };
  },
  frag(id, children = [], props = {}) {
    const finalProps = { ...props };
    const childArray = normaliseChildren(children);
    return {
      _type: "fragment",
      props: finalProps,
      children: childArray,
      key: id
    };
  },
  resolveTag(tag) {
    const s = String(tag ?? "");
    if (s === "") {
      console.warn("<piko:element> resolved to an empty tag name, falling back to <div>");
      return "div";
    }
    if (rejectedPikoElementTargets[s]) {
      console.warn(`<piko:element> cannot target '${s}', falling back to <div>`);
      return "div";
    }
    if (pikoTagMap[s]) {
      return pikoTagMap[s];
    }
    return s;
  },
  pikoEl(rawTag, id, props = {}, children = [], moduleName = "") {
    const tag = dom.resolveTag(rawTag);
    const finalProps = { ...props };
    const rawTagStr = String(rawTag ?? "");
    const pikoNs = getPiko();
    if (pikoLinkTags[rawTagStr]) {
      finalProps["piko:a"] = "";
      const href = String(finalProps["href"] ?? "");
      if (href && pikoNs?.nav) {
        finalProps["onClick"] = (e) => {
          pikoNs.nav.navigateTo(href, e);
        };
      }
    }
    if (pikoAssetTags[rawTagStr] && typeof finalProps["src"] === "string" && pikoNs?.assets) {
      finalProps["src"] = pikoNs.assets.resolve(finalProps["src"], moduleName || void 0);
    }
    return dom.el(tag, id, finalProps, children);
  }
};
const rejectedPikoElementTargets = {
  "piko:partial": true,
  "piko:slot": true,
  "piko:element": true
};
const pikoTagMap = {
  "piko:a": "a",
  "piko:img": "img",
  "piko:svg": "piko-svg-inline",
  "piko:picture": "picture",
  "piko:video": "video"
};
const pikoLinkTags = {
  "piko:a": true
};
const pikoAssetTags = {
  "piko:img": true,
  "piko:svg": true,
  "piko:picture": true,
  "piko:video": true
};
function normaliseChildren(children) {
  if (children == null) {
    return [];
  }
  if (Array.isArray(children)) {
    return children.flat(Infinity).filter(isDef);
  }
  return [children].filter(isDef);
}
function isDef(x) {
  return x != null;
}
function isUndef(x) {
  return x === void 0 || x === null;
}
function isVNodeHidden(vnode) {
  if (vnode.props && Object.prototype.hasOwnProperty.call(vnode.props, "_c")) {
    return !vnode.props._c;
  }
  return false;
}
function areSameVNode(vnode1, vnode2) {
  return vnode1.key === vnode2.key && vnode1._type === vnode2._type && vnode1.tag === vnode2.tag;
}
function createKeyToOldIdxMap(children, beginIndex, endIndex) {
  const map = /* @__PURE__ */ new Map();
  for (let i = beginIndex; i <= endIndex; i++) {
    const childVNode = children[i];
    if (childVNode?.key != null) {
      map.set(childVNode.key, i);
    }
  }
  return map;
}
function getHiddenNodeType(vnode) {
  return vnode._type === "fragment" ? "fragment" : "node";
}
function parseClassData(value) {
  let classes = "";
  if (typeof value === "string" && value) {
    classes += ` ${value}`;
  } else if (Array.isArray(value)) {
    for (const item of value) {
      const nestedClass = parseClassData(item);
      if (nestedClass) {
        classes += ` ${nestedClass}`;
      }
    }
  } else if (typeof value === "object" && value !== null) {
    const objValue = value;
    for (const key in objValue) {
      if (Object.prototype.hasOwnProperty.call(objValue, key) && objValue[key]) {
        classes += ` ${key}`;
      }
    }
  }
  return classes.trim();
}
function parseStyleData(styleValue) {
  if (!styleValue) {
    return "";
  }
  if (typeof styleValue === "string") {
    return styleValue;
  }
  if (typeof styleValue === "object") {
    const entries = Object.entries(styleValue).filter(([, value]) => value != null);
    if (entries.length === 0) {
      return "";
    }
    return `${entries.map(([key, value]) => {
      const cssKey = key.replace(/([A-Z])/g, "-$1").toLowerCase();
      return `${cssKey}: ${value}`;
    }).join("; ")};`;
  }
  return "";
}
function combineClasses(staticClass, dynamicClass) {
  const sClass = staticClass.trim();
  const dClass = dynamicClass.trim();
  if (!sClass && !dClass) {
    return "";
  }
  if (!sClass) {
    return dClass;
  }
  if (!dClass) {
    return sClass;
  }
  return `${sClass} ${dClass}`;
}
function combineStyles(staticStyle, dynamicStyle) {
  const sStyle = staticStyle.replace(/;+\s*$/, "").trim();
  const dStyle = dynamicStyle.replace(/^;+|;+\s*$/g, "").trim();
  if (!sStyle && !dStyle) {
    return "";
  }
  if (!sStyle) {
    return `${dStyle};`;
  }
  if (!dStyle) {
    return `${sStyle};`;
  }
  return `${sStyle}; ${dStyle};`;
}
const SVG_NS = "http://www.w3.org/2000/svg";
const elementReplacements = /* @__PURE__ */ new WeakMap();
function registerElementReplacement(originalElement, replacementElement, options) {
  const entry = {
    element: replacementElement
  };
  if (options?.watchProps && originalElement instanceof Element) {
    entry.watchedProps = options.watchProps;
    entry.watchedValues = {};
    for (const prop of options.watchProps) {
      entry.watchedValues[prop] = originalElement.getAttribute(prop);
    }
  }
  elementReplacements.set(originalElement, entry);
}
function replaceElementWithTracking(originalElement, replacementElement, options) {
  registerElementReplacement(originalElement, replacementElement, options);
  originalElement.replaceWith(replacementElement);
}
function hasWatchedPropsChanged(entry, newProps) {
  if (!entry.watchedProps || !entry.watchedValues) {
    return false;
  }
  for (const prop of entry.watchedProps) {
    const oldValue = entry.watchedValues[prop];
    const newValue = newProps?.[prop];
    if (oldValue !== newValue) {
      return true;
    }
  }
  return false;
}
const RANDOM_KEY_RADIX = 36;
const RANDOM_KEY_SLICE = 2;
const PREFIX_PE_LENGTH = 3;
const PREFIX_ON_LENGTH = 2;
function patchSameFragment(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs) {
  const newIsHidden = isVNodeHidden(newVNode);
  const oldIsHidden = isVNodeHidden(oldVNode);
  newVNode.elm = oldElm;
  if (newIsHidden) {
    patchHiddenFragment(oldVNode, newVNode, oldElm, oldIsHidden, parentElement, nextSiblingNode, refs);
  } else if (oldIsHidden && oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
    parentElement.replaceChild(createElm(newVNode, refs), oldElm);
    newVNode.elm = void 0;
  } else if (!oldIsHidden) {
    updateChildren(parentElement, oldVNode.children ?? [], newVNode.children ?? [], refs, nextSiblingNode);
    newVNode.elm = void 0;
  } else {
    removeVNode(oldVNode, parentElement, refs);
    parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
    newVNode.elm = void 0;
  }
}
function patchElementWithReplacement(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs) {
  const entry = elementReplacements.get(oldElm);
  if (entry?.element.parentNode === parentElement) {
    if (hasWatchedPropsChanged(entry, newVNode.props)) {
      parentElement.removeChild(entry.element);
      clearElmRefsRecursive(oldVNode, refs);
      parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
    } else {
      newVNode.elm = entry.element;
    }
  } else {
    clearElmRefsRecursive(oldVNode, refs);
    parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
  }
}
function patchSameVNode(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs) {
  newVNode.elm = oldElm;
  if (isVNodeHidden(newVNode)) {
    patchHiddenElement(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
    return;
  }
  if (oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
    const newContent = createElm(newVNode, refs);
    if (oldElm.parentNode === parentElement) {
      parentElement.replaceChild(newContent, oldElm);
    } else {
      parentElement.insertBefore(newContent, nextSiblingNode);
    }
    return;
  }
  if (oldElm?.parentNode === parentElement) {
    patchVNode(oldElm, oldVNode, newVNode, refs);
    return;
  }
  if (oldElm) {
    patchElementWithReplacement(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
    return;
  }
  parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
}
function patch(oldVNode, newVNode, refs, parentElement, nextSiblingNode) {
  if (newVNode == null) {
    if (oldVNode != null) {
      removeVNode(oldVNode, parentElement, refs);
    }
    return;
  }
  if (oldVNode === newVNode) {
    return;
  }
  const oldElm = oldVNode?.elm;
  if (newVNode._type === "fragment") {
    if (oldVNode?._type === "fragment" && oldVNode.key === newVNode.key) {
      patchSameFragment(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
    } else {
      if (oldVNode) {
        removeVNode(oldVNode, parentElement, refs);
      }
      const newIsHidden = isVNodeHidden(newVNode);
      const newContent = createElm(newVNode, refs);
      parentElement.insertBefore(newContent, nextSiblingNode);
      newVNode.elm = newIsHidden ? newContent : void 0;
    }
    return;
  }
  if (oldVNode && areSameVNode(oldVNode, newVNode)) {
    patchSameVNode(oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs);
  } else {
    if (oldVNode) {
      removeVNode(oldVNode, parentElement, refs);
    }
    parentElement.insertBefore(createElm(newVNode, refs), nextSiblingNode);
  }
}
function patchHiddenFragment(oldVNode, newVNode, oldElm, oldIsHidden, parentElement, nextSiblingNode, refs) {
  if (!oldIsHidden) {
    removeVNode(oldVNode, parentElement, refs);
    const placeholder = createElm(newVNode, refs);
    parentElement.insertBefore(placeholder, nextSiblingNode);
    newVNode.elm = placeholder;
  } else if (oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
    const message = `hidden fragment _k=${newVNode.key ?? `err-H2H`}`;
    if (oldElm.nodeValue !== message) {
      oldElm.nodeValue = message;
    }
  } else {
    if (oldElm?.parentNode === parentElement) {
      parentElement.removeChild(oldElm);
    }
    clearElmRefsRecursive(oldVNode, refs);
    const placeholder = createElm(newVNode, refs);
    parentElement.insertBefore(placeholder, nextSiblingNode);
    newVNode.elm = placeholder;
  }
}
function patchHiddenElement(_oldVNode, newVNode, oldElm, parentElement, nextSiblingNode, refs) {
  if (oldElm && oldElm.nodeType !== Node.COMMENT_NODE) {
    const placeholder = createElm(newVNode, refs);
    if (oldElm.parentNode === parentElement) {
      parentElement.replaceChild(placeholder, oldElm);
    } else {
      parentElement.insertBefore(placeholder, nextSiblingNode);
    }
    newVNode.elm = placeholder;
  } else if (oldElm && oldElm.nodeType === Node.COMMENT_NODE) {
    const message = `hidden node _k=${newVNode.key ?? `err-elemH2H`}`;
    if (oldElm.nodeValue !== message) {
      oldElm.nodeValue = message;
    }
  } else if (!oldElm) {
    const placeholder = createElm(newVNode, refs);
    parentElement.insertBefore(placeholder, nextSiblingNode);
    newVNode.elm = placeholder;
  }
}
function createElm(vnode, refs, isSvg) {
  if (isVNodeHidden(vnode)) {
    vnode.elm = document.createComment(`hidden ${getHiddenNodeType(vnode)} _k=${vnode.key ?? `err-no-key-hidden-${Math.random().toString(RANDOM_KEY_RADIX).slice(RANDOM_KEY_SLICE)}`}`);
    return vnode.elm;
  }
  switch (vnode._type) {
    case "text":
      vnode.elm = document.createTextNode(vnode.text ?? "");
      return vnode.elm;
    case "comment":
      vnode.elm = document.createComment(vnode.text ?? "");
      return vnode.elm;
    case "fragment": {
      const fragmentDoc = document.createDocumentFragment();
      if (vnode.children) {
        for (const child of vnode.children) {
          fragmentDoc.appendChild(createElm(child, refs, isSvg));
        }
      }
      return fragmentDoc;
    }
    case "element":
      return createElementNode(vnode, refs, isSvg);
    default:
      vnode.elm = document.createTextNode("");
      return vnode.elm;
  }
}
function createElementNode(elVNode, refs, isSvg) {
  const tagIsSvg = elVNode.tag === "svg";
  const childSvg = isSvg === true || tagIsSvg === true;
  const elementNode = childSvg ? document.createElementNS(SVG_NS, elVNode.tag) : document.createElement(elVNode.tag);
  elVNode.elm = elementNode;
  patchProps(elementNode, {}, elVNode.props ?? {}, refs);
  if (elVNode.html != null) {
    elementNode.innerHTML = elVNode.html;
  } else if (elVNode.children) {
    for (const child of elVNode.children) {
      elementNode.appendChild(createElm(child, refs, childSvg));
    }
  }
  return elementNode;
}
function clearElmRefsRecursive(vnode, refs) {
  if (!vnode) {
    return;
  }
  if (vnode.children) {
    for (const child of vnode.children) {
      clearElmRefsRecursive(child, refs);
    }
  }
  if (refs && vnode.props?._ref && typeof vnode.props._ref === "string") {
    const refKey = vnode.props._ref;
    if (refs[refKey] === vnode.elm) {
      delete refs[refKey];
    }
  }
  vnode.elm = void 0;
}
function removeVNode(vnodeToRemove, parentElement, refs = null) {
  if (!vnodeToRemove) {
    return;
  }
  if (vnodeToRemove._type === "fragment" && !isVNodeHidden(vnodeToRemove)) {
    if (vnodeToRemove.children) {
      for (const child of vnodeToRemove.children) {
        removeVNode(child, parentElement, refs);
      }
    }
  } else if (vnodeToRemove.elm?.parentNode === parentElement) {
    parentElement.removeChild(vnodeToRemove.elm);
  }
  clearElmRefsRecursive(vnodeToRemove, refs);
}
function addVNodes(parentElement, referenceNode, vnodesToAdd, startIndex, endIndex, refs) {
  for (let i = startIndex; i <= endIndex; i++) {
    const childVNode = vnodesToAdd[i];
    if (childVNode) {
      const createdDomMaterial = createElm(childVNode, refs);
      parentElement.insertBefore(createdDomMaterial, referenceNode);
      if (childVNode._type === "fragment" && !isVNodeHidden(childVNode)) {
        childVNode.elm = void 0;
      }
    }
  }
}
function getFirstDomElementRecursive(vnode) {
  if (!vnode) {
    return null;
  }
  if (vnode.elm && vnode.elm.nodeType !== Node.DOCUMENT_FRAGMENT_NODE) {
    return vnode.elm;
  }
  if (vnode._type !== "fragment" || isVNodeHidden(vnode) || !vnode.children) {
    return null;
  }
  for (const child of vnode.children) {
    const firstChildDom = getFirstDomElementRecursive(child);
    if (firstChildDom) {
      return firstChildDom;
    }
  }
  return null;
}
function patchElementVNode(domElement, oldVNode, newVNode, refs) {
  if (domElement.nodeType === Node.COMMENT_NODE && !isVNodeHidden(newVNode)) {
    console.error("PPElement Error: patchVNode attempting to patch a comment as a visible element.", {
      oldVNode,
      newVNode,
      domElement
    });
    return;
  }
  patchProps(domElement, oldVNode.props ?? {}, newVNode.props ?? {}, refs);
  if (newVNode.html != null) {
    if (oldVNode.children && oldVNode.children.length > 0) {
      domElement.textContent = "";
    }
    if (oldVNode.html !== newVNode.html) {
      domElement.innerHTML = newVNode.html;
    }
  } else {
    const oldChildren = oldVNode.children ?? [];
    const newChildren = newVNode.children ?? [];
    if (oldChildren.length > 0 || newChildren.length > 0) {
      updateChildren(domElement, oldChildren, newChildren, refs, null);
    }
  }
}
function patchVNode(domElement, oldVNode, newVNode, refs) {
  newVNode.elm = domElement;
  if (oldVNode === newVNode) {
    return;
  }
  if (isVNodeHidden(newVNode) && newVNode._type !== "comment") {
    if (domElement.nodeType === Node.COMMENT_NODE) {
      const message = `hidden node _k=${newVNode.key ?? "no-key"}`;
      if (domElement.nodeValue !== message) {
        domElement.nodeValue = message;
      }
    }
    return;
  }
  if (newVNode._type === "text") {
    if (oldVNode.text !== newVNode.text) {
      domElement.nodeValue = newVNode.text ?? "";
    }
  } else if (newVNode._type === "comment") {
    if (oldVNode.text !== newVNode.text) {
      domElement.nodeValue = newVNode.text ?? "";
    }
  } else if (newVNode._type === "element") {
    patchElementVNode(domElement, oldVNode, newVNode, refs);
  }
}
function patchAndRelocateToEnd(oldVNode, newVNode, nextOldSibling, oldEndVNode, refs, parentElement, overallInsertBeforeNode) {
  const nextOldSiblingElm = getFirstDomElementRecursive(nextOldSibling);
  patch(oldVNode, newVNode, refs, parentElement, nextOldSiblingElm ?? overallInsertBeforeNode);
  const domToMove = newVNode.elm ?? getFirstDomElementRecursive(newVNode);
  const referenceForMove = getFirstDomElementRecursive(oldEndVNode)?.nextSibling ?? overallInsertBeforeNode;
  if (domToMove) {
    parentElement.insertBefore(domToMove, referenceForMove);
  }
}
function patchAndRelocateToStart(oldVNode, newVNode, oldStartVNode, refs, parentElement, overallInsertBeforeNode) {
  const referenceForPatch = getFirstDomElementRecursive(oldStartVNode);
  patch(oldVNode, newVNode, refs, parentElement, referenceForPatch ?? overallInsertBeforeNode);
  const domToMove = newVNode.elm ?? getFirstDomElementRecursive(newVNode);
  if (domToMove) {
    parentElement.insertBefore(domToMove, referenceForPatch ?? overallInsertBeforeNode);
  }
}
function removeRemainingOldChildren(oldChildren, startIdx, endIdx, parentElement, refs) {
  for (let i = startIdx; i <= endIdx; i++) {
    const child = oldChildren[i];
    if (child) {
      removeVNode(child, parentElement, refs);
    }
  }
}
function addRemainingNewChildren(newChildren, startIdx, endIdx, parentElement, refs, overallInsertBeforeNode) {
  const insertRef = getFirstDomElementRecursive(newChildren[endIdx + 1]) ?? overallInsertBeforeNode;
  addVNodes(parentElement, insertRef, newChildren, startIdx, endIdx, refs);
}
function resolveKeyedChild(oldChildren, oldKeyToIdxMap, newStartVNode, oldStartVNode, refs, parentElement, overallInsertBeforeNode) {
  const idxInOld = newStartVNode.key != null ? oldKeyToIdxMap.get(newStartVNode.key) : void 0;
  const referenceForInsert = getFirstDomElementRecursive(oldStartVNode);
  if (isUndef(idxInOld)) {
    patch(null, newStartVNode, refs, parentElement, referenceForInsert ?? overallInsertBeforeNode);
    return;
  }
  const vnodeToMove = oldChildren[idxInOld];
  if (vnodeToMove && areSameVNode(vnodeToMove, newStartVNode)) {
    patch(vnodeToMove, newStartVNode, refs, parentElement, referenceForInsert ?? overallInsertBeforeNode);
    oldChildren[idxInOld] = void 0;
    const domToActuallyMove = newStartVNode.elm ?? getFirstDomElementRecursive(newStartVNode);
    if (domToActuallyMove && referenceForInsert !== domToActuallyMove) {
      parentElement.insertBefore(domToActuallyMove, referenceForInsert ?? overallInsertBeforeNode);
    }
    return;
  }
  patch(null, newStartVNode, refs, parentElement, referenceForInsert ?? overallInsertBeforeNode);
  if (vnodeToMove) {
    removeVNode(vnodeToMove, parentElement, refs);
  }
}
function updateChildren(parentElement, oldChildren, newChildren, refs, overallInsertBeforeNode) {
  let oldStartIdx = 0, newStartIdx = 0;
  let oldEndIdx = oldChildren.length - 1;
  let newEndIdx = newChildren.length - 1;
  let oldStartVNode = oldChildren[0];
  let oldEndVNode = oldChildren[oldEndIdx];
  let newStartVNode = newChildren[0];
  let newEndVNode = newChildren[newEndIdx];
  let oldKeyToIdxMap;
  while (oldStartIdx <= oldEndIdx && newStartIdx <= newEndIdx) {
    if (isUndef(oldStartVNode)) {
      oldStartVNode = oldChildren[++oldStartIdx];
      continue;
    }
    if (isUndef(oldEndVNode)) {
      oldEndVNode = oldChildren[--oldEndIdx];
      continue;
    }
    if (isUndef(newStartVNode)) {
      newStartVNode = newChildren[++newStartIdx];
      continue;
    }
    if (isUndef(newEndVNode)) {
      newEndVNode = newChildren[--newEndIdx];
      continue;
    }
    if (areSameVNode(oldStartVNode, newStartVNode)) {
      const ref = getFirstDomElementRecursive(oldChildren[oldStartIdx + 1]);
      patch(oldStartVNode, newStartVNode, refs, parentElement, ref ?? overallInsertBeforeNode);
      oldStartVNode = oldChildren[++oldStartIdx];
      newStartVNode = newChildren[++newStartIdx];
    } else if (areSameVNode(oldEndVNode, newEndVNode)) {
      patch(oldEndVNode, newEndVNode, refs, parentElement, overallInsertBeforeNode);
      oldEndVNode = oldChildren[--oldEndIdx];
      newEndVNode = newChildren[--newEndIdx];
    } else if (areSameVNode(oldStartVNode, newEndVNode)) {
      patchAndRelocateToEnd(
        oldStartVNode,
        newEndVNode,
        oldChildren[oldStartIdx + 1],
        oldEndVNode,
        refs,
        parentElement,
        overallInsertBeforeNode
      );
      oldStartVNode = oldChildren[++oldStartIdx];
      newEndVNode = newChildren[--newEndIdx];
    } else if (areSameVNode(oldEndVNode, newStartVNode)) {
      patchAndRelocateToStart(
        oldEndVNode,
        newStartVNode,
        oldStartVNode,
        refs,
        parentElement,
        overallInsertBeforeNode
      );
      oldEndVNode = oldChildren[--oldEndIdx];
      newStartVNode = newChildren[++newStartIdx];
    } else {
      oldKeyToIdxMap ??= createKeyToOldIdxMap(oldChildren, oldStartIdx, oldEndIdx);
      resolveKeyedChild(
        oldChildren,
        oldKeyToIdxMap,
        newStartVNode,
        oldStartVNode,
        refs,
        parentElement,
        overallInsertBeforeNode
      );
      newStartVNode = newChildren[++newStartIdx];
    }
  }
  if (oldStartIdx <= oldEndIdx) {
    removeRemainingOldChildren(oldChildren, oldStartIdx, oldEndIdx, parentElement, refs);
  } else if (newStartIdx <= newEndIdx) {
    addRemainingNewChildren(newChildren, newStartIdx, newEndIdx, parentElement, refs, overallInsertBeforeNode);
  }
}
function removeStaleProps(htmlElement, oldProps, newProps, refs) {
  for (const propName in oldProps) {
    if (!(propName in newProps) || oldProps[propName] !== newProps[propName]) {
      if (propName.startsWith("?")) {
        htmlElement.removeAttribute(propName.slice(1));
      } else if (propName.startsWith("on")) {
        const parsed = parseEventPropKey(propName, PREFIX_ON_LENGTH);
        toggleListener(htmlElement, parsed.eventName, oldProps[propName], false, parsed.listenerOptions);
      } else if (propName.startsWith("pe:")) {
        const parsed = parseEventPropKey(propName, PREFIX_PE_LENGTH);
        toggleListener(htmlElement, parsed.eventName, oldProps[propName], false, parsed.listenerOptions);
      } else if (propName === "_ref" && refs && oldProps[propName] && refs[oldProps[propName]] === htmlElement) {
        delete refs[oldProps[propName]];
      } else if (!["_k", "_c", "_s", "class", "_class", "style", "_style"].includes(propName)) {
        htmlElement.removeAttribute(propName);
      }
    }
  }
}
function applyAttributeValue(htmlElement, propName, newValue) {
  if (newValue == null || newValue === false) {
    htmlElement.removeAttribute(propName);
    return;
  }
  if ((htmlElement.tagName === "INPUT" || htmlElement.tagName === "TEXTAREA" || htmlElement.tagName === "SELECT") && propName === "value") {
    if (htmlElement.value !== String(newValue)) {
      htmlElement.value = String(newValue);
    }
    return;
  }
  if (typeof newValue === "object") {
    try {
      htmlElement.setAttribute(propName, JSON.stringify(newValue));
    } catch (error) {
      console.warn("[renderer] Failed to JSON.stringify prop, falling back to String():", {
        propName,
        value: newValue,
        error
      });
      htmlElement.setAttribute(propName, String(newValue));
    }
    return;
  }
  htmlElement.setAttribute(propName, String(newValue));
}
function applyPropValue(htmlElement, propName, oldValue, newValue, refs) {
  if (propName.startsWith("?")) {
    const realAttrName = propName.slice(1);
    if (newValue) {
      htmlElement.setAttribute(realAttrName, "");
    } else {
      htmlElement.removeAttribute(realAttrName);
    }
    return;
  }
  if (propName.startsWith("on")) {
    const parsed = parseEventPropKey(propName, PREFIX_ON_LENGTH);
    if (oldValue !== newValue) {
      toggleListener(htmlElement, parsed.eventName, oldValue, false, parsed.listenerOptions);
      toggleListener(htmlElement, parsed.eventName, newValue, true, parsed.listenerOptions);
    }
    return;
  }
  if (propName.startsWith("pe:")) {
    const parsed = parseEventPropKey(propName, PREFIX_PE_LENGTH);
    if (oldValue !== newValue) {
      toggleListener(htmlElement, parsed.eventName, oldValue, false, parsed.listenerOptions);
      toggleListener(htmlElement, parsed.eventName, newValue, true, parsed.listenerOptions);
    }
    return;
  }
  if (propName === "_ref") {
    if (!refs) {
      return;
    }
    if (oldValue && refs[oldValue] === htmlElement) {
      delete refs[oldValue];
    }
    if (newValue && typeof newValue === "string") {
      refs[newValue] = htmlElement;
    }
    return;
  }
  applyAttributeValue(htmlElement, propName, newValue);
}
function applyNewProps(htmlElement, oldProps, newProps, refs) {
  for (const propName in newProps) {
    const newValue = newProps[propName];
    const oldValue = oldProps[propName];
    if (["_k", "_c", "_s", "class", "_class", "style", "_style"].includes(propName)) {
      continue;
    }
    if (propName !== "value" && newValue === oldValue && typeof newValue !== "function") {
      continue;
    }
    if (propName !== "value" && newValue === oldValue) {
      continue;
    }
    applyPropValue(htmlElement, propName, oldValue, newValue, refs);
  }
}
function reconcileClassAttribute(htmlElement, newProps) {
  const staticClassValue = newProps.class ?? "";
  const dynamicClassValue = newProps._class ?? null;
  const finalClass = combineClasses(staticClassValue, parseClassData(dynamicClassValue));
  if (finalClass.trim()) {
    if (htmlElement.getAttribute("class") !== finalClass.trim()) {
      htmlElement.setAttribute("class", finalClass.trim());
    }
  } else {
    htmlElement.removeAttribute("class");
  }
}
function reconcileStyleAttribute(htmlElement, newProps) {
  const staticStyleValue = newProps.style ?? "";
  const dynamicStyleValue = newProps._style ?? null;
  const finalStyle = combineStyles(staticStyleValue, parseStyleData(dynamicStyleValue));
  if (finalStyle.trim()) {
    if (htmlElement.getAttribute("style") !== finalStyle.trim()) {
      htmlElement.setAttribute("style", finalStyle.trim());
    }
  } else {
    htmlElement.removeAttribute("style");
  }
}
function applyVisibility(htmlElement, newProps) {
  const shouldShow = newProps._s !== false;
  if (!shouldShow) {
    if (htmlElement.style.display !== "none") {
      htmlElement.style.display = "none";
    }
  } else if (htmlElement.style.display === "none") {
    htmlElement.style.display = "";
  }
}
function patchProps(htmlElement, oldProps, newProps, refs) {
  if (!htmlElement) {
    console.error("PPElement Error: patchProps called with undefined htmlElement.", { oldProps, newProps });
    return;
  }
  removeStaleProps(htmlElement, oldProps, newProps, refs);
  applyNewProps(htmlElement, oldProps, newProps, refs);
  reconcileClassAttribute(htmlElement, newProps);
  reconcileStyleAttribute(htmlElement, newProps);
  applyVisibility(htmlElement, newProps);
}
function parseEventPropKey(propName, prefixLen) {
  const raw = propName.slice(prefixLen);
  const delimiterIndex = raw.indexOf("$");
  if (delimiterIndex === -1) {
    return { eventName: raw.toLowerCase() };
  }
  const eventName = raw.slice(0, delimiterIndex).toLowerCase();
  const opts = {};
  for (const s of raw.slice(delimiterIndex + 1).split("$")) {
    if (s === "capture") {
      opts.capture = true;
    }
    if (s === "passive") {
      opts.passive = true;
    }
  }
  return { eventName, listenerOptions: opts };
}
function toggleListener(htmlElement, eventName, handler, add, listenerOptions) {
  if (!handler) {
    return;
  }
  const options = { ...listenerOptions };
  if ((eventName === "focus" || eventName === "blur") && options.capture === void 0) {
    options.capture = true;
  }
  const method = add ? "addEventListener" : "removeEventListener";
  if (Array.isArray(handler)) {
    for (const func of handler) {
      if (typeof func === "function") {
        htmlElement[method](eventName, func, options);
      }
    }
  } else if (typeof handler === "function") {
    htmlElement[method](eventName, handler, options);
  }
}
function propertyToAttributeName(propertyName) {
  return propertyName.replace(/[A-Z]/g, (match) => `-${match.toLowerCase()}`);
}
function attributeToPropertyNameFn(attributeName, propTypes) {
  for (const propName in propTypes) {
    if (propertyToAttributeName(propName) === attributeName) {
      return propName;
    }
  }
  return attributeName.replace(/-([a-z])/g, (_match, letter) => letter.toUpperCase());
}
function shouldReflectProperty(propDef) {
  if (!propDef) {
    return false;
  }
  if (propDef.reflectToAttribute === true) {
    return true;
  }
  if (propDef.reflectToAttribute === false) {
    return false;
  }
  return propDef.type === "string" || propDef.type === "number" || propDef.type === "boolean";
}
function createPropTypeRegistry(options) {
  const propTypes = options.propTypes ?? {};
  return {
    get(propertyName) {
      return propTypes[propertyName];
    },
    getPropertyNames() {
      return Object.keys(propTypes);
    },
    deriveObservedAttributes() {
      const attributesToObserve = [];
      for (const propName in propTypes) {
        const propDef = propTypes[propName];
        if (shouldReflectProperty(propDef)) {
          attributesToObserve.push(propertyToAttributeName(propName));
        }
      }
      return attributesToObserve;
    },
    getDefaultValue(propertyName) {
      const propDef = propTypes[propertyName];
      if (propDef?.default === void 0) {
        return void 0;
      }
      return typeof propDef.default === "function" ? propDef.default() : propDef.default;
    },
    shouldReflect(propertyName) {
      return shouldReflectProperty(propTypes[propertyName]);
    },
    propertyToAttributeName,
    attributeToPropertyName(attributeName) {
      return attributeToPropertyNameFn(attributeName, propTypes);
    }
  };
}
function createStateManager(options) {
  let context;
  const changedPropsSet = /* @__PURE__ */ new Set();
  return {
    getState() {
      return context?.state;
    },
    setState(partialState) {
      if (!context?.state) {
        console.warn(`PPElement ${options.tagName}: setState called before state was initialised.`);
        return;
      }
      Object.assign(context.state, partialState);
    },
    getContext() {
      return context;
    },
    setContext(ctx) {
      context = ctx;
    },
    hasState() {
      return context?.state !== void 0;
    },
    getChangedProps() {
      return changedPropsSet;
    },
    clearChangedProps() {
      const copy = new Set(changedPropsSet);
      changedPropsSet.clear();
      return copy;
    },
    recordChange(propertyName) {
      changedPropsSet.add(propertyName);
    }
  };
}
function createFormAssociation(options) {
  let internals;
  if (options.formAssociated) {
    internals = options.host.attachInternals();
  }
  return {
    getInternals() {
      return internals;
    },
    isFormAssociated() {
      return options.formAssociated;
    }
  };
}
function createBehaviourApplicator(options) {
  const { host, enabledBehaviours, getBehaviour: getBehaviour2 } = options;
  return {
    applyBehaviours() {
      for (const behaviourName of enabledBehaviours) {
        const setupFunction = getBehaviour2(behaviourName);
        if (setupFunction) {
          setupFunction(host);
        } else {
          console.warn(
            `PPElement: Unknown behaviour "${behaviourName}" enabled for ${host.tagName}.`
          );
        }
      }
    }
  };
}
const RESET_CSS = `*, *::before, *::after { margin:0; padding:0; box-sizing:border-box; } :host { display: block; }`;
function createShadowDOMService(options) {
  const { host, componentCSS, mode = "open", delegatesFocus = false } = options;
  return {
    ensureShadowRoot() {
      if (host.shadowRoot) {
        return host.shadowRoot;
      }
      const shadow = host.attachShadow({ mode, delegatesFocus, serializable: true });
      const resetStyleEl = document.createElement("style");
      resetStyleEl.textContent = RESET_CSS;
      shadow.appendChild(resetStyleEl);
      if (typeof componentCSS === "string" && componentCSS.trim()) {
        const userStyleElement = document.createElement("style");
        userStyleElement.textContent = componentCSS;
        shadow.appendChild(userStyleElement);
      }
      return shadow;
    },
    getShadowRoot() {
      return host.shadowRoot;
    },
    hasShadowRoot() {
      return host.shadowRoot !== null;
    }
  };
}
function buildSlotSelector(slotName) {
  return slotName ? `slot[name="${slotName}"]` : `slot:not([name])`;
}
function createSlotManager(options) {
  const { getShadowRoot, tagName } = options;
  const pendingListeners = [];
  function doAttach(shadowRoot, slotName, callback) {
    const selector = buildSlotSelector(slotName);
    const slotElement = shadowRoot.querySelector(selector);
    if (!slotElement) {
      console.warn(
        `PPElement: Slot "${selector}" not found in ${tagName} for attaching listener.`
      );
      return;
    }
    const handleSlotChange = () => {
      callback(slotElement.assignedElements({ flatten: true }));
    };
    slotElement.addEventListener("slotchange", handleSlotChange);
    handleSlotChange();
  }
  return {
    getSlottedElements(slotName = "") {
      const shadowRoot = getShadowRoot();
      if (!shadowRoot) {
        return [];
      }
      const selector = buildSlotSelector(slotName);
      const slotElement = shadowRoot.querySelector(selector);
      return slotElement?.assignedElements({ flatten: true }) ?? [];
    },
    attachSlotListener(slotName, callback) {
      const shadowRoot = getShadowRoot();
      if (!shadowRoot) {
        pendingListeners.push({ slotName, callback });
        return;
      }
      doAttach(shadowRoot, slotName, callback);
    },
    flushPendingListeners() {
      if (pendingListeners.length === 0) {
        return;
      }
      const shadowRoot = getShadowRoot();
      if (!shadowRoot) {
        return;
      }
      const listeners = pendingListeners.splice(0);
      for (const { slotName, callback } of listeners) {
        doAttach(shadowRoot, slotName, callback);
      }
    },
    hasSlotContent(slotName = "") {
      return this.getSlottedElements(slotName).length > 0;
    }
  };
}
function createLifecycleManager() {
  const onConnectedCallbacks = [];
  const onDisconnectedCallbacks = [];
  const onBeforeRenderCallbacks = [];
  const onAfterRenderCallbacks = [];
  const onUpdatedCallbacks = [];
  const onCleanupCallbacks = [];
  let connectedOnce = false;
  return {
    onConnected(callback) {
      onConnectedCallbacks.push(callback);
    },
    onDisconnected(callback) {
      onDisconnectedCallbacks.push(callback);
    },
    onBeforeRender(callback) {
      onBeforeRenderCallbacks.push(callback);
    },
    onAfterRender(callback) {
      onAfterRenderCallbacks.push(callback);
    },
    onUpdated(callback) {
      onUpdatedCallbacks.push(callback);
    },
    onCleanup(callback) {
      onCleanupCallbacks.push(callback);
    },
    executeConnected() {
      if (connectedOnce) {
        return;
      }
      connectedOnce = true;
      onConnectedCallbacks.forEach((cb) => cb());
    },
    executeDisconnected() {
      onDisconnectedCallbacks.forEach((cb) => cb());
    },
    executeBeforeRender() {
      onBeforeRenderCallbacks.forEach((cb) => cb());
    },
    executeAfterRender() {
      onAfterRenderCallbacks.forEach((cb) => cb());
    },
    executeUpdated(changedProperties) {
      onUpdatedCallbacks.forEach((cb) => cb(changedProperties));
    },
    executeCleanups() {
      onCleanupCallbacks.forEach((cb) => cb());
      onCleanupCallbacks.length = 0;
    },
    resetConnectedState() {
      connectedOnce = false;
    },
    hasConnectedOnce() {
      return connectedOnce;
    }
  };
}
function parseNumberAttribute(attributeValue, defaultValue) {
  const num = parseFloat(attributeValue);
  if (isNaN(num)) {
    return typeof defaultValue === "number" ? defaultValue : 0;
  }
  return num;
}
function parseJsonAttribute(attributeValue, typeHint, defaultValue) {
  try {
    return JSON.parse(attributeValue);
  } catch {
    if (defaultValue !== void 0) {
      return defaultValue;
    }
    return typeHint === "array" ? [] : null;
  }
}
function translateValue(typeHint, attributeValue, defaultValue, isNullable) {
  if (attributeValue === null) {
    if (typeHint === "boolean") {
      return false;
    }
    return isNullable ? null : defaultValue;
  }
  switch (typeHint) {
    case "boolean":
      return attributeValue !== "false";
    case "number":
      return parseNumberAttribute(attributeValue, defaultValue);
    case "string":
      return attributeValue;
    case "array":
    case "object":
    case "json":
      return parseJsonAttribute(attributeValue, typeHint, defaultValue);
    default:
      return attributeValue;
  }
}
function valueToAttributeString(propType, propertyValue) {
  if (propType === "boolean") {
    return propertyValue === true ? "true" : "false";
  }
  if (propType === "json" || propType === "object" || propType === "array") {
    return JSON.stringify(propertyValue);
  }
  return String(propertyValue);
}
function applyToState(propertyName, attributeValue, options, syncState) {
  const { propTypeRegistry, getState, getDefaults } = options;
  const propDef = propTypeRegistry.get(propertyName);
  const state = getState();
  if (!propDef || !state) {
    return;
  }
  syncState.applyingToState = true;
  try {
    const defaultValue = getDefaults()[propertyName];
    const typedValue = translateValue(
      propDef.type ?? "any",
      attributeValue,
      defaultValue,
      propDef.nullable ?? false
    );
    if (state[propertyName] !== typedValue) {
      state[propertyName] = typedValue;
    }
  } finally {
    syncState.applyingToState = false;
  }
}
function reflectToAttribute(propertyName, propertyValue, options, syncState) {
  if (syncState.applyingToState && !syncState.initialising) {
    return;
  }
  const { host, propTypeRegistry } = options;
  const propDef = propTypeRegistry.get(propertyName);
  if (!shouldReflectProperty(propDef)) {
    return;
  }
  syncState.reflectingToAttribute = true;
  try {
    const attributeName = propTypeRegistry.propertyToAttributeName(propertyName);
    const propType = propDef?.type ?? "any";
    if (propertyValue === null || propertyValue === void 0) {
      if (host.hasAttribute(attributeName)) {
        host.removeAttribute(attributeName);
      }
    } else if (propType === "boolean") {
      host.toggleAttribute(attributeName, propertyValue === true);
    } else {
      const stringValue = valueToAttributeString(propType, propertyValue);
      if (host.getAttribute(attributeName) !== stringValue) {
        host.setAttribute(attributeName, stringValue);
      }
    }
  } finally {
    syncState.reflectingToAttribute = false;
  }
}
function createAttributeSyncService(options) {
  const syncState = { applyingToState: false, reflectingToAttribute: false, initialising: true };
  return {
    applyAttributeToState(propertyName, attributeValue) {
      applyToState(propertyName, attributeValue, options, syncState);
    },
    reflectStateToAttribute(propertyName, propertyValue) {
      reflectToAttribute(propertyName, propertyValue, options, syncState);
    },
    handleAttributeChanged(attributeName, _oldValue, newValue) {
      if (syncState.reflectingToAttribute || !options.getState() || syncState.initialising) {
        return;
      }
      const propertyName = options.propTypeRegistry.attributeToPropertyName(attributeName);
      if (options.propTypeRegistry.get(propertyName)) {
        this.applyAttributeToState(propertyName, newValue);
      }
    },
    translateAttributeValue(typeHint, attributeValue, propertyName, isNullable = false) {
      return translateValue(typeHint, attributeValue, options.getDefaults()[propertyName], isNullable);
    },
    syncAllAttributesToState() {
      const propNames = options.propTypeRegistry.getPropertyNames();
      for (const attribute of Array.from(options.host.attributes)) {
        const propertyName = options.propTypeRegistry.attributeToPropertyName(attribute.name);
        if (propNames.includes(propertyName)) {
          this.applyAttributeToState(propertyName, attribute.value);
        }
      }
    },
    reflectAllStateToAttributes() {
      const state = options.getState();
      if (!state) {
        return;
      }
      for (const propName in state) {
        this.reflectStateToAttribute(propName, state[propName]);
      }
    },
    isApplyingToState: () => syncState.applyingToState,
    isReflectingToAttribute: () => syncState.reflectingToAttribute,
    isInitialising: () => syncState.initialising,
    setInitialising: (value) => {
      syncState.initialising = value;
    }
  };
}
function reflectChangedProps(ctx) {
  const { stateManager, attributeSyncService } = ctx.options;
  const changedProps = stateManager.getChangedProps();
  if (changedProps.size === 0) {
    return;
  }
  const changedPropsCopy = stateManager.clearChangedProps();
  const state = stateManager.getState();
  if (state) {
    changedPropsCopy.forEach((propName) => {
      if (Object.prototype.hasOwnProperty.call(state, propName)) {
        attributeSyncService.reflectStateToAttribute(propName, state[propName]);
      }
    });
  }
  ctx.options.lifecycleManager.executeUpdated(changedPropsCopy);
}
function executeRender(ctx) {
  const { getShadowRoot, isConnected, renderVDOM, refs, lifecycleManager, onRenderComplete } = ctx.options;
  const shadowRoot = getShadowRoot();
  if (!shadowRoot || !isConnected()) {
    return ctx.oldVDOM;
  }
  lifecycleManager.executeBeforeRender();
  const newVirtualTree = renderVDOM();
  patch(ctx.oldVDOM, newVirtualTree, refs, shadowRoot, null);
  reflectChangedProps(ctx);
  lifecycleManager.executeAfterRender();
  onRenderComplete?.();
  return newVirtualTree;
}
function createRenderScheduler(options) {
  let oldVDOM = null;
  let renderScheduled = false;
  let pendingAfterInit = false;
  const ctx = {
    get oldVDOM() {
      return oldVDOM;
    },
    options
  };
  return {
    scheduleRender() {
      if (options.isInitialising()) {
        pendingAfterInit = true;
        return;
      }
      if (!options.isConnected()) {
        return;
      }
      if (!renderScheduled) {
        renderScheduled = true;
        queueMicrotask(() => {
          renderScheduled = false;
          if (options.isConnected()) {
            oldVDOM = executeRender(ctx);
            options.lifecycleManager.executeConnected();
          }
        });
      }
    },
    render() {
      oldVDOM = executeRender(ctx);
    },
    getOldVDOM() {
      return oldVDOM;
    },
    isScheduled() {
      return renderScheduled;
    },
    hasRendered() {
      return oldVDOM !== null;
    },
    setPendingAfterInit(value) {
      pendingAfterInit = value;
    },
    hasPendingAfterInit() {
      return pendingAfterInit;
    }
  };
}
const _PPElement = class _PPElement extends HTMLElement {
  /** Creates a new PPElement instance and initialises all composed services. */
  constructor() {
    super();
    this.refs = {};
    this._initServices();
  }
  /** Returns the previous virtual node from the last render. */
  get oldVDOM() {
    return this._renderScheduler.getOldVDOM();
  }
  /** Returns whether a render is currently scheduled. */
  get renderScheduled() {
    return this._renderScheduler.isScheduled();
  }
  /** No-op setter kept for backward compatibility with tests. */
  set renderScheduled(_value) {
  }
  /** Returns the computed default values from prop type definitions. */
  get defaults() {
    return this._getDefaults();
  }
  /** Returns the set of property names that have changed since the last render. */
  get changedPropsSet() {
    return this._stateManager.getChangedProps();
  }
  /**
   * Creates and wires all composed services in dependency order.
   */
  _initServices() {
    const constructor = this.constructor;
    this._propTypeRegistry = createPropTypeRegistry({
      propTypes: constructor.propTypes
    });
    this._stateManager = createStateManager({
      tagName: this.tagName
    });
    this._formAssociation = createFormAssociation({
      host: this,
      formAssociated: constructor.formAssociated
    });
    this._behaviourApplicator = createBehaviourApplicator({
      host: this,
      enabledBehaviours: constructor.enabledBehaviours,
      getBehaviour
    });
    this._shadowDOMService = createShadowDOMService({
      host: this,
      componentCSS: constructor.css,
      delegatesFocus: constructor.formAssociated
    });
    this._slotManager = createSlotManager({
      getShadowRoot: () => this.shadowRoot,
      tagName: this.tagName
    });
    this._lifecycleManager = createLifecycleManager();
    this._attributeSyncService = createAttributeSyncService({
      host: this,
      propTypeRegistry: this._propTypeRegistry,
      getState: () => this._stateManager.getState(),
      getDefaults: () => this._getDefaults()
    });
    this._renderScheduler = createRenderScheduler({
      getShadowRoot: () => this.shadowRoot,
      isConnected: () => this.isConnected,
      isInitialising: () => this._attributeSyncService.isInitialising(),
      renderVDOM: () => this.renderVDOM(),
      refs: this.refs,
      lifecycleManager: this._lifecycleManager,
      stateManager: this._stateManager,
      attributeSyncService: this._attributeSyncService,
      onRenderComplete: () => this._slotManager.flushPendingListeners()
    });
    this._behaviourApplicator.applyBehaviours();
  }
  /**
   * Computes and caches default values from prop type definitions.
   *
   * @returns The cached defaults record.
   */
  _getDefaults() {
    if (this._defaults) {
      return this._defaults;
    }
    this._defaults = {};
    for (const propName of this._propTypeRegistry.getPropertyNames()) {
      const defaultValue = this._propTypeRegistry.getDefaultValue(propName);
      if (defaultValue !== void 0) {
        this._defaults[propName] = defaultValue;
      }
    }
    return this._defaults;
  }
  /**
   * Returns the list of attribute names to observe for changes.
   *
   * @returns The attribute names derived from prop type definitions.
   */
  static get observedAttributes() {
    const registry = createPropTypeRegistry({ propTypes: this.propTypes });
    return registry.deriveObservedAttributes();
  }
  /**
   * Returns the component CSS to inject into the shadow root.
   *
   * @returns The CSS string, or undefined if none.
   */
  static get css() {
    return void 0;
  }
  /** Returns the component state context for backward compatibility. */
  get $$ctx() {
    return this._stateManager.getContext();
  }
  /** Returns the ElementInternals instance for form-associated components. */
  get internals() {
    return this._formAssociation.getInternals();
  }
  /** Returns the current component state object. */
  get state() {
    return this._stateManager.getState();
  }
  /**
   * Merges partial state into the component state and schedules a render.
   *
   * @param partialState - The state properties to update.
   */
  setState(partialState) {
    this._stateManager.setState(partialState);
  }
  /**
   * Registers a callback to run when the component connects to the DOM.
   *
   * @param callback - The function to call on connection.
   */
  onConnected(callback) {
    this._lifecycleManager.onConnected(callback);
  }
  /**
   * Registers a callback to run when the component disconnects from the DOM.
   *
   * @param callback - The function to call on disconnection.
   */
  onDisconnected(callback) {
    this._lifecycleManager.onDisconnected(callback);
  }
  /**
   * Registers a callback to run before each render.
   *
   * @param callback - The function to call before rendering.
   */
  onBeforeRender(callback) {
    this._lifecycleManager.onBeforeRender(callback);
  }
  /**
   * Registers a callback to run after each render.
   *
   * @param callback - The function to call after rendering.
   */
  onAfterRender(callback) {
    this._lifecycleManager.onAfterRender(callback);
  }
  /**
   * Registers a callback to run when component properties change.
   *
   * @param callback - The function to call with the set of changed property names.
   */
  onUpdated(callback) {
    this._lifecycleManager.onUpdated(callback);
  }
  /**
   * Registers a cleanup function to run when the component disconnects.
   *
   * Cleanups run after onDisconnected callbacks, then the array is cleared.
   * This allows co-locating setup and teardown logic.
   *
   * @param callback - The cleanup function to call on disconnection.
   */
  onCleanup(callback) {
    this._lifecycleManager.onCleanup(callback);
  }
  /**
   * Returns the elements assigned to a named slot.
   *
   * @param slotName - The slot name, or empty string for the default slot.
   * @returns The array of slotted elements.
   */
  getSlottedElements(slotName = "") {
    return this._slotManager.getSlottedElements(slotName);
  }
  /**
   * Attaches a listener for slot content changes.
   *
   * @param slotName - The slot name to watch.
   * @param callback - The function to call when slot content changes.
   */
  attachSlotListener(slotName, callback) {
    this._slotManager.attachSlotListener(slotName, callback);
  }
  /**
   * Checks whether a slot has any assigned content.
   *
   * @param slotName - The slot name, or empty string for the default slot.
   * @returns True if the slot has content.
   */
  hasSlotContent(slotName = "") {
    return this._slotManager.hasSlotContent(slotName);
  }
  /**
   * Returns the virtual DOM tree for this component.
   *
   * Subclasses must override this method with their template logic.
   *
   * @returns The root virtual node.
   */
  renderVDOM() {
    console.warn(`PPElement: renderVDOM() called on base class for ${this.tagName}.`);
    return dom.cmt(`No VDOM for ${this.tagName}`, `${this.tagName}-default-vdom`);
  }
  /** Performs an immediate synchronous render. */
  render() {
    this._renderScheduler.render();
  }
  /** Schedules an asynchronous render on the next microtask. */
  scheduleRender() {
    this._renderScheduler.scheduleRender();
  }
  /**
   * Initialises the component with a state context.
   *
   * Sets up state defaults, synchronises HTML attributes to state,
   * reflects state back to attributes, and triggers the first render
   * if the element is already connected.
   *
   * @param optsFromInstance - The state context provided by the compiled component.
   */
  init(optsFromInstance) {
    this._attributeSyncService.setInitialising(true);
    this._renderScheduler.setPendingAfterInit(false);
    this._stateManager.setContext(optsFromInstance);
    this._shadowDOMService.ensureShadowRoot();
    const state = this._stateManager.getState();
    if (state) {
      const defaults = this._getDefaults();
      for (const propName of this._propTypeRegistry.getPropertyNames()) {
        if (!(propName in state) && defaults[propName] !== void 0) {
          state[propName] = defaults[propName];
        }
      }
    }
    this._attributeSyncService.syncAllAttributesToState();
    this._attributeSyncService.reflectAllStateToAttributes();
    this._attributeSyncService.setInitialising(false);
    this._stateManager.clearChangedProps();
    if (this._renderScheduler.hasPendingAfterInit()) {
      this.scheduleRender();
    } else if (this.isConnected && !this._renderScheduler.hasRendered()) {
      this.render();
      this._lifecycleManager.executeConnected();
    }
  }
  /** Called when the element is inserted into the DOM. */
  connectedCallback() {
    if (!this._stateManager.hasState()) {
      console.warn(
        `PPElement ${this.tagName}: connectedCallback - init() incomplete or $$ctx not set.`
      );
    }
    if (this._stateManager.hasState() && !this._attributeSyncService.isInitialising() && this.isConnected) {
      if (!this._renderScheduler.hasRendered()) {
        this.scheduleRender();
      } else {
        this._lifecycleManager.executeConnected();
      }
    }
  }
  /** Called when the element is removed from the DOM. */
  disconnectedCallback() {
    this._lifecycleManager.executeDisconnected();
    this._lifecycleManager.executeCleanups();
    this._lifecycleManager.resetConnectedState();
  }
  /**
   * Called when an observed attribute changes.
   *
   * Skips processing while reflecting to attributes or during initialisation.
   *
   * @param attributeName - The name of the changed attribute.
   * @param _oldValue - The previous attribute value.
   * @param newValue - The new attribute value.
   */
  attributeChangedCallback(attributeName, _oldValue, newValue) {
    if (this._attributeSyncService.isReflectingToAttribute() || !this._stateManager.getState() || this._attributeSyncService.isInitialising()) {
      return;
    }
    const propertyName = this._propTypeRegistry.attributeToPropertyName(attributeName);
    const propDef = this._propTypeRegistry.get(propertyName);
    if (propDef) {
      this.applyHtmlAttributeToState(propertyName, newValue);
    }
  }
  /**
   * Applies an HTML attribute value to the component state.
   *
   * @param propertyName - The property name mapped from the attribute.
   * @param attributeValue - The new attribute value.
   */
  applyHtmlAttributeToState(propertyName, attributeValue) {
    this._attributeSyncService.applyAttributeToState(propertyName, attributeValue);
  }
  /**
   * Reflects a state property value back to an HTML attribute.
   *
   * @param propertyName - The property name to reflect.
   * @param propertyValue - The property value to set as an attribute.
   */
  reflectStatePropertyToAttribute(propertyName, propertyValue) {
    this._attributeSyncService.reflectStateToAttribute(propertyName, propertyValue);
  }
  /**
   * Translates a raw attribute string into a typed property value.
   *
   * @param typeHint - The expected type hint for conversion.
   * @param attributeValue - The raw attribute value.
   * @param propertyName - The target property name.
   * @param isNullable - Whether the property accepts null values.
   * @returns The converted property value.
   */
  translateAttributeValue(typeHint, attributeValue, propertyName, isNullable = false) {
    return this._attributeSyncService.translateAttributeValue(
      typeHint,
      attributeValue,
      propertyName,
      isNullable
    );
  }
  /**
   * Converts an HTML attribute name to a component property name.
   *
   * @param attributeName - The attribute name to convert.
   * @returns The corresponding property name.
   */
  attributeNameToPropertyName(attributeName) {
    return this._propTypeRegistry.attributeToPropertyName(attributeName);
  }
  /**
   * Converts a property name to an HTML attribute name.
   *
   * @param propertyName - The property name to convert.
   * @returns The corresponding attribute name.
   */
  static propertyNameToAttributeName(propertyName) {
    return propertyToAttributeName(propertyName);
  }
};
_PPElement.enabledBehaviours = [];
_PPElement.formAssociated = false;
let PPElement = _PPElement;
const svgCache = /* @__PURE__ */ new Map();
const fetchPromises = /* @__PURE__ */ new Map();
class PikoSvgInline extends HTMLElement {
  constructor() {
    super(...arguments);
    this._src = "";
    this._abortController = null;
  }
  /** Returns the list of attributes to observe for changes. */
  static get observedAttributes() {
    return ["src"];
  }
  /** Returns the current SVG source URL. */
  get src() {
    return this._src;
  }
  /**
   * Sets the SVG source URL and triggers a reload.
   *
   * @param value - The new source URL.
   */
  set src(value) {
    if (this._src !== value) {
      this._src = value;
      this.setAttribute("src", value);
      void this._loadSvg();
    }
  }
  /** Called when the element is inserted into the DOM. */
  connectedCallback() {
    const srcAttr = this.getAttribute("src");
    if (srcAttr && srcAttr !== this._src) {
      this._src = srcAttr;
      void this._loadSvg();
    }
  }
  /** Called when the element is removed from the DOM. */
  disconnectedCallback() {
    this._abortController?.abort();
    this._abortController = null;
  }
  /**
   * Called when an observed attribute changes.
   *
   * @param name - The attribute name.
   * @param oldValue - The previous attribute value.
   * @param newValue - The new attribute value.
   */
  attributeChangedCallback(name, oldValue, newValue) {
    if (name === "src" && newValue !== oldValue && newValue !== this._src) {
      this._src = newValue ?? "";
      void this._loadSvg();
    }
  }
  /**
   * Fetches and inlines the SVG from the current source URL.
   *
   * Aborts any previous in-flight request before starting a new one.
   */
  async _loadSvg() {
    const src = this._src;
    if (!src) {
      return;
    }
    this._abortController?.abort();
    this._abortController = new AbortController();
    const signal = this._abortController.signal;
    try {
      const svgContent = await this._fetchSvg(src, signal);
      if (signal.aborted) {
        return;
      }
      this._inlineSvg(svgContent);
    } catch (err) {
      if (signal.aborted) {
        return;
      }
      console.warn(`PikoSvgInline: Failed to load SVG from ${src}`, err);
      this.innerHTML = `<!-- piko-svg-inline error: failed to load ${src} -->`;
    }
  }
  /**
   * Fetches SVG content with caching and deduplication.
   *
   * Returns cached content if available. Deduplicates concurrent requests
   * for the same URL.
   *
   * @param src - The SVG source URL.
   * @param signal - The abort signal for cancellation.
   * @returns The SVG content string.
   */
  async _fetchSvg(src, signal) {
    const cached = svgCache.get(src);
    if (cached) {
      return cached;
    }
    const existingPromise = fetchPromises.get(src);
    if (existingPromise) {
      return existingPromise;
    }
    const fetchPromise = this._doFetch(src, signal);
    fetchPromises.set(src, fetchPromise);
    try {
      const result = await fetchPromise;
      svgCache.set(src, result);
      return result;
    } finally {
      fetchPromises.delete(src);
    }
  }
  /**
   * Performs the HTTP fetch and validates the response is SVG.
   *
   * @param src - The SVG source URL.
   * @param signal - The abort signal for cancellation.
   * @returns The validated SVG content string.
   */
  async _doFetch(src, signal) {
    const response = await fetch(src, { signal });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    const text = await response.text();
    if (!text.includes("<svg")) {
      throw new Error("Response is not valid SVG");
    }
    return text;
  }
  /**
   * Parses SVG content and replaces this element with the inline SVG.
   *
   * Copies attributes from the host element to the SVG, merging classes
   * and preserving SVG-specific attributes like viewBox. Registers the
   * replacement with the VDOM renderer and watches the src attribute
   * for changes.
   *
   * @param svgContent - The raw SVG markup string.
   */
  _inlineSvg(svgContent) {
    const parser = new DOMParser();
    const doc = parser.parseFromString(svgContent, "image/svg+xml");
    const svgElement = doc.querySelector("svg");
    if (!svgElement) {
      console.warn("PikoSvgInline: No SVG element found in response");
      return;
    }
    for (const attr of Array.from(this.attributes)) {
      if (attr.name === "src") {
        continue;
      }
      if (attr.name === "class") {
        const existingClass = svgElement.getAttribute("class") ?? "";
        const mergedClass = existingClass ? `${existingClass} ${attr.value}` : attr.value;
        svgElement.setAttribute("class", mergedClass);
      } else if (!svgElement.hasAttribute(attr.name)) {
        svgElement.setAttribute(attr.name, attr.value);
      }
    }
    replaceElementWithTracking(this, svgElement, { watchProps: ["src"] });
  }
}
function registerPikoSvgInline() {
  if (!customElements.get("piko-svg-inline")) {
    customElements.define("piko-svg-inline", PikoSvgInline);
  }
}
const arrayMutatorMethods = ["push", "pop", "shift", "unshift", "splice", "sort", "reverse"];
function makeReactive(target, context, parentProp) {
  if (typeof target !== "object" || target === null) {
    return target;
  }
  if (target instanceof Node) {
    return target;
  }
  if (Array.isArray(target)) {
    return createArrayProxy(target, context, parentProp);
  }
  return createObjectProxy(target, context);
}
function createArrayProxy(arr, context, parentProp) {
  return new Proxy(arr, {
    get(target, prop, receiver) {
      const value = Reflect.get(target, prop, receiver);
      if (typeof prop === "string" && arrayMutatorMethods.includes(prop) && typeof value === "function") {
        return function(...args) {
          const result = value.apply(target, args);
          if (context?.changedPropsSet && parentProp) {
            context.changedPropsSet.add(parentProp);
          }
          if (context?.scheduleRender) {
            context.scheduleRender();
          }
          return result;
        };
      }
      if (typeof value === "object" && value !== null && !(value instanceof Node)) {
        return makeReactive(value, context, parentProp);
      }
      return value;
    },
    set(target, prop, value) {
      target[prop] = value;
      if (context?.changedPropsSet && parentProp) {
        context.changedPropsSet.add(parentProp);
      }
      if (context?.scheduleRender) {
        context.scheduleRender();
      }
      return true;
    }
  });
}
function createObjectProxy(target, context) {
  return new Proxy(target, {
    get(proxyTarget, prop, receiver) {
      const value = Reflect.get(proxyTarget, prop, receiver);
      if (typeof value === "object" && value !== null && !(value instanceof Node)) {
        const propKey = typeof prop === "string" ? prop : String(prop);
        return makeReactive(value, context, propKey);
      }
      return value;
    },
    set(proxyTarget, prop, value) {
      const oldVal = proxyTarget[prop];
      if (oldVal === value && typeof value !== "object") {
        return true;
      }
      proxyTarget[prop] = value;
      if (context?.changedPropsSet && typeof prop === "string") {
        context.changedPropsSet.add(prop);
      }
      if (context?.scheduleRender) {
        context.scheduleRender();
      }
      return true;
    }
  });
}
registerPikoSvgInline();
export {
  PPElement,
  dom,
  makeReactive,
  registerBehaviour,
  registerPikoSvgInline
};
//# sourceMappingURL=ppframework.components.es.js.map
