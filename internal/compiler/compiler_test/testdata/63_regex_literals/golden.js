/*
--- BEGIN AST DUMP ---

<div>
  <p class="result" [p-text: state.validationResult] />
  <button class="test-integer" [Events: p-on:click.="testInteger" {P: testInteger}]>
    "Test Integer"
  </button>
  <button class="test-double" [Events: p-on:click.="testDouble" {P: testDouble}]>
    "Test Double"
  </button>
  <button class="test-email" [Events: p-on:click.="testEmail" {P: testEmail}]>
    "Test Email"
  </button>
</div>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"validationResult": "Ready"};
        const state = makeReactive($$initialState, contextParam);
        const simplePattern = /^[a-z]+$/;
        const caseInsensitive = /hello/gi;
        const ALLOWED_CHARS_REGEX = {"integer": /^[0-9]+$/, "double": /^[\d.]+$/, "number": /^-?[\d.]+$/};
        const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        function testInteger() {
            const testValue = "12345";
            if (ALLOWED_CHARS_REGEX.integer.test(testValue)) {
                state.validationResult = "Integer: Valid";
            } else {
                state.validationResult = "Integer: Invalid";
            }
        }
        function testDouble() {
            const testValue = "123.45";
            if (ALLOWED_CHARS_REGEX.double.test(testValue)) {
                state.validationResult = "Double: Valid";
            } else {
                state.validationResult = "Double: Invalid";
            }
        }
        function testEmail() {
            const testValue = "test@example.com";
            if (emailPattern.test(testValue)) {
                state.validationResult = "Email: Valid";
            } else {
                state.validationResult = "Email: Invalid";
            }
        }
        return {"state": state, "$$initialState": $$initialState, "testInteger": testInteger, "testDouble": testDouble, "testEmail": testEmail};
    }
    class RegexLiteralsTestElement extends PPElement {
        constructor () {
            super();
            this._dir_click_testInteger_evt_1 = (e) => {
                this.$$ctx.testInteger.call(this, e);
            };
            this._dir_click_testDouble_evt_2 = (e) => {
                this.$$ctx.testDouble.call(this, e);
            };
            this._dir_click_testEmail_evt_3 = (e) => {
                this.$$ctx.testEmail.call(this, e);
            };
        }
        static get propTypes () {
            return {"validationResult": {"type": "string"}};
        }
        static get defaultProps () {
            return {"validationResult": "Ready"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"class": "result"}, [dom.txt(String(this.$$ctx.state.validationResult), "r.0:0:txt")]), dom.el("button", "r.0:1", {"class": "test-integer", "onClick": this._dir_click_testInteger_evt_1}, dom.txt("Test Integer", "r.0:1:0")), dom.el("button", "r.0:2", {"class": "test-double", "onClick": this._dir_click_testDouble_evt_2}, dom.txt("Test Double", "r.0:2:0")), dom.el("button", "r.0:3", {"class": "test-email", "onClick": this._dir_click_testEmail_evt_3}, dom.txt("Test Email", "r.0:3:0"))]));
        }
        static get css () {
            return ":host{display: block}.result{font-weight: bold;margin-bottom: 10px}button{margin-right: 5px}";
        }
    }
    customElements.define("regex-literals-test", RegexLiteralsTestElement);
})();