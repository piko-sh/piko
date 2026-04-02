/* Mock actions for compiler browser integration tests. */

import { ActionBuilder, createActionBuilder, registerActionFunction } from '/_piko/dist/ppframework.core.es.js';

export function testGreet(input) {
  return createActionBuilder('test.Greet', input);
}

export function testSubmit(input) {
  return createActionBuilder('test.Submit', input);
}

export const action = {
  test: {
    greet: testGreet,
    submit: testSubmit
  }
};

registerActionFunction('test.Greet', testGreet);
registerActionFunction('test.Submit', testSubmit);
