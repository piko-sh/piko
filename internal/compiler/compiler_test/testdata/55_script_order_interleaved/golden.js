/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ state.status }}
      ": "
      {{ state.value }}
    </RichText>
  </p>
  <button [Events: p-on:click.="processA" {P: processA}]>
    "Process A"
  </button>
  <button [Events: p-on:click.="processB" {P: processB}]>
    "Process B"
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
        const CONFIG_A = "alpha";
        function processA() {
            state.status = CONFIG_A;
            state.value = helperMultiply(state.value);
        }
        const $$initialState = {"status": "idle", "value": 1};
        const state = makeReactive($$initialState, contextParam);
        function processB() {
            state.status = CONFIG_B;
            state.value = helperAdd(state.value);
        }
        const CONFIG_B = "beta";
        function helperMultiply(n) {
            return n * MULTIPLIER;
        }
        const MULTIPLIER = 2;
        function helperAdd(n) {
            return n + ADDEND;
        }
        const ADDEND = 10;
        return {"state": state, "$$initialState": $$initialState, "processA": processA, "processB": processB, "helperMultiply": helperMultiply, "helperAdd": helperAdd};
    }
    class OrderInterleavedElement extends PPElement {
        constructor () {
            super();
            this._dir_click_processA_evt_1 = (e) => {
                this.$$ctx.processA.call(this, e);
            };
            this._dir_click_processB_evt_2 = (e) => {
                this.$$ctx.processB.call(this, e);
            };
        }
        static get propTypes () {
            return {"status": {"type": "string"}, "value": {"type": "number"}};
        }
        static get defaultProps () {
            return {"status": "idle", "value": 1};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.state.status ?? "") + ": " + String(this.$$ctx.state.value ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_processA_evt_1}, dom.txt("Process A", "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_processB_evt_2}, dom.txt("Process B", "r.0:2:0"))]));
        }
    }
    customElements.define("order-interleaved", OrderInterleavedElement);
})();