(function() {
    const form = this.tagName === 'FORM' ? this : this.closest('form');
    if (!form) return {};
    const formData = new FormData(form);
    const data = {};
    for (const [key, value] of formData.entries()) {
        if (data[key]) {
            if (Array.isArray(data[key])) {
                data[key].push(value);
            } else {
                data[key] = [data[key], value];
            }
        } else {
            data[key] = value;
        }
    }
    return data;
})
