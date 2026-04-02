/*
--- BEGIN AST DUMP ---

<div>
  <label>
    <input type="checkbox" [p-model: state.isChecked] />
    "Is Checked"
  </label>
  <p>
    <RichText>
      "Status: "
      {{ (state.isChecked ? "Checked" : "Not Checked") }}
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
        const $$initialState = {"isChecked": false};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PModelCheckboxElement extends PPElement {
        constructor () {
            super();
            this._dir_change___internal_model_updater_evt_1 = (event) => {
                this.$$ctx.state.isChecked = (event.originalTarget || event.target).checked;
                if (this._updateFormState) this._updateFormState();
            };
        }
        static get propTypes () {
            return {"isChecked": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"isChecked": false};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("label", "r.0:0", {}, dom.frag("r.0:0_f", [dom.el("input", "r.0:0:0", {"?checked": this.$$ctx.state.isChecked, "onChange": this._dir_change___internal_model_updater_evt_1, "type": "checkbox"}, null), dom.txt(" Is Checked ", "r.0:0:1")])), dom.el("p", "r.0:1", {}, dom.txt("Status: " + String((this.$$ctx.state.isChecked ? "Checked" : "Not Checked") ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("p-model-checkbox", PModelCheckboxElement);
})();