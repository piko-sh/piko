(() => {
    return {
        scrollHeight: Math.max(
            document.body.scrollHeight,
            document.documentElement.scrollHeight
        ),
        viewportWidth: window.innerWidth,
        viewportHeight: window.innerHeight
    };
})()
