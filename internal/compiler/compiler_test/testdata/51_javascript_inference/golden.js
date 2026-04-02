/*
--- BEGIN AST DUMP ---

<div>
  <h2>
    "JavaScript Type Inference Test"
  </h2>
  <p>
    "Count:"
    <span [p-text: state.count] />
  </p>
  <p>
    "Message:"
    <span [p-text: state.message] />
  </p>
  <p [p-if: state.active]>
    "Active Status"
  </p>
  <button :disabled="state.disabled" {P: state.disabled}>
    "Button"
  </button>
  <ul>
    <li [p-for: item in state.items] [p-key: item]>
      <RichText>
        {{ item }}
      </RichText>
    </li>
  </ul>
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
        const $$initialState = {"count": 42, "message": "Hello from JavaScript", "active": true, "disabled": false, "items": ["Alpha", "Beta", "Gamma"], "data": {"nested": "object"}};
        const state = makeReactive($$initialState, contextParam);
        function increment() {
            state.count++;
        }
        return {"state": state, "$$initialState": $$initialState, "increment": increment};
    }
    class JsInferenceTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"active": {"type": "boolean"}, "count": {"type": "number"}, "data": {"type": "object"}, "disabled": {"type": "boolean"}, "items": {"type": "array", "itemType": "string"}, "message": {"type": "string"}};
        }
        static get defaultProps () {
            return {"active": true, "count": 42, "data": {"nested": "object"}, "disabled": false, "items": ["Alpha", "Beta", "Gamma"], "message": "Hello from JavaScript"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("h2", "r.0:0", {}, dom.txt("JavaScript Type Inference Test", "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.frag("r.0:1_f", [dom.txt("Count: ", "r.0:1:0"), dom.el("span", "r.0:1:1", {}, [dom.txt(String(this.$$ctx.state.count), "r.0:1:1:txt")])])), dom.el("p", "r.0:2", {}, dom.frag("r.0:2_f", [dom.txt("Message: ", "r.0:2:0"), dom.el("span", "r.0:2:1", {}, [dom.txt(String(this.$$ctx.state.message), "r.0:2:1:txt")])])), this.$$ctx.state.active ? dom.el("p", "r.0:3", {}, dom.txt("Active Status", "r.0:3:0")) : null, dom.el("button", "r.0:4", {"?disabled": (this.$$ctx.state.disabled)}, dom.txt("Button", "r.0:4:0")), dom.el("ul", "r.0:5", {}, (Array.isArray(this.$$ctx.state.items) ? this.$$ctx.state.items : this.$$ctx.state.items && typeof this.$$ctx.state.items === "object" ? Object.entries(this.$$ctx.state.items) : []).map((item) => {
                return dom.el("li", "r.0:5:0." + String(item ?? ""), {}, dom.txt(String(item ?? ""), "r.0:5:0." + String(item ?? "") + ":0"));
            }))]));
        }
        static get css () {
            return "h2{color: #333}";
        }
    }
    customElements.define("js-inference-test", JsInferenceTestElement);
})();