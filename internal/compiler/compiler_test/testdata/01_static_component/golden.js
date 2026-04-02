/*
--- BEGIN AST DUMP ---

<div class="container">
  <h1>
    "Static Component"
  </h1>
  <p>
    "This component has no reactive state."
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
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class StaticComponentElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "container"}, dom.frag("r.0_f", [dom.el("h1", "r.0:0", {}, dom.txt("Static Component", "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("This component has no reactive state.", "r.0:1:0"))]));
        }
        static get css () {
            return ".container{padding: 1rem;border: 1px solid #eee;border-radius: 4px}h1{color: #333}";
        }
    }
    customElements.define("static-component", StaticComponentElement);
})();