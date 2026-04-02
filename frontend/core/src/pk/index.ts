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

export {piko} from './namespace';

export {
    _runPageCleanup,
    _initCleanupObserver,
    _registerLifecycle,
    _executeConnected,
    _executeDisconnected,
    _executeBeforeRender,
    _executeAfterRender,
    _executeUpdated,
    _executeConnectedForPartials,
    _hasLifecycleCallbacks,
} from './lifecycle';

export {_createPKContext} from './context';
export type {PKContext} from './context';

export {action, isActionDescriptor, createActionError, createActionBuilder, ActionBuilder, registerActionFunction, getActionFunction} from './action';
export type {ActionDescriptor, ActionError, ActionMethod, RetryConfig, RetryBackoff} from './action';

export {
    handleAction,
    onActionError,
    clearGlobalErrorHandler,
    clearAllDebounceTimers,
} from '../core/ActionExecutor';
export type {GlobalActionErrorHandler} from '../core/ActionExecutor';

export {bus} from './bus';
export {getGlobalPageContext} from '../services/PageContext';

export type {PartialHandle} from './partial';
export type {PKLifecycleCallbacks} from './lifecycle';
export type {PikoDebugAPI, PartialDebugInfo} from './debug';
export type {ReloadOptions, ReloadGroupOptions, AutoRefreshOptions, CascadeNode, CascadeOptions} from './coordination';
export type {SSEOptions, SSESubscription} from './sse';
export type {DispatchOptions, EventListener} from './events';
export type {TraceConfig, TraceEntry, TraceMetrics} from './trace';
export type {FormDataHandle, ValidationRule, ValidationRules, ValidationResult} from './form';
export type {LoadingOptions, RetryOptions, RetryResult} from './ui';
export type {NavigateOptions, RouteInfo, NavigationGuard} from './navigation';
export type {WhenVisibleOptions, AbortableOperation, PollingOptions, MutationWatchOptions} from './advanced';
