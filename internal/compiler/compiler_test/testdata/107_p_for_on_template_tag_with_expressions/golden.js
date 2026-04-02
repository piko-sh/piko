/*
--- BEGIN AST DUMP ---

<dl>
  <Fragment>
    <dt>
      <span [p-text: index] />
      <RichText>
        ": "
        {{ item.term }}
      </RichText>
    </dt>
    <dd>
      <RichText>
        {{ item.definition }}
      </RichText>
    </dd>
  </Fragment>
</dl>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"entries": [{"id": 1, "term": "HTML", "definition": "HyperText Markup Language"}, {"id": 2, "term": "CSS", "definition": "Cascading Style Sheets"}, {"id": 3, "term": "JS", "definition": "JavaScript"}]};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PForTemplateExprElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"entries": {"type": "array", "itemType": "object"}};
        }
        static get defaultProps () {
            return {"entries": [{"id": 1, "term": "HTML", "definition": "HyperText Markup Language"}, {"id": 2, "term": "CSS", "definition": "Cascading Style Sheets"}, {"id": 3, "term": "JS", "definition": "JavaScript"}]};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("dl", "r.0", {}, (Array.isArray(this.$$ctx.state.entries) ? this.$$ctx.state.entries : this.$$ctx.state.entries && typeof this.$$ctx.state.entries === "object" ? Object.entries(this.$$ctx.state.entries) : []).map((item, index) => {
                return dom.frag(`${"r.0:0." + String(item.id ?? "")}_f`, [dom.el("dt", "r.0:0." + String(item.id ?? "") + ":0", {}, dom.frag(`${"r.0:0." + String(item.id ?? "") + ":0"}_f`, [dom.el("span", "r.0:0." + String(item.id ?? "") + ":0:0", {}, [dom.txt(String(index), "r.0:0." + String(item.id ?? "") + ":0:0" + ":txt")]), dom.txt(": " + String(item.term ?? ""), "r.0:0." + String(item.id ?? "") + ":0:1")])), dom.el("dd", "r.0:0." + String(item.id ?? "") + ":1", {}, dom.txt(String(item.definition ?? ""), "r.0:0." + String(item.id ?? "") + ":1:0"))]);
            }));
        }
    }
    customElements.define("p-for-template-expr", PForTemplateExprElement);
})();