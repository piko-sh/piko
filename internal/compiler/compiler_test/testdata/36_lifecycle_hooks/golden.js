/*
--- BEGIN AST DUMP ---

<div>
  <p>
    <RichText>
      "Scroll Y: "
      {{ state.scrollY }}
    </RichText>
  </p>
  <p>
    <RichText>
      "Update Count: "
      {{ state.updateCount }}
    </RichText>
  </p>
  <button [Events: p-on:click.="triggerUpdate" {P: triggerUpdate}]>
    "Trigger Update"
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
        const $$initialState = {"scrollY": 0, "updateCount": 0, "name": "initial"};
        const state = makeReactive($$initialState, contextParam);
        const handleScroll = () => {
            console.log("Scroll handler fired");
            state.scrollY = Math.round(window.scrollY);
        };
        pkc.onConnected(() => {
            console.log("Component Connected!");
            window.addEventListener("scroll", handleScroll);
            handleScroll();
        });
        pkc.onDisconnected(() => {
            console.log("Component Disconnected!");
            window.removeEventListener("scroll", handleScroll);
        });
        pkc.onUpdated((changedProps) => {
            console.log("Component Updated. Changed props:", Array.from(changedProps).join(", "));
            if (changedProps.has("name")) {
                state.updateCount++;
            }
        });
        function triggerUpdate() {
            state.name = "updated";
        }
        return {"state": state, "$$initialState": $$initialState, "handleScroll": handleScroll, "triggerUpdate": triggerUpdate};
    }
    class LifecycleHooksElement extends PPElement {
        constructor () {
            super();
            this._dir_click_triggerUpdate_evt_1 = (e) => {
                this.$$ctx.triggerUpdate.call(this, e);
            };
        }
        static get propTypes () {
            return {"name": {"type": "string"}, "scrollY": {"type": "number"}, "updateCount": {"type": "number"}};
        }
        static get defaultProps () {
            return {"name": "initial", "scrollY": 0, "updateCount": 0};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("p", "r.0:0", {}, dom.txt("Scroll Y: " + String(this.$$ctx.state.scrollY ?? ""), "r.0:0:0")), dom.el("p", "r.0:1", {}, dom.txt("Update Count: " + String(this.$$ctx.state.updateCount ?? ""), "r.0:1:0")), dom.el("button", "r.0:2", {"onClick": this._dir_click_triggerUpdate_evt_1}, dom.txt("Trigger Update", "r.0:2:0"))]));
        }
    }
    customElements.define("lifecycle-hooks", LifecycleHooksElement);
})();