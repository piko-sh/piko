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

/**
 * Stands in for /_piko/dist/ppframework.core.es.js when generated PK modules are loaded
 * from source. It re-exports the runtime from the same source modules the bundle entry
 * (src/main.ts) exports it from, without starting the framework as importing main.ts
 * would.
 */

export {_createPKContext, _createPagePKContext} from '@/pk/context';
export {_currentPageVisit, _initCleanupObserver, _registerLifecycle, _runPageCleanup} from '@/pk/lifecycle';
export {getGlobalPageContext} from '@/services/PageContext';
