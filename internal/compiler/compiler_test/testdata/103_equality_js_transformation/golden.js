/*
--- BEGIN AST DUMP ---

<div>
  <h1>
    "Equality Operator JS Transformation"
  </h1>
  <p id="strict-eq" [p-if: (state.count == 5)]>
    "Count is strictly 5"
  </p>
  <p id="strict-ne" [p-if: (state.status != "error")]>
    "Status is not error"
  </p>
  <p id="loose-eq" [p-if: (state.count ~= "5")]>
    "Count loosely equals '5'"
  </p>
  <p id="loose-ne" [p-if: (state.count !~= "0")]>
    "Count loosely not zero"
  </p>
  <p id="truthy" [p-if: ~state.message]>
    "Has message"
  </p>
  <p id="combined" [p-if: ((state.count == 5) && ~state.message)]>
    "Count is 5 and has message"
  </p>
  <p :class="state.status == 'active' ? 'enabled' : 'disabled'" {P: ((state.status == "active") ? "enabled" : "disabled")}>
    "Status class"
  </p>
  <p :data-has-msg="~state.message ? 'yes' : 'no'" {P: (~state.message ? "yes" : "no")}>
    "Truthy ternary"
  </p>
  <div :data-is-five="state.count == 5" {P: (state.count == 5)}>
    "Strict eq in attr"
  </div>
  <div :data-loose-eq="state.count ~= '5'" {P: (state.count ~= "5")}>
    "Loose eq in attr"
  </div>
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
        const $$initialState = {"count": 5, "status": "active", "message": "Hello"};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class EqualityJsTransformationElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"count": {"type": "number"}, "message": {"type": "string"}, "status": {"type": "string"}};
        }
        static get defaultProps () {
            return {"count": 5, "message": "Hello", "status": "active"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("h1", "r.0:0", {}, dom.txt("Equality Operator JS Transformation", "r.0:0:0")), this.$$ctx.state.count === 5 ? dom.el("p", "r.0:1", {"id": "strict-eq"}, dom.txt("Count is strictly 5", "r.0:1:0")) : null, this.$$ctx.state.status !== "error" ? dom.el("p", "r.0:2", {"id": "strict-ne"}, dom.txt("Status is not error", "r.0:2:0")) : null, this.$$ctx.state.count == "5" ? dom.el("p", "r.0:3", {"id": "loose-eq"}, dom.txt("Count loosely equals '5'", "r.0:3:0")) : null, this.$$ctx.state.count != "0" ? dom.el("p", "r.0:4", {"id": "loose-ne"}, dom.txt("Count loosely not zero", "r.0:4:0")) : null, !!this.$$ctx.state.message ? dom.el("p", "r.0:5", {"id": "truthy"}, dom.txt("Has message", "r.0:5:0")) : null, this.$$ctx.state.count === 5 && !!this.$$ctx.state.message ? dom.el("p", "r.0:6", {"id": "combined"}, dom.txt("Count is 5 and has message", "r.0:6:0")) : null, dom.el("p", "r.0:7", {"class": (this.$$ctx.state.status === "active" ? "enabled" : "disabled")}, dom.txt("Status class", "r.0:7:0")), dom.el("p", "r.0:8", {"data-has-msg": (!!this.$$ctx.state.message ? "yes" : "no")}, dom.txt("Truthy ternary", "r.0:8:0")), dom.el("div", "r.0:9", {"data-is-five": (this.$$ctx.state.count === 5)}, dom.txt("Strict eq in attr", "r.0:9:0")), dom.el("div", "r.0:10", {"data-loose-eq": (this.$$ctx.state.count == "5")}, dom.txt("Loose eq in attr", "r.0:10:0"))]));
        }
    }
    customElements.define("equality-js-transformation", EqualityJsTransformationElement);
})();