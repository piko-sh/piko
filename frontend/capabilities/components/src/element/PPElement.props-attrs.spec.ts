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

import { describe, it, expect, beforeEach, afterEach, vi, MockInstance } from 'vitest';
import { PPElement, PropTypeDefinition } from '@/element';
import { dom } from '@/vdom';
import type { VirtualNode } from '@/vdom';
import { makeReactive } from '@/reactivity';

vi.mock('@/core/PPFramework', () => ({
  PPFramework: {
    navigateTo: vi.fn(),
  },
}));

interface TestPropsState {
  stringProp?: string;
  numberProp?: number;
  booleanProp?: boolean;
  reflectedString?: string;
  reflectedNumber?: number;
  reflectedBoolean?: boolean;
  jsonProp?: { data: string } | null;
  nonReflectedProp?: string;
  propWithDefault?: string;
}

type ApplyHtmlAttributeToStateFn = PPElement['applyHtmlAttributeToState'];
type ReflectStatePropertyToAttributeFn = PPElement['reflectStatePropertyToAttribute'];
type TranslateAttributeValueFn = PPElement['translateAttributeValue'];
type ScheduleRenderFn = () => void;


class PropsAttrsTestElement extends PPElement {
  applyHtmlAttributeToStateSpy!: MockInstance<ApplyHtmlAttributeToStateFn>;
  reflectStatePropertyToAttributeSpy!: MockInstance<ReflectStatePropertyToAttributeFn>;
  translateAttributeValueSpy!: MockInstance<TranslateAttributeValueFn>;
  scheduleRenderSpy!: MockInstance<ScheduleRenderFn>;


  static override get propTypes(): Record<string, PropTypeDefinition> | undefined {
    return {
      stringProp: { type: 'string' },
      numberProp: { type: 'number' },
      booleanProp: { type: 'boolean' },
      reflectedString: { type: 'string', reflectToAttribute: true },
      reflectedNumber: { type: 'number', reflectToAttribute: true },
      reflectedBoolean: { type: 'boolean', reflectToAttribute: true },
      jsonProp: { type: 'json', reflectToAttribute: true },
      nonReflectedProp: { type: 'string', reflectToAttribute: false },
      propWithDefault: { type: 'string', default: 'default value' },
    };
  }

  constructor() {
    super();
  }

  public testSetup(initialState?: Partial<TestPropsState>) {
    const constructor = this.constructor as typeof PropsAttrsTestElement;
    const propTypes = constructor.propTypes || {};
    const programmaticDefaults = {} as Partial<TestPropsState>;

    for (const key of Object.keys(propTypes) as Array<keyof TestPropsState>) {
      const propDef = propTypes[key];
      if (propDef && propDef.default !== undefined) {
        programmaticDefaults[key] = typeof propDef.default === 'function'
          ? (propDef.default as () => any)()
          : propDef.default;
      }
    }

    const baseInitialState: TestPropsState = {
      stringProp: undefined,
      numberProp: undefined,
      booleanProp: undefined,
      reflectedString: 'initial reflected',
      reflectedNumber: 100,
      reflectedBoolean: true,
      jsonProp: { data: 'initial json' },
      nonReflectedProp: 'not reflected initial',
      propWithDefault: 'default value',
      ...programmaticDefaults,
      ...initialState,
    };

    super.init({
      state: makeReactive(baseInitialState, this as any) as Record<string, unknown>,
      $$initialState: { ...baseInitialState } as Record<string, unknown>,
    });

    this.applyHtmlAttributeToStateSpy = vi.spyOn(this as any, 'applyHtmlAttributeToState');
    this.reflectStatePropertyToAttributeSpy = vi.spyOn(this as any, 'reflectStatePropertyToAttribute');
    this.translateAttributeValueSpy = vi.spyOn(this as any, 'translateAttributeValue');
    this.scheduleRenderSpy = vi.spyOn(this as any, 'scheduleRender');
  }

  public testTranslateAttributeValue(typeHint: string, attributeValue: string | null, propertyName: string): any {
    return this['translateAttributeValue'](typeHint, attributeValue, propertyName);
  }
  public testApplyHtmlAttributeToState(propertyName: string, attributeValue: string | null) {
    this['applyHtmlAttributeToState'](propertyName, attributeValue);
  }
  public testReflectStatePropertyToAttribute(propertyName: string, propertyValue: any) {
    this['reflectStatePropertyToAttribute'](propertyName, propertyValue);
  }

  override renderVDOM(): VirtualNode {
    return dom.el('div', 'props-test-root', {
      'data-stringprop': this.state?.stringProp,
      'data-reflectedstring': this.state?.reflectedString
    });
  }

  override scheduleRender(): void {
    super.scheduleRender();
  }
}
customElements.define('props-attrs-test-element', PropsAttrsTestElement);


describe('PPElement Props and Attributes', () => {
  let element: PropsAttrsTestElement;
  let host: HTMLElement;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    element = document.createElement('props-attrs-test-element') as PropsAttrsTestElement;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    if (element.parentNode) element.parentNode.removeChild(element);
    if (host.parentNode) host.parentNode.removeChild(host);
  });

  describe('static get observedAttributes()', () => {
    it('should correctly derive observedAttributes from propTypes with reflection', () => {
      const observed = PropsAttrsTestElement.observedAttributes;
      expect(observed).toContain('reflected-string');
      expect(observed).toContain('reflected-number');
      expect(observed).toContain('reflected-boolean');
      expect(observed).toContain('json-prop');

      expect(observed).toContain('string-prop');
      expect(observed).toContain('number-prop');
      expect(observed).toContain('boolean-prop');
      expect(observed).toContain('prop-with-default');

      expect(observed).not.toContain('non-reflected-prop');
    });
  });

  describe('translateAttributeValue()', () => {
    beforeEach(() => {
      element.testSetup();
    });

    it('should translate string attributes correctly', () => {
      expect(element.testTranslateAttributeValue('string', 'hello', 'stringProp')).toBe('hello');
      expect(element.testTranslateAttributeValue('string', '', 'stringProp')).toBe('');
    });

    it('should translate number attributes correctly', () => {
      expect(element.testTranslateAttributeValue('number', '123', 'numberProp')).toBe(123);
      expect(element.testTranslateAttributeValue('number', '123.45', 'numberProp')).toBe(123.45);
      const numDefault = (PropsAttrsTestElement.propTypes!['numberProp'] as PropTypeDefinition)?.default ?? 0;
      expect(element.testTranslateAttributeValue('number', 'invalid', 'numberProp')).toBe(numDefault);
    });

    it('should translate boolean attributes ("true", "false", empty string, absence)', () => {
      expect(element.testTranslateAttributeValue('boolean', 'true', 'booleanProp')).toBe(true);
      expect(element.testTranslateAttributeValue('boolean', '', 'booleanProp')).toBe(true);
      expect(element.testTranslateAttributeValue('boolean', 'false', 'booleanProp')).toBe(false);
      const boolDefault = (PropsAttrsTestElement.propTypes!['booleanProp'] as PropTypeDefinition)?.default ?? false;
      expect(element.testTranslateAttributeValue('boolean', null, 'booleanProp')).toBe(boolDefault);
    });

    it('should translate JSON attributes correctly', () => {
      const jsonObj = { data: 'value' };
      expect(element.testTranslateAttributeValue('json', JSON.stringify(jsonObj), 'jsonProp')).toEqual(jsonObj);
      const jsonDefault = (PropsAttrsTestElement.propTypes!['jsonProp'] as PropTypeDefinition)?.default ?? null;
      expect(element.testTranslateAttributeValue('json', 'not json', 'jsonProp')).toEqual(jsonDefault);
    });

    it('should return prop default value for null attribute if defined', () => {
      expect(element.testTranslateAttributeValue('string', null, 'propWithDefault')).toBe('default value');
    });
  });

  describe('attributeChangedCallback() & applyHtmlAttributeToState()', () => {
    beforeEach(() => {
      element.testSetup({ reflectedString: 'initial' });
      host.appendChild(element);
    });

    it('attributeChangedCallback should call applyHtmlAttributeToState for observed reflected attributes', () => {
      element.setAttribute('reflected-string', 'new value');
      expect(element.applyHtmlAttributeToStateSpy).toHaveBeenCalledWith('reflectedString', 'new value');
    });

    it('applyHtmlAttributeToState should update state and trigger scheduleRender if value changed', () => {
      element.scheduleRenderSpy.mockClear();
      element.testApplyHtmlAttributeToState('reflectedString', 'from apply');
      expect(element.state!.reflectedString).toBe('from apply');
      expect(element.scheduleRenderSpy).toHaveBeenCalled();
    });

    it('applyHtmlAttributeToState should not trigger scheduleRender if value is the same', () => {
      element.state!.reflectedString = 'same value';
      element.scheduleRenderSpy.mockClear();
      element.testApplyHtmlAttributeToState('reflectedString', 'same value');
      expect(element.scheduleRenderSpy).not.toHaveBeenCalled();
    });
  });

  describe('reflectStatePropertyToAttribute()', () => {
    beforeEach(() => {
      element.testSetup();
      host.appendChild(element);
    });

    it('should set string attribute when reflectedString state changes', () => {
      element.testReflectStatePropertyToAttribute('reflectedString', 'reflected value');
      expect(element.getAttribute('reflected-string')).toBe('reflected value');
    });

    it('should set number attribute (as string) when reflectedNumber state changes', () => {
      element.testReflectStatePropertyToAttribute('reflectedNumber', 789);
      expect(element.getAttribute('reflected-number')).toBe('789');
    });

    it('should add boolean attribute when true and remove when false', () => {
      element.testReflectStatePropertyToAttribute('reflectedBoolean', true);
      expect(element.hasAttribute('reflected-boolean')).toBe(true);

      element.testReflectStatePropertyToAttribute('reflectedBoolean', false);
      expect(element.hasAttribute('reflected-boolean')).toBe(false);
    });

    it('should set JSON attribute (as string) when jsonProp state changes', () => {
      const newJson = { data: "new json" };
      element.testReflectStatePropertyToAttribute('jsonProp', newJson);
      expect(element.getAttribute('json-prop')).toBe(JSON.stringify(newJson));
    });

    it('should remove attribute if reflected state property becomes null/undefined', () => {
      element.setAttribute('reflected-string', 'has value');
      element.testReflectStatePropertyToAttribute('reflectedString', undefined);
      expect(element.hasAttribute('reflected-string')).toBe(false);

      element.setAttribute('reflected-string', 'has value again');
      element.testReflectStatePropertyToAttribute('reflectedString', null);
      expect(element.hasAttribute('reflected-string')).toBe(false);
    });

    it('should not set attribute for non-reflected props', () => {
      element.testReflectStatePropertyToAttribute('nonReflectedProp', 'should not reflect');
      expect(element.hasAttribute('non-reflected-prop')).toBe(false);
    });
  });

  describe('End-to-End Syncing', () => {
    beforeEach(async () => {
      element.testSetup({ reflectedBoolean: false });
      host.appendChild(element);
      await new Promise(r => requestAnimationFrame(()=> setTimeout(r,0)));
      element.scheduleRenderSpy.mockClear();
    });

    it('setting a reflected attribute should update the state property', () => {
      element.setAttribute('reflected-boolean', 'true');
      expect(element.state!.reflectedBoolean).toBe(true);
    });

    it('setting a reflected state property should update the attribute after render', async () => {
      element.state!.reflectedBoolean = true;

      await new Promise(resolve => requestAnimationFrame(resolve));
      await new Promise(resolve => setTimeout(resolve, 0));

      expect(element.hasAttribute('reflected-boolean')).toBe(true);

      element.state!.reflectedBoolean = false;
      await new Promise(resolve => requestAnimationFrame(resolve));
      await new Promise(resolve => setTimeout(resolve, 0));
      expect(element.hasAttribute('reflected-boolean')).toBe(false);
    });

    it('default values should be applied if no attribute, and overridden by attributes during init', () => {
      const elWithDefault = document.createElement('props-attrs-test-element') as PropsAttrsTestElement;
      elWithDefault.setAttribute('prop-with-default', 'attr override during init');
      elWithDefault.testSetup({});

      expect(elWithDefault.state!.propWithDefault).toBe('attr override during init');

      const elWithDefaultOnly = document.createElement('props-attrs-test-element') as PropsAttrsTestElement;
      elWithDefaultOnly.testSetup({});
      expect(elWithDefaultOnly.state!.propWithDefault).toBe('default value');
    });
  });
});
