/*
--- BEGIN AST DUMP ---

<div [p-context: "app"]>
  <header [p-key: "header"] />
  <main>
    <section [p-for: item in state.items] [p-key: item.id] [p-context: `item_ctx_${item.id}`]>
      <p>
        <RichText>
          {{ item.text }}
        </RichText>
      </p>
    </section>
  </main>
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
        const $$initialState = {"items": [{"id": 101, "text": "A"}, {"id": 202, "text": "B"}]};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class KeyContextElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"items": {"type": "array", "itemType": "object"}};
        }
        static get defaultProps () {
            return {"items": [{"id": 101, "text": "A"}, {"id": 202, "text": "B"}]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "app.0", {}, dom.frag("app.0_f", [dom.el("header", "app.0:0.header", {}, null), dom.el("main", "app.0:1", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("section", "item_ctx_" + String(item.id ?? "") + "." + String(item.id ?? ""), {}, dom.el("p", "item_ctx_" + String(item.id ?? "") + "." + String(item.id ?? "") + ":0", {}, dom.txt(String(item.text ?? ""), "item_ctx_" + String(item.id ?? "") + "." + String(item.id ?? "") + ":0:0")));
            }))]));
        }
    }
    customElements.define("key-context", KeyContextElement);
})();