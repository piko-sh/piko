(() => {
    const frames = document.querySelectorAll('iframe');
    return Array.from(frames).map(f => ({
        name: f.name || '',
        src: f.src || '',
        id: f.id || ''
    }));
})()
