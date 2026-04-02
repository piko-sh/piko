/*
--- BEGIN AST DUMP ---

<div>
  <button id="toggle" [Events: p-on:click.="toggle" {P: toggle}]>
    "Toggle"
  </button>
  <p id="status">
    <RichText>
      "Active: "
      {{ state.isActive }}
    </RichText>
  </p>
  <p id="count">
    <RichText>
      "Count: "
      {{ state.count }}
    </RichText>
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
        const $$initialState = {"isActive": false, "count": 0};
        const state = makeReactive($$initialState, contextParam);
        function toggle() {
            state.isActive = !state.isActive;
            state.count++;
        }
        return {"state": state, "$$initialState": $$initialState, "toggle": toggle};
    }
    class PClassDirectiveElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggle_evt_1 = (e) => {
                this.$$ctx.toggle.call(this, e);
            };
        }
        static get propTypes () {
            return {"count": {"type": "number"}, "isActive": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"count": 0, "isActive": false};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "toggle", "onClick": this._dir_click_toggle_evt_1}, dom.txt("Toggle", "r.0:0:0")), dom.el("p", "r.0:1", {"id": "status"}, dom.txt("Active: " + String(this.$$ctx.state.isActive ?? ""), "r.0:1:0")), dom.el("p", "r.0:2", {"id": "count"}, dom.txt("Count: " + String(this.$$ctx.state.count ?? ""), "r.0:2:0"))]));
        }
    }
    customElements.define("p-class-directive", PClassDirectiveElement);
})();