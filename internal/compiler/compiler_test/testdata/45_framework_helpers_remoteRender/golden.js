/*
--- BEGIN AST DUMP ---

<button [Events: p-on:click.="updatePartial" {P: updatePartial}]>
  "Update Partial"
</button>

--- END AST DUMP ---
*/

import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        function updatePartial() {
            piko.partials.render({"src": "/my/partial", "querySelector": "#target-div", "patchMethod": "morph"});
        }
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState, "updatePartial": updatePartial};
    }
    class RemoteRenderHelperElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updatePartial_evt_1 = (e) => {
                this.$$ctx.updatePartial.call(this, e);
            };
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("button", "r.0", {"onClick": this._dir_click_updatePartial_evt_1}, dom.txt("Update Partial", "r.0:0"));
        }
    }
    customElements.define("remote-render-helper", RemoteRenderHelperElement);
})();