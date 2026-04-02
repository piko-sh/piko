/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Count: "
      {{ state.count }}
    </RichText>
  </p>
  <button [Events: p-on:click.="increment" {P: increment}]>
    "+"
  </button>
  <button [Events: p-on:click.="decrement" {P: decrement}]>
    "-"
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
        function increment() {
            state.count += STEP;
        }
        function decrement() {
            state.count -= STEP;
        }
        const $$initialState = {"count": 0};
        const state = makeReactive($$initialState, contextParam);
        const STEP = 1;
        const MAX_VALUE = 100;
        return {"state": state, "$$initialState": $$initialState, "increment": increment, "decrement": decrement};
    }
    class OrderFuncsFirstElement extends PPElement {
        constructor () {
            super();
            this._dir_click_increment_evt_1 = (e) => {
                this.$$ctx.increment.call(this, e);
            };
            this._dir_click_decrement_evt_2 = (e) => {
                this.$$ctx.decrement.call(this, e);
            };
        }
        static get propTypes () {
            return {"count": {"type": "number"}};
        }
        static get defaultProps () {
            return {"count": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Count: " + String(this.$$ctx.state.count ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_increment_evt_1}, dom.txt("+", "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_decrement_evt_2}, dom.txt("-", "r.0:2:0"))]));
        }
    }
    customElements.define("order-funcs-first", OrderFuncsFirstElement);
})();