import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement, dom, makeReactive } from "/_piko/dist/ppframework.components.es.js";
import { action } from "/_piko/assets/pk-js/pk/actions.gen.js";
;
(() => {
    function instance(contextParam) {
        const pkc = this;
        const $$initialState = {"message": "Hello from script only"};
        const state = makeReactive($$initialState, contextParam);
        function logMessage() {
            console.log(state.message);
        }
        return {"state": state, "$$initialState": $$initialState, "logMessage": logMessage};
    }
    class NoTemplateComponentElement extends PPElement {
        constructor () {
            super();
        }
        static get propTypes () {
            return {"message": {"type": "string"}};
        }
        static get defaultProps () {
            return {"message": "Hello from script only"};
        }
        connectedCallback () {
            this.init(instance.call(this, this));
            super.connectedCallback();
        }
    }
    customElements.define("no-template-component", NoTemplateComponentElement);
})();