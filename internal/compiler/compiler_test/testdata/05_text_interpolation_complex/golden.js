/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Total: "
      {{ (state.price * (1 + state.tax)) }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Message: "
      {{ getMessage() }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Status: "
      {{ (state.isActive ? "Active" : "Inactive") }}
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
        const $$initialState = {"price": 100, "tax": 0.07, "isActive": true, "name": "Product"};
        const state = makeReactive($$initialState, contextParam);
        function getMessage() {
            return `The ${state.name} is ready.`;
        }
        return {"state": state, "$$initialState": $$initialState, "getMessage": getMessage};
    }
    class TextInterpComplexElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"isActive": {"type": "boolean"}, "name": {"type": "string"}, "price": {"type": "number"}, "tax": {"type": "number"}};
        }
        static get defaultProps () {
            return {"isActive": true, "name": "Product", "price": 100, "tax": 0.07};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Total: " + String(this.$$ctx.state.price * (1 + this.$$ctx.state.tax) ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Message: " + String(this.$$ctx.getMessage() ?? ""), "r.0:1:0")), dom.el("p", "r.0:2", {}, dom.txt("Status: " + String((this.$$ctx.state.isActive ? "Active" : "Inactive") ?? ""), "r.0:2:0"))]));
        }
    }
    customElements.define("text-interp-complex", TextInterpComplexElement);
})();