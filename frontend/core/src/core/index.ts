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

export {PPFramework, RegisterHelper} from './PPFramework';
export type {PPFrameworkOptions, PPHelper} from './PPFramework';

export {
    browserAPIs,
    browserDOMOperations,
    browserWindowOperations,
    browserHTTPOperations,
    createBrowserAPIs
} from './BrowserAPIs';
export type {
    BrowserAPIs,
    DOMOperations,
    WindowOperations,
    HTTPOperations
} from './BrowserAPIs';

export {default as fragmentMorpher} from './fragmentMorpher';
export type {MorphOptions, NodeKey} from './fragmentMorpher';

export {createFetchClient} from './FetchClient';
export type {FetchClient, FetchClientOptions, FetchClientDependencies, FetchResult} from './FetchClient';

export {createRouter} from './Router';
export type {Router, RouterConfig, RouterDependencies, NavigateOptions} from './Router';

export {createRemoteRenderer} from './RemoteRenderer';
export type {RemoteRenderer, RemoteRendererDependencies, RemoteRenderOptions, PatchTarget} from './RemoteRenderer';

export {createModalManager} from './ModalManager';
export type {ModalManager, ModalRequestOptions} from './ModalManager';

export {addFragmentQuery, buildRemoteUrl, isSameDomain} from './URLUtils';

export {
    handleAction,
    onActionError,
    clearGlobalErrorHandler,
    clearAllDebounceTimers,
    callServerActionDirect
} from './ActionExecutor';
export type {
    GlobalActionErrorHandler,
    DirectCallOptions,
    DirectCallResponse
} from './ActionExecutor';

export {readSSEStream} from './SSEStreamReader';
export type {SSEStreamCallbacks} from './SSEStreamReader';
