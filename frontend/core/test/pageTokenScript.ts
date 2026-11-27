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
 * Adds a script element that carries a page token the way a browser shows it after load.
 * The attribute is present but empty, and only the `nonce` property holds the token.
 * jsdom keeps the property and the attribute in step, so the property is set directly.
 *
 * @param token - The page token.
 * @param parent - Where to add the script. Defaults to the document head.
 * @returns The added script.
 */
export function addPageTokenScript(token: string, parent: ParentNode = document.head): HTMLScriptElement {
    const script = document.createElement('script');
    script.setAttribute('nonce', '');
    Object.defineProperty(script, 'nonce', {value: token, configurable: true});
    parent.prepend(script);
    return script;
}
