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

package i18n_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/maths"
)

func getTranslation(store *Store, locale, key string, pool *StrBufPool) *Translation {
	entry, found := store.Get(locale, key)
	if !found {
		entry, found = store.Get(store.DefaultLocale(), key)
	}
	if !found {
		return NewTranslationFromString(key, key, pool)
	}
	return NewTranslationWithLocale(key, entry, pool, locale)
}

func TestIntegration_FullPipeline(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting":     "Hello, ${name}!",
		"cart.items":   "You have one item|You have ${count} items",
		"order.total":  "Your total is ${total}",
		"product.info": "${name}: ${price} (${quantity} in stock)",
	})
	store.AddTranslations("fr-FR", map[string]string{
		"greeting": "Bonjour, ${name}!",
	})

	pool := NewStrBufPool(256)

	t.Run("simple greeting", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "greeting", pool)
		trans.StringVar("name", "Alice")
		result := trans.String()
		assert.Equal(t, "Hello, Alice!", result)
	})

	t.Run("plural singular", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "cart.items", pool)
		trans.Count(1)
		result := trans.String()
		assert.Equal(t, "You have one item", result)
	})

	t.Run("plural many", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "cart.items", pool)
		trans.Count(5)
		result := trans.String()
		assert.Equal(t, "You have 5 items", result)
	})

	t.Run("multiple variables", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "product.info", pool)
		trans.StringVar("name", "Widget")
		trans.FloatVar("price", 19.99)
		trans.IntVar("quantity", 42)
		result := trans.String()
		assert.Equal(t, "Widget: 19.99 (42 in stock)", result)
	})

	t.Run("locale fallback", func(t *testing.T) {
		trans := getTranslation(store, "fr-FR", "greeting", pool)
		trans.StringVar("name", "Pierre")
		assert.Equal(t, "Bonjour, Pierre!", trans.String())

		trans = getTranslation(store, "fr-FR", "cart.items", pool)
		trans.Count(3)
		assert.Equal(t, "You have 3 items", trans.String())
	})

	t.Run("missing key with fallback", func(t *testing.T) {
		trans := NewTranslationFromString("missing.key", "Fallback text", pool)
		result := trans.String()
		assert.Equal(t, "Fallback text", result)
	})

	t.Run("missing key without fallback", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "missing.key", pool)
		result := trans.String()
		assert.Equal(t, "missing.key", result)
	})
}

func TestIntegration_DecimalType(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"price":       "Price: ${amount}",
		"calculation": "Result: ${value}",
	})
	pool := NewStrBufPool(256)

	t.Run("simple decimal", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "price", pool)
		trans.DecimalVar("amount", maths.NewDecimalFromString("19.99"))
		result := trans.String()
		assert.Equal(t, "Price: 19.99", result)
	})

	t.Run("decimal from int", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "price", pool)
		trans.DecimalVar("amount", maths.NewDecimalFromInt(100))
		result := trans.String()
		assert.Equal(t, "Price: 100", result)
	})

	t.Run("decimal calculation", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "calculation", pool)
		value := maths.NewDecimalFromString("10.5").Multiply(maths.NewDecimalFromInt(3))
		trans.DecimalVar("value", value)
		result := trans.String()
		assert.Equal(t, "Result: 31.5", result)
	})

	t.Run("high precision decimal", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "price", pool)
		trans.DecimalVar("amount", maths.NewDecimalFromString("3.14159265358979323846"))
		result := trans.String()
		assert.Contains(t, result, "3.14159265358979323846")
	})
}

func TestIntegration_MoneyType(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"total":    "Total: ${amount}",
		"discount": "You saved: ${savings}",
	})
	pool := NewStrBufPool(256)

	t.Run("money GBP", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "total", pool)
		trans.MoneyVar("amount", maths.NewMoneyFromString("99.99", "GBP"))
		result := trans.String()
		assert.Contains(t, result, "99.99")
	})

	t.Run("money USD", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "total", pool)
		trans.MoneyVar("amount", maths.NewMoneyFromString("49.99", "USD"))
		result := trans.String()
		assert.Contains(t, result, "49.99")
	})

	t.Run("money from int", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "total", pool)
		trans.MoneyVar("amount", maths.NewMoneyFromInt(100, "GBP"))
		result := trans.String()
		assert.Contains(t, result, "100")
	})

	t.Run("money arithmetic", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "discount", pool)
		original := maths.NewMoneyFromString("100.00", "GBP")
		discounted := maths.NewMoneyFromString("80.00", "GBP")
		savings := original.Subtract(discounted)
		trans.MoneyVar("savings", savings)
		result := trans.String()
		assert.Contains(t, result, "20")
	})
}

func TestIntegration_BigIntType(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"count":  "Total: ${n}",
		"result": "Calculation: ${value}",
	})
	pool := NewStrBufPool(256)

	t.Run("simple bigint", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "count", pool)
		trans.BigIntVar("n", maths.NewBigIntFromInt(1000000))
		result := trans.String()
		assert.Equal(t, "Total: 1000000", result)
	})

	t.Run("large bigint", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "count", pool)
		trans.BigIntVar("n", maths.NewBigIntFromString("9999999999999999999999999999"))
		result := trans.String()
		assert.Equal(t, "Total: 9999999999999999999999999999", result)
	})

	t.Run("bigint arithmetic", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "result", pool)
		value := maths.NewBigIntFromInt(1000).Multiply(maths.NewBigIntFromInt(1000))
		trans.BigIntVar("value", value)
		result := trans.String()
		assert.Equal(t, "Calculation: 1000000", result)
	})
}

func TestIntegration_MixedTypes(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"order.summary": "${customer} ordered ${quantity} x ${product} for ${total}",
		"complex":       "String: ${s}, Int: ${i}, Float: ${f}, Decimal: ${d}, Money: ${m}, BigInt: ${b}",
	})
	pool := NewStrBufPool(256)

	t.Run("order summary", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "order.summary", pool)
		trans.StringVar("customer", "Alice")
		trans.IntVar("quantity", 3)
		trans.StringVar("product", "Widget")
		trans.MoneyVar("total", maths.NewMoneyFromString("59.97", "GBP"))
		result := trans.String()
		assert.Contains(t, result, "Alice")
		assert.Contains(t, result, "3")
		assert.Contains(t, result, "Widget")
		assert.Contains(t, result, "59.97")
	})

	t.Run("all types", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "complex", pool)
		trans.StringVar("s", "hello")
		trans.IntVar("i", 42)
		trans.FloatVar("f", 3.14)
		trans.DecimalVar("d", maths.NewDecimalFromString("1.23456"))
		trans.MoneyVar("m", maths.NewMoneyFromString("9.99", "GBP"))
		trans.BigIntVar("b", maths.NewBigIntFromString("12345678901234567890"))
		result := trans.String()

		assert.Contains(t, result, "hello")
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "3.14")
		assert.Contains(t, result, "1.23456")
		assert.Contains(t, result, "9.99")
		assert.Contains(t, result, "12345678901234567890")
	})
}

func TestIntegration_BufferReuse(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, ${name}!",
	})
	pool := NewStrBufPool(256)

	trans1 := getTranslation(store, "en-GB", "greeting", pool)
	trans1.StringVar("name", "Alice")
	result1 := trans1.String()
	assert.Equal(t, "Hello, Alice!", result1)

	trans2 := getTranslation(store, "en-GB", "greeting", pool)
	trans2.StringVar("name", "Bob")
	result2 := trans2.String()
	assert.Equal(t, "Hello, Bob!", result2)

	assert.NotEqual(t, result1, result2)
}

func TestIntegration_StrBufPool(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, ${name}!",
	})
	pool := NewStrBufPool(256)

	for range 100 {
		trans := getTranslation(store, "en-GB", "greeting", pool)
		trans.StringVar("name", "User")
		result := trans.String()
		assert.Equal(t, "Hello, User!", result)
	}
}

func TestIntegration_ComplexPluralisation(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"items": "one item|${count} items",
	})
	store.AddTranslations("ru-RU", map[string]string{
		"items": "${count} яблоко|${count} яблока|${count} яблок",
	})
	pool := NewStrBufPool(256)

	t.Run("english plural", func(t *testing.T) {
		tests := []struct {
			expected string
			count    int
		}{
			{count: 0, expected: "0 items"},
			{count: 1, expected: "one item"},
			{count: 2, expected: "2 items"},
			{count: 100, expected: "100 items"},
		}

		for _, tc := range tests {
			trans := getTranslation(store, "en-GB", "items", pool)
			trans.Count(tc.count)
			result := trans.String()
			assert.Equal(t, tc.expected, result, "count=%d", tc.count)
		}
	})

	t.Run("russian plural", func(t *testing.T) {
		tests := []struct {
			expected string
			count    int
		}{
			{count: 1, expected: "1 яблоко"},
			{count: 2, expected: "2 яблока"},
			{count: 5, expected: "5 яблок"},
			{count: 21, expected: "21 яблоко"},
			{count: 22, expected: "22 яблока"},
		}

		for _, tc := range tests {
			trans := getTranslation(store, "ru-RU", "items", pool)
			trans.Count(tc.count)
			result := trans.String()
			assert.Equal(t, tc.expected, result, "count=%d", tc.count)
		}
	})
}

func TestIntegration_FluentChaining(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"receipt": "${customer} bought ${quantity} x ${item} for ${total} on ${date}",
	})
	pool := NewStrBufPool(256)

	result := getTranslation(store, "en-GB", "receipt", pool).
		StringVar("customer", "Alice").
		IntVar("quantity", 2).
		StringVar("item", "Widget").
		MoneyVar("total", maths.NewMoneyFromString("39.98", "GBP")).
		StringVar("date", "2024-01-15").
		String()

	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "2")
	assert.Contains(t, result, "Widget")
	assert.Contains(t, result, "39.98")
	assert.Contains(t, result, "2024-01-15")
}

func TestIntegration_ErrorHandling(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"value": "Value: ${v}",
	})
	pool := NewStrBufPool(256)

	t.Run("decimal error", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "value", pool)
		errDecimal := maths.NewDecimalFromString("invalid")
		require.Error(t, errDecimal.Err())
		trans.DecimalVar("v", errDecimal)
		result := trans.String()
		assert.Equal(t, "Value: 0", result)
	})

	t.Run("bigint error", func(t *testing.T) {
		trans := getTranslation(store, "en-GB", "value", pool)
		errBigInt := maths.NewBigIntFromString("not_a_number")
		require.Error(t, errBigInt.Err())
		trans.BigIntVar("v", errBigInt)
		result := trans.String()
		assert.Equal(t, "Value: 0", result)
	})
}

func BenchmarkIntegration_SimpleTranslation(b *testing.B) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, ${name}!",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en-GB", "greeting")
	b.ResetTimer()

	for b.Loop() {
		trans := NewTranslationWithLocale("greeting", entry, pool, "en-GB")
		trans.StringVar("name", "Alice")
		_ = trans.String()
	}
}

func BenchmarkIntegration_WithDecimal(b *testing.B) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"price": "Price: ${amount}",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en-GB", "price")
	price := maths.NewDecimalFromString("19.99")
	b.ResetTimer()

	for b.Loop() {
		trans := NewTranslationWithLocale("price", entry, pool, "en-GB")
		trans.DecimalVar("amount", price)
		_ = trans.String()
	}
}

func BenchmarkIntegration_WithMoney(b *testing.B) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"total": "Total: ${amount}",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en-GB", "total")
	amount := maths.NewMoneyFromString("99.99", "GBP")
	b.ResetTimer()

	for b.Loop() {
		trans := NewTranslationWithLocale("total", entry, pool, "en-GB")
		trans.MoneyVar("amount", amount)
		_ = trans.String()
	}
}

func BenchmarkIntegration_ComplexTemplate(b *testing.B) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"order": "${customer} ordered ${quantity} x ${product} for ${total}",
	})
	pool := NewStrBufPool(256)
	entry, _ := store.Get("en-GB", "order")
	total := maths.NewMoneyFromString("59.97", "GBP")
	b.ResetTimer()

	for b.Loop() {
		trans := NewTranslationWithLocale("order", entry, pool, "en-GB")
		trans.StringVar("customer", "Alice")
		trans.IntVar("quantity", 3)
		trans.StringVar("product", "Widget")
		trans.MoneyVar("total", total)
		_ = trans.String()
	}
}
