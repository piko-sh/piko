(() => {
    if (!window.__networkRequests) {
        window.__networkRequests = [];
        const originalFetch = window.fetch;
        window.fetch = async function(...args) {
            const url = typeof args[0] === 'string' ? args[0] : args[0].url;
            const method = args[1]?.method || 'GET';
            const startTime = Date.now();
            try {
                const response = await originalFetch.apply(this, args);
                window.__networkRequests.push({
                    url: url,
                    method: method,
                    status: response.status,
                    timestamp: startTime,
                    type: 'fetch'
                });
                return response;
            } catch (e) {
                window.__networkRequests.push({
                    url: url,
                    method: method,
                    status: 0,
                    timestamp: startTime,
                    type: 'fetch',
                    error: e.message
                });
                throw e;
            }
        };

        const originalXHROpen = XMLHttpRequest.prototype.open;
        const originalXHRSend = XMLHttpRequest.prototype.send;
        XMLHttpRequest.prototype.open = function(method, url) {
            this.__method = method;
            this.__url = url;
            this.__startTime = Date.now();
            return originalXHROpen.apply(this, arguments);
        };
        XMLHttpRequest.prototype.send = function() {
            const xhr = this;
            xhr.addEventListener('loadend', function() {
                window.__networkRequests.push({
                    url: xhr.__url,
                    method: xhr.__method,
                    status: xhr.status,
                    timestamp: xhr.__startTime,
                    type: 'xhr'
                });
            });
            return originalXHRSend.apply(this, arguments);
        };
    }
})()
