(function() {
    var REFRESH_INTERVAL_MS = 90000;

    function refreshRecaptchaTokens() {
        grecaptcha.ready(function() {
            document.querySelectorAll(
                "input[data-captcha-provider=recaptcha_v3][data-captcha-initialized]"
            ).forEach(function(input) {
                grecaptcha.execute(input.dataset.captchaSitekey, {
                    action: input.dataset.captchaAction || "submit"
                }).then(function(token) {
                    input.value = token;
                });
            });
        });
    }

    function initRecaptchas() {
        grecaptcha.ready(function() {
            document.querySelectorAll(
                "input[data-captcha-provider=recaptcha_v3]:not([data-captcha-initialized])"
            ).forEach(function(input) {
                input.setAttribute("data-captcha-initialized", "true");
                grecaptcha.execute(input.dataset.captchaSitekey, {
                    action: input.dataset.captchaAction || "submit"
                }).then(function(token) {
                    input.value = token;
                });
            });
        });
    }

    document.addEventListener("piko:widgetinit", initRecaptchas);

    if (typeof grecaptcha !== "undefined") {
        initRecaptchas();
    } else {
        window.__pikoCaptchaRecaptchaReady = initRecaptchas;
    }

    setInterval(refreshRecaptchaTokens, REFRESH_INTERVAL_MS);
})();
