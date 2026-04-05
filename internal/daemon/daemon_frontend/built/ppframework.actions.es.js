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
const HTTP_STATUS_UNPROCESSABLE$1 = 422;
const HTTP_STATUS_UNAUTHORIZED = 401;
const HTTP_STATUS_FORBIDDEN$1 = 403;
function createActionError(status, message, validationErrors, data, helpers) {
  return {
    status,
    message,
    validationErrors,
    data,
    _helpers: helpers,
    get isNetworkError() {
      return this.status === 0;
    },
    get isValidationError() {
      return this.status === HTTP_STATUS_UNPROCESSABLE$1 && this.validationErrors !== void 0;
    },
    get isAuthError() {
      return this.status === HTTP_STATUS_UNAUTHORIZED || this.status === HTTP_STATUS_FORBIDDEN$1;
    }
  };
}
function isActionDescriptor(value) {
  return value !== null && typeof value === "object" && typeof value.action === "string";
}
const BLOCK_DELIMITER = "\n\n";
const DEFAULT_EVENT_TYPE = "message";
const COMPLETE_EVENT_TYPE = "complete";
const ERROR_EVENT_TYPE = "error";
const EVENT_LINE_PREFIX = "event: ";
const DATA_LINE_PREFIX = "data: ";
const ID_LINE_PREFIX = "id: ";
function parseSSEBlock(block) {
  const trimmed = block.trim();
  if (!trimmed || trimmed.startsWith(":")) {
    return null;
  }
  let eventType = "";
  let rawData = "";
  let id;
  for (const line of trimmed.split("\n")) {
    if (line.startsWith(EVENT_LINE_PREFIX)) {
      eventType = line.substring(EVENT_LINE_PREFIX.length);
    } else if (line.startsWith(DATA_LINE_PREFIX)) {
      rawData = line.substring(DATA_LINE_PREFIX.length);
    } else if (line.startsWith(ID_LINE_PREFIX)) {
      id = line.substring(ID_LINE_PREFIX.length);
    }
  }
  if (!eventType && !rawData) {
    return null;
  }
  if (!eventType) {
    eventType = DEFAULT_EVENT_TYPE;
  }
  let data = rawData;
  try {
    data = JSON.parse(rawData);
  } catch {
  }
  return { eventType, data, id };
}
function processSSEBlock(block, callbacks) {
  const parsed = parseSSEBlock(block);
  if (!parsed) {
    return null;
  }
  if (parsed.eventType === COMPLETE_EVENT_TYPE) {
    return { completeData: parsed.data };
  }
  if (parsed.eventType === ERROR_EVENT_TYPE) {
    const errorData = parsed.data;
    const message = typeof errorData?.message === "string" ? errorData.message : "SSE stream error";
    throw createActionError(0, message, void 0, parsed.data);
  }
  callbacks.onEvent(parsed.data, parsed.eventType);
  if (parsed.id && callbacks.onEventId) {
    callbacks.onEventId(parsed.id);
  }
  return null;
}
async function consumeSSEStream(reader, callbacks) {
  const decoder = new TextDecoder();
  let buffer = "";
  let completeData = void 0;
  let receivedComplete = false;
  for (; ; ) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split(BLOCK_DELIMITER);
    buffer = parts.pop() ?? "";
    for (const part of parts) {
      const result = processSSEBlock(part, callbacks);
      if (!result) {
        continue;
      }
      completeData = result.completeData;
      receivedComplete = true;
    }
  }
  return { completeData, receivedComplete };
}
async function readSSEStream(body, callbacks, _signal) {
  const reader = body.getReader();
  try {
    const { completeData, receivedComplete } = await consumeSSEStream(reader, callbacks);
    if (!receivedComplete) {
      throw createActionError(0, "SSE stream ended without completion");
    }
    return completeData;
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw createActionError(0, "Request cancelled");
    }
    if (error !== null && typeof error === "object" && "status" in error) {
      throw error;
    }
    const message = error instanceof Error ? error.message : "SSE connection lost";
    throw createActionError(0, message);
  } finally {
    try {
      reader.releaseLock();
    } catch {
    }
  }
}
const CSRF_TOKEN_META_NAME = "csrf-token";
const CSRF_EPHEMERAL_META_NAME = "csrf-ephemeral";
function getCSRFTokenFromMeta() {
  return document.querySelector(`meta[name="${CSRF_TOKEN_META_NAME}"]`)?.content ?? null;
}
function getCSRFEphemeralFromMeta() {
  return document.querySelector(`meta[name="${CSRF_EPHEMERAL_META_NAME}"]`)?.content ?? null;
}
const LOADING_CLASS = "pk-loading";
const ARIA_BUSY_ATTR = "aria-busy";
function applyLoadingIndicator(el) {
  el.classList.add(LOADING_CLASS);
  el.setAttribute(ARIA_BUSY_ATTR, "true");
}
function removeLoadingIndicator(el) {
  el.classList.remove(LOADING_CLASS);
  el.removeAttribute(ARIA_BUSY_ATTR);
}
const ActionsHookEvent = {
  ACTION_START: "action:start",
  ACTION_COMPLETE: "action:complete"
};
const HTTP_STATUS_UNPROCESSABLE = 422;
const HTTP_STATUS_FORBIDDEN = 403;
const CSRF_ERROR_EXPIRED = "csrf_expired";
const CSRF_ERROR_INVALID = "csrf_invalid";
const DEFAULT_RETRY_BASE_DELAY = 1e3;
const MAX_RETRY_DELAY = 3e4;
const DEFAULT_SSE_RECONNECT_DELAY = 3e3;
const MAX_SSE_RECONNECT_DELAY = 3e4;
const HTTP_STATUS_TIMEOUT = 408;
const HTTP_STATUS_SERVER_ERROR = 500;
const HTTP_STATUS_OK = 200;
const RANDOM_STRING_RADIX = 36;
const RANDOM_STRING_SLICE_START = 2;
const RANDOM_STRING_SLICE_END = 9;
const debounceTimers = /* @__PURE__ */ new Map();
let hookManager = null;
let formStateManager = null;
let helperRegistry = null;
function setActionExecutorDependencies(deps) {
  if (deps.hookManager) {
    hookManager = deps.hookManager;
  }
  if (deps.formStateManager) {
    formStateManager = deps.formStateManager;
  }
  if (deps.helperRegistry) {
    helperRegistry = deps.helperRegistry;
  }
}
function getCSRFTokens(element) {
  const actionToken = element?.getAttribute("data-csrf-action-token") ?? getCSRFTokenFromMeta();
  const ephemeralToken = element?.getAttribute("data-csrf-ephemeral-token") ?? getCSRFEphemeralFromMeta();
  return { actionToken, ephemeralToken };
}
function isCSRFError(status, responseData) {
  return status === HTTP_STATUS_FORBIDDEN && (responseData.error === CSRF_ERROR_EXPIRED || responseData.error === CSRF_ERROR_INVALID);
}
function attemptCSRFRecovery(responseData, element, retryAction) {
  if (responseData.error === CSRF_ERROR_INVALID) {
    window.location.reload();
    return true;
  }
  const partial = element.closest("[partial_src]");
  if (partial) {
    partial.dispatchEvent(new CustomEvent("refresh-partial", {
      bubbles: false,
      detail: {
        afterMorph: () => {
          const refreshedEl = partial.querySelector("[data-csrf-action-token]");
          if (refreshedEl) {
            retryAction();
          } else {
            console.warn("[ActionExecutor] Could not find element with CSRF token after partial refresh");
          }
        }
      }
    }));
    return true;
  }
  window.location.reload();
  return true;
}
function validateForm(element, event) {
  const form = element.closest("form");
  if (!form) {
    return true;
  }
  if (form.noValidate) {
    return true;
  }
  const submitter = event?.submitter;
  if (submitter?.formNoValidate) {
    return true;
  }
  return form.reportValidity();
}
function clearPreviousErrors(form) {
  form.querySelectorAll("[error]").forEach((el) => {
    el.removeAttribute("error");
  });
}
function applyServerErrors(form, errors) {
  clearPreviousErrors(form);
  for (const [fieldName, messages] of Object.entries(errors)) {
    const errorMessage = messages.join(", ");
    const fields = form.querySelectorAll(`[name="${fieldName}"]`);
    if (fields.length > 0) {
      fields.forEach((field) => {
        field.setAttribute("error", errorMessage);
      });
    }
  }
}
function showLoading(target, element) {
  if (target === true) {
    applyLoadingIndicator(element);
  } else if (typeof target === "string") {
    const el = document.querySelector(target);
    if (el) {
      applyLoadingIndicator(el);
    }
  } else if (target instanceof HTMLElement) {
    applyLoadingIndicator(target);
  }
}
function hideLoading(target, element) {
  if (target === true) {
    removeLoadingIndicator(element);
  } else if (typeof target === "string") {
    const el = document.querySelector(target);
    if (el) {
      removeLoadingIndicator(el);
    }
  } else if (target instanceof HTMLElement) {
    removeLoadingIndicator(target);
  }
}
function calculateRetryDelay(attempt, config) {
  const backoff = config.backoff ?? "exponential";
  if (backoff === "linear") {
    return Math.min(DEFAULT_RETRY_BASE_DELAY * (attempt + 1), MAX_RETRY_DELAY);
  }
  return Math.min(DEFAULT_RETRY_BASE_DELAY * Math.pow(2, attempt), MAX_RETRY_DELAY);
}
function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
async function executeWithRetry(actionName, args, method, actionToken, ephemeralToken, retryConfig, options) {
  const maxAttempts = retryConfig?.attempts ?? 1;
  let lastError = null;
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      return await executeServerAction(actionName, args, method, actionToken, ephemeralToken, options);
    } catch (error) {
      if (error instanceof Error && !("status" in error)) {
        lastError = createActionError(0, error.message);
      } else {
        lastError = error;
      }
      const isTimeout = lastError.status === HTTP_STATUS_TIMEOUT;
      const isCancelled = lastError.status === 0 && lastError.message === "Request cancelled";
      const isRetryable = (lastError.status === 0 || lastError.status >= HTTP_STATUS_SERVER_ERROR || isTimeout) && !isCancelled;
      if (!isRetryable || attempt >= maxAttempts - 1) {
        throw lastError;
      }
      const retryDelay = calculateRetryDelay(attempt, retryConfig ?? {});
      await delay(retryDelay);
    }
  }
  throw lastError;
}
function buildActionBody(args, ephemeralToken) {
  const headers = {};
  const bodyData = {};
  if (args.length > 0) {
    if (args.length === 1 && typeof args[0] === "object" && args[0] !== null) {
      Object.assign(bodyData, args[0]);
    } else {
      bodyData["args"] = args.map((v, i) => ({ [i]: v })).reduce((acc, b) => ({ ...acc, ...b }), {});
    }
  }
  if (ephemeralToken) {
    bodyData["_csrf_ephemeral_token"] = ephemeralToken;
  }
  const hasFiles = Object.values(bodyData).some(
    (v) => v instanceof File || v instanceof Blob
  );
  if (hasFiles) {
    const formData = new FormData();
    for (const [key, value] of Object.entries(bodyData)) {
      if (value instanceof File) {
        formData.append(key, value, value.name);
      } else if (value instanceof Blob) {
        formData.append(key, value);
      } else if (value !== void 0 && value !== null) {
        formData.append(key, String(value));
      }
    }
    return { body: formData, headers };
  }
  headers["Content-Type"] = "application/json";
  return { body: JSON.stringify(bodyData), headers };
}
function createRequestAbortController(options) {
  const controller = new AbortController();
  let timeoutId;
  if (options?.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort());
    }
  }
  if (options?.timeout && options.timeout > 0) {
    timeoutId = setTimeout(() => controller.abort(), options.timeout);
  }
  return { controller, timeoutId };
}
async function parseActionResponse(response) {
  let responseData;
  try {
    responseData = await response.json();
  } catch {
    throw createActionError(
      response.status,
      "Failed to parse server response",
      void 0,
      void 0
    );
  }
  if (!response.ok) {
    const validationErrors = response.status === HTTP_STATUS_UNPROCESSABLE ? responseData.errors : void 0;
    throw createActionError(
      response.status,
      responseData.message ?? responseData.error ?? `Action failed with status ${response.status}`,
      validationErrors,
      responseData.error ?? responseData.data,
      responseData._helpers
    );
  }
  return responseData;
}
async function executeServerAction(actionName, args, method, actionToken, ephemeralToken, options) {
  const { body, headers } = buildActionBody(args, ephemeralToken);
  if (actionToken) {
    headers["X-CSRF-Action-Token"] = actionToken;
  }
  const { controller, timeoutId } = createRequestAbortController(options);
  try {
    const response = await fetch(`/_piko/actions/${actionName}`, {
      method,
      headers,
      credentials: "same-origin",
      body,
      signal: controller.signal
    });
    return await parseActionResponse(response);
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      const isTimeout = options?.timeout && !options.signal?.aborted;
      throw createActionError(
        isTimeout ? HTTP_STATUS_TIMEOUT : 0,
        isTimeout ? "Request timeout" : "Request cancelled",
        void 0,
        void 0
      );
    }
    throw error;
  } finally {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
  }
}
function getDebounceKey(actionName, element) {
  element.dataset.ppActionId ??= `action-${Date.now()}-${Math.random().toString(RANDOM_STRING_RADIX).slice(RANDOM_STRING_SLICE_START, RANDOM_STRING_SLICE_END)}`;
  return `${actionName}:${element.dataset.ppActionId}`;
}
function clearDebounce(key) {
  const timer = debounceTimers.get(key);
  if (timer) {
    clearTimeout(timer);
    debounceTimers.delete(key);
  }
}
async function executeHelpers(helpers, element, event) {
  if (!helperRegistry) {
    return;
  }
  const errors = [];
  for (const helper of helpers) {
    try {
      const args = (helper.args ?? []).map((a) => String(a));
      await helperRegistry.execute(helper.name, element, event, args);
    } catch (error) {
      console.error(`[ActionExecutor] Helper "${helper.name}" failed:`, error);
      errors.push(error);
    }
  }
  if (errors.length > 0) {
    throw new AggregateError(errors, `${errors.length} helper(s) failed`);
  }
}
async function invokeSuccessCallback(descriptor, response, element, event) {
  if (!descriptor.onSuccess) {
    return;
  }
  try {
    const data = response.data ?? response;
    const next = descriptor.onSuccess(data);
    if (isActionDescriptor(next)) {
      await handleAction(next, element, event);
    }
  } catch (error) {
    console.error("[ActionExecutor] onSuccess callback failed:", error);
    throw error;
  }
}
function validateAndEmitStart(descriptor, element, form, actionStartTime, event) {
  if (form) {
    clearPreviousErrors(form);
    if (!validateForm(element, event)) {
      hookManager?.emit(ActionsHookEvent.ACTION_COMPLETE, {
        action: descriptor.action,
        method: descriptor.method ?? "POST",
        elementTag: element.tagName.toLowerCase(),
        success: false,
        statusCode: 0,
        duration: Date.now() - actionStartTime,
        timestamp: Date.now(),
        validationFailed: true
      });
      return false;
    }
  }
  hookManager?.emit(ActionsHookEvent.ACTION_START, {
    action: descriptor.action,
    method: descriptor.method ?? "POST",
    elementTag: element.tagName.toLowerCase(),
    timestamp: actionStartTime
  });
  return true;
}
async function executeServerRequest(descriptor, element) {
  const { actionToken, ephemeralToken } = getCSRFTokens(element);
  if (descriptor.onProgress) {
    const sseOptions = { timeout: descriptor.timeout, signal: descriptor.signal };
    let data;
    if (descriptor.retryStream) {
      data = await executeServerActionSSEWithRetry({
        actionName: descriptor.action,
        args: descriptor.args ?? [],
        method: descriptor.method ?? "POST",
        actionToken,
        ephemeralToken,
        onProgress: descriptor.onProgress,
        retryConfig: descriptor.retryStream,
        options: sseOptions
      });
    } else {
      data = await executeServerActionSSE({
        actionName: descriptor.action,
        args: descriptor.args ?? [],
        method: descriptor.method ?? "POST",
        actionToken,
        ephemeralToken,
        onProgress: descriptor.onProgress,
        options: sseOptions
      });
    }
    return { data, status: HTTP_STATUS_OK };
  }
  return await executeWithRetry(
    descriptor.action,
    descriptor.args ?? [],
    descriptor.method ?? "POST",
    actionToken,
    ephemeralToken,
    descriptor.retry,
    { timeout: descriptor.timeout, signal: descriptor.signal }
  );
}
async function handleActionSuccess(descriptor, response, element, event, form, actionStartTime) {
  if (!descriptor.shouldSuppressHelpers && response._helpers && response._helpers.length > 0) {
    await executeHelpers(response._helpers, element, event);
  }
  if (form && formStateManager) {
    formStateManager.markFormClean(form);
  }
  hookManager?.emit(ActionsHookEvent.ACTION_COMPLETE, {
    action: descriptor.action,
    method: descriptor.method ?? "POST",
    elementTag: element.tagName.toLowerCase(),
    success: true,
    statusCode: response.status ?? HTTP_STATUS_OK,
    duration: Date.now() - actionStartTime,
    timestamp: Date.now()
  });
  await invokeSuccessCallback(descriptor, response, element, event);
}
function invokeErrorHandlers(actionError, descriptor, rawError) {
  if (descriptor.onError) {
    try {
      descriptor.onError(actionError);
    } catch (callbackError) {
      console.error("[ActionExecutor] onError callback failed:", callbackError);
    }
  } else {
    console.error("[ActionExecutor] Action failed:", rawError);
  }
}
async function handleActionFailure(descriptor, error, element, event, form, actionStartTime) {
  const actionError = error;
  const errorCode = typeof actionError.data === "string" ? actionError.data : void 0;
  if (isCSRFError(actionError.status, { error: errorCode })) {
    const recovered = attemptCSRFRecovery(
      { error: errorCode },
      element,
      () => {
        void executeAction(descriptor, element, event);
      }
    );
    if (recovered) {
      return true;
    }
  }
  if (actionError._helpers && actionError._helpers.length > 0) {
    await executeHelpers(actionError._helpers, element, event);
  }
  if (actionError.status === HTTP_STATUS_UNPROCESSABLE && actionError.validationErrors && form) {
    applyServerErrors(form, actionError.validationErrors);
  }
  hookManager?.emit(ActionsHookEvent.ACTION_COMPLETE, {
    action: descriptor.action,
    method: descriptor.method ?? "POST",
    elementTag: element.tagName.toLowerCase(),
    success: false,
    statusCode: actionError.status,
    duration: Date.now() - actionStartTime,
    timestamp: Date.now()
  });
  invokeErrorHandlers(actionError, descriptor, error);
  return false;
}
async function executeAction(descriptor, element, event) {
  const actionStartTime = Date.now();
  const form = element.closest("form");
  if (!validateAndEmitStart(descriptor, element, form, actionStartTime, event)) {
    return;
  }
  if (descriptor.optimistic) {
    try {
      descriptor.optimistic();
    } catch (error) {
      console.error("[ActionExecutor] Optimistic update failed:", error);
    }
  }
  if (descriptor.loading !== void 0) {
    showLoading(descriptor.loading, element);
  }
  try {
    const response = await executeServerRequest(descriptor, element);
    await handleActionSuccess(descriptor, response, element, event, form, actionStartTime);
  } catch (error) {
    const recovered = await handleActionFailure(descriptor, error, element, event, form, actionStartTime);
    if (recovered) {
      return;
    }
    throw error;
  } finally {
    if (descriptor.onComplete) {
      try {
        descriptor.onComplete();
      } catch (error) {
        console.error("[ActionExecutor] onComplete callback failed:", error);
      }
    }
    if (descriptor.loading !== void 0) {
      hideLoading(descriptor.loading, element);
    }
  }
}
async function handleAction(descriptor, element, event) {
  if (descriptor.debounce && descriptor.debounce > 0) {
    const debounceKey = getDebounceKey(descriptor.action, element);
    clearDebounce(debounceKey);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        debounceTimers.delete(debounceKey);
        executeAction(descriptor, element, event).then(resolve).catch((err) => {
          console.error("[ActionExecutor] Debounced action failed:", err);
          reject(err instanceof Error ? err : new Error(String(err)));
        });
      }, descriptor.debounce);
      debounceTimers.set(debounceKey, timer);
    });
  }
  return executeAction(descriptor, element, event);
}
const SSE_CONTENT_TYPE = "text/event-stream";
const SSE_ACCEPT_HEADER = "text/event-stream";
function calculateSSEReconnectDelay(attempt, config) {
  const baseDelay = config.baseDelay ?? DEFAULT_SSE_RECONNECT_DELAY;
  const maxDelay = config.maxDelay ?? MAX_SSE_RECONNECT_DELAY;
  const backoff = config.backoff ?? "linear";
  if (backoff === "linear") {
    return Math.min(baseDelay * (attempt + 1), maxDelay);
  }
  return Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
}
function buildSSEHeaders(actionToken, ephemeralToken, lastEventId) {
  const headers = {
    "Content-Type": "application/json",
    "Accept": SSE_ACCEPT_HEADER
  };
  if (actionToken && ephemeralToken) {
    headers["X-CSRF-Action-Token"] = actionToken;
  }
  if (lastEventId) {
    headers["Last-Event-ID"] = lastEventId;
  }
  return headers;
}
function buildSSEUrl(actionName, actionToken, ephemeralToken) {
  let url = `/_piko/actions/${actionName}`;
  if (actionToken && ephemeralToken) {
    url += `?_csrf_ephemeral_token=${encodeURIComponent(ephemeralToken)}`;
  }
  return url;
}
function buildSSEBody(args) {
  const bodyData = {};
  if (args.length > 0) {
    if (args.length === 1 && typeof args[0] === "object" && args[0] !== null) {
      Object.assign(bodyData, args[0]);
    } else {
      bodyData["args"] = args.map((v, i) => ({ [i]: v })).reduce((acc, b) => ({ ...acc, ...b }), {});
    }
  }
  return bodyData;
}
function setupAbortControl(options) {
  const controller = new AbortController();
  let timeoutId;
  if (options?.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort());
    }
  }
  if (options?.timeout && options.timeout > 0) {
    timeoutId = setTimeout(() => controller.abort(), options.timeout);
  }
  return { controller, timeoutId };
}
async function throwSSEErrorResponse(response) {
  let responseData;
  try {
    responseData = await response.json();
  } catch {
    throw createActionError(response.status, `Action failed with status ${response.status}`);
  }
  const validationErrors = response.status === HTTP_STATUS_UNPROCESSABLE ? responseData.errors : void 0;
  throw createActionError(
    response.status,
    responseData.message ?? responseData.error ?? `Action failed with status ${response.status}`,
    validationErrors,
    responseData.error ?? responseData.data,
    responseData._helpers
  );
}
function rethrowAsActionError(error, options) {
  if (error instanceof DOMException && error.name === "AbortError") {
    const isTimeout = options?.timeout && !options.signal?.aborted;
    throw createActionError(
      isTimeout ? HTTP_STATUS_TIMEOUT : 0,
      isTimeout ? "Request timeout" : "Request cancelled"
    );
  }
  throw error;
}
async function executeServerActionSSEInternal(params) {
  const { actionName, args, method, actionToken, ephemeralToken, onProgress, options, lastEventId, onEventId } = params;
  const headers = buildSSEHeaders(actionToken, ephemeralToken, lastEventId);
  const url = buildSSEUrl(actionName, actionToken, ephemeralToken);
  const bodyData = buildSSEBody(args);
  const { controller, timeoutId } = setupAbortControl(options);
  try {
    const response = await fetch(url, {
      method,
      headers,
      credentials: "same-origin",
      body: JSON.stringify(bodyData),
      signal: controller.signal
    });
    if (!response.ok) {
      await throwSSEErrorResponse(response);
    }
    const contentType = response.headers.get("Content-Type") ?? "";
    if (contentType.startsWith(SSE_CONTENT_TYPE) && response.body) {
      return await readSSEStream(response.body, { onEvent: onProgress, onEventId }, controller.signal);
    }
    let responseData;
    try {
      responseData = await response.json();
    } catch {
      throw createActionError(response.status, "Failed to parse server response");
    }
    return responseData;
  } catch (error) {
    return rethrowAsActionError(error, options);
  } finally {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
  }
}
async function executeServerActionSSE(params) {
  return executeServerActionSSEInternal(params);
}
async function executeServerActionSSEWithRetry(params) {
  const { actionName, args, method, actionToken, ephemeralToken, onProgress, retryConfig, options } = params;
  let reconnectCount = 0;
  let lastEventId;
  const maxReconnects = retryConfig.maxReconnects;
  for (; ; ) {
    try {
      let isReconnection = reconnectCount > 0;
      const wrappedOnProgress = (data, eventType) => {
        if (isReconnection) {
          reconnectCount = 0;
          isReconnection = false;
        }
        onProgress(data, eventType);
      };
      const result = await executeServerActionSSEInternal({
        actionName,
        args,
        method,
        actionToken,
        ephemeralToken,
        onProgress: wrappedOnProgress,
        options,
        lastEventId,
        onEventId: (id) => {
          lastEventId = id;
        }
      });
      return result;
    } catch (error) {
      const actionError = error;
      if (actionError.message === "Request cancelled") {
        throw error;
      }
      if (actionError.data !== void 0) {
        throw error;
      }
      if (actionError.status !== 0) {
        throw error;
      }
      if (reconnectCount >= maxReconnects) {
        throw createActionError(
          0,
          `SSE stream failed after ${reconnectCount} reconnection attempts`
        );
      }
      retryConfig.onDisconnect?.();
      const reconnectDelay = calculateSSEReconnectDelay(reconnectCount, retryConfig);
      await delay(reconnectDelay);
      if (options?.signal?.aborted) {
        throw createActionError(0, "Request cancelled");
      }
      reconnectCount++;
      retryConfig.onReconnect?.(reconnectCount);
    }
  }
}
function createActionsCapability(services) {
  setActionExecutorDependencies({
    hookManager: services.hookManager,
    formStateManager: services.formStateManager ?? void 0,
    helperRegistry: services.helperRegistry
  });
  return {
    handleAction
  };
}
waitForPiko("actions").then((pk) => {
  pk._registerCapability("actions", createActionsCapability);
}).catch((err) => {
  console.error("[piko/actions] Failed to initialise:", err);
});
//# sourceMappingURL=ppframework.actions.es.js.map
