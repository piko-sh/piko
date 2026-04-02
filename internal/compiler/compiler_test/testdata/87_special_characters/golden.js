/*
--- BEGIN AST DUMP ---

<div>
  <p id="html-entities">
    "<div> & \"quotes\""
  </p>
  <p id="unicode">
    "Unicode: \u2603 \u2764"
  </p>
  <p id="from-state">
    <RichText>
      {{ state.specialText }}
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
        const $$initialState = {"specialText": "Text with <special> & \"chars\""};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class SpecialCharactersElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"specialText": {"type": "string"}};
        }
        static get defaultProps () {
            return {"specialText": "Text with <special> & \"chars\""};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"id": "html-entities"}, dom.txt("<div> & \"quotes\"", "r.0:0:0")), dom.el("p", "r.0:1", {"id": "unicode"}, dom.txt("Unicode: \\u2603 \\u2764", "r.0:1:0")), dom.el("p", "r.0:2", {"id": "from-state"}, dom.txt(String(this.$$ctx.state.specialText ?? ""), "r.0:2:0"))]));
        }
    }
    customElements.define("special-characters", SpecialCharactersElement);
})();