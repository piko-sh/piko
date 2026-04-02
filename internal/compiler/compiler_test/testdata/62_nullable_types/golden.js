/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Name: "
      {{ (state.name ?? "No name") }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Count: "
      {{ (state.count ?? "Not set") }}
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
        const $$initialState = {"name": null, "count": 0, "items": null};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PpUnnamedElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"count": {"type": "number", "nullable": true}, "items": {"type": "array", "itemType": "string", "nullable": true}, "name": {"type": "string", "nullable": true}};
        }
        static get defaultProps () {
            return {"count": 0, "items": null, "name": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Name: " + String(this.$$ctx.state.name ?? "No name" ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Count: " + String(this.$$ctx.state.count ?? "Not set" ?? ""), "r.0:1:0"))]));
        }
    }
    customElements.define("pp-unnamed", PpUnnamedElement);
})();