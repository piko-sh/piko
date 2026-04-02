(() => {
    const isScrollable = (el) => {
        const style = window.getComputedStyle(el);
        const ov = style.overflowY;
        return (ov === 'auto' || ov === 'scroll') && el.scrollHeight > el.clientHeight;
    };

    const docEl = document.documentElement;
    const body = document.body;
    if (docEl.scrollHeight > docEl.clientHeight + 10) {
        const htmlOv = window.getComputedStyle(docEl).overflowY;
        const bodyOv = window.getComputedStyle(body).overflowY;
        if (htmlOv !== 'hidden' && bodyOv !== 'hidden') {
            return { selector: null, scrollHeight: docEl.scrollHeight, clientHeight: docEl.clientHeight };
        }
    }

    let best = null;
    let bestArea = 0;
    const walk = (el) => {
        if (isScrollable(el)) {
            const rect = el.getBoundingClientRect();
            const area = rect.width * rect.height;
            if (area > bestArea) {
                bestArea = area;
                best = el;
            }
        }
        for (const child of el.children) {
            walk(child);
        }
    };
    walk(body);

    if (!best) {
        return { selector: null, scrollHeight: docEl.scrollHeight, clientHeight: docEl.clientHeight };
    }

    const buildSelector = (el) => {
        if (el.id) return '#' + el.id;
        let path = '';
        let current = el;
        while (current && current !== document.body && current !== document.documentElement) {
            let seg = current.tagName.toLowerCase();
            if (current.id) {
                return '#' + current.id + (path ? ' > ' + path : '');
            }
            if (current.className && typeof current.className === 'string') {
                const classes = current.className.trim().split(/\s+/).filter(c => !c.startsWith('ng-'));
                if (classes.length > 0) {
                    seg += '.' + classes.slice(0, 2).join('.');
                }
            }
            path = path ? seg + ' > ' + path : seg;
            current = current.parentElement;
        }
        return path;
    };

    return {
        selector: buildSelector(best),
        scrollHeight: best.scrollHeight,
        clientHeight: best.clientHeight
    };
})()
