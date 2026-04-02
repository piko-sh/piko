(() => {
    const OriginalIO = window.IntersectionObserver;
    if (!OriginalIO) return;

    window.IntersectionObserver = class {
        constructor(callback, options) {
            this._callback = callback;
            this._real = new OriginalIO((entries, obs) => {
                callback(entries, this);
            }, options);
        }

        observe(target) {
            this._real.observe(target);

            setTimeout(() => {
                try {
                    this._callback([{
                        target: target,
                        isIntersecting: true,
                        intersectionRatio: 1.0,
                        boundingClientRect: target.getBoundingClientRect(),
                        intersectionRect: target.getBoundingClientRect(),
                        rootBounds: null,
                        time: performance.now(),
                    }], this);
                } catch (e) {}
            }, 10);
        }

        unobserve(target) { this._real.unobserve(target); }
        disconnect() { this._real.disconnect(); }
        takeRecords() { return this._real.takeRecords(); }
    };
})()
