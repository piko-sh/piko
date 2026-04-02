/*
--- BEGIN AST DUMP ---

<div>
  <input id="quantity" max="100" min="0" type="number" [p-model: state.quantity] />
  <button id="increment" [Events: p-on:click.="increment" {P: increment}]>
    "+"
  </button>
  <button id="decrement" [Events: p-on:click.="decrement" {P: decrement}]>
    "-"
  </button>
  <p id="display">
    <RichText>
      "Quantity: "
      {{ state.quantity }}
    </RichText>
  </p>
  <p id="doubled">
    <RichText>
      "Doubled: "
      {{ (state.quantity * 2) }}
    </RichText>
  </p>
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
        const $$initialState = {"quantity": 5};
        const state = makeReactive($$initialState, contextParam);
        function increment() {
            state.quantity++;
        }
        function decrement() {
            state.quantity--;
        }
        return {"state": state, "$$initialState": $$initialState, "increment": increment, "decrement": decrement};
    }
    class PModelNumberElement extends PPElement {
        constructor () {
            super();
            this._dir_input___internal_model_updater_evt_1 = (event) => {
                this.$$ctx.state.quantity = (event.originalTarget || event.target).value;
                if (this._updateFormState) this._updateFormState();
            };
            this._dir_click_increment_evt_2 = (e) => {
                this.$$ctx.increment.call(this, e);
            };
            this._dir_click_decrement_evt_3 = (e) => {
                this.$$ctx.decrement.call(this, e);
            };
        }
        static get propTypes () {
            return {"quantity": {"type": "number"}};
        }
        static get defaultProps () {
            return {"quantity": 5};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("input", "r.0:0", {"id": "quantity", "max": "100", "min": "0", "onInput": this._dir_input___internal_model_updater_evt_1, "type": "number", "value": this.$$ctx.state.quantity}, null), dom.el("button", "r.0:1", {"id": "increment", "onClick": this._dir_click_increment_evt_2}, dom.txt("+", "r.0:1:0")), dom.el("button", "r.0:2", {"id": "decrement", "onClick": this._dir_click_decrement_evt_3}, dom.txt("-", "r.0:2:0")), dom.el("p", "r.0:3", {"id": "display"}, dom.txt("Quantity: " + String(this.$$ctx.state.quantity ?? ""), "r.0:3:0")), dom.el("p", "r.0:4", {"id": "doubled"}, dom.txt("Doubled: " + String(this.$$ctx.state.quantity * 2 ?? ""), "r.0:4:0"))]));
        }
    }
    customElements.define("p-model-number", PModelNumberElement);
})();