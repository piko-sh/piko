/*
--- BEGIN AST DUMP ---

<div>
  <div id="html-content" [p-html: state.htmlContent] />
  <button id="change" [Events: p-on:click.="changeContent" {P: changeContent}]>
    "Change Content"
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
        const $$initialState = {"htmlContent": "<strong>Bold</strong> and <em>italic</em>"};
        const state = makeReactive($$initialState, contextParam);
        function changeContent() {
            state.htmlContent = "<ul><li>Item 1</li><li>Item 2</li></ul>";
        }
        return {"state": state, "$$initialState": $$initialState, "changeContent": changeContent};
    }
    class PHtmlComplexElement extends PPElement {
        constructor () {
            super();
            this._dir_click_changeContent_evt_1 = (e) => {
                this.$$ctx.changeContent.call(this, e);
            };
        }
        static get propTypes () {
            return {"htmlContent": {"type": "string"}};
        }
        static get defaultProps () {
            return {"htmlContent": "<strong>Bold</strong> and <em>italic</em>"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {"id": "html-content"}, [dom.html(String(this.$$ctx.state.htmlContent), "r.0:0:html")]), dom.el("button", "r.0:1", {"id": "change", "onClick": this._dir_click_changeContent_evt_1}, dom.txt("Change Content", "r.0:1:0"))]));
        }
    }
    customElements.define("p-html-complex", PHtmlComplexElement);
})();