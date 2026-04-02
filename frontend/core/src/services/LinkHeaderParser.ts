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

/** Parses and applies HTTP Link headers for resource preloading. */
export interface LinkHeaderParser {
    /**
     * Parses a Link header string and adds corresponding link elements to the document head.
     * Skips links that already exist in the document.
     * @param linkHeader - The raw Link header string to parse.
     */
    parseAndApply(linkHeader: string): void;
}

/**
 * Creates a LinkHeaderParser for processing HTTP Link headers into DOM link elements.
 * @returns A new LinkHeaderParser instance.
 */
export function createLinkHeaderParser(): LinkHeaderParser {
    return {
        parseAndApply(linkHeader: string) {
            if (!linkHeader) {
                return;
            }

            const links = linkHeader.split(/,\s*/);

            links.forEach(link => {
                const parts = link.split(';');
                const urlMatch = parts[0].trim().match(/<(.+)>/);
                if (!urlMatch) {
                    return;
                }

                const url = urlMatch[1];
                const params: Record<string, string> = {};

                for (let i = 1; i < parts.length; i++) {
                    const paramParts = parts[i].trim().split('=');
                    const key = paramParts[0];
                    params[key] = paramParts[1] ? paramParts[1].replace(/"/g, '') : 'true';
                }

                if (document.querySelector(`link[href="${url}"]`)) {
                    return;
                }

                const linkEl = document.createElement('link');
                linkEl.href = url;

                if (params.rel) {
                    linkEl.rel = params.rel;
                }
                if (params.as) {
                    linkEl.setAttribute('as', params.as);
                }
                if ('crossorigin' in params) {
                    linkEl.crossOrigin = params.crossorigin === 'true' ? '' : params.crossorigin;
                }
                if (params.type) {
                    linkEl.type = params.type;
                }

                document.head.appendChild(linkEl);
            });
        }
    };
}
