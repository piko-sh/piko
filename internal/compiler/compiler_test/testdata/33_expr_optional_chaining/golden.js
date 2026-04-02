/*
--- BEGIN AST DUMP ---

<div>
  <p id="p1">
    <RichText>
      "User Name: "
      {{ state.user?.name }}
    </RichText>
  </p>
  <p id="p2">
    <RichText>
      "Friend Name: "
      {{ state.user?.friend?.name }}
    </RichText>
  </p>
  <p id="p3">
    <RichText>
      "City: "
      {{ state.user?.address?.city }}
    </RichText>
  </p>
  <button [Events: p-on:click.="addUserDetails" {P: addUserDetails}]>
    "Add Details"
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
        const $$initialState = {"user": {"name": "Alice", "friend": null}};
        const state = makeReactive($$initialState, contextParam);
        function addUserDetails() {
            state.user.friend = {"name": "Bob"};
            state.user.address = {"city": "New York"};
        }
        return {"state": state, "$$initialState": $$initialState, "addUserDetails": addUserDetails};
    }
    class OptionalChainingElement extends PPElement {
        constructor () {
            super();
            this._dir_click_addUserDetails_evt_1 = (e) => {
                this.$$ctx.addUserDetails.call(this, e);
            };
        }
        static get propTypes () {
            return {"user": {"type": "object"}};
        }
        static get defaultProps () {
            return {"user": {"name": "Alice", "friend": null}};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {"id": "p1"}, dom.txt("User Name: " + String(this.$$ctx.state.user?.name ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {"id": "p2"}, dom.txt("Friend Name: " + String(this.$$ctx.state.user?.friend?.name ?? ""), "r.0:1:0")), dom.el("p", "r.0:2", {"id": "p3"}, dom.txt("City: " + String(this.$$ctx.state.user?.address?.city ?? ""), "r.0:2:0")), dom.el("button", "r.0:3", {"onClick": this._dir_click_addUserDetails_evt_1}, dom.txt("Add Details", "r.0:3:0"))]));
        }
    }
    customElements.define("optional-chaining", OptionalChainingElement);
})();