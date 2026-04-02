/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Value: "
      {{ state.value }}
    </RichText>
  </p>
  <button [Events: p-on:click.="doubleValue" {P: doubleValue}]>
    "Double"
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
        const double = (n) => {
            return n * 2;
        };
        const BASE = 5;
        const DOUBLED_BASE = double(BASE);
        const $$initialState = {"value": DOUBLED_BASE};
        const state = makeReactive($$initialState, contextParam);
        function doubleValue() {
            state.value = double(state.value);
        }
        return {"state": state, "$$initialState": $$initialState, "double": double, "doubleValue": doubleValue};
    }
    class ArrowFnImmediateElement extends PPElement {
        constructor () {
            super();
            this._dir_click_doubleValue_evt_1 = (e) => {
                this.$$ctx.doubleValue.call(this, e);
            };
        }
        static get propTypes () {
            return {"value": {"type": "number"}};
        }
        static get defaultProps () {
            return {"value": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Value: " + String(this.$$ctx.state.value ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_doubleValue_evt_1}, dom.txt("Double", "r.0:1:0"))]));
        }
    }
    customElements.define("arrow-fn-immediate", ArrowFnImmediateElement);
})();