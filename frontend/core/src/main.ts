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

import {PPFramework} from '@/core';
// Side-effect: registers core helpers (submitForm, submitModalForm, resetForm, redirect, emitEvent, dispatchEvent).
import '@/helpers/formHelpers';
import '@/helpers/redirect';
import '@/helpers/emitEvent';
import '@/helpers/dispatchEvent';

// Imported directly from source modules to avoid Rollup circular chunk warnings.
import {_initCleanupObserver, _registerLifecycle, _runPageCleanup} from '@/pk/lifecycle';
import {ActionBuilder, createActionBuilder, registerActionFunction} from '@/pk/action';
import {getGlobalPageContext} from '@/services/PageContext';
import {bus} from '@/pk/bus';
import {piko} from '@/pk/namespace';
import {createRefs as _createRefs} from '@/pk/refs';
import {_createPKContext} from '@/pk/context';

export {piko};

export {ActionBuilder, createActionBuilder, registerActionFunction};

export {_registerLifecycle, _runPageCleanup, _initCleanupObserver};
export {getGlobalPageContext};
export {bus};
export {_createRefs};
export {_createPKContext};

export type {
    PPHelper, PPFrameworkOptions, NavigateOptions, RemoteRenderOptions, PatchTarget, FetchResult
} from '@/core';
export type {
    PartialHandle, ReloadOptions, ReloadGroupOptions, AutoRefreshOptions,
    CascadeNode, CascadeOptions, SSEOptions, SSESubscription,
    DispatchOptions, EventListener as PKEventListener,
    TraceConfig, TraceEntry, TraceMetrics,
    FormDataHandle, ValidationRule, ValidationRules, ValidationResult,
    LoadingOptions, RetryOptions, RetryResult,
    NavigateOptions as PKNavigateOptions, RouteInfo, NavigationGuard,
    WhenVisibleOptions, AbortableOperation, PollingOptions, MutationWatchOptions
} from '@/pk';

_initCleanupObserver();

PPFramework.init();

piko._markReady();
