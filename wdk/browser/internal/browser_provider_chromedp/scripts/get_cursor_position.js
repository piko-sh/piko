(function() {
	const sel = window.getSelection();
	if (!sel.rangeCount) return -1;

	const range = sel.getRangeAt(0);
	if (!range.collapsed) return -1; // Return -1 if there's a selection, not a cursor

	let offset = 0;
	const walker = document.createTreeWalker(this, NodeFilter.SHOW_TEXT);
	let node;

	while (node = walker.nextNode()) {
		if (node === range.startContainer) {
			return offset + range.startOffset;
		}
		offset += node.length;
	}

	return -1;
})
