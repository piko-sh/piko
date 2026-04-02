/*
--- BEGIN AST DUMP ---

<div>
  <piko:element is="span">
    "Static Content"
  </piko:element>
  <piko:element :is="state.tag" {P: state.tag}>
    "Dynamic Content"
  </piko:element>
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
        const $$initialState = {"tag": "h3"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PikoElementTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"tag": {"type": "string"}};
        }
        static get defaultProps () {
            return {"tag": "h3"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("span", "r.0:0", {}, dom.txt("Static Content", "r.0:0:0")), dom.pikoEl(this.$$ctx.state.tag, "r.0:1", {}, dom.txt("Dynamic Content", "r.0:1:0"), "")]));
        }
    }
    customElements.define("piko-element-test", PikoElementTestElement);
})();