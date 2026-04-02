(() => {
	const sel = window.getSelection();
	if (sel.rangeCount > 0) {
		sel.collapseToEnd();
	}
})()
