(function() {
    function initTurnstileCaptchas() {
        document.querySelectorAll(
            "[data-captcha-provider=turnstile]:not([data-captcha-initialized])"
        ).forEach(function(container) {
            container.setAttribute("data-captcha-initialized", "true");
            var fieldName = container.dataset.captchaField;
            var input = document.querySelector("input[name=\"" + fieldName + "\"]");
            if (!input) return;
            turnstile.render("#" + container.id, {
                sitekey: container.dataset.captchaSitekey,
                theme: container.dataset.captchaTheme || "light",
                size: container.dataset.captchaSize || "normal",
                callback: function(token) { input.value = token; },
                "expired-callback": function() { input.value = ""; }
            });
        });
    }

    document.addEventListener("piko:widgetinit", initTurnstileCaptchas);

    if (typeof turnstile !== "undefined") {
        initTurnstileCaptchas();
    } else {
        window.__pikoCaptchaTurnstileReady = initTurnstileCaptchas;
    }
})();
