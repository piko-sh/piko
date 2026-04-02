(() => {
    const selectors = [
        'mio-cookie-notice mwc-button',
        'mio-cookie-notice button',
        '#CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll',
        '#CybotCookiebotDialogBodyButtonAccept',
        '#onetrust-accept-btn-handler',
        '[data-cky-tag="accept-button"]',
        '.osano-cm-accept-all',
        '.cm-btn-accept',
        '#cmplz-cookiebanner-container .cmplz-accept',
        '[class*="cookie"] button[class*="accept"]',
        '[class*="cookie"] button[class*="agree"]',
        '[class*="cookie"] button[class*="allow"]',
        '[class*="consent"] button[class*="accept"]',
        '[class*="consent"] button[class*="agree"]',
        '[id*="cookie"] button[class*="accept"]',
        '[id*="cookie-accept"]',
        '[id*="cookie-agree"]',
        'button[id*="accept-cookie"]',
    ];

    // Try each selector.
    for (const sel of selectors) {
        try {
            const btn = document.querySelector(sel);
            if (btn) {
                btn.click();
                return true;
            }
        } catch (e) {}
    }

    const acceptPatterns = [
        /^okay[,.]?\s*got it$/i,
        /^accept\s*(all|cookies)?$/i,
        /^agree(\s+(&|and)\s+continue)?$/i,
        /^allow\s*(all|cookies)?$/i,
        /^got it$/i,
        /^i\s*(understand|agree|accept)$/i,
        /^ok$/i,
        /^continue$/i,
    ];

    const candidates = document.querySelectorAll(
        'button, [role="button"], a[class*="button"], mwc-button'
    );

    for (const el of candidates) {
        const text = (el.textContent || '').trim();
        for (const pattern of acceptPatterns) {
            if (pattern.test(text)) {
                el.click();
                return true;
            }
        }
    }

    return false;
})()
