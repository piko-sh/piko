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
import { createDOMBinder, type DOMBinder, type DOMBinderCallbacks } from '@/services/DOMBinder';
import { createHelperRegistry, type HelperRegistry } from '@/services/HelperRegistry';
import * as PageContext from '@/services/PageContext';
import * as ActionExecutor from '@/core/ActionExecutor';
import * as ActionModule from '@/pk/action';
import { action } from '@/pk/action';

describe('DOMBinder', () => {
    let domBinder: DOMBinder;
    let helperRegistry: HelperRegistry;
    let callbacks: DOMBinderCallbacks;
    let root: HTMLDivElement;

    beforeEach(() => {
        helperRegistry = createHelperRegistry();
        callbacks = {
            onNavigate: vi.fn(),
            onOpenModal: vi.fn()
        };
        domBinder = createDOMBinder(helperRegistry, callbacks);

        root = document.createElement('div');
        document.body.appendChild(root);
    });

    afterEach(() => {
        root.remove();
        vi.clearAllMocks();
    });

    describe('bindLinks', () => {
        it('should bind click handler to piko:a anchors', () => {
            const link = document.createElement('a');
            link.setAttribute('piko:a', '');
            link.setAttribute('href', '/test-page');
            root.appendChild(link);

            domBinder.bindLinks(root);
            link.click();

            expect(callbacks.onNavigate).toHaveBeenCalledWith('/test-page', expect.any(MouseEvent));
        });

        it('should prevent default on piko:a clicks', () => {
            const link = document.createElement('a');
            link.setAttribute('piko:a', '');
            link.setAttribute('href', '/test');
            root.appendChild(link);

            domBinder.bindLinks(root);

            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
            const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
            link.dispatchEvent(clickEvent);

            expect(preventDefaultSpy).toHaveBeenCalled();
        });

        it('should not navigate if href is missing', () => {
            const link = document.createElement('a');
            link.setAttribute('piko:a', '');
            root.appendChild(link);

            domBinder.bindLinks(root);
            link.click();

            expect(callbacks.onNavigate).not.toHaveBeenCalled();
        });

        it('should rebind links without duplicating handlers', () => {
            const link = document.createElement('a');
            link.setAttribute('piko:a', '');
            link.setAttribute('href', '/test');
            root.appendChild(link);

            domBinder.bindLinks(root);
            domBinder.bindLinks(root);
            link.click();

            expect(callbacks.onNavigate).toHaveBeenCalledTimes(1);
        });
    });

    describe('bindActions', () => {
        it('should dispatch helper when function is in helper registry', () => {
            const helperFn = vi.fn();
            helperRegistry.register('myHelper', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myHelper', a: [{ t: 's', v: 'arg1' }] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledWith(button, expect.any(Event), 'arg1');
        });

        it('should warn when function not found in page context or helper registry', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'unknownFunction', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('not found'));
            warnSpy.mockRestore();
        });

        it('should warn when p-event:customname handler function is not found', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'customAction', a: [] }));
            div.setAttribute('p-event:mycustom', payload);
            root.appendChild(div);

            domBinder.bindActions(root);
            div.dispatchEvent(new CustomEvent('mycustom'));

            expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Function "customAction" not found for p-event handler'));

            warnSpy.mockRestore();
        });

        it('should not prevent default on action events without .prevent modifier', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);

            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
            const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
            button.dispatchEvent(clickEvent);

            expect(preventDefaultSpy).not.toHaveBeenCalled();
            warnSpy.mockRestore();
        });

        it('should call preventDefault when .prevent modifier is used', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.prevent', payload);
            root.appendChild(button);

            domBinder.bindActions(root);

            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
            const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
            button.dispatchEvent(clickEvent);

            expect(preventDefaultSpy).toHaveBeenCalled();
            warnSpy.mockRestore();
        });

        it('should call stopPropagation when .stop modifier is used', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.stop', payload);
            root.appendChild(button);

            domBinder.bindActions(root);

            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
            const stopPropagationSpy = vi.spyOn(clickEvent, 'stopPropagation');
            button.dispatchEvent(clickEvent);

            expect(stopPropagationSpy).toHaveBeenCalled();
            warnSpy.mockRestore();
        });

        it('should fire handler only once when .once modifier is used', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const helperFn = vi.fn();
            helperRegistry.register('testOnce', helperFn as never);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'testOnce', a: [] }));
            button.setAttribute('p-on:click.once', payload);
            root.appendChild(button);

            domBinder.bindActions(root);

            button.click();
            button.click();
            button.click();

            expect(helperFn).toHaveBeenCalledTimes(1);
            warnSpy.mockRestore();
        });

        it('should not fire handler when .self modifier is used and event comes from child', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const helperFn = vi.fn();
            helperRegistry.register('testSelf', helperFn as never);

            const container = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'testSelf', a: [] }));
            container.setAttribute('p-on:click.self', payload);
            const child = document.createElement('button');
            container.appendChild(child);
            root.appendChild(container);

            domBinder.bindActions(root);

            child.click();
            expect(helperFn).not.toHaveBeenCalled();

            container.click();
            expect(helperFn).toHaveBeenCalledTimes(1);
            warnSpy.mockRestore();
        });

        it('should apply multiple modifiers when composing .prevent.stop', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.prevent.stop', payload);
            root.appendChild(button);

            domBinder.bindActions(root);

            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
            const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
            const stopPropagationSpy = vi.spyOn(clickEvent, 'stopPropagation');
            button.dispatchEvent(clickEvent);

            expect(preventDefaultSpy).toHaveBeenCalled();
            expect(stopPropagationSpy).toHaveBeenCalled();
            warnSpy.mockRestore();
        });

        it('should handle invalid base64 payload gracefully', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            const button = document.createElement('button');
            button.setAttribute('p-on:click', 'invalid-base64!!!');
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Could not decode action payload'),
                expect.any(Object)
            );
            errorSpy.mockRestore();
        });

        it('should mark bound elements to prevent double-binding', () => {
            const helperFn = vi.fn();
            helperRegistry.register('myAction', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledTimes(1);
        });

        it('should warn when client-side handler is not found', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'action', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('not found'));
            warnSpy.mockRestore();
        });
    });

    describe('modal binding (p-modal:selector)', () => {
        it('should bind click to open modal', () => {
            const button = document.createElement('button');
            button.setAttribute('p-modal:selector', '#my-modal');
            button.setAttribute('p-modal:title', 'Confirm');
            button.setAttribute('p-modal:message', 'Are you sure?');
            button.setAttribute('p-modal:confirm_label', 'Yes');
            button.setAttribute('p-modal:cancel_label', 'No');
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(callbacks.onOpenModal).toHaveBeenCalledWith({
                selector: '#my-modal',
                params: expect.any(Map),
                title: 'Confirm',
                message: 'Are you sure?',
                cancelLabel: 'No',
                confirmLabel: 'Yes',
                confirmAction: '',
                element: button
            });
        });

        it('should collect p-modal-param:* attributes', () => {
            const button = document.createElement('button');
            button.setAttribute('p-modal:selector', '#modal');
            button.setAttribute('p-modal-param:id', '123');
            button.setAttribute('p-modal-param:name', 'test');
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            const callArgs = (callbacks.onOpenModal as ReturnType<typeof vi.fn>).mock.calls[0][0];
            expect(callArgs.params.get('id')).toBe('123');
            expect(callArgs.params.get('name')).toBe('test');
        });
    });

    describe('bind (combined)', () => {
        it('should bind both links and actions', () => {
            const helperFn = vi.fn();
            helperRegistry.register('action', helperFn);

            const link = document.createElement('a');
            link.setAttribute('piko:a', '');
            link.setAttribute('href', '/page');
            root.appendChild(link);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'action', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bind(root);

            link.click();
            expect(callbacks.onNavigate).toHaveBeenCalled();

            button.click();
            expect(helperFn).toHaveBeenCalled();
        });
    });

    describe('URI scheme handling', () => {
        describe('native URI schemes', () => {
            const nativeSchemes = [
                { scheme: 'tel:', href: 'tel:01234567890', description: 'telephone' },
                { scheme: 'mailto:', href: 'mailto:user@example.com', description: 'email' },
                { scheme: 'sms:', href: 'sms:+1234567890?body=Hello', description: 'SMS' },
                { scheme: 'geo:', href: 'geo:51.5074,-0.1278', description: 'geographic' },
                { scheme: 'webcal:', href: 'webcal://example.com/calendar.ics', description: 'calendar' },
                { scheme: 'facetime:', href: 'facetime:user@example.com', description: 'FaceTime' },
                { scheme: 'facetime-audio:', href: 'facetime-audio:user@example.com', description: 'FaceTime audio' },
                { scheme: 'skype:', href: 'skype:username?call', description: 'Skype' },
                { scheme: 'whatsapp:', href: 'whatsapp://send?phone=1234567890', description: 'WhatsApp' },
                { scheme: 'viber:', href: 'viber://chat?number=1234567890', description: 'Viber' },
                { scheme: 'maps:', href: 'maps://maps.apple.com/?q=London', description: 'Apple Maps' },
                { scheme: 'comgooglemaps:', href: 'comgooglemaps://?q=London', description: 'Google Maps iOS' }
            ];

            it.each(nativeSchemes)(
                'should let browser handle $description links ($scheme) without SPA navigation',
                ({ href }) => {
                    const link = document.createElement('a');
                    link.setAttribute('piko:a', '');
                    link.setAttribute('href', href);
                    root.appendChild(link);

                    domBinder.bindLinks(root);
                    link.click();

                    expect(callbacks.onNavigate).not.toHaveBeenCalled();
                }
            );

            it.each(nativeSchemes)(
                'should not prevent default for $description links ($scheme)',
                ({ href }) => {
                    const link = document.createElement('a');
                    link.setAttribute('piko:a', '');
                    link.setAttribute('href', href);
                    root.appendChild(link);

                    domBinder.bindLinks(root);

                    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
                    const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
                    link.dispatchEvent(clickEvent);

                    expect(preventDefaultSpy).not.toHaveBeenCalled();
                }
            );

            it('should handle native schemes case-insensitively', () => {
                const link = document.createElement('a');
                link.setAttribute('piko:a', '');
                link.setAttribute('href', 'TEL:01234567890');
                root.appendChild(link);

                domBinder.bindLinks(root);
                link.click();

                expect(callbacks.onNavigate).not.toHaveBeenCalled();
            });
        });

        describe('blocked URI schemes', () => {
            const blockedSchemes = [
                { scheme: 'javascript:', href: 'javascript:alert("xss")', description: 'JavaScript' },
                { scheme: 'data:', href: 'data:text/html,<script>alert("xss")</script>', description: 'data URI' },
                { scheme: 'blob:', href: 'blob:https://example.com/uuid', description: 'blob URI' },
                { scheme: 'file:', href: 'file:///etc/passwd', description: 'file URI' }
            ];

            it.each(blockedSchemes)(
                'should block $description links ($scheme) entirely',
                ({ href }) => {
                    const link = document.createElement('a');
                    link.setAttribute('piko:a', '');
                    link.setAttribute('href', href);
                    root.appendChild(link);

                    domBinder.bindLinks(root);
                    link.click();

                    expect(callbacks.onNavigate).not.toHaveBeenCalled();
                }
            );

            it.each(blockedSchemes)(
                'should prevent default for blocked $description links ($scheme)',
                ({ href }) => {
                    const link = document.createElement('a');
                    link.setAttribute('piko:a', '');
                    link.setAttribute('href', href);
                    root.appendChild(link);

                    domBinder.bindLinks(root);

                    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
                    const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
                    link.dispatchEvent(clickEvent);

                    expect(preventDefaultSpy).toHaveBeenCalled();
                }
            );

            it.each(blockedSchemes)(
                'should warn when blocking $description links ($scheme)',
                ({ href }) => {
                    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

                    const link = document.createElement('a');
                    link.setAttribute('piko:a', '');
                    link.setAttribute('href', href);
                    root.appendChild(link);

                    domBinder.bindLinks(root);
                    link.click();

                    expect(warnSpy).toHaveBeenCalledWith(
                        expect.stringContaining('Blocked navigation to dangerous URI scheme'),
                        href
                    );
                    warnSpy.mockRestore();
                }
            );

            it('should handle blocked schemes case-insensitively', () => {
                const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

                const link = document.createElement('a');
                link.setAttribute('piko:a', '');
                link.setAttribute('href', 'JAVASCRIPT:alert("xss")');
                root.appendChild(link);

                domBinder.bindLinks(root);

                const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
                const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
                link.dispatchEvent(clickEvent);

                expect(preventDefaultSpy).toHaveBeenCalled();
                expect(callbacks.onNavigate).not.toHaveBeenCalled();
                warnSpy.mockRestore();
            });
        });

        describe('normal URLs', () => {
            it('should use SPA navigation for relative paths', () => {
                const link = document.createElement('a');
                link.setAttribute('piko:a', '');
                link.setAttribute('href', '/about');
                root.appendChild(link);

                domBinder.bindLinks(root);
                link.click();

                expect(callbacks.onNavigate).toHaveBeenCalledWith('/about', expect.any(MouseEvent));
            });

            it('should use SPA navigation for absolute HTTP URLs', () => {
                const link = document.createElement('a');
                link.setAttribute('piko:a', '');
                link.setAttribute('href', 'https://example.com/page');
                root.appendChild(link);

                domBinder.bindLinks(root);
                link.click();

                expect(callbacks.onNavigate).toHaveBeenCalledWith('https://example.com/page', expect.any(MouseEvent));
            });

            it('should use SPA navigation for fragment-only URLs', () => {
                const link = document.createElement('a');
                link.setAttribute('piko:a', '');
                link.setAttribute('href', '#section');
                root.appendChild(link);

                domBinder.bindLinks(root);
                link.click();

                expect(callbacks.onNavigate).toHaveBeenCalledWith('#section', expect.any(MouseEvent));
            });
        });
    });

    describe('action descriptor detection', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
        });

        it('should call handleAction when handler returns an action descriptor', () => {
            const mockHandler = vi.fn(() => action('testAction', 123));
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'myHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
            expect(handleActionSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    action: 'testAction',
                    args: [123]
                }),
                button,
                expect.any(Event)
            );
        });

        it('should not call handleAction when handler returns non-action value', () => {
            const mockHandler = vi.fn(() => 'not an action');
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'regularHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'regularHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
            expect(handleActionSpy).not.toHaveBeenCalled();
        });

        it('should not call handleAction when handler returns undefined', () => {
            const mockHandler = vi.fn(() => undefined);
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'voidHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'voidHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
            expect(handleActionSpy).not.toHaveBeenCalled();
        });

        it('should not call handleAction when handler returns null', () => {
            const mockHandler = vi.fn(() => null);
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'nullHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'nullHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
            expect(handleActionSpy).not.toHaveBeenCalled();
        });

        it('should pass action descriptor with all builder options', () => {
            const mockHandler = vi.fn(() =>
                action('fullAction', 'arg1', 'arg2')
                    .setMethod('PUT')
                    .setLoading(true)
                    .setDebounce(300)
                    .setRetry(3, 'exponential')
            );
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'fullHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'fullHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(handleActionSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    action: 'fullAction',
                    args: ['arg1', 'arg2'],
                    method: 'PUT',
                    loading: true,
                    debounce: 300,
                    retry: { attempts: 3, backoff: 'exponential' }
                }),
                button,
                expect.any(Event)
            );
        });

        it('should handle action descriptor from custom events (p-event)', () => {
            const mockHandler = vi.fn(() => action('customEventAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'customHandler' ? mockHandler : undefined,
                hasFunction: (name: string) => name === 'customHandler',
                getExportedFunctions: () => ['customHandler']
            });

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'customHandler', a: [] }));
            div.setAttribute('p-event:mycustom', payload);
            root.appendChild(div);

            domBinder.bindActions(root);
            div.dispatchEvent(new CustomEvent('mycustom'));

            expect(mockHandler).toHaveBeenCalled();
            expect(handleActionSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    action: 'customEventAction'
                }),
                div,
                expect.any(CustomEvent)
            );
        });

        it('should pass event to handler function when $event placeholder is used', () => {
            const mockHandler = vi.fn((event: Event) => {
                expect(event).toBeInstanceOf(MouseEvent);
                return action('eventAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'eventHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'eventHandler', a: [{ t: 'e' }] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalledWith(expect.any(Event), expect.any(Event));
        });

        it('should pass scoping event even when no $event placeholder is used', () => {
            const mockHandler = vi.fn((_scopingEvent: Event) => {
                expect(_scopingEvent).toBeInstanceOf(Event);
                return action('noEventAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'noEventHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'noEventHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalledWith(expect.any(Event));
        });

        it('should pass resolved arguments to handler with $event first', () => {
            const mockHandler = vi.fn((_scopingEvent: Event, _resolvedEvent: Event, arg1: string, arg2: number) => {
                expect(_scopingEvent).toBeInstanceOf(Event);
                expect(_resolvedEvent).toBeInstanceOf(Event);
                expect(arg1).toBe('static-value');
                expect(arg2).toBe(42);
                return action('argsAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'argsHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'argsHandler',
                a: [{ t: 'e' }, { t: 's', v: 'static-value' }, { t: 's', v: 42 }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
        });

        it('should pass $event at any position in arguments', () => {
            const mockHandler = vi.fn((_scopingEvent: Event, arg1: string, event: Event, arg2: number) => {
                expect(_scopingEvent).toBeInstanceOf(Event);
                expect(arg1).toBe('first');
                expect(event).toBeInstanceOf(Event);
                expect(arg2).toBe(123);
                return action('middleEventAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'middleEventHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'middleEventHandler',
                a: [{ t: 's', v: 'first' }, { t: 'e' }, { t: 's', v: 123 }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
        });

        it('should handle errors in handler gracefully', () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const mockHandler = vi.fn(() => {
                throw new Error('Handler error');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'errorHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'errorHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);

            expect(() => button.click()).not.toThrow();
            expect(consoleSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in page handler'),
                expect.any(Error)
            );
            expect(handleActionSpy).not.toHaveBeenCalled();

            consoleSpy.mockRestore();
        });
    });

    describe('$form injection via resolveArgsWithEvent', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
        });

        it('should inject FormDataHandle when $form is used inside a form', () => {
            const mockHandler = vi.fn((_event: Event, formHandle: unknown) => {
                expect(formHandle).toBeDefined();
                expect(typeof (formHandle as { toObject: () => unknown }).toObject).toBe('function');
                return action('formAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'formHandler' ? mockHandler : undefined
            });

            const form = document.createElement('form');
            const input = document.createElement('input');
            input.name = 'username';
            input.value = 'testuser';
            form.appendChild(input);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'formHandler',
                a: [{ t: 'f' }]
            }));
            button.setAttribute('p-on:click', payload);
            form.appendChild(button);
            root.appendChild(form);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
        });

        it('should return null when $form is used but no form ancestor exists', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const mockHandler = vi.fn((_event: Event, formVal: unknown) => {
                expect(formVal).toBeNull();
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'noFormHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'noFormHandler',
                a: [{ t: 'f' }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('$form used but no form ancestor found'),
                expect.any(HTMLElement)
            );
            warnSpy.mockRestore();
        });

        it('should inject $event and $form together at correct positions', () => {
            const mockHandler = vi.fn((_scopeEvent: Event, eventArg: Event, formArg: unknown, staticArg: string) => {
                expect(eventArg).toBeInstanceOf(Event);
                expect(formArg).toBeDefined();
                expect(staticArg).toBe('extra');
                return action('combinedAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'combinedHandler' ? mockHandler : undefined
            });

            const form = document.createElement('form');
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'combinedHandler',
                a: [{ t: 'e' }, { t: 'f' }, { t: 's', v: 'extra' }]
            }));
            button.setAttribute('p-on:click', payload);
            form.appendChild(button);
            root.appendChild(form);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalled();
        });
    });

    describe('$form injection via resolveArgsForAction (plain object)', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;
        let getActionFunctionSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
            getActionFunctionSpy = vi.spyOn(ActionModule, 'getActionFunction');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
            getActionFunctionSpy.mockRestore();
        });

        it('should convert $form to plain object via toObject() for registered action functions', () => {
            const mockActionFn = vi.fn((formObj: unknown) => {
                expect(typeof formObj).toBe('object');
                expect(formObj).not.toBeNull();
                return action('registeredAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: () => undefined,
                hasFunction: () => false,
                getExportedFunctions: () => [],
                getScopedFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(mockActionFn);

            const form = document.createElement('form');
            const input = document.createElement('input');
            input.name = 'email';
            input.value = 'test@example.com';
            form.appendChild(input);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'email.Contact',
                a: [{ t: 'f' }]
            }));
            button.setAttribute('p-on:click', payload);
            form.appendChild(button);
            root.appendChild(form);

            domBinder.bindActions(root);
            button.click();

            expect(mockActionFn).toHaveBeenCalled();
        });

        it('should return empty object when $form used in action function but no form ancestor', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const mockActionFn = vi.fn((formObj: unknown) => {
                expect(formObj).toEqual({});
                return action('noFormAction');
            });
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: () => undefined,
                hasFunction: () => false,
                getExportedFunctions: () => [],
                getScopedFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(mockActionFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'email.Send',
                a: [{ t: 'f' }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockActionFn).toHaveBeenCalled();
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('$form used but no form ancestor found'),
                expect.any(HTMLElement)
            );
            warnSpy.mockRestore();
        });
    });

    describe('helper execution', () => {
        it('should convert $event arg to event.type string for helpers', () => {
            const helperFn = vi.fn();
            helperRegistry.register('eventTypeHelper', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'eventTypeHelper',
                a: [{ t: 'e' }, { t: 's', v: 'other' }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledWith(button, expect.any(Event), 'click', 'other');
        });

        it('should convert $form arg to JSON string for helpers', () => {
            const helperFn = vi.fn();
            helperRegistry.register('formJsonHelper', helperFn);

            const form = document.createElement('form');
            const input = document.createElement('input');
            input.name = 'field1';
            input.value = 'value1';
            form.appendChild(input);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'formJsonHelper',
                a: [{ t: 'f' }]
            }));
            button.setAttribute('p-on:click', payload);
            form.appendChild(button);
            root.appendChild(form);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledWith(button, expect.any(Event), expect.any(String));
            const jsonArg = helperFn.mock.calls[0][2] as string;
            const parsed = JSON.parse(jsonArg) as Record<string, unknown>;
            expect(parsed.field1).toBe('value1');
        });

        it('should return empty string for $form in helper when no form ancestor', () => {
            const helperFn = vi.fn();
            helperRegistry.register('noFormHelper', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'noFormHelper',
                a: [{ t: 'f' }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledWith(button, expect.any(Event), '');
        });

        it('should handle async helper that rejects gracefully', async () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const asyncHelper = vi.fn().mockRejectedValue(new Error('async fail'));
            helperRegistry.register('asyncFailHelper', asyncHelper as never);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'asyncFailHelper',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            await vi.waitFor(() => {
                expect(errorSpy).toHaveBeenCalledWith(
                    expect.stringContaining('Async helper execution failed'),
                    expect.any(Error)
                );
            });
            errorSpy.mockRestore();
        });

        it('should handle helper that throws synchronously', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const throwingHelper = vi.fn(() => { throw new Error('sync fail'); });
            helperRegistry.register('throwHelper', throwingHelper as never);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'throwHelper',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            expect(() => button.click()).not.toThrow();
            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Helper execution failed'),
                expect.any(Error)
            );
            errorSpy.mockRestore();
        });

        it('should convert static arg values to strings for helpers', () => {
            const helperFn = vi.fn();
            helperRegistry.register('stringifyHelper', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'stringifyHelper',
                a: [{ t: 's', v: 42 }, { t: 's', v: true }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledWith(button, expect.any(Event), '42', 'true');
        });
    });

    describe('dispatchIfActionDescriptor', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
        });

        it('should dispatch ActionDescriptor from a Promise-returning handler', async () => {
            const mockHandler = vi.fn(() => Promise.resolve(action('asyncAction', 'data')));
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'asyncHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'asyncHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            await vi.waitFor(() => {
                expect(handleActionSpy).toHaveBeenCalledWith(
                    expect.objectContaining({ action: 'asyncAction', args: ['data'] }),
                    button,
                    expect.any(Event)
                );
            });
        });

        it('should log error when sync action handleAction rejects', async () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            handleActionSpy.mockRejectedValue(new Error('action failed'));
            const mockHandler = vi.fn(() => action('failingAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'failHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'failHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            await vi.waitFor(() => {
                expect(errorSpy).toHaveBeenCalledWith(
                    expect.stringContaining('Action execution failed'),
                    expect.any(Error)
                );
            });
            errorSpy.mockRestore();
        });

        it('should log error when async action promise rejects', async () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const mockHandler = vi.fn(() => Promise.reject(new Error('promise rejected')));
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'rejectHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'rejectHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            await vi.waitFor(() => {
                expect(errorSpy).toHaveBeenCalledWith(
                    expect.stringContaining('Async action execution failed'),
                    expect.any(Error)
                );
            });
            errorSpy.mockRestore();
        });

        it('should not dispatch when promise resolves to non-ActionDescriptor', async () => {
            const mockHandler = vi.fn(() => Promise.resolve('just a string'));
            getGlobalPageContextSpy.mockReturnValue({
                getFunction: (name: string) => name === 'stringPromiseHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'stringPromiseHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            await new Promise(resolve => setTimeout(resolve, 10));
            expect(handleActionSpy).not.toHaveBeenCalled();
        });
    });

    describe('handleExplicitPartialCall', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
        });

        it('should broadcast @partial-name.fn() to all instances of the partial', () => {
            const fn1 = vi.fn(() => action('partialAction'));
            const fn2 = vi.fn(() => action('partialAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getFunctionsByPartialName: (partialName: string, fnName: string) => {
                    if (partialName === 'myPartial' && fnName === 'doStuff') {
                        return [fn1, fn2];
                    }
                    return [];
                },
                getRegisteredPartialNames: () => ['myPartial']
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: '@myPartial.doStuff',
                a: [{ t: 's', v: 'arg1' }]
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(fn1).toHaveBeenCalledWith(expect.any(Event), 'arg1');
            expect(fn2).toHaveBeenCalledWith(expect.any(Event), 'arg1');
        });

        it('should warn with suggestion when partial is not found', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                getFunctionsByPartialName: () => [],
                getRegisteredPartialNames: () => ['myPartial', 'otherPartial']
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: '@myPartail.doStuff',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Partial "myPartail" not found')
            );
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Did you mean "@myPartial"')
            );
            warnSpy.mockRestore();
        });

        it('should warn without suggestion when no partial names are close enough', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                getFunctionsByPartialName: () => [],
                getRegisteredPartialNames: () => ['completelyDifferent']
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: '@xyz.fn',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Partial "xyz" not found')
            );
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Registered partials: completelyDifferent')
            );
            warnSpy.mockRestore();
        });

        it('should handle error in partial function gracefully', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const throwingFn = vi.fn(() => { throw new Error('partial error'); });
            getGlobalPageContextSpy.mockReturnValue({
                getFunctionsByPartialName: () => [throwingFn],
                getRegisteredPartialNames: () => ['myPartial']
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: '@myPartial.doStuff',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            expect(() => button.click()).not.toThrow();
            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in @myPartial.doStuff'),
                expect.any(Error)
            );
            errorSpy.mockRestore();
        });
    });

    describe('handleImplicitScopeCall', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;
        let getActionFunctionSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
            getActionFunctionSpy = vi.spyOn(ActionModule, 'getActionFunction');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
            getActionFunctionSpy.mockRestore();
        });

        it('should prefer scoped function when element is inside a partial', () => {
            const scopedFn = vi.fn(() => action('scopedAction'));
            const globalFn = vi.fn(() => action('globalAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: (name: string, partialId: string) => {
                    if (name === 'myFn' && partialId === 'partial-123') {
                        return scopedFn;
                    }
                    return undefined;
                },
                getFunction: (name: string) => name === 'myFn' ? globalFn : undefined
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'partial-123');
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myFn', a: [] }));
            button.setAttribute('p-on:click', payload);
            partial.appendChild(button);
            root.appendChild(partial);

            domBinder.bindActions(root);
            button.click();

            expect(scopedFn).toHaveBeenCalled();
            expect(globalFn).not.toHaveBeenCalled();
        });

        it('should fall back to global function when scoped function is not found', () => {
            const globalFn = vi.fn(() => action('globalAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: (name: string) => name === 'globalFn' ? globalFn : undefined
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'partial-456');
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'globalFn', a: [] }));
            button.setAttribute('p-on:click', payload);
            partial.appendChild(button);
            root.appendChild(partial);

            domBinder.bindActions(root);
            button.click();

            expect(globalFn).toHaveBeenCalled();
        });

        it('should fall back to helper registry when neither scoped nor global function found', () => {
            const helperFn = vi.fn();
            helperRegistry.register('fallbackHelper', helperFn);
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'fallbackHelper', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledWith(button, expect.any(Event));
        });

        it('should fall back to action function registry when page context and helpers miss', () => {
            const mockActionFn = vi.fn(() => action('registeredAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(mockActionFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'email.Contact', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockActionFn).toHaveBeenCalled();
            expect(handleActionSpy).toHaveBeenCalled();
        });

        it('should warn with suggestion when function not found in any registry', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined,
                getExportedFunctions: () => ['handleClick', 'handleSubmit']
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'handleClck', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Function "handleClck" not found')
            );
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Did you mean "handleClick"')
            );
            warnSpy.mockRestore();
        });

        it('should include partial scope search note when element is inside a partial', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined,
                getExportedFunctions: () => []
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'scope-789');
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'missing', a: [] }));
            button.setAttribute('p-on:click', payload);
            partial.appendChild(button);
            root.appendChild(partial);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('searched partial scope and global')
            );
            warnSpy.mockRestore();
        });
    });

    describe('handleCustomEventNoModifier', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;
        let getActionFunctionSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
            getActionFunctionSpy = vi.spyOn(ActionModule, 'getActionFunction');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
            getActionFunctionSpy.mockRestore();
        });

        it('should use page context function for p-event when available', () => {
            const mockFn = vi.fn(() => action('pageAction'));
            getGlobalPageContextSpy.mockReturnValue({
                hasFunction: (name: string) => name === 'onCustom',
                getFunction: (name: string) => name === 'onCustom' ? mockFn : undefined,
                getExportedFunctions: () => ['onCustom']
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'onCustom', a: [] }));
            div.setAttribute('p-event:mycustom', payload);
            root.appendChild(div);

            domBinder.bindActions(root);
            div.dispatchEvent(new CustomEvent('mycustom'));

            expect(mockFn).toHaveBeenCalled();
        });

        it('should fall back to helper for p-event when page context has no match', () => {
            const helperFn = vi.fn();
            helperRegistry.register('customHelper', helperFn);
            getGlobalPageContextSpy.mockReturnValue({
                hasFunction: () => false,
                getFunction: () => undefined,
                getExportedFunctions: () => []
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'customHelper', a: [] }));
            div.setAttribute('p-event:mycustom', payload);
            root.appendChild(div);

            domBinder.bindActions(root);
            div.dispatchEvent(new CustomEvent('mycustom'));

            expect(helperFn).toHaveBeenCalled();
        });

        it('should fall back to action function registry for p-event', () => {
            const mockActionFn = vi.fn(() => action('registeredEventAction'));
            getGlobalPageContextSpy.mockReturnValue({
                hasFunction: () => false,
                getFunction: () => undefined,
                getExportedFunctions: () => []
            });
            getActionFunctionSpy.mockReturnValue(mockActionFn);

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'email.Send', a: [] }));
            div.setAttribute('p-event:mycustom', payload);
            root.appendChild(div);

            domBinder.bindActions(root);
            div.dispatchEvent(new CustomEvent('mycustom'));

            expect(mockActionFn).toHaveBeenCalled();
            expect(handleActionSpy).toHaveBeenCalled();
        });

        it('should warn with suggestion when p-event function not found', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                hasFunction: () => false,
                getFunction: () => undefined,
                getExportedFunctions: () => ['onCustomEvent']
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'onCustomEvnt', a: [] }));
            div.setAttribute('p-event:mycustom', payload);
            root.appendChild(div);

            domBinder.bindActions(root);
            div.dispatchEvent(new CustomEvent('mycustom'));

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Did you mean "onCustomEvent"')
            );
            warnSpy.mockRestore();
        });
    });

    describe('parseFunctionReference (via bindActions)', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
        });

        it('should parse @partial.fn as explicit partial call', () => {
            const partialFn = vi.fn(() => action('partialAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getFunctionsByPartialName: (partialName: string, fnName: string) => {
                    if (partialName === 'sidebar' && fnName === 'toggle') {
                        return [partialFn];
                    }
                    return [];
                },
                getRegisteredPartialNames: () => ['sidebar']
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: '@sidebar.toggle',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(partialFn).toHaveBeenCalled();
        });

        it('should treat plain function name as implicit scope call', () => {
            const globalFn = vi.fn(() => action('plainAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: (name: string) => name === 'handleClick' ? globalFn : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'handleClick',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(globalFn).toHaveBeenCalled();
        });

        it('should treat @ without dot as plain function name', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined,
                getExportedFunctions: () => []
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: '@nodot',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Function "@nodot" not found')
            );
            warnSpy.mockRestore();
        });
    });

    describe('empty event name in createActionHandler', () => {
        it('should not bind when event name is empty after splitting', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myFn', a: [] }));
            button.setAttribute('p-on:', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            expect(warnSpy).not.toHaveBeenCalled();
            warnSpy.mockRestore();
        });
    });

    describe('data-pk-action-method attribute', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;
        let getActionFunctionSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
            getActionFunctionSpy = vi.spyOn(ActionModule, 'getActionFunction');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
            getActionFunctionSpy.mockRestore();
        });

        it('should default to POST when data-pk-action-method is not set', () => {
            const mockActionFn = vi.fn(() => action('testAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(mockActionFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'testAction', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockActionFn).toHaveBeenCalled();
        });
    });

    describe('payload with no args field', () => {
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            getGlobalPageContextSpy.mockRestore();
        });

        it('should handle payload where a field is undefined', () => {
            const mockHandler = vi.fn();
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: (name: string) => name === 'noArgsHandler' ? mockHandler : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'noArgsHandler' }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(mockHandler).toHaveBeenCalledWith(expect.any(Event));
        });
    });

    describe('p-modal:confirm_action attribute', () => {
        it('should include confirmAction in modal options', () => {
            const button = document.createElement('button');
            button.setAttribute('p-modal:selector', '#delete-modal');
            button.setAttribute('p-modal:confirm_action', 'deleteItem');
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(callbacks.onOpenModal).toHaveBeenCalledWith(
                expect.objectContaining({
                    selector: '#delete-modal',
                    confirmAction: 'deleteItem'
                })
            );
        });
    });

    describe('multiple handlers on same event', () => {
        it('should invoke multiple p-on handlers for the same element and event', () => {
            const helperFn = vi.fn();
            helperRegistry.register('multiHelper', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'multiHelper', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(helperFn).toHaveBeenCalledTimes(1);
        });
    });

    describe('event handler wrapping error catching', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
        });

        it('should catch and log errors from event handler dispatch', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            const throwingFn = vi.fn((): never => { throw new Error('handler boom'); });
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: (name: string) => name === 'boomHandler' ? throwingFn : undefined
            });

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'boomHandler', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            expect(() => button.click()).not.toThrow();

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in page handler'),
                expect.any(Error)
            );
            errorSpy.mockRestore();
        });
    });

    describe('executeRegisteredAction error handling', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;
        let getActionFunctionSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
            getActionFunctionSpy = vi.spyOn(ActionModule, 'getActionFunction');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
            getActionFunctionSpy.mockRestore();
        });

        it('should catch and log errors from registered action functions', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const throwingAction = vi.fn(() => { throw new Error('action boom'); });
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(throwingAction);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'broken.Action', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            expect(() => button.click()).not.toThrow();

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in action handler'),
                expect.any(Error)
            );
            errorSpy.mockRestore();
        });
    });

    describe('findPartialScope traversal', () => {
        let handleActionSpy: ReturnType<typeof vi.spyOn>;
        let getGlobalPageContextSpy: ReturnType<typeof vi.spyOn>;
        let getActionFunctionSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);
            getGlobalPageContextSpy = vi.spyOn(PageContext, 'getGlobalPageContext');
            getActionFunctionSpy = vi.spyOn(ActionModule, 'getActionFunction');
        });

        afterEach(() => {
            handleActionSpy.mockRestore();
            getGlobalPageContextSpy.mockRestore();
            getActionFunctionSpy.mockRestore();
        });

        it('should find partial scope on a grandparent element', () => {
            const scopedFn = vi.fn(() => action('deepAction'));
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: (name: string, partialId: string) => {
                    if (name === 'deepFn' && partialId === 'deep-partial') {
                        return scopedFn;
                    }
                    return undefined;
                },
                getFunction: () => undefined
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const grandparent = document.createElement('div');
            grandparent.setAttribute('partial', 'deep-partial');
            const parent = document.createElement('div');
            grandparent.appendChild(parent);
            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'deepFn', a: [] }));
            button.setAttribute('p-on:click', payload);
            parent.appendChild(button);
            root.appendChild(grandparent);

            domBinder.bindActions(root);
            button.click();

            expect(scopedFn).toHaveBeenCalled();
        });

        it('should return undefined partialId when no partial ancestor exists', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            getGlobalPageContextSpy.mockReturnValue({
                getScopedFunction: () => undefined,
                getFunction: () => undefined,
                getExportedFunctions: () => []
            });
            getActionFunctionSpy.mockReturnValue(undefined);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'orphanFn', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Function "orphanFn" not found')
            );
            const warnMessage = warnSpy.mock.calls[0][0] as string;
            expect(warnMessage).not.toContain('searched partial scope');
            warnSpy.mockRestore();
        });
    });

    describe('.passive and .capture event modifiers', () => {
        it('should pass { passive: true } to addEventListener when .passive modifier is used', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.passive', payload);
            root.appendChild(button);

            const addEventListenerSpy = vi.spyOn(button, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'click',
                expect.any(Function),
                expect.objectContaining({ passive: true })
            );
            const options = addEventListenerSpy.mock.calls[0][2] as AddEventListenerOptions;
            expect(options.capture).toBeUndefined();

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });

        it('should pass { capture: true } to addEventListener when .capture modifier is used', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.capture', payload);
            root.appendChild(button);

            const addEventListenerSpy = vi.spyOn(button, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'click',
                expect.any(Function),
                expect.objectContaining({ capture: true })
            );
            const options = addEventListenerSpy.mock.calls[0][2] as AddEventListenerOptions;
            expect(options.passive).toBeUndefined();

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });

        it('should pass { passive: true, capture: true } when both .passive.capture modifiers are used', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.passive.capture', payload);
            root.appendChild(button);

            const addEventListenerSpy = vi.spyOn(button, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'click',
                expect.any(Function),
                expect.objectContaining({ passive: true, capture: true })
            );

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });

        it('should not pass listener options when no .passive or .capture modifier is present', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            const addEventListenerSpy = vi.spyOn(button, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'click',
                expect.any(Function),
                undefined
            );

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });

        it('should fire handler only once AND register as passive when .passive.once modifiers are used', () => {
            const helperFn = vi.fn();
            helperRegistry.register('passiveOnceHelper', helperFn);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'passiveOnceHelper', a: [] }));
            button.setAttribute('p-on:click.passive.once', payload);
            root.appendChild(button);

            const addEventListenerSpy = vi.spyOn(button, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'click',
                expect.any(Function),
                expect.objectContaining({ passive: true })
            );

            button.click();
            button.click();
            button.click();

            expect(helperFn).toHaveBeenCalledTimes(1);

            addEventListenerSpy.mockRestore();
        });

        it('should apply .capture modifier with p-event bindings', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'customAction', a: [] }));
            div.setAttribute('p-event:mycustom.capture', payload);
            root.appendChild(div);

            const addEventListenerSpy = vi.spyOn(div, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'mycustom',
                expect.any(Function),
                expect.objectContaining({ capture: true })
            );

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });

        it('should apply .passive modifier with p-event bindings', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const div = document.createElement('div');
            const payload = btoa(JSON.stringify({ f: 'customAction', a: [] }));
            div.setAttribute('p-event:mycustom.passive', payload);
            root.appendChild(div);

            const addEventListenerSpy = vi.spyOn(div, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'mycustom',
                expect.any(Function),
                expect.objectContaining({ passive: true })
            );

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });

        it('should combine .passive with .prevent and .stop modifiers', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({ f: 'myAction', a: [] }));
            button.setAttribute('p-on:click.passive.prevent.stop', payload);
            root.appendChild(button);

            const addEventListenerSpy = vi.spyOn(button, 'addEventListener');
            domBinder.bindActions(root);

            expect(addEventListenerSpy).toHaveBeenCalledWith(
                'click',
                expect.any(Function),
                expect.objectContaining({ passive: true })
            );

            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
            const preventDefaultSpy = vi.spyOn(clickEvent, 'preventDefault');
            const stopPropagationSpy = vi.spyOn(clickEvent, 'stopPropagation');
            button.dispatchEvent(clickEvent);

            expect(preventDefaultSpy).toHaveBeenCalled();
            expect(stopPropagationSpy).toHaveBeenCalled();

            addEventListenerSpy.mockRestore();
            warnSpy.mockRestore();
        });
    });

    describe('helper execution with async return value', () => {
        it('should handle helper returning a resolved promise', async () => {
            const asyncHelper = vi.fn().mockResolvedValue(undefined);
            helperRegistry.register('asyncOkHelper', asyncHelper as never);

            const button = document.createElement('button');
            const payload = btoa(JSON.stringify({
                f: 'asyncOkHelper',
                a: []
            }));
            button.setAttribute('p-on:click', payload);
            root.appendChild(button);

            domBinder.bindActions(root);
            button.click();

            expect(asyncHelper).toHaveBeenCalled();
        });
    });
});
