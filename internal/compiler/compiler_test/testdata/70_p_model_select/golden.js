/*
--- BEGIN AST DUMP ---

<div>
  <select id="fruit-select" [p-model: state.selectedFruit]>
    <option value="apple">
      "Apple"
    </option>
    <option value="banana">
      "Banana"
    </option>
    <option value="cherry">
      "Cherry"
    </option>
  </select>
  <p id="selected">
    <RichText>
      "Selected: "
      {{ state.selectedFruit }}
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
        const $$initialState = {"selectedFruit": "banana"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PModelSelectElement extends PPElement {
        constructor () {
            super();
            this._dir_input___internal_model_updater_evt_1 = (event) => {
                this.$$ctx.state.selectedFruit = (event.originalTarget || event.target).value;
                if (this._updateFormState) this._updateFormState();
            };
        }
        static get propTypes () {
            return {"selectedFruit": {"type": "string"}};
        }
        static get defaultProps () {
            return {"selectedFruit": "banana"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("select", "r.0:0", {"id": "fruit-select", "onInput": this._dir_input___internal_model_updater_evt_1, "value": this.$$ctx.state.selectedFruit}, dom.frag("r.0:0_f", [dom.el("option", "r.0:0:0", {"value": "apple"}, dom.txt("Apple", "r.0:0:0:0")), dom.el("option", "r.0:0:1", {"value": "banana"}, dom.txt("Banana", "r.0:0:1:0")), dom.el("option", "r.0:0:2", {"value": "cherry"}, dom.txt("Cherry", "r.0:0:2:0"))])), dom.el("p", "r.0:1", {"id": "selected"}, dom.txt("Selected: " + String(this.$$ctx.state.selectedFruit ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("p-model-select", PModelSelectElement);
})();