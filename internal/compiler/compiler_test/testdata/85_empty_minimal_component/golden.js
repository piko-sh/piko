/*
--- BEGIN AST DUMP ---

<div id="root">
  "Minimal"
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
    class EmptyMinimalComponentElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"id": "root"}, dom.txt("Minimal", "r.0:0"));
        }
    }
    customElements.define("empty-minimal-component", EmptyMinimalComponentElement);
})();