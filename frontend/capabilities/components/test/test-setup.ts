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

import { beforeEach } from 'vitest';

// Provide a minimal piko namespace stub so tests that call dom.pikoEl()
// (which accesses window.piko.nav and window.piko.asset) do not fail.
(window as Record<string, unknown>).piko = {
  nav: {
    navigateTo(_url: string, _event?: Event): void { /* noop */ },
    navigate(_url: string): Promise<void> { return Promise.resolve(); },
  },
  asset: {
    resolve(src: string, _moduleName?: string): string { return src; },
  },
};

beforeEach(() => {
  document.body.innerHTML = '';
  document.head.innerHTML = '';

  const appRoot = document.createElement('div');
  appRoot.id = 'app';
  document.body.appendChild(appRoot);
});
