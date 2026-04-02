/*
--- BEGIN AST DUMP ---

<div>
  <div [p-class: {"is-active": true, "is-dark": (state.theme == "dark")}]>
    "Class Object"
  </div>
  <div [p-class: ["base", state.theme, "font-bold"]]>
    "Class Array"
  </div>
  <p [Events: p-on:click.="log(['click', { x: $event.clientX, y: $event.clientY }])" {P: log(["click", {"x": $event.clientX, "y": $event.clientY}])}]>
    "Click to Log"
  </p>
  <button [Events: p-on:click.="changeTheme" {P: changeTheme}]>
    "Change Theme"
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
        const $$initialState = {"theme": "dark"};
        const state = makeReactive($$initialState, contextParam);
        function changeTheme() {
            state.theme = "light";
        }
        function log(data) {
            console.log("Logged data:", JSON.stringify(data));
        }
        return {"state": state, "$$initialState": $$initialState, "changeTheme": changeTheme, "log": log};
    }
    class LiteralsInExprElement extends PPElement {
        constructor () {
            super();
            this._dir_click_log_evt_1 = (e) => {
                this.$$ctx.log.call(this, ["click", {"x": $event.clientX, "y": $event.clientY}]);
            };
            this._dir_click_changeTheme_evt_2 = (e) => {
                this.$$ctx.changeTheme.call(this, e);
            };
        }
        static get propTypes () {
            return {"theme": {"type": "string"}};
        }
        static get defaultProps () {
            return {"theme": "dark"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {"_class": {"is-active": true, "is-dark": this.$$ctx.state.theme === "dark"}}, dom.txt("Class Object", "r.0:0:0")), dom.el("div", "r.0:1", {"_class": ["base", this.$$ctx.state.theme, "font-bold"]}, dom.txt("Class Array", "r.0:1:0")), dom.el("p", "r.0:2", {"onClick": this._dir_click_log_evt_1}, dom.txt("Click to Log", "r.0:2:0")), dom.el("button", "r.0:3", {"onClick": this._dir_click_changeTheme_evt_2}, dom.txt("Change Theme", "r.0:3:0"))]));
        }
    }
    customElements.define("literals-in-expr", LiteralsInExprElement);
})();