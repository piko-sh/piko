/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Icon: "
      {{ state.icon }}
    </RichText>
  </p>
  <button [Events: p-on:click.="setStatus('error')" {P: setStatus("error")}]>
    "Set to Error"
  </button>
  <button [Events: p-on:click.="setStatus('success')" {P: setStatus("success")}]>
    "Set to Success"
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
        const ICONS = {"success": "✅", "error": "❌"};
        const $$initialState = {"status": "success", "icon": ICONS.success};
        const state = makeReactive($$initialState, contextParam);
        function setStatus(newStatus) {
            state.status = newStatus;
            state.icon = ICONS[newStatus];
        }
        return {"state": state, "$$initialState": $$initialState, "setStatus": setStatus};
    }
    class NonStateLookupElement extends PPElement {
        constructor () {
            super();
            this._dir_click_setStatus_evt_1 = (e) => {
                this.$$ctx.setStatus.call(this, "error");
            };
            this._dir_click_setStatus_evt_2 = (e) => {
                this.$$ctx.setStatus.call(this, "success");
            };
        }
        static get propTypes () {
            return {"icon": {"type": "any"}, "status": {"type": "string"}};
        }
        static get defaultProps () {
            return {"icon": null, "status": "success"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Icon: " + String(this.$$ctx.state.icon ?? ""), "r.0:0:0")), dom.el("button", "r.0:1", {"onClick": this._dir_click_setStatus_evt_1}, dom.txt("Set to Error", "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_setStatus_evt_2}, dom.txt("Set to Success", "r.0:2:0"))]));
        }
    }
    customElements.define("non-state-lookup", NonStateLookupElement);
})();