/*
--- BEGIN AST DUMP ---

<div class="container">
  <svg class="icon" preserveAspectRatio="xMidYMid meet" viewBox="0 0 100 100">
    <defs>
      <lineargradient gradientTransform="rotate(45)" gradientUnits="userSpaceOnUse" id="grad" spreadMethod="pad">
        <stop offset="0%" stop-color="#6750a4" />
        <stop offset="100%" stop-color="#eaddff" />
      </lineargradient>
    </defs>
    <circle cx="50" cy="50" fill="url(#grad)" pathLength="100" r="40" />
    <text lengthAdjust="spacing" text-anchor="middle" textLength="60" x="50" y="55">
      "Hello"
    </text>
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
    class SvgCaseTestElement extends PPElement {
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
            return dom.el("div", "r.0", {"class": "container"}, dom.el("svg", "r.0:0", {"class": "icon", "preserveAspectRatio": "xMidYMid meet", "viewBox": "0 0 100 100"}, dom.frag("r.0:0_f", [dom.el("defs", "r.0:0:0", {}, dom.el("lineargradient", "r.0:0:0:0", {"gradientTransform": "rotate(45)", "gradientUnits": "userSpaceOnUse", "id": "grad", "spreadMethod": "pad"}, dom.frag("r.0:0:0:0_f", [dom.el("stop", "r.0:0:0:0:0", {"offset": "0%", "stop-color": "#6750a4"}, null), dom.el("stop", "r.0:0:0:0:1", {"offset": "100%", "stop-color": "#eaddff"}, null)]))), dom.el("circle", "r.0:0:1", {"cx": "50", "cy": "50", "fill": "url(#grad)", "pathLength": "100", "r": "40"}, null), dom.el("text", "r.0:0:2", {"lengthAdjust": "spacing", "text-anchor": "middle", "textLength": "60", "x": "50", "y": "55"}, dom.txt("Hello", "r.0:0:2:0"))])));
        }
        static get css () {
            return ".icon{width: 100px;height: 100px}";
        }
    }
    customElements.define("svg-case-test", SvgCaseTestElement);
})();