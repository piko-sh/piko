// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package typegen_domain

import (
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// ActionManifestProvider provides access to the action manifest from the last successful
// build. This abstraction allows the TypeInfoService to remain decoupled from the
// coordinator implementation.
type ActionManifestProvider interface {
	// GetLastSuccessfulBuild returns the most recent successful build result.
	//
	// Returns *ProjectAnnotationResult which contains the build data if found.
	// Returns bool which is true if a successful build exists, or false if not.
	GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool)
}

// TypeInfoService provides TypeScript type information for LSP intellisense. It is the
// single source of truth for piko.* and action.* completions.
type TypeInfoService struct {
	// actionProvider gives access to actions found by the coordinator. It may be nil if not
	// set, in which case completions will be empty.
	actionProvider ActionManifestProvider
}

// TypeInfoServiceOption configures a TypeInfoService.
type TypeInfoServiceOption func(*TypeInfoService)

// NewTypeInfoService creates a new TypeInfoService.
//
// Takes opts (...TypeInfoServiceOption) which configures the service.
//
// Returns *TypeInfoService which is the configured service ready for use.
func NewTypeInfoService(opts ...TypeInfoServiceOption) *TypeInfoService {
	s := &TypeInfoService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetPikoCompletions returns completions for the piko namespace.
//
// Takes namespace (string) which specifies the piko API namespace to query.
//
// Returns []CompletionItem which contains the available completions, or nil if the
// namespace is not found.
func (*TypeInfoService) GetPikoCompletions(namespace string) []CompletionItem {
	key := "piko"
	if namespace != "" {
		key = "piko." + namespace
	}

	funcs, ok := pikoAPICompletions[key]
	if !ok {
		return nil
	}

	items := make([]CompletionItem, len(funcs))
	for i, f := range funcs {
		kind := CompletionKindFunction
		if f.IsProperty {
			kind = CompletionKindProperty
		}
		items[i] = CompletionItem{
			Label:         f.Label,
			Detail:        f.Signature,
			Documentation: f.Description,
			InsertText:    f.Label,
			Kind:          kind,
		}
	}
	return items
}

// GetPikoSubNamespaces returns the available piko sub-namespaces.
//
// Returns []string which contains all valid piko sub-namespace identifiers.
func (*TypeInfoService) GetPikoSubNamespaces() []string {
	return PikoSubNamespaces
}

// GetActionCompletions returns completions for registered actions matching a prefix.
//
// If an ActionManifestProvider is configured, it returns real actions from the last
// successful build. Otherwise, it returns nil.
//
// Takes prefix (string) which filters actions by their TypeScript function name. Pass an
// empty string to return all actions.
//
// Returns []CompletionItem which contains matching action completions, or nil if no
// provider is configured or no build results are available.
func (s *TypeInfoService) GetActionCompletions(prefix string) []CompletionItem {
	if s.actionProvider == nil {
		return nil
	}

	result, ok := s.actionProvider.GetLastSuccessfulBuild()
	if !ok || result == nil || result.VirtualModule == nil {
		return nil
	}

	manifest := result.VirtualModule.ActionManifest
	if manifest == nil {
		return nil
	}

	items := make([]CompletionItem, 0, len(manifest.Actions))
	for i := range manifest.Actions {
		if prefix != "" && !strings.HasPrefix(manifest.Actions[i].TSFunctionName, prefix) {
			continue
		}

		detail := buildActionSignature(&manifest.Actions[i])

		items = append(items, CompletionItem{
			Label:         manifest.Actions[i].TSFunctionName,
			Detail:        detail,
			Documentation: manifest.Actions[i].Description,
			InsertText:    manifest.Actions[i].TSFunctionName,
			Kind:          CompletionKindFunction,
		})
	}
	return items
}

// pikoAPIFunction describes a function or property in the piko namespace.
type pikoAPIFunction struct {
	// Label is the display name shown in completion lists.
	Label string

	// Signature is the function signature shown in completion details.
	Signature string

	// Description provides human-readable documentation for the function.
	Description string

	// IsProperty indicates whether the function behaves as a property accessor.
	IsProperty bool
}

var (
	// PikoSubNamespaces lists all available sub-namespaces under piko. Exported so LSP can
	// use it for trigger detection.
	PikoSubNamespaces = []string{
		"nav",
		"form",
		"ui",
		"event",
		"partials",
		"sse",
		"timing",
		"util",
		"trace",
	}

	// pikoAPICompletions contains all static piko.* API completions.
	pikoAPICompletions = map[string][]pikoAPIFunction{
		"piko": {
			{
				Label:       "refs",
				Signature:   "refs: Record<string, HTMLElement | null>",
				Description: "Global refs proxy for accessing elements by p-ref attribute",
				IsProperty:  true,
			},
			{
				Label:       "createRefs",
				Signature:   "createRefs(scope?: Element): Record<string, HTMLElement | null>",
				Description: "Creates a scoped refs proxy for a container element",
				IsProperty:  false,
			},
			{
				Label:       "partial",
				Signature:   "partial(name: string): PartialHandle",
				Description: "Gets a handle for a server-side partial by name",
				IsProperty:  false,
			},
			{
				Label:       "bus",
				Signature:   "bus: EventBus",
				Description: "Event bus for cross-component communication",
				IsProperty:  true,
			},
			{
				Label:       "onCleanup",
				Signature:   "onCleanup(fn: () => void, scope?: Element): void",
				Description: "Registers a cleanup function to run on navigation or element removal",
				IsProperty:  false,
			},
		},
		"piko.nav": {
			{
				Label:       "navigate",
				Signature:   "navigate(url: string, options?: NavigateOptions): Promise<void>",
				Description: "Navigates to a new URL using SPA navigation",
				IsProperty:  false,
			},
			{
				Label:       "back",
				Signature:   "back(): void",
				Description: "Navigates back in history",
				IsProperty:  false,
			},
			{
				Label:       "forward",
				Signature:   "forward(): void",
				Description: "Navigates forward in history",
				IsProperty:  false,
			},
			{
				Label:       "go",
				Signature:   "go(delta: number): void",
				Description: "Navigates to a specific point in history",
				IsProperty:  false,
			},
			{
				Label:       "current",
				Signature:   "current(): RouteInfo",
				Description: "Gets information about the current route",
				IsProperty:  false,
			},
			{
				Label:       "buildUrl",
				Signature:   "buildUrl(path: string, query?: Record<string, string>): string",
				Description: "Builds a URL with query parameters",
				IsProperty:  false,
			},
			{
				Label:       "updateQuery",
				Signature:   "updateQuery(params: Record<string, string | null>): void",
				Description: "Updates query parameters without full navigation",
				IsProperty:  false,
			},
			{
				Label:       "guard",
				Signature:   "guard(fn: NavigationGuard): () => void",
				Description: "Registers a navigation guard",
				IsProperty:  false,
			},
			{
				Label:       "matchPath",
				Signature:   "matchPath(pattern: string): boolean",
				Description: "Checks if the current path matches a pattern",
				IsProperty:  false,
			},
			{
				Label:       "extractParams",
				Signature:   "extractParams(pattern: string): Record<string, string> | null",
				Description: "Extracts path parameters from the current URL",
				IsProperty:  false,
			},
		},
		"piko.form": {
			{
				Label:       "data",
				Signature:   "data(form: string | HTMLFormElement): FormDataHandle",
				Description: "Creates a FormDataHandle for easy form data access",
				IsProperty:  false,
			},
			{
				Label:       "validate",
				Signature:   "validate(form: string | HTMLFormElement, rules: ValidationRules): ValidationResult",
				Description: "Validates a form against a set of rules",
				IsProperty:  false,
			},
			{
				Label:       "reset",
				Signature:   "reset(form: string | HTMLFormElement): void",
				Description: "Resets a form to its initial state",
				IsProperty:  false,
			},
			{
				Label:       "setValues",
				Signature:   "setValues(form: string | HTMLFormElement, values: Record<string, unknown>): void",
				Description: "Sets form field values programmatically",
				IsProperty:  false,
			},
		},
		"piko.ui": {
			{
				Label:       "loading",
				Signature:   "loading<T>(target: string | Element, promise: Promise<T>): Promise<T>",
				Description: "Wraps a promise with automatic loading state management",
				IsProperty:  false,
			},
			{
				Label:       "withLoading",
				Signature:   "withLoading<T>(target: string | Element, fn: () => Promise<T>): Promise<T>",
				Description: "Shows a loading indicator while a function executes",
				IsProperty:  false,
			},
			{
				Label:       "withRetry",
				Signature:   "withRetry<T>(fn: () => Promise<T>, options?: RetryOptions): Promise<T>",
				Description: "Wraps an async function with retry logic",
				IsProperty:  false,
			},
		},
		"piko.event": {
			{
				Label:       "dispatch",
				Signature:   "dispatch(target: string | Element, event: string, data?: unknown): void",
				Description: "Dispatches a custom event to a target element",
				IsProperty:  false,
			},
			{
				Label:       "listen",
				Signature:   "listen(target: string | Element, event: string, handler: (data: unknown) => void): () => void",
				Description: "Listens for custom events on a target element",
				IsProperty:  false,
			},
			{
				Label:       "listenOnce",
				Signature:   "listenOnce(target: string | Element, event: string, handler: (data: unknown) => void): () => void",
				Description: "Listens for an event once, then automatically unsubscribes",
				IsProperty:  false,
			},
			{
				Label:       "waitFor",
				Signature:   "waitFor(target: string | Element, event: string, timeout?: number): Promise<unknown>",
				Description: "Creates a promise that resolves when an event is received",
				IsProperty:  false,
			},
		},
		"piko.partials": {
			{
				Label:       "reload",
				Signature:   "reload(name: string, options?: ReloadOptions): Promise<void>",
				Description: "Reloads a single partial with retry and debounce support",
				IsProperty:  false,
			},
			{
				Label:       "reloadGroup",
				Signature:   "reloadGroup(names: string[], options?: GroupReloadOptions): Promise<void>",
				Description: "Reloads multiple partials in parallel or sequential order",
				IsProperty:  false,
			},
			{
				Label:       "reloadCascade",
				Signature:   "reloadCascade(dependencies: CascadeDependencies): Promise<void>",
				Description: "Reloads partials in dependency order (cascade)",
				IsProperty:  false,
			},
			{
				Label:       "autoRefresh",
				Signature:   "autoRefresh(name: string, options: AutoRefreshOptions): () => void",
				Description: "Sets up automatic refresh for a partial at specified intervals",
				IsProperty:  false,
			},
		},
		"piko.sse": {
			{
				Label:       "subscribe",
				Signature:   "subscribe(channel: string, options: SSEOptions): () => void",
				Description: "Subscribes to server-sent events",
				IsProperty:  false,
			},
			{
				Label:       "create",
				Signature:   "create(channel: string, options: SSEOptions): SSESubscription",
				Description: "Creates an SSE subscription with state tracking",
				IsProperty:  false,
			},
		},
		"piko.timing": {
			{
				Label:       "debounce",
				Signature:   "debounce<T extends (...args: any[]) => void>(fn: T, delay: number): T",
				Description: "Creates a debounced version of a function",
				IsProperty:  false,
			},
			{
				Label:       "throttle",
				Signature:   "throttle<T extends (...args: any[]) => void>(fn: T, delay: number): T",
				Description: "Creates a throttled version of a function",
				IsProperty:  false,
			},
			{
				Label:       "debounceAsync",
				Signature:   "debounceAsync<T>(fn: () => Promise<T>, delay: number): () => Promise<T>",
				Description: "Creates a debounced version of an async function",
				IsProperty:  false,
			},
			{
				Label:       "throttleAsync",
				Signature:   "throttleAsync<T>(fn: () => Promise<T>, delay: number): () => Promise<T>",
				Description: "Creates a throttled version of an async function",
				IsProperty:  false,
			},
			{
				Label:       "timeout",
				Signature:   "timeout(ms: number): AbortableOperation<void>",
				Description: "Creates a timeout promise that can be cancelled",
				IsProperty:  false,
			},
			{
				Label:       "poll",
				Signature:   "poll<T>(fn: () => Promise<T>, interval: number, options?: PollOptions): AbortableOperation<T>",
				Description: "Polls a function at specified intervals",
				IsProperty:  false,
			},
			{
				Label:       "nextFrame",
				Signature:   "nextFrame(): Promise<void>",
				Description: "Returns a promise that resolves on the next animation frame",
				IsProperty:  false,
			},
			{
				Label:       "waitFrames",
				Signature:   "waitFrames(count: number): Promise<void>",
				Description: "Waits for multiple animation frames",
				IsProperty:  false,
			},
		},
		"piko.util": {
			{
				Label:       "whenVisible",
				Signature:   "whenVisible(target: string | Element, callback: () => void, options?: IntersectionOptions): () => void",
				Description: "Executes a callback when an element becomes visible",
				IsProperty:  false,
			},
			{
				Label:       "withAbortSignal",
				Signature:   "withAbortSignal<T>(fn: (signal: AbortSignal) => Promise<T>): AbortableOperation<T>",
				Description: "Creates a cancellable async operation",
				IsProperty:  false,
			},
			{
				Label:       "watchMutations",
				Signature:   "watchMutations(target: string | Element, callback: MutationCallback, options?: MutationObserverInit): () => void",
				Description: "Watches for DOM mutations on an element",
				IsProperty:  false,
			},
			{
				Label:       "whenIdle",
				Signature:   "whenIdle(callback: () => void, options?: IdleOptions): () => void",
				Description: "Executes a function when the browser is idle",
				IsProperty:  false,
			},
			{
				Label:       "deferred",
				Signature:   "deferred<T>(): DeferredPromise<T>",
				Description: "Creates a deferred promise with external resolve/reject",
				IsProperty:  false,
			},
			{
				Label:       "once",
				Signature:   "once<T extends (...args: any[]) => any>(fn: T): T",
				Description: "Creates a function that only executes once",
				IsProperty:  false,
			},
		},
		"piko.trace": {
			{
				Label:       "enable",
				Signature:   "enable(config?: Partial<TraceConfig>): void",
				Description: "Enables tracing with optional configuration",
				IsProperty:  false,
			},
			{
				Label:       "disable",
				Signature:   "disable(): void",
				Description: "Disables tracing",
				IsProperty:  false,
			},
			{
				Label:       "isEnabled",
				Signature:   "isEnabled(): boolean",
				Description: "Checks if tracing is enabled",
				IsProperty:  false,
			},
			{
				Label:       "clear",
				Signature:   "clear(): void",
				Description: "Clears all trace entries",
				IsProperty:  false,
			},
			{
				Label:       "getEntries",
				Signature:   "getEntries(): TraceEntry[]",
				Description: "Gets all trace entries",
				IsProperty:  false,
			},
			{
				Label:       "getMetrics",
				Signature:   "getMetrics(): Record<string, TraceMetrics>",
				Description: "Gets aggregated metrics from trace entries",
				IsProperty:  false,
			},
			{
				Label:       "log",
				Signature:   "log(category: string, data?: unknown): void",
				Description: "Logs a custom trace entry",
				IsProperty:  false,
			},
		},
	}
)

// WithActionProvider sets the action manifest provider for real action completions.
//
// Takes provider (ActionManifestProvider) which supplies action manifests.
//
// Returns TypeInfoServiceOption which configures the service with the provider.
func WithActionProvider(provider ActionManifestProvider) TypeInfoServiceOption {
	return func(s *TypeInfoService) {
		s.actionProvider = provider
	}
}

// buildActionSignature constructs a TypeScript function signature for an action.
//
// Takes action (*annotator_dto.ActionDefinition) which provides the action metadata
// including input and output types.
//
// Returns string which is the formatted TypeScript signature.
func buildActionSignature(action *annotator_dto.ActionDefinition) string {
	params := buildCallParamList(action.CallParams)

	var outputType string
	if action.OutputType != nil {
		outputType = action.OutputType.TSType
		if outputType == "" {
			outputType = action.OutputType.Name
		}
	}

	if outputType != "" {
		return action.TSFunctionName + "(" + params + "): ActionBuilder<" + outputType + ">"
	}
	return action.TSFunctionName + "(" + params + "): ActionBuilder<void>"
}

// buildCallParamList builds a comma-separated list of TypeScript parameter types from a
// slice of ActionTypeInfo.
//
// Takes params ([]annotator_dto.ActionTypeInfo) which are the call parameters.
//
// Returns string which is the comma-separated parameter types.
func buildCallParamList(params []annotator_dto.ActionTypeInfo) string {
	if len(params) == 0 {
		return ""
	}

	parts := make([]string, 0, len(params))
	for i := range params {
		typeName := params[i].TSType
		if typeName == "" {
			typeName = params[i].Name
		}
		parts = append(parts, typeName)
	}
	return strings.Join(parts, ", ")
}
