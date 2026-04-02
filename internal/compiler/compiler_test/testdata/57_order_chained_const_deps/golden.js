/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Total: "
      {{ state.total }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Sum: "
      {{ state.sum }}
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
        const A = 1;
        const B = A + 1;
        const C = B * 2;
        const D = A + B + C;
        const E = D * D;
        const BASE_RATE = 0.1;
        const MULTIPLIER = 100;
        const ADJUSTED_RATE = BASE_RATE * MULTIPLIER;
        const FINAL_RATE = ADJUSTED_RATE + D;
        const $$initialState = {"total": E, "sum": FINAL_RATE};
        const state = makeReactive($$initialState, contextParam);
        function recalculate() {
            state.total = A + B + C + D + E;
        }
        return {"state": state, "$$initialState": $$initialState, "recalculate": recalculate};
    }
    class ChainedConstDepsElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"sum": {"type": "number"}, "total": {"type": "number"}};
        }
        static get defaultProps () {
            return {"sum": null, "total": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Total: " + String(this.$$ctx.state.total ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Sum: " + String(this.$$ctx.state.sum ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("chained-const-deps", ChainedConstDepsElement);
})();