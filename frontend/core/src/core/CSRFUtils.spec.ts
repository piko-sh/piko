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

import {describe, it, expect, beforeEach, afterEach} from 'vitest';
import {
    getCSRFTokenFromMeta,
    getCSRFEphemeralFromMeta,
    getCSRFTokensFromMeta
} from '@/core/CSRFUtils';

describe('CSRFUtils', () => {
    let originalHead: HTMLHeadElement;

    beforeEach(() => {
        originalHead = document.head.cloneNode(true) as HTMLHeadElement;
        document.querySelectorAll('meta[name="csrf-token"], meta[name="csrf-ephemeral"]')
            .forEach(el => el.remove());
    });

    afterEach(() => {
        document.head.innerHTML = originalHead.innerHTML;
    });

    describe('getCSRFTokenFromMeta()', () => {
        it('should return null when no csrf-token meta tag exists', () => {
            const result = getCSRFTokenFromMeta();

            expect(result).toBeNull();
        });

        it('should return token when csrf-token meta tag exists', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-token';
            meta.content = 'test-action-token-123';
            document.head.appendChild(meta);

            const result = getCSRFTokenFromMeta();

            expect(result).toBe('test-action-token-123');
        });

        it('should return empty string when meta tag has empty content', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-token';
            meta.content = '';
            document.head.appendChild(meta);

            const result = getCSRFTokenFromMeta();

            expect(result).toBe('');
        });

        it('should return first meta tag value if multiple exist', () => {
            const meta1 = document.createElement('meta');
            meta1.name = 'csrf-token';
            meta1.content = 'first-token';
            document.head.appendChild(meta1);

            const meta2 = document.createElement('meta');
            meta2.name = 'csrf-token';
            meta2.content = 'second-token';
            document.head.appendChild(meta2);

            const result = getCSRFTokenFromMeta();

            expect(result).toBe('first-token');
        });

        it('should handle special characters in token', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-token';
            meta.content = 'token+with/special=chars==';
            document.head.appendChild(meta);

            const result = getCSRFTokenFromMeta();

            expect(result).toBe('token+with/special=chars==');
        });
    });

    describe('getCSRFEphemeralFromMeta()', () => {
        it('should return null when no csrf-ephemeral meta tag exists', () => {
            const result = getCSRFEphemeralFromMeta();

            expect(result).toBeNull();
        });

        it('should return token when csrf-ephemeral meta tag exists', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-ephemeral';
            meta.content = 'test-ephemeral-token-456';
            document.head.appendChild(meta);

            const result = getCSRFEphemeralFromMeta();

            expect(result).toBe('test-ephemeral-token-456');
        });

        it('should return empty string when meta tag has empty content', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-ephemeral';
            meta.content = '';
            document.head.appendChild(meta);

            const result = getCSRFEphemeralFromMeta();

            expect(result).toBe('');
        });
    });

    describe('getCSRFTokensFromMeta()', () => {
        it('should return both null when no meta tags exist', () => {
            const result = getCSRFTokensFromMeta();

            expect(result).toEqual({
                actionToken: null,
                ephemeralToken: null
            });
        });

        it('should return both tokens when both meta tags exist', () => {
            const actionMeta = document.createElement('meta');
            actionMeta.name = 'csrf-token';
            actionMeta.content = 'action-token-value';
            document.head.appendChild(actionMeta);

            const ephemeralMeta = document.createElement('meta');
            ephemeralMeta.name = 'csrf-ephemeral';
            ephemeralMeta.content = 'ephemeral-token-value';
            document.head.appendChild(ephemeralMeta);

            const result = getCSRFTokensFromMeta();

            expect(result).toEqual({
                actionToken: 'action-token-value',
                ephemeralToken: 'ephemeral-token-value'
            });
        });

        it('should return only action token when ephemeral is missing', () => {
            const actionMeta = document.createElement('meta');
            actionMeta.name = 'csrf-token';
            actionMeta.content = 'action-token-only';
            document.head.appendChild(actionMeta);

            const result = getCSRFTokensFromMeta();

            expect(result).toEqual({
                actionToken: 'action-token-only',
                ephemeralToken: null
            });
        });

        it('should return only ephemeral token when action is missing', () => {
            const ephemeralMeta = document.createElement('meta');
            ephemeralMeta.name = 'csrf-ephemeral';
            ephemeralMeta.content = 'ephemeral-token-only';
            document.head.appendChild(ephemeralMeta);

            const result = getCSRFTokensFromMeta();

            expect(result).toEqual({
                actionToken: null,
                ephemeralToken: 'ephemeral-token-only'
            });
        });
    });

    describe('meta tag placement', () => {
        it('should find meta tags in body (non-standard but possible)', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-token';
            meta.content = 'body-token';
            document.body.appendChild(meta);

            const result = getCSRFTokenFromMeta();

            expect(result).toBe('body-token');

            meta.remove();
        });

        it('should find meta tags regardless of other attributes', () => {
            const meta = document.createElement('meta');
            meta.name = 'csrf-token';
            meta.content = 'token-with-attrs';
            meta.setAttribute('data-custom', 'value');
            meta.id = 'csrf-meta';
            document.head.appendChild(meta);

            const result = getCSRFTokenFromMeta();

            expect(result).toBe('token-with-attrs');
        });
    });
});
