package main

import (
	"crypto/rand"
	"os"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/wdk/captcha/captcha_provider_hcaptcha"
	"piko.sh/piko/wdk/captcha/captcha_provider_hmac_challenge"
	"piko.sh/piko/wdk/captcha/captcha_provider_recaptcha_v3"
	"piko.sh/piko/wdk/captcha/captcha_provider_turnstile"
	"piko.sh/piko/wdk/logger"
)

func main() {
	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.AddPrettyOutput()

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic(err)
	}

	hmacProvider, err := captcha_provider_hmac_challenge.NewProvider(captcha_provider_hmac_challenge.Config{
		Secret: secret,
	})
	if err != nil {
		panic(err)
	}

	turnstilePassProvider, err := captcha_provider_turnstile.NewProvider(captcha_provider_turnstile.Config{
		SiteKey:   "1x00000000000000000000AA",
		SecretKey: "1x0000000000000000000000000000000AA",
	})
	if err != nil {
		panic(err)
	}

	turnstileFailProvider, err := captcha_provider_turnstile.NewProvider(captcha_provider_turnstile.Config{
		SiteKey:   "2x00000000000000000000AB",
		SecretKey: "2x0000000000000000000000000000000AB",
	})
	if err != nil {
		panic(err)
	}

	recaptchaPassProvider, err := captcha_provider_recaptcha_v3.NewProvider(captcha_provider_recaptcha_v3.Config{
		SiteKey:   "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI",
		SecretKey: "6LeIxAcTAAAAAGG-vFI1TnRWxMZNFuojJ4WifJWe",
	})
	if err != nil {
		panic(err)
	}

	hcaptchaPassProvider, err := captcha_provider_hcaptcha.NewProvider(captcha_provider_hcaptcha.Config{
		SiteKey:   "10000000-ffff-ffff-ffff-000000000001",
		SecretKey: "0x0000000000000000000000000000000000000000",
	})
	if err != nil {
		panic(err)
	}

	hcaptchaFailProvider, err := captcha_provider_hcaptcha.NewProvider(captcha_provider_hcaptcha.Config{
		SiteKey:   "20000000-ffff-ffff-ffff-000000000002",
		SecretKey: "0x0000000000000000000000000000000000000000",
	})
	if err != nil {
		panic(err)
	}

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
		piko.WithCaptchaProvider("hmac_challenge", hmacProvider),
		piko.WithCaptchaProvider("turnstile_pass", turnstilePassProvider),
		piko.WithCaptchaProvider("turnstile_fail", turnstileFailProvider),
		piko.WithCaptchaProvider("recaptcha_pass", recaptchaPassProvider),
		piko.WithCaptchaProvider("hcaptcha_pass", hcaptchaPassProvider),
		piko.WithCaptchaProvider("hcaptcha_fail", hcaptchaFailProvider),
		piko.WithDefaultCaptchaProvider("hmac_challenge"),
	)
	if err := server.Run(command); err != nil {
		panic(err)
	}
}
