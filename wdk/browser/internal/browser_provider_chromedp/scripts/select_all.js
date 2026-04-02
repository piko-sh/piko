(function() {
	const range = document.createRange();
	range.selectNodeContents(this);
	const sel = window.getSelection();
	sel.removeAllRanges();
	sel.addRange(range);
})
