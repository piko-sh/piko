/*
--- BEGIN AST DUMP ---

<div>
  <p>
    "Count:"
    <span [p-text: state.count] />
  </p>
  <button [Events: p-on:click.="increment" {P: increment}]>
    "Increment"
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
        const $$initialState = {"count": 0};
        const state = makeReactive($$initialState, contextParam);
        function increment() {
            state.count++;
        }
        return {"state": state, "$$initialState": $$initialState, "increment": increment};
    }
    class ReactiveCounterElement extends PPElement {
        constructor () {
            super();
            this._dir_click_increment_evt_1 = (e) => {
                this.$$ctx.increment.call(this, e);
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
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.frag("r.0:0_f", [dom.txt("Count: ", "r.0:0:0"), dom.el("span", "r.0:0:1", {}, [dom.txt(String(this.$$ctx.state.count), "r.0:0:1:txt")])])), dom.el("button", "r.0:1", {"onClick": this._dir_click_increment_evt_1}, dom.txt("Increment", "r.0:1:0"))]));
        }
    }
    customElements.define("reactive-counter", ReactiveCounterElement);
})();