(async () => {
    window.eventLog = [];
    const { piko } = await import('/_piko/dist/ppframework.core.es.js');
    piko.bus.on('*', (detail, eventName) => {
        window.eventLog.push({ event: eventName, detail: detail, timestamp: Date.now() });
    });
})()
