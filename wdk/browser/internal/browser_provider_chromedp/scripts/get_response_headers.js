(() => {
    // Get headers from performance entries for the current document
    const entries = performance.getEntriesByType('navigation');
    if (entries.length === 0) return {};
    const entry = entries[0];
    // serverTiming contains some response info but not full headers
    // Full headers require network monitoring
    return {
        contentType: document.contentType || '',
        lastModified: document.lastModified || ''
    };
})()
