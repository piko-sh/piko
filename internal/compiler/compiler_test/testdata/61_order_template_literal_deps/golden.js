/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      {{ state.greeting }}
    </RichText>
  </p>
  <p>
    <RichText>
      {{ state.fullUrl }}
    </RichText>
  </p>
  <button [Events: p-on:click.="updateName" {P: updateName}]>
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
        const firstName = "John";
        const lastName = "Doe";
        const fullName = firstName + " " + lastName;
        const title = "Mr.";
        const formalName = title + " " + fullName;
        const greeting = "Hello, " + formalName + "!";
        const protocol = "https";
        const domain = "example.com";
        const path = "/api/v1";
        const baseUrl = protocol + "://" + domain;
        const fullUrl = baseUrl + path + "/users";
        const $$initialState = {"greeting": greeting, "fullUrl": fullUrl};
        const state = makeReactive($$initialState, contextParam);
        function updateName() {
            state.greeting = "Welcome back, " + formalName + "!";
        }
        return {"state": state, "$$initialState": $$initialState, "updateName": updateName};
    }
    class TemplateLiteralDepsElement extends PPElement {
        constructor () {
            super();
            this._dir_click_updateName_evt_1 = (e) => {
                this.$$ctx.updateName.call(this, e);
            };
        }
        static get propTypes () {
            return {"fullUrl": {"type": "string"}, "greeting": {"type": "string"}};
        }
        static get defaultProps () {
            return {"fullUrl": null, "greeting": null};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt(String(this.$$ctx.state.greeting ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt(String(this.$$ctx.state.fullUrl ?? ""), "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_updateName_evt_1}, dom.txt("Update", "r.0:2:0"))]));
        }
    }
    customElements.define("template-literal-deps", TemplateLiteralDepsElement);
})();