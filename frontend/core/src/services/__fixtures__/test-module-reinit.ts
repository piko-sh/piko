/** Test fixture function for reinit module. */
export function reinitFunc() { return 'reinit-result'; }

/** Tracks how many times the reinit hook has been invoked. */
let reinitCallCount = 0;
/** The page visit the reinit hook was last called with. */
let lastReinitVisit: object | undefined;
/** Reinitialisation hook for the test fixture. */
// eslint-disable-next-line @typescript-eslint/naming-convention
export function __reinit__(visit?: object) { reinitCallCount++; lastReinitVisit = visit; }
/** Returns the page visit the reinit hook was last called with. */
export function getLastReinitVisit() { return lastReinitVisit; }
/** Returns the number of times reinit has been called. */
export function getReinitCallCount() { return reinitCallCount; }
