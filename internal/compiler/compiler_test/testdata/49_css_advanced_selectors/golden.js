/*
--- BEGIN AST DUMP ---

<div class="inside">
  "Inside text"
</div>
<slot />

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
    class CssAdvancedElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.frag("root_fragment", [dom.el("div", "r.0", {"class": "inside"}, dom.txt("Inside text", "r.0:0")), dom.el("slot", "r.1", {}, null)]);
        }
        static get css () {
            return ":host-context(.theme-dark) .inside{color: white;background-color: black}::slotted(.slotted-item){font-style: italic}.unused{display: none}";
        }
    }
    customElements.define("css-advanced", CssAdvancedElement);
})();