/*
--- BEGIN AST DUMP ---

<div class="icon">
  "glyph"
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
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class CssEscapeLiteralsElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "icon"}, dom.txt("glyph", "r.0:0"));
        }
        static get css () {
            return ".icon:before{content: \"\\eb1c\"}.w-1\\/2{width: 50%}.md\\:flex{display: flex}.dash:before{content: \"\\2014\"}.sub:before{content: \"${danger}\"}.quote:before{content: '\"x\"'}.tail{color: red}";
        }
    }
    customElements.define("css-escape-literals", CssEscapeLiteralsElement);
})();