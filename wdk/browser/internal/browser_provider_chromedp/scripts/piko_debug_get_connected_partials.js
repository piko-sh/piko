(() => {
    const debug = window.__pikoDebug;
    if (!debug) return [];
    const partials = debug.getAllConnectedPartials();
    return partials.map(el => {
        const id = el.getAttribute('partial') || el.getAttribute('data-partial');
        const name = el.getAttribute('data-partial-name');
        return name ? '[data-partial-name="' + name + '"]' : '[partial="' + id + '"]';
    });
})()
