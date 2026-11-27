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
 * Claims the window for this runtime before any other runtime module runs.
 *
 * main.ts imports this module first, so it is evaluated before every module that sets
 * window globals or binds events. When another runtime already runs in the window, the
 * claim throws, which stops the rest of this runtime from being evaluated at all.
 */

import {claimRuntime} from '@/core/runtimeGuard';
import {runtimeVersionGate} from '@/core/RuntimeVersion';

if (typeof window !== 'undefined') {
    claimRuntime(
        {
            win: window,
            reload: () => window.location.reload(),
            claimReload: () => runtimeVersionGate.claimReload()
        },
        {version: runtimeVersionGate.ownVersion, moduleURL: import.meta.url}
    );
}
