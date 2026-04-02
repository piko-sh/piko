(() => {
    if (!window.__pikoDebug || !window.__pikoDebug.isAvailable()) {
        return { error: 'Debug API not available' };
    }
    const components = window.__pikoDebug.getAllComponents();
    return components.map(c => ({
        selector: c.selector || '',
        name: c.name || '',
        isConnected: c.isConnected || false
    }));
})()
