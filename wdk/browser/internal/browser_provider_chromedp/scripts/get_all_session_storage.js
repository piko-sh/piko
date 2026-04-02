(() => {
    const items = {};
    for (let i = 0; i < sessionStorage.length; i++) {
        const key = sessionStorage.key(i);
        items[key] = sessionStorage.getItem(key);
    }
    return items;
})()
