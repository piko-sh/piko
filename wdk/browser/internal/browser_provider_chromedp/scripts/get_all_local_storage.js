(() => {
    const items = {};
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        items[key] = localStorage.getItem(key);
    }
    return items;
})()
