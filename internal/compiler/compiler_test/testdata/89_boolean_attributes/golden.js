/*
--- BEGIN AST DUMP ---

<div>
  <button id="toggle" [Events: p-on:click.="toggle" {P: toggle}]>
    "Toggle"
  </button>
  <input id="input" type="text" :disabled="state.isDisabled" {P: state.isDisabled} />
  <button id="submit" :disabled="state.isDisabled" {P: state.isDisabled}>
    "Submit"
  </button>
  <p id="status">
    <RichText>
      "Disabled: "
      {{ state.isDisabled }}
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
        const $$initialState = {"isDisabled": true};
        const state = makeReactive($$initialState, contextParam);
        function toggle() {
            state.isDisabled = !state.isDisabled;
        }
        return {"state": state, "$$initialState": $$initialState, "toggle": toggle};
    }
    class BooleanAttributesElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggle_evt_1 = (e) => {
                this.$$ctx.toggle.call(this, e);
            };
        }
        static get propTypes () {
            return {"isDisabled": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"isDisabled": true};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "toggle", "onClick": this._dir_click_toggle_evt_1}, dom.txt("Toggle", "r.0:0:0")), dom.el("input", "r.0:1", {"?disabled": (this.$$ctx.state.isDisabled), "id": "input", "type": "text"}, null), dom.el("button", "r.0:2", {"?disabled": (this.$$ctx.state.isDisabled), "id": "submit"}, dom.txt("Submit", "r.0:2:0")), dom.el("p", "r.0:3", {"id": "status"}, dom.txt("Disabled: " + String(this.$$ctx.state.isDisabled ?? ""), "r.0:3:0"))]));
        }
    }
    customElements.define("boolean-attributes", BooleanAttributesElement);
})();