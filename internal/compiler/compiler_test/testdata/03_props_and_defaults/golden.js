/*
--- BEGIN AST DUMP ---

<div class="user-card">
  <h3>
    <span [p-text: state.title] />
    <span [p-text: state.name] />
  </h3>
  <p>
    "Age:"
    <span [p-text: state.age] />
  </p>
  <p [p-if: state.isAdmin]>
    "Admin User"
  </p>
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
        const $$initialState = {"title": "User", "name": "Guest", "age": 99, "isAdmin": false};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PropsComponentElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"age": {"type": "number"}, "isAdmin": {"type": "boolean"}, "name": {"type": "string"}, "title": {"type": "string"}};
        }
        static get defaultProps () {
            return {"age": 99, "isAdmin": false, "name": "Guest", "title": "User"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "user-card"}, dom.frag("r.0_f", [dom.el("h3", "r.0:0", {}, dom.frag("r.0:0_f", [dom.el("span", "r.0:0:0", {}, [dom.txt(String(this.$$ctx.state.title), "r.0:0:0:txt")]), dom.ws("r.0:0:1"), dom.el("span", "r.0:0:2", {}, [dom.txt(String(this.$$ctx.state.name), "r.0:0:2:txt")])])), dom.el("p", "r.0:1", {}, dom.frag("r.0:1_f", [dom.txt("Age: ", "r.0:1:0"), dom.el("span", "r.0:1:1", {}, [dom.txt(String(this.$$ctx.state.age), "r.0:1:1:txt")])])), this.$$ctx.state.isAdmin ? dom.el("p", "r.0:2", {}, dom.txt("Admin User", "r.0:2:0")) : null]));
        }
    }
    customElements.define("props-component", PropsComponentElement);
})();