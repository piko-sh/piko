// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

//go:build bench

package i18n_test_bench

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/i18n/i18n_adapters"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/wdk/maths"
	"piko.sh/piko/wdk/safedisk"
)

type testCase struct {
	name string
	path string
}

func setupBenchmark(b *testing.B, tc testCase) config.ServerConfig {
	b.Helper()
	i18nDir := filepath.Join(tc.path, "i18n")
	absI18nDir, err := filepath.Abs(i18nDir)
	require.NoError(b, err)

	baseDir := filepath.Dir(absI18nDir)

	serverConfig := config.ServerConfig{
		I18nDefaultLocale: new("en"),
		Paths: config.PathsConfig{
			BaseDir:       &baseDir,
			I18nSourceDir: new("i18n"),
		},
	}

	return serverConfig
}

func setupStore() *i18n_domain.Store {
	store := i18n_domain.NewStore("en-GB")

	store.AddTranslations("en-GB", map[string]string{

		"common.hello":       "Hello",
		"common.welcome":     "Welcome to our application",
		"common.goodbye":     "Goodbye",
		"navigation.home":    "Home",
		"navigation.about":   "About Us",
		"navigation.contact": "Contact",

		"greeting.personal": "Hello, ${name}!",
		"greeting.formal":   "Dear ${title} ${lastName},",
		"order.summary":     "Order #${orderId}: ${itemCount} items totalling ${total}",
		"user.profile":      "User: ${username} (${email})",
		"dashboard.welcome": "Welcome back, ${name}! You have ${notifications} new notifications.",
		"invoice.header":    "Invoice #${invoiceId} for ${customerName} dated ${date}",

		"items.count":         "one item|${count} items",
		"notifications.count": "You have one notification|You have ${count} notifications",
		"cart.items":          "Your cart contains one item|Your cart contains ${count} items",
		"files.selected":      "${count} file selected|${count} files selected",
		"messages.unread":     "one unread message|${count} unread messages",

		"common.brand":     "Piko WDK",
		"footer.copyright": "© 2025 @common.brand. All rights reserved.",
		"header.title":     "@common.brand - Dashboard",

		"stats.percentage": "Completion: ${percentage}%",
		"price.formatted":  "Price: ${price}",
		"date.event":       "Event on ${eventDate}",

		"email.confirmation": "Dear ${customerName}, your order #${orderId} has been confirmed. " +
			"Total: ${total}. Expected delivery: ${deliveryDate}. " +
			"Thank you for shopping with @common.brand!",
	})

	store.AddTranslations("de-DE", map[string]string{
		"common.hello":   "Hallo",
		"common.welcome": "Willkommen in unserer Anwendung",
		"items.count":    "ein Artikel|${count} Artikel",
	})

	store.AddTranslations("ru-RU", map[string]string{
		"items.count": "${count} яблоко|${count} яблока|${count} яблок",
	})

	store.AddTranslations("ar", map[string]string{
		"items.count": "صفر|واحد|اثنان|قليل|كثير|آخر",
	})

	return store
}

func BenchmarkFSService_Loading(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			name: entry.Name(),
			path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.name, func(b *testing.B) {
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				b.StopTimer()
				serverConfig := setupBenchmark(b, tc)
				testSandbox, _ := safedisk.NewNoOpSandbox(*serverConfig.Paths.BaseDir, safedisk.ModeReadOnly)
				b.StartTimer()

				service, err := i18n_adapters.NewFSService(ctx, testSandbox, *serverConfig.I18nDefaultLocale, *serverConfig.Paths.I18nSourceDir)

				b.StopTimer()
				testSandbox.Close()
				require.NoError(b, err, "NewFSService failed unexpectedly")
				require.NotNil(b, service, "Service should not be nil")
				b.StartTimer()
			}
		})
	}
}

func BenchmarkStore_Get(b *testing.B) {
	store := setupStore()

	keys := []string{
		"common.hello",
		"greeting.personal",
		"items.count",
		"email.confirmation",
	}

	b.Run("existing_key", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			key := keys[i%len(keys)]
			entry, found := store.Get("en-GB", key)
			if !found || entry == nil {
				b.Fatal("Expected to find entry")
			}
			i++
		}
	})

	b.Run("missing_key", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, found := store.Get("en-GB", "nonexistent.key")
			if found {
				b.Fatal("Expected not to find entry")
			}
		}
	})

	b.Run("fallback_locale", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {

			entry, found := store.Get("en-AU", "common.hello")
			if !found || entry == nil {
				b.Fatal("Expected to find entry via fallback")
			}
		}
	})
}

func BenchmarkStore_ResolveMessage(b *testing.B) {
	store := setupStore()
	scope := map[string]any{"name": "Test User"}

	b.Run("simple_message", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result, found := store.ResolveMessage("common.brand", "en-GB", scope, 0)
			if !found || result == "" {
				b.Fatal("Expected to resolve message")
			}
		}
	})

	b.Run("with_interpolation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result, found := store.ResolveMessage("greeting.personal", "en-GB", scope, 0)
			if !found || result == "" {
				b.Fatal("Expected to resolve message")
			}
		}
	})
}

func BenchmarkParseTemplate(b *testing.B) {
	b.Run("literal_only", func(b *testing.B) {
		template := "Hello, World! This is a simple template."
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			parts, errs := i18n_domain.ParseTemplate(template)
			if len(errs) > 0 || len(parts) == 0 {
				b.Fatal("Parse failed")
			}
		}
	})

	b.Run("single_expression", func(b *testing.B) {
		template := "Hello, ${name}!"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			parts, errs := i18n_domain.ParseTemplate(template)
			if len(errs) > 0 || len(parts) == 0 {
				b.Fatal("Parse failed")
			}
		}
	})

	b.Run("multiple_expressions", func(b *testing.B) {
		template := "Dear ${title} ${lastName}, your order #${orderId} is ready."
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			parts, errs := i18n_domain.ParseTemplate(template)
			if len(errs) > 0 || len(parts) == 0 {
				b.Fatal("Parse failed")
			}
		}
	})

	b.Run("with_linked_message", func(b *testing.B) {
		template := "Welcome to @common.brand and @contact.email"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			parts, errs := i18n_domain.ParseTemplate(template)
			if len(errs) > 0 || len(parts) == 0 {
				b.Fatal("Parse failed")
			}
		}
	})

	b.Run("complex_mixed", func(b *testing.B) {
		template := "Dear ${customerName}, your order #${orderId} totalling ${total} " +
			"will be delivered on ${deliveryDate}. Thank you!"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			parts, errs := i18n_domain.ParseTemplate(template)
			if len(errs) > 0 || len(parts) == 0 {
				b.Fatal("Parse failed")
			}
		}
	})
}

func BenchmarkRender(b *testing.B) {
	store := setupStore()
	buffer := i18n_domain.NewStrBuf(256)

	b.Run("literal_only", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "common.hello")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.Render(entry, nil, nil, "en-GB", buffer)
			if result == "" {
				b.Fatal("Render failed")
			}
		}
	})

	b.Run("single_var", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "greeting.personal")
		vars := map[string]any{"name": "John"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.Render(entry, vars, nil, "en-GB", buffer)
			if result == "" {
				b.Fatal("Render failed")
			}
		}
	})

	b.Run("multiple_vars", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "order.summary")
		vars := map[string]any{
			"orderId":   "ORD-12345",
			"itemCount": 5,
			"total":     "$149.99",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.Render(entry, vars, nil, "en-GB", buffer)
			if result == "" {
				b.Fatal("Render failed")
			}
		}
	})

	b.Run("with_plural_singular", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "items.count")
		count := 1
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.Render(entry, nil, &count, "en-GB", buffer)
			if result == "" {
				b.Fatal("Render failed")
			}
		}
	})

	b.Run("with_plural_many", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "items.count")
		vars := map[string]any{"count": 42}
		count := 42
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.Render(entry, vars, &count, "en-GB", buffer)
			if result == "" {
				b.Fatal("Render failed")
			}
		}
	})

	b.Run("complex_template", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "email.confirmation")
		vars := map[string]any{
			"customerName": "John Smith",
			"orderId":      "ORD-98765",
			"total":        "$299.99",
			"deliveryDate": "January 15, 2025",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.Render(entry, vars, nil, "en-GB", buffer)
			if result == "" {
				b.Fatal("Render failed")
			}
		}
	})
}

func BenchmarkSelectPluralForm(b *testing.B) {
	twoForms := []string{"one item", "${count} items"}
	threeForms := []string{"${count} яблоко", "${count} яблока", "${count} яблок"}
	sixForms := []string{"صفر", "واحد", "اثنان", "قليل", "كثير", "آخر"}

	b.Run("english_singular", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.SelectPluralForm(1, "en", twoForms)
			if result == "" {
				b.Fatal("SelectPluralForm failed")
			}
		}
	})

	b.Run("english_plural", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.SelectPluralForm(42, "en", twoForms)
			if result == "" {
				b.Fatal("SelectPluralForm failed")
			}
		}
	})

	b.Run("russian_complex", func(b *testing.B) {
		counts := []int{1, 2, 5, 21, 22, 25}
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			count := counts[i%len(counts)]
			result := i18n_domain.SelectPluralForm(count, "ru", threeForms)
			if result == "" {
				b.Fatal("SelectPluralForm failed")
			}
			i++
		}
	})

	b.Run("arabic_six_forms", func(b *testing.B) {
		counts := []int{0, 1, 2, 5, 11, 100}
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			count := counts[i%len(counts)]
			result := i18n_domain.SelectPluralForm(count, "ar", sixForms)
			if result == "" {
				b.Fatal("SelectPluralForm failed")
			}
			i++
		}
	})
}

func BenchmarkSplitPluralForms(b *testing.B) {
	b.Run("two_forms", func(b *testing.B) {
		template := "one item|${count} items"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			forms := i18n_domain.SplitPluralForms(template)
			if len(forms) != 2 {
				b.Fatal("Expected 2 forms")
			}
		}
	})

	b.Run("three_forms", func(b *testing.B) {
		template := "${count} яблоко|${count} яблока|${count} яблок"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			forms := i18n_domain.SplitPluralForms(template)
			if len(forms) != 3 {
				b.Fatal("Expected 3 forms")
			}
		}
	})

	b.Run("with_escaped_pipe", func(b *testing.B) {
		template := "value || pipe|other form"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			forms := i18n_domain.SplitPluralForms(template)
			if len(forms) != 2 {
				b.Fatal("Expected 2 forms")
			}
		}
	})
}

func BenchmarkHasPluralForms(b *testing.B) {
	b.Run("has_plural", func(b *testing.B) {
		template := "one item|${count} items"
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if !i18n_domain.HasPluralForms(template) {
				b.Fatal("Expected to have plural forms")
			}
		}
	})

	b.Run("no_plural", func(b *testing.B) {
		template := "Hello, ${name}! Welcome to our app."
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if i18n_domain.HasPluralForms(template) {
				b.Fatal("Expected no plural forms")
			}
		}
	})
}

func BenchmarkFormatDateTime(b *testing.B) {
	testTime := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)

	b.Run("short_en_GB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "en-GB", i18n_domain.DateTimeStyleShort, false, false)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})

	b.Run("medium_en_GB", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "en-GB", i18n_domain.DateTimeStyleMedium, false, false)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})

	b.Run("long_en_US", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "en-US", i18n_domain.DateTimeStyleLong, false, false)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})

	b.Run("full_de_DE", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "de-DE", i18n_domain.DateTimeStyleFull, false, false)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})

	b.Run("date_only", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "en-GB", i18n_domain.DateTimeStyleShort, true, false)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})

	b.Run("time_only", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "en-GB", i18n_domain.DateTimeStyleShort, false, true)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})

	b.Run("locale_fallback", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result := i18n_domain.FormatDateTime(testTime, "en-AU", i18n_domain.DateTimeStyleMedium, false, false)
			if result == "" {
				b.Fatal("FormatDateTime failed")
			}
		}
	})
}

func BenchmarkDateTime_FluentAPI(b *testing.B) {
	testTime := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)

	b.Run("create_and_format", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			dt := i18n_domain.NewDateTime(testTime)
			result := dt.Format("en-GB")
			if result == "" {
				b.Fatal("DateTime.Format failed")
			}
		}
	})

	b.Run("chained_style", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			dt := i18n_domain.NewDateTime(testTime).Short().DateOnly()
			result := dt.Format("en-GB")
			if result == "" {
				b.Fatal("DateTime.Format failed")
			}
		}
	})

	b.Run("with_utc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			dt := i18n_domain.NewDateTime(testTime).UTC().Long()
			result := dt.Format("en-GB")
			if result == "" {
				b.Fatal("DateTime.Format failed")
			}
		}
	})
}

func BenchmarkTranslation_String(b *testing.B) {
	store := setupStore()
	pool := i18n_domain.NewStrBufPool(256)

	b.Run("literal_only", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "common.hello")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslation("common.hello", entry, pool)
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})

	b.Run("with_string_var", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "greeting.personal")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslation("greeting.personal", entry, pool).
				StringVar("name", "John")
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})

	b.Run("with_int_var", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "notifications.count")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslationWithLocale("notifications.count", entry, pool, "en-GB").
				IntVar("count", 5).
				Count(5)
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})

	b.Run("with_multiple_vars", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "order.summary")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslation("order.summary", entry, pool).
				StringVar("orderId", "ORD-12345").
				IntVar("itemCount", 5).
				StringVar("total", "$149.99")
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})

	b.Run("with_datetime_var", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "date.event")
		testTime := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslationWithLocale("date.event", entry, pool, "en-GB").
				DateTimeVar("eventDate", i18n_domain.NewDateTime(testTime).Long().DateOnly())
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})

	b.Run("with_money_var", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "price.formatted")
		price := maths.NewMoneyFromFloat(149.99, "GBP")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslationWithLocale("price.formatted", entry, pool, "en-GB").
				MoneyVar("price", price)
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})

	b.Run("complex_email", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "email.confirmation")
		deliveryDate := i18n_domain.NewDateTime(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)).
			Long().DateOnly()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			trans := i18n_domain.NewTranslationWithLocale("email.confirmation", entry, pool, "en-GB").
				StringVar("customerName", "John Smith").
				StringVar("orderId", "ORD-98765").
				StringVar("total", "$299.99").
				DateTimeVar("deliveryDate", deliveryDate)
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
		}
	})
}

func BenchmarkTranslation_Count(b *testing.B) {
	store := setupStore()
	pool := i18n_domain.NewStrBufPool(256)

	counts := []int{0, 1, 2, 5, 10, 21, 22, 100}

	b.Run("english_varying", func(b *testing.B) {
		entry, _ := store.Get("en-GB", "items.count")
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			count := counts[i%len(counts)]
			trans := i18n_domain.NewTranslationWithLocale("items.count", entry, pool, "en-GB").
				IntVar("count", count).
				Count(count)
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
			i++
		}
	})

	b.Run("russian_varying", func(b *testing.B) {
		entry, _ := store.Get("ru-RU", "items.count")
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			count := counts[i%len(counts)]
			trans := i18n_domain.NewTranslationWithLocale("items.count", entry, pool, "ru-RU").
				IntVar("count", count).
				Count(count)
			result := trans.String()
			if result == "" {
				b.Fatal("Translation.String failed")
			}
			i++
		}
	})
}

func BenchmarkStrBuf_Operations(b *testing.B) {
	b.Run("write_string_short", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(64)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteString("Hello")
		}
	})

	b.Run("write_string_long", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(256)
		longString := "This is a much longer string that contains more content for testing purposes."
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteString(longString)
		}
	})

	b.Run("write_int", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(64)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteInt(12345)
		}
	})

	b.Run("write_int64", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(64)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteInt64(9223372036854775807)
		}
	})

	b.Run("write_float", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(64)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteFloat(3.14159265359)
		}
	})

	b.Run("write_mixed", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(256)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteString("Order #")
			buffer.WriteInt(12345)
			buffer.WriteString(": Total $")
			buffer.WriteFloat(149.99)
			buffer.WriteString(" (")
			buffer.WriteInt(5)
			buffer.WriteString(" items)")
		}
	})

	b.Run("write_any_string", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(64)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteAny("Hello")
		}
	})

	b.Run("write_any_int", func(b *testing.B) {
		buffer := i18n_domain.NewStrBuf(64)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer.Reset()
			buffer.WriteAny(12345)
		}
	})
}

func BenchmarkStrBufPool(b *testing.B) {
	pool := i18n_domain.NewStrBufPool(256)

	b.Run("get_put", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			buffer := pool.Get()
			buffer.WriteString("Hello, World!")
			pool.Put(buffer)
		}
	})

	b.Run("get_use_put", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			buffer := pool.Get()
			buffer.WriteString("Order #")
			buffer.WriteInt(i)
			buffer.WriteString(": $")
			buffer.WriteFloat(149.99)
			_ = buffer.String()
			pool.Put(buffer)
			i++
		}
	})
}

func BenchmarkStrBuf_String(b *testing.B) {
	buffer := i18n_domain.NewStrBuf(256)
	buffer.WriteString("Hello, World! This is a test string for benchmarking.")

	b.Run("safe_string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = buffer.String()
		}
	})

	b.Run("unsafe_string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = buffer.UnsafeString()
		}
	})
}

func BenchmarkStore_GetParallel(b *testing.B) {
	store := setupStore()
	keys := []string{
		"common.hello",
		"greeting.personal",
		"items.count",
		"order.summary",
		"email.confirmation",
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := keys[i%len(keys)]
			_, _ = store.Get("en-GB", key)
			i++
		}
	})
}

func BenchmarkTranslation_StringParallel(b *testing.B) {
	store := setupStore()
	pool := i18n_domain.NewStrBufPool(256)
	entry, _ := store.Get("en-GB", "greeting.personal")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			trans := i18n_domain.NewTranslation("greeting.personal", entry, pool).
				StringVar("name", "John")
			_ = trans.String()
		}
	})
}

func BenchmarkStrBufPool_Parallel(b *testing.B) {
	pool := i18n_domain.NewStrBufPool(256)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get()
			buffer.WriteString("Hello, ")
			buffer.WriteString("World!")
			_ = buffer.String()
			pool.Put(buffer)
		}
	})
}
