/*
--- BEGIN AST DUMP ---

<div>
  <input type="text" [p-model: state.name] />
  <p>
    <RichText>
      "Hello, "
      {{ state.name }}
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
        const $$initialState = {"name": "World"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PModelTextElement extends PPElement {
        constructor () {
            super();
            this._dir_input___internal_model_updater_evt_1 = (event) => {
                this.$$ctx.state.name = (event.originalTarget || event.target).value;
                if (this._updateFormState) this._updateFormState();
            };
        }
        static get propTypes () {
            return {"name": {"type": "string"}};
        }
        static get defaultProps () {
            return {"name": "World"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("input", "r.0:0", {"onInput": this._dir_input___internal_model_updater_evt_1, "type": "text", "value": this.$$ctx.state.name}, null), dom.el("p", "r.0:1", {}, dom.txt("Hello, " + String(this.$$ctx.state.name ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("p-model-text", PModelTextElement);
})();