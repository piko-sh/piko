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

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createSyncPartialManager, type SyncPartialManager, type SyncPartialCallbacks } from './SyncPartialManager';

class MockIntersectionObserver {
    private readonly callback: IntersectionObserverCallback;
    private elements = new Set<Element>();
    static instances: MockIntersectionObserver[] = [];

    constructor(callback: IntersectionObserverCallback, _options?: IntersectionObserverInit) {
        this.callback = callback;
        MockIntersectionObserver.instances.push(this);
    }

    observe(target: Element) {
        this.elements.add(target);
    }

    unobserve(target: Element) {
        this.elements.delete(target);
    }

    disconnect() {
        this.elements.clear();
    }

    triggerIntersection(entries: Partial<IntersectionObserverEntry>[]) {
        const fullEntries = entries.map(entry => ({
            boundingClientRect: {} as DOMRectReadOnly,
            intersectionRatio: entry.isIntersecting ? 1 : 0,
            intersectionRect: {} as DOMRectReadOnly,
            isIntersecting: false,
            rootBounds: null,
            target: document.createElement('div'),
            time: Date.now(),
            ...entry
        })) as IntersectionObserverEntry[];
        this.callback(fullEntries, this as unknown as IntersectionObserver);
    }
}

describe('SyncPartialManager', () => {
    let manager: SyncPartialManager;
    let callbacks: SyncPartialCallbacks;
    let onRemoteRenderSpy: ReturnType<typeof vi.fn>;
    let rootElement: HTMLElement;

    beforeEach(() => {
        vi.useFakeTimers();

        vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
        MockIntersectionObserver.instances = [];

        onRemoteRenderSpy = vi.fn().mockResolvedValue(undefined);
        callbacks = {
            onRemoteRender: onRemoteRenderSpy as unknown as SyncPartialCallbacks['onRemoteRender']
        };

        rootElement = document.createElement('div');
        document.body.appendChild(rootElement);
    });

    afterEach(() => {
        rootElement.remove();
        vi.useRealTimers();
        vi.unstubAllGlobals();
    });

    const createSyncContainer = (partialSrc: string): HTMLElement => {
        const container = document.createElement('div');
        container.setAttribute('partial_mode', 'sync');
        container.setAttribute('partial_src', partialSrc);
        return container;
    };

    describe('bind()', () => {
        it('should bind sync containers within root element', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            expect(container.getAttribute('pk-sync-bound')).toBe('true');
        });

        it('should warn and skip containers without partial_src', () => {
            manager = createSyncPartialManager(callbacks);
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const container = document.createElement('div');
            container.setAttribute('partial_mode', 'sync');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            expect(consoleSpy).toHaveBeenCalledWith(
                'SyncPartialManager: A sync container is missing its "partial_src" attribute.',
                container
            );
            expect(container.getAttribute('pk-sync-bound')).toBeNull();

            consoleSpy.mockRestore();
        });

        it('should not re-bind already bound containers', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            container.setAttribute('pk-sync-bound', 'true');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            expect(MockIntersectionObserver.instances).toHaveLength(1);
        });

        it('should observe containers with IntersectionObserver', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            vi.advanceTimersByTime(32);

            expect(MockIntersectionObserver.instances).toHaveLength(1);
        });

        it('should bind multiple containers', () => {
            manager = createSyncPartialManager(callbacks);
            const container1 = createSyncContainer('/api/partial1');
            const container2 = createSyncContainer('/api/partial2');
            rootElement.appendChild(container1);
            rootElement.appendChild(container2);

            manager.bind(rootElement);

            expect(container1.getAttribute('pk-sync-bound')).toBe('true');
            expect(container2.getAttribute('pk-sync-bound')).toBe('true');
        });
    });

    describe('input event handling', () => {
        it('should debounce input events and call onRemoteRender', async () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const input = document.createElement('input');
            container.appendChild(input);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            input.dispatchEvent(new Event('input', { bubbles: true }));

            expect(onRemoteRenderSpy).not.toHaveBeenCalled();

            vi.advanceTimersByTime(16);
            vi.advanceTimersByTime(400);

            expect(onRemoteRenderSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    src: '/api/partial',
                    patchMethod: 'morph',
                    childrenOnly: true,
                    patchLocation: container
                })
            );
        });

        it('should reset debounce timer on subsequent input events', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const input = document.createElement('input');
            container.appendChild(input);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            input.dispatchEvent(new Event('input', { bubbles: true }));
            vi.advanceTimersByTime(16);
            vi.advanceTimersByTime(200);

            input.dispatchEvent(new Event('input', { bubbles: true }));
            vi.advanceTimersByTime(16);
            vi.advanceTimersByTime(200);

            expect(onRemoteRenderSpy).not.toHaveBeenCalled();

            vi.advanceTimersByTime(200);

            expect(onRemoteRenderSpy).toHaveBeenCalledTimes(1);
        });
    });

    describe('change event handling', () => {
        it('should trigger immediate update for SELECT elements', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const select = document.createElement('select');
            container.appendChild(select);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            select.dispatchEvent(new Event('change', { bubbles: true }));
            vi.advanceTimersByTime(16);

            expect(onRemoteRenderSpy).toHaveBeenCalled();
        });

        it('should trigger immediate update for INPUT elements (non-text)', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const input = document.createElement('input');
            input.type = 'checkbox';
            container.appendChild(input);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            input.dispatchEvent(new Event('change', { bubbles: true }));
            vi.advanceTimersByTime(16);

            expect(onRemoteRenderSpy).toHaveBeenCalled();
        });

        it('should NOT trigger immediate update for text INPUT elements', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const input = document.createElement('input');
            input.type = 'text';
            container.appendChild(input);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            input.dispatchEvent(new Event('change', { bubbles: true }));
            vi.advanceTimersByTime(16);

            expect(onRemoteRenderSpy).not.toHaveBeenCalled();
        });

        it('should NOT trigger for non-trigger elements', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const div = document.createElement('div');
            container.appendChild(div);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            div.dispatchEvent(new Event('change', { bubbles: true }));
            vi.advanceTimersByTime(16);

            expect(onRemoteRenderSpy).not.toHaveBeenCalled();
        });

        it('should trigger for PP-SELECT elements', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const ppSelect = document.createElement('pp-select');
            container.appendChild(ppSelect);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            ppSelect.dispatchEvent(new Event('change', { bubbles: true }));
            vi.advanceTimersByTime(16);

            expect(onRemoteRenderSpy).toHaveBeenCalled();
        });

        it('should trigger for PP-CHECKBOX elements', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            const ppCheckbox = document.createElement('pp-checkbox');
            container.appendChild(ppCheckbox);
            rootElement.appendChild(container);

            manager.bind(rootElement);

            ppCheckbox.dispatchEvent(new Event('change', { bubbles: true }));
            vi.advanceTimersByTime(16);

            expect(onRemoteRenderSpy).toHaveBeenCalled();
        });
    });

    describe('refresh-partial event handling', () => {
        it('should trigger update on refresh-partial event', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            container.dispatchEvent(new CustomEvent('refresh-partial', { bubbles: false }));

            expect(onRemoteRenderSpy).toHaveBeenCalled();
        });

        it('should use custom formData from event detail', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            const customFormData = { name: ['test'] };
            container.dispatchEvent(new CustomEvent('refresh-partial', {
                bubbles: false,
                detail: { formData: customFormData }
            }));

            expect(onRemoteRenderSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    formData: customFormData
                })
            );
        });

        it('should call afterMorph callback from event detail', async () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            const afterMorphSpy = vi.fn();
            container.dispatchEvent(new CustomEvent('refresh-partial', {
                bubbles: false,
                detail: { afterMorph: afterMorphSpy }
            }));

            await vi.runAllTimersAsync();

            expect(afterMorphSpy).toHaveBeenCalled();
        });

        it('should stop propagation of refresh-partial event', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            const parentListener = vi.fn();
            rootElement.addEventListener('refresh-partial', parentListener);

            container.dispatchEvent(new CustomEvent('refresh-partial', { bubbles: true }));

            expect(parentListener).not.toHaveBeenCalled();
        });
    });

    describe('IntersectionObserver visibility handling', () => {
        it('should trigger refresh when container becomes visible', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            vi.advanceTimersByTime(32);

            const observer = MockIntersectionObserver.instances[0];
            observer.triggerIntersection([{
                target: container,
                isIntersecting: true
            }]);

            vi.advanceTimersByTime(150);

            expect(onRemoteRenderSpy).toHaveBeenCalled();
        });

        it('should debounce multiple visibility changes', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);
            vi.advanceTimersByTime(32);

            const observer = MockIntersectionObserver.instances[0];

            observer.triggerIntersection([{ target: container, isIntersecting: true }]);
            vi.advanceTimersByTime(50);

            observer.triggerIntersection([{ target: container, isIntersecting: true }]);
            vi.advanceTimersByTime(150);

            expect(onRemoteRenderSpy).toHaveBeenCalledTimes(1);
        });

        it('should not trigger when container becomes hidden', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);
            vi.advanceTimersByTime(32);

            const observer = MockIntersectionObserver.instances[0];

            observer.triggerIntersection([{ target: container, isIntersecting: true }]);
            vi.advanceTimersByTime(150);

            onRemoteRenderSpy.mockClear();

            observer.triggerIntersection([{ target: container, isIntersecting: false }]);
            vi.advanceTimersByTime(150);

            expect(onRemoteRenderSpy).not.toHaveBeenCalled();
        });

        it('should trigger again when re-entering viewport', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);
            vi.advanceTimersByTime(32);

            const observer = MockIntersectionObserver.instances[0];

            observer.triggerIntersection([{ target: container, isIntersecting: true }]);
            vi.advanceTimersByTime(150);

            observer.triggerIntersection([{ target: container, isIntersecting: false }]);
            vi.advanceTimersByTime(150);

            onRemoteRenderSpy.mockClear();

            observer.triggerIntersection([{ target: container, isIntersecting: true }]);
            vi.advanceTimersByTime(150);

            expect(onRemoteRenderSpy).toHaveBeenCalledTimes(1);
        });
    });

    describe('form data gathering', () => {
        it('should gather form data when container is inside a form', () => {
            manager = createSyncPartialManager(callbacks);
            const form = document.createElement('form');
            const container = createSyncContainer('/api/partial');
            const input = document.createElement('input');
            input.name = 'username';
            input.value = 'testuser';
            form.appendChild(input);
            form.appendChild(container);
            rootElement.appendChild(form);

            manager.bind(rootElement);

            container.dispatchEvent(new CustomEvent('refresh-partial', { bubbles: false }));

            expect(onRemoteRenderSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    formData: expect.objectContaining({
                        username: ['testuser']
                    })
                })
            );
        });

        it('should pass undefined formData when no form is present', () => {
            manager = createSyncPartialManager(callbacks);
            const container = createSyncContainer('/api/partial');
            rootElement.appendChild(container);

            manager.bind(rootElement);

            container.dispatchEvent(new CustomEvent('refresh-partial', { bubbles: false }));

            expect(onRemoteRenderSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    formData: undefined
                })
            );
        });
    });
});
