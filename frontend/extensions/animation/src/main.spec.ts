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

import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('main', () => {
    beforeEach(() => {
        vi.resetModules();
        delete (window as unknown as Record<string, unknown>).__piko_animation;
        delete (window as unknown as Record<string, unknown>).__piko_registerTimelineAction;
    });

    it('should expose setupAnimation on window.__piko_animation', async () => {
        await import('./main');

        expect((window as unknown as Record<string, unknown>).__piko_animation).toBeTypeOf('function');
    });

    it('should expose registerTimelineAction on window.__piko_registerTimelineAction', async () => {
        await import('./main');

        expect((window as unknown as Record<string, unknown>).__piko_registerTimelineAction).toBeTypeOf('function');
    });
});
