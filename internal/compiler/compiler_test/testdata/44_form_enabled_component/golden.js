/*
--- BEGIN AST DUMP ---

<input type="text" :name="state.name" {P: state.name} :required="state.required" {P: state.required} [p-model: state.value] [p-ref: nativeInput] />

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"value": "initial value", "name": "custom-field", "required": true};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class FormComponentElement extends PPElement {
        constructor () {
            super();
            this._dir_input___internal_model_updater_evt_1 = (event) => {
                this.$$ctx.state.value = (event.originalTarget || event.target).value;
                if (this._updateFormState) this._updateFormState();
            };
        }
        static "formAssociated" = true;
        static "enabledBehaviours" = ["form"];
        static get propTypes () {
            return {"name": {"type": "string"}, "required": {"type": "boolean"}, "value": {"type": "string"}};
        }
        static get defaultProps () {
            return {"name": "custom-field", "required": true, "value": "initial value"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("input", "r.0", {"?required": (this.$$ctx.state.required), "_ref": "nativeInput", "name": (this.$$ctx.state.name), "onInput": this._dir_input___internal_model_updater_evt_1, "type": "text", "value": this.$$ctx.state.value}, null);
        }
    }
    customElements.define("form-component", FormComponentElement);
})();