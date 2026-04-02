/*
--- BEGIN AST DUMP ---

<div class="container">
  <h1 class="title">
    "Scoped Styles"
  </h1>
  <p class="description">
    "This component has scoped CSS"
  </p>
  <button class="action-btn" id="btn">
    "Action"
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
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class CssScopedElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "container"}, dom.frag("r.0_f", [dom.el("h1", "r.0:0", {"class": "title"}, dom.txt("Scoped Styles", "r.0:0:0")), dom.el("p", "r.0:1", {"class": "description"}, dom.txt("This component has scoped CSS", "r.0:1:0")), dom.el("button", "r.0:2", {"class": "action-btn", "id": "btn"}, dom.txt("Action", "r.0:2:0"))]));
        }
        static get css () {
            return ".container{padding: 16px;border: 1px solid #ccc}.title{font-size: 24px;margin-bottom: 8px}.description{color: #666}.action-btn{background: blue;color: white;padding: 8px 16px;border: none;cursor: pointer}.action-btn:hover{background: darkblue}";
        }
    }
    customElements.define("css-scoped", CssScopedElement);
})();