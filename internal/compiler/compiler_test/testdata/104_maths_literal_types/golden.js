/*
--- BEGIN AST DUMP ---

<div>
  <h1>
    "Maths and Temporal Literal Types"
  </h1>
  <p id="decimal" [p-text: 19.99d]>
    "decimal"
  </p>
  <p id="bigint" [p-text: 42n]>
    "bigint"
  </p>
  <p id="rune" [p-text: r'A']>
    "rune"
  </p>
  <p id="date" [p-text: d'2026-01-15']>
    "date"
  </p>
  <p id="time" [p-text: t'14:30:45']>
    "time"
  </p>
  <p id="datetime" [p-text: dt'2026-01-15T14:30:45Z']>
    "datetime"
  </p>
  <p id="duration" [p-text: du'1h30m']>
    "duration"
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
        const $$initialState = {};
        const state = makeReactive($$initialState, contextParam);
        return {"state": state, "$$initialState": $$initialState};
    }
    class MathsLiteralTypesElement extends PPElement {
        constructor () {
            super();
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
        renderVDOM () {
            return dom.el("div", "r.0", {}, dom.frag("r.0_f", [dom.el("h1", "r.0:0", {}, dom.txt("Maths and Temporal Literal Types", "r.0:0:0")), dom.el("p", "r.0:1", {"id": "decimal"}, [dom.txt(String(19.99), "r.0:1:txt")]), dom.el("p", "r.0:2", {"id": "bigint"}, [dom.txt(String("42"), "r.0:2:txt")]), dom.el("p", "r.0:3", {"id": "rune"}, [dom.txt(String("A"), "r.0:3:txt")]), dom.el("p", "r.0:4", {"id": "date"}, [dom.txt(String("2026-01-15"), "r.0:4:txt")]), dom.el("p", "r.0:5", {"id": "time"}, [dom.txt(String("14:30:45"), "r.0:5:txt")]), dom.el("p", "r.0:6", {"id": "datetime"}, [dom.txt(String("2026-01-15T14:30:45Z"), "r.0:6:txt")]), dom.el("p", "r.0:7", {"id": "duration"}, [dom.txt(String("1h30m"), "r.0:7:txt")])]));
        }
    }
    customElements.define("maths-literal-types", MathsLiteralTypesElement);
})();