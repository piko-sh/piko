/*
--- BEGIN AST DUMP ---

<div>
  <a :href="`/users/${state.user.id}`" {P: `/users/${state.user.id}`}>
    "User Link"
  </a>
  <p :data-info="`Name: ${state.user.name}, Role: ${state.user.role}`" {P: `Name: ${state.user.name}, Role: ${state.user.role}`}>
    "User Info"
  </p>
  <button [Events: p-on:click.="updateUser" {P: updateUser}]>
    "Update User"
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
        const $$initialState = {"user": {"id": "abc-123", "name": "Jane Doe", "role": "admin"}};
        const state = makeReactive($$initialState, contextParam);
        function updateUser() {
            state.user.id = "xyz-789";
            state.user.name = "John Smith";
        }
        return {"state": state, "$$initialState": $$initialState, "updateUser": updateUser};
    }
    class TemplateLiteralsElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updateUser_evt_1 = (e) => {
                this.$$ctx.updateUser.call(this, e);
            };
        }
        static get propTypes () {
            return {"user": {"type": "object"}};
        }
        static get defaultProps () {
            return {"user": {"id": "abc-123", "name": "Jane Doe", "role": "admin"}};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("a", "r.0:0", {"href": ("/users/" + String(this.$$ctx.state.user.id ?? ""))}, dom.txt("User Link", "r.0:0:0")), dom.el("p", "r.0:1", {"data-info": ("Name: " + String(this.$$ctx.state.user.name ?? "") + ", Role: " + String(this.$$ctx.state.user.role ?? ""))}, dom.txt("User Info", "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_updateUser_evt_1}, dom.txt("Update User", "r.0:2:0"))]));
        }
    }
    customElements.define("template-literals", TemplateLiteralsElement);
})();