/*
--- BEGIN AST DUMP ---

<div>
  <div id="target" [Events: p-on:click.="onClick" {P: onClick} p-on:mouseenter.="onEnter" {P: onEnter} p-on:mouseleave.="onLeave" {P: onLeave}]>
    "Hover and click me"
  </div>
  <p id="hover-status">
    <RichText>
      {{ state.hoverStatus }}
    </RichText>
  </p>
  <p id="click-count">
    <RichText>
      "Clicks: "
      {{ state.clickCount }}
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
        const $$initialState = {"hoverStatus": "Not hovering", "clickCount": 0};
        const state = makeReactive($$initialState, contextParam);
        function onEnter() {
            state.hoverStatus = "Hovering";
        }
        function onLeave() {
            state.hoverStatus = "Not hovering";
        }
        function onClick() {
            state.clickCount++;
        }
        return {"state": state, "$$initialState": $$initialState, "onEnter": onEnter, "onLeave": onLeave, "onClick": onClick};
    }
    class MultipleEventHandlersElement extends PPElement {
        constructor () {
            super();
            this._dir_click_onClick_evt_1 = (e) => {
                this.$$ctx.onClick.call(this, e);
            };
            this._dir_mouseenter_onEnter_evt_2 = (e) => {
                this.$$ctx.onEnter.call(this, e);
            };
            this._dir_mouseleave_onLeave_evt_3 = (e) => {
                this.$$ctx.onLeave.call(this, e);
            };
        }
        static get propTypes () {
            return {"clickCount": {"type": "number"}, "hoverStatus": {"type": "string"}};
        }
        static get defaultProps () {
            return {"clickCount": 0, "hoverStatus": "Not hovering"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("div", "r.0:0", {"id": "target", "onClick": this._dir_click_onClick_evt_1, "onMouseenter": this._dir_mouseenter_onEnter_evt_2, "onMouseleave": this._dir_mouseleave_onLeave_evt_3}, dom.txt(" Hover and click me ", "r.0:0:0")), dom.el("p", "r.0:1", {"id": "hover-status"}, dom.txt(String(this.$$ctx.state.hoverStatus ?? ""), "r.0:1:0")), dom.el("p", "r.0:2", {"id": "click-count"}, dom.txt("Clicks: " + String(this.$$ctx.state.clickCount ?? ""), "r.0:2:0"))]));
        }
    }
    customElements.define("multiple-event-handlers", MultipleEventHandlersElement);
})();