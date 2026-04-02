/*
--- BEGIN AST DUMP ---

<div>
  <h1 [p-ref: title] [p-timeline:hidden]>
    "Hello"
  </h1>
  <pre>
    <code [p-ref: codeLine] [p-timeline:hidden]>
      "piko create my-app"
    </code>
  </pre>
  <div [p-ref: toast] [p-timeline:anchor: title] [p-timeline:hidden]>
    "Toast"
  </div>
</div>

--- END AST DUMP ---
*/

import "/_piko/dist/ppframework.animation.es.js";
import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"time": 0};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class PpDemoAnimElement extends PPElement {
        static "$$timeline" = [{"action": "show", "ref": "title", "time": 0.5}, {"action": "show", "ref": "codeLine", "time": 1.5}, {"action": "type", "ref": "codeLine", "time": 1.5, "speed": 50}];
        constructor () {
            super();
        }
        static "enabledBehaviours" = ["animation"];
        static get propTypes () {
            return {"time": {"type": "number"}};
        }
        static get defaultProps () {
            return {"time": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("h1", "r.0:0", {"_ref": "title", "p-timeline-hidden": ""}, dom.txt("Hello", "r.0:0:0")), dom.el("pre", "r.0:1", {}, dom.el("code", "r.0:1:0", {"_ref": "codeLine", "p-timeline-hidden": ""}, dom.txt("piko create my-app", "r.0:1:0:0"))), dom.el("div", "r.0:2", {"_ref": "toast", "p-timeline-anchor": "title", "p-timeline-hidden": ""}, dom.txt("Toast", "r.0:2:0"))]));
        }
    }
    customElements.define("pp-demo-anim", PpDemoAnimElement);
})();