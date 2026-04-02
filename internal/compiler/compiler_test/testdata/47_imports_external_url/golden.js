/*
--- BEGIN AST DUMP ---

<p>
  "Using an external module"
</p>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
import confetti from "https://esm.sh/canvas-confetti";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        pkc.onConnected(() => {
            confetti({"particleCount": 1, "spread": 1, "origin": {"y": 0.5}});
            console.log("Confetti function called");
        });
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class ExternalImportElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("p", "r.0", {}, dom.txt("Using an external module", "r.0:0"));
        }
    }
    customElements.define("external-import", ExternalImportElement);
})();