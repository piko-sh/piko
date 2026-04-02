/*
--- BEGIN AST DUMP ---

<div>
  <a>
    "Link"
  </a>
  <img />
  <button [Events: p-on:click.="update" {P: update}]>
    "Update Attributes"
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
        const $$initialState = {"url": "https://example.com/initial", "id": 123, "imgSrc": "/logo_initial.png"};
        const state = makeReactive($$initialState, contextParam);
        function update() {
            state.url = "https://example.com/updated";
            state.id = 456;
            state.imgSrc = "/logo_updated.png";
        }
        return {"state": state, "$$initialState": $$initialState, "update": update};
    }
    class AttrPBindElement extends PPElement {
        constructor () {
            super();
            this._dir_click_update_evt_1 = (e) => {
                this.$$ctx.update.call(this, e);
            };
        }
        static get propTypes () {
            return {"id": {"type": "number"}, "imgSrc": {"type": "string"}, "url": {"type": "string"}};
        }
        static get defaultProps () {
            return {"id": 123, "imgSrc": "/logo_initial.png", "url": "https://example.com/initial"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("a", "r.0:0", {"data-id": (this.$$ctx.state.id), "href": (this.$$ctx.state.url)}, dom.txt("Link", "r.0:0:0")), dom.el("img", "r.0:1", {"src": (this.$$ctx.state.imgSrc)}, null), dom.el("button", "r.0:2", {"onClick": this._dir_click_update_evt_1}, dom.txt("Update Attributes", "r.0:2:0"))]));
        }
    }
    customElements.define("attr-p-bind", AttrPBindElement);
})();