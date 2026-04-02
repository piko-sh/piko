(() => {
    return {
        scrollHeight: Math.max(document.body.scrollHeight, document.documentElement.scrollHeight),
        clientHeight: window.innerHeight
    };
})()
