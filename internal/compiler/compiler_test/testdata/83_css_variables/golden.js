/*
--- BEGIN AST DUMP ---

<div>
  <button id="toggle" [Events: p-on:click.="toggle" {P: toggle}]>
    "Toggle Theme"
  </button>
  <div class="themed-box" id="box">
    "Themed Box"
  </div>
  <p id="status">
    <RichText>
      "Theme: "
      {{ (state.isDark ? "Dark" : "Light") }}
    </RichText>
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
        const $$initialState = {"isDark": false};
        const state = makeReactive($$initialState, contextParam);
        function toggle() {
            state.isDark = !state.isDark;
        }
        return {"state": state, "$$initialState": $$initialState, "toggle": toggle};
    }
    class CssVariablesElement extends PPElement {
        constructor () {
            super();
            this._dir_click_toggle_evt_1 = (e) => {
                this.$$ctx.toggle.call(this, e);
            };
        }
        static get propTypes () {
            return {"isDark": {"type": "boolean"}};
        }
        static get defaultProps () {
            return {"isDark": false};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "toggle", "onClick": this._dir_click_toggle_evt_1}, dom.txt("Toggle Theme", "r.0:0:0")), dom.el("div", "r.0:1", {"class": "themed-box", "id": "box"}, dom.txt("Themed Box", "r.0:1:0")), dom.el("p", "r.0:2", {"id": "status"}, dom.txt("Theme: " + String((this.$$ctx.state.isDark ? "Dark" : "Light") ?? ""), "r.0:2:0"))]));
        }
        static get css () {
            return ":host{--bg-color: #ffffff;--text-color: #000000}:host(.dark){--bg-color: #1a1a1a;--text-color: #ffffff}.themed-box{background-color: var(--bg-color);color: var(--text-color);padding: 20px;border: 1px solid currentColor}";
        }
    }
    customElements.define("css-variables", CssVariablesElement);
})();