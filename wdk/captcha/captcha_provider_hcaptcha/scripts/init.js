(function() {
    function initHcaptchas() {
        document.querySelectorAll(
            "[data-captcha-provider=hcaptcha]:not([data-captcha-initialized])"
        ).forEach(function(container) {
            container.setAttribute("data-captcha-initialized", "true");
            var fieldName = container.dataset.captchaField;
            var input = document.querySelector("input[name=\"" + fieldName + "\"]");
            if (!input) return;
            hcaptcha.render(container.id, {
                sitekey: container.dataset.captchaSitekey,
                theme: container.dataset.captchaTheme || "light",
                size: container.dataset.captchaSize || "normal",
                callback: function(token) { input.value = token; },
                "expired-callback": function() { input.value = ""; }
            });
        });
    }

    document.addEventListener("piko:widgetinit", initHcaptchas);

    if (typeof hcaptcha !== "undefined") {
        initHcaptchas();
    } else {
        window.__pikoCaptchaHcaptchaReady = initHcaptchas;
    }
})();
