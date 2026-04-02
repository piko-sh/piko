(function() {
	const sel = window.getSelection();
	if (!sel.rangeCount) return { start: -1, end: -1 };

	const range = sel.getRangeAt(0);

	let startOffset = -1;
	let endOffset = -1;
	let currentOffset = 0;

	const walker = document.createTreeWalker(this, NodeFilter.SHOW_TEXT);
	let node;

	while (node = walker.nextNode()) {
		if (node === range.startContainer) {
			startOffset = currentOffset + range.startOffset;
		}
		if (node === range.endContainer) {
			endOffset = currentOffset + range.endOffset;
			break;
		}
		currentOffset += node.length;
	}

	return { start: startOffset, end: endOffset };
})
