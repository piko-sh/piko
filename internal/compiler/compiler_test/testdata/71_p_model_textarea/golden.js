/*
--- BEGIN AST DUMP ---

<div>
  <textarea id="content" [p-model: state.content] />
  <p id="length">
    <RichText>
      "Length: "
      {{ state.content.length }}
    </RichText>
  </p>
  <p id="preview">
    <RichText>
      {{ state.content }}
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
        const $$initialState = {"content": "Initial text"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PModelTextareaElement extends PPElement {
        constructor () {
            super();
            this._dir_input___internal_model_updater_evt_1 = (event) => {
                this.$$ctx.state.content = (event.originalTarget || event.target).value;
                if (this._updateFormState) this._updateFormState();
            };
        }
        static get propTypes () {
            return {"content": {"type": "string"}};
        }
        static get defaultProps () {
            return {"content": "Initial text"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("textarea", "r.0:0", {"id": "content", "onInput": this._dir_input___internal_model_updater_evt_1, "value": this.$$ctx.state.content}, null), dom.el("p", "r.0:1", {"id": "length"}, dom.txt("Length: " + String(this.$$ctx.state.content.length ?? ""), "r.0:1:0")), dom.el("p", "r.0:2", {"id": "preview"}, dom.txt(String(this.$$ctx.state.content ?? ""), "r.0:2:0"))]));
        }
    }
    customElements.define("p-model-textarea", PModelTextareaElement);
})();