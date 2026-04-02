/*
--- BEGIN AST DUMP ---

<div class="progress">
  <svg class="track" viewBox="0 0 48 48">
    <circle class="bg" cx="24" cy="24" fill="none" r="20" stroke-width="4" />
    <circle class="indicator" cx="24" cy="24" fill="none" r="20" stroke-linecap="round" stroke-width="4" />
  </svg>
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
        const $$initialState = {"value": 0};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class RawSvgTestElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"value": {"type": "number"}};
        }
        static get defaultProps () {
            return {"value": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {"class": "progress"}, dom.el("svg", "r.0:0", {"class": "track", "viewBox": "0 0 48 48"}, dom.frag("r.0:0_f", [dom.el("circle", "r.0:0:0", {"class": "bg", "cx": "24", "cy": "24", "fill": "none", "r": "20", "stroke-width": "4"}, null), dom.el("circle", "r.0:0:1", {"class": "indicator", "cx": "24", "cy": "24", "fill": "none", "r": "20", "stroke-linecap": "round", "stroke-width": "4"}, null)])));
        }
        static get css () {
            return ".track{width: 48px;height: 48px}.bg{stroke: #e6e0e9}.indicator{stroke: #6750a4}";
        }
    }
    customElements.define("raw-svg-test", RawSvgTestElement);
})();