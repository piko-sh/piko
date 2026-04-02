/*
--- BEGIN AST DUMP ---

<div>
  <!--  This is a comment  -->
  <p id="before">
    "Before comment"
  </p>
  <!--  Another comment with special chars: <>&\"'  -->
  <p id="after">
    "After comment"
  </p>
  <!-- \n            Multi-line\n            comment\n         -->
  <p id="final">
    "Final text"
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
    class HtmlCommentsElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.cmt(" This is a comment ", "r.0:0"), dom.el("p", "r.0:1", {"id": "before"}, dom.txt("Before comment", "r.0:1:0")), dom.cmt(" Another comment with special chars: <>&\"' ", "r.0:2"), dom.el("p", "r.0:3", {"id": "after"}, dom.txt("After comment", "r.0:3:0")), dom.cmt("\n            Multi-line\n            comment\n        ", "r.0:4"), dom.el("p", "r.0:5", {"id": "final"}, dom.txt("Final text", "r.0:5:0"))]));
        }
    }
    customElements.define("html-comments", HtmlCommentsElement);
})();