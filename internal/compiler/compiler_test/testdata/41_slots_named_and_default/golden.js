/*
--- BEGIN AST DUMP ---

<div class="card">
  <header>
    <slot name="header">
      "Default Header"
    </slot>
  </header>
  <main>
    <slot>
      "Default Content"
    </slot>
  </main>
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
    class SlotsComponentElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "card"}, dom.frag("r.0_f", [dom.el("header", "r.0:0", {}, dom.el("slot", "r.0:0:0", {"name": "header"}, dom.txt("Default Header", "r.0:0:0:0"))), dom.el("main", "r.0:1", {}, dom.el("slot", "r.0:1:0", {}, dom.txt("Default Content", "r.0:1:0:0")))]));
        }
    }
    customElements.define("slots-component", SlotsComponentElement);
})();