/*
--- BEGIN AST DUMP ---

<div>
  <div [p-html: state.rawHtml]>
    "This will be replaced."
  </div>
  <button [Events: p-on:click.="updateHtml" {P: updateHtml}]>
    "Update"
  </button>
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
        const $$initialState = {"rawHtml": "<em>Initial HTML</em>"};
        const state = makeReactive($$initialState, contextParam);
        function updateHtml() {
            state.rawHtml = "<strong>Updated HTML</strong> with a <a href='#'>link</a>.";
        }
        return {"state": state, "$$initialState": $$initialState, "updateHtml": updateHtml};
    }
    class PHtmlDirElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updateHtml_evt_1 = (e) => {
                this.$$ctx.updateHtml.call(this, e);
            };
        }
        static get propTypes () {
            return {"rawHtml": {"type": "string"}};
        }
        static get defaultProps () {
            return {"rawHtml": "<em>Initial HTML</em>"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {}, [dom.html(String(this.$$ctx.state.rawHtml), "r.0:0:html")]), dom.el("button", "r.0:1", {"onClick": this._dir_click_updateHtml_evt_1}, dom.txt("Update", "r.0:1:0"))]));
        }
    }
    customElements.define("p-html-dir", PHtmlDirElement);
})();