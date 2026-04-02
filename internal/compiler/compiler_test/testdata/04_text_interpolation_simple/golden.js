/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Name: "
      {{ state.name }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Email: "
      {{ state.user.email }}
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
        const $$initialState = {"name": "Alice", "user": {"email": "alice@example.com"}};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class TextInterpSimpleElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"name": {"type": "string"}, "user": {"type": "object"}};
        }
        static get defaultProps () {
            return {"name": "Alice", "user": {"email": "alice@example.com"}};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Name: " + String(this.$$ctx.state.name ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Email: " + String(this.$$ctx.state.user.email ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("text-interp-simple", TextInterpSimpleElement);
})();