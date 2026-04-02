/*
--- BEGIN AST DUMP ---

<div>
  <p style="font-weight: bold;" [p-style: {"color": state.activeColor, "fontSize": (state.size + "px")}]>
    "Styled Text"
  </p>
  <button [Events: p-on:click.="updateStyle" {P: updateStyle}]>
    "Update Style"
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
        const $$initialState = {"activeColor": "blue", "size": 16};
        const state = makeReactive($$initialState, contextParam);
        function updateStyle() {
            state.activeColor = "red";
            state.size = 24;
        }
        return {"state": state, "$$initialState": $$initialState, "updateStyle": updateStyle};
    }
    class PStyleObjectElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updateStyle_evt_1 = (e) => {
                this.$$ctx.updateStyle.call(this, e);
            };
        }
        static get propTypes () {
            return {"activeColor": {"type": "string"}, "size": {"type": "number"}};
        }
        static get defaultProps () {
            return {"activeColor": "blue", "size": 16};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"_style": {"color": this.$$ctx.state.activeColor, "fontSize": this.$$ctx.state.size + "px"}, "style": "font-weight: bold;"}, dom.txt(" Styled Text ", "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_updateStyle_evt_1}, dom.txt("Update Style", "r.0:1:0"))]));
        }
    }
    customElements.define("p-style-object", PStyleObjectElement);
})();