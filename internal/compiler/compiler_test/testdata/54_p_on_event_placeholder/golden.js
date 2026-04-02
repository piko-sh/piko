/*
--- BEGIN AST DUMP ---

<div>
  <button id="btn1" [Events: p-on:click.="handleEvent($event)" {P: handleEvent($event)}]>
    "Event Only"
  </button>
  <button id="btn2" [Events: p-on:click.="handleFirst($event, 'a', 'b')" {P: handleFirst($event, "a", "b")}]>
    "Event First"
  </button>
  <button id="btn3" [Events: p-on:click.="handleMiddle('before', $event, 'after')" {P: handleMiddle("before", $event, "after")}]>
    "Event Middle"
  </button>
  <button id="btn4" [Events: p-on:click.="handleLast('a', 'b', $event)" {P: handleLast("a", "b", $event)}]>
    "Event Last"
  </button>
  <button id="btn5" [Events: p-on:click.="noEvent()" {P: noEvent()}]>
    "No Event"
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
        function handleEvent(event) {
            console.log("Event:", event.type);
        }
        function handleFirst(event, a, b) {
            console.log("First:", event.type, a, b);
        }
        function handleMiddle(before, event, after) {
            console.log("Middle:", before, event.type, after);
        }
        function handleLast(a, b, event) {
            console.log("Last:", a, b, event.type);
        }
        function noEvent() {
            console.log("No event received");
        }
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState, "handleEvent": handleEvent, "handleFirst": handleFirst, "handleMiddle": handleMiddle, "handleLast": handleLast, "noEvent": noEvent};
    }
    class EventPlaceholderElement extends PPElement {
        constructor () {
            super();
            this._dir_click_handleEvent_evt_1 = (e) => {
                this.$$ctx.handleEvent.call(this, e);
            };
            this._dir_click_handleFirst_evt_2 = (e) => {
                this.$$ctx.handleFirst.call(this, e, "a", "b");
            };
            this._dir_click_handleMiddle_evt_3 = (e) => {
                this.$$ctx.handleMiddle.call(this, "before", e, "after");
            };
            this._dir_click_handleLast_evt_4 = (e) => {
                this.$$ctx.handleLast.call(this, "a", "b", e);
            };
            this._dir_click_noEvent_evt_5 = (e) => {
                this.$$ctx.noEvent.call(this);
            };
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("button", "r.0:0", {"id": "btn1", "onClick": this._dir_click_handleEvent_evt_1}, dom.txt("Event Only", "r.0:0:0")), dom.el("button", "r.0:1", {"id": "btn2", "onClick": this._dir_click_handleFirst_evt_2}, dom.txt("Event First", "r.0:1:0")), dom.el("button", "r.0:2", {"id": "btn3", "onClick": this._dir_click_handleMiddle_evt_3}, dom.txt("Event Middle", "r.0:2:0")), dom.el("button", "r.0:3", {"id": "btn4", "onClick": this._dir_click_handleLast_evt_4}, dom.txt("Event Last", "r.0:3:0")), dom.el("button", "r.0:4", {"id": "btn5", "onClick": this._dir_click_noEvent_evt_5}, dom.txt("No Event", "r.0:4:0"))]));
        }
    }
    customElements.define("event-placeholder", EventPlaceholderElement);
})();