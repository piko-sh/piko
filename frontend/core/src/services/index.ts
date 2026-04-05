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

export {createErrorDisplay} from './ErrorDisplay';
export type {ErrorDisplay, ErrorDisplayOptions} from './ErrorDisplay';

export {createHelperRegistry} from './HelperRegistry';
export type {HelperRegistry, PPHelper} from './HelperRegistry';

export {createSpriteSheetManager} from './SpriteSheetManager';
export type {SpriteSheetManager} from './SpriteSheetManager';

export {createModuleLoader, initModuleLoaderFromPage} from './ModuleLoader';
export type {ModuleLoader} from './ModuleLoader';

export {createLinkHeaderParser} from './LinkHeaderParser';
export type {LinkHeaderParser} from './LinkHeaderParser';

export {createDOMBinder} from './DOMBinder';
export type {DOMBinder, DOMBinderCallbacks, ActionArg, OpenModalOptions} from './DOMBinder';

export {createHookManager, HookEvent} from './HookManager';
export type {
    HookManager,
    HooksAPI,
    HookEventType,
    HookCallback,
    HookOptions,
    HookPayloads,
    FrameworkReadyPayload,
    PageViewPayload,
    NavigationPayload,
    NavigationCompletePayload,
    NavigationErrorPayload,
    ActionPayload,
    ActionCompletePayload,
    ModalPayload,
    PartialRenderPayload,
    FormStatePayload,
    NetworkPayload,
    ErrorPayload
} from './HookManager';

export {createNetworkStatus} from './NetworkStatus';
export type {NetworkStatus, NetworkStatusCallback, NetworkStatusDependencies} from './NetworkStatus';

export {createPageContext, getGlobalPageContext, resetGlobalPageContext, findClosestMatch} from './PageContext';
export type {PageContext, PageContextOptions} from './PageContext';
