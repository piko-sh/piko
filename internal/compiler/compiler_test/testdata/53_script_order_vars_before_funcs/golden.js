/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ state.message }}
    </RichText>
  </p>
  <button [Events: p-on:click.="greet" {P: greet}]>
    "Greet"
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
        const $$initialState = {"message": "Hello"};
        const state = makeReactive($$initialState, contextParam);
        const PREFIX = "Greeting: ";
        const SUFFIX = "!";
        function greet() {
            state.message = PREFIX + "World" + SUFFIX;
        }
        function reset() {
            state.message = "Hello";
        }
        return {"state": state, "$$initialState": $$initialState, "greet": greet, "reset": reset};
    }
    class OrderVarsFirstElement extends PPElement {
        constructor () {
            super();
            this._dir_click_greet_evt_1 = (e) => {
                this.$$ctx.greet.call(this, e);
            };
        }
        static get propTypes () {
            return {"message": {"type": "string"}};
        }
        static get defaultProps () {
            return {"message": "Hello"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.state.message ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_greet_evt_1}, dom.txt("Greet", "r.0:1:0"))]));
        }
    }
    customElements.define("order-vars-first", OrderVarsFirstElement);
})();