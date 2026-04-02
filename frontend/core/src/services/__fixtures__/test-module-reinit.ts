/** Test fixture function for reinit module. */
export function reinitFunc() { return 'reinit-result'; }

/** Tracks how many times the reinit hook has been invoked. */
let reinitCallCount = 0;
/** Reinitialisation hook for the test fixture. */
// eslint-disable-next-line @typescript-eslint/naming-convention
export function __reinit__() { reinitCallCount++; }
/** Returns the number of times reinit has been called. */
export function getReinitCallCount() { return reinitCallCount; }
