---
title: Linguistics API
description: Tokenisation, normalisation, stemming, phonetic encoding, and fuzzy matching for text analysis.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 250
---

# Linguistics API

The linguistics package provides text analysis primitives used by the cache search layer and available to user code for light-weight NLP (tokenising, stemming, phonetic encoding, fuzzy match). Language support covers English, Dutch, French, German, Hungarian, Norwegian, Russian, Spanish, and Swedish. Source of truth: [`wdk/linguistics/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/linguistics/facade.go).

## Analyser

```go
func DefaultConfig() AnalyserConfig
func DefaultConfigForLanguage(language string) AnalyserConfig
func NewAnalyser(config AnalyserConfig, opts ...Option) *Analyser
func NewTokeniser(config AnalyserConfig) *Tokeniser
func NewNormaliser(preserveCase bool) *Normaliser
func NewPhoneticEncoder(maxLength int) *PhoneticEncoder
```

## Options

```go
func WithStemmer(s StemmerPort) Option
func WithTokeniser(t TokeniserPort) Option
func WithPhoneticEncoder(p PhoneticEncoderPort) Option
func WithStopWordsProvider(p StopWordsProviderPort) Option
func WithLanguage(language string) Option
```

## Factories

Register or look up per-language backends:

```go
func RegisterStemmerFactory(language string, factory func() (StemmerPort, error))
func RegisterPhoneticEncoderFactory(language string, factory func() (PhoneticEncoderPort, error))
func RegisterStopWordsProviderFactory(name string, factory func() (StopWordsProviderPort, error))
func CreateStemmer(language string) StemmerPort
func CreatePhoneticEncoder(language string) PhoneticEncoderPort
func CreateStopWordsProvider(name string) StopWordsProviderPort
func SupportedLanguages() []string
func ValidateLanguage(language string) string
```

Use the no-op variants (`NewNoOpStemmer`, `NewNoOpPhoneticEncoder`, `NewNoOpStopWordsProvider`) when a pipeline needs structural compatibility without transformation.

## String similarity

```go
func Jaro(a, b string) float64
func JaroWinkler(a, b string, boostThreshold float64, prefixSize int) float64
func FuzzyMatch(text, pattern string, threshold float64, caseSensitive bool) (bool, float64)
func SoundexEncode(word string) string
```

## Constants

| Group | Values |
|---|---|
| Languages | `LanguageEnglish`, `LanguageDutch`, `LanguageGerman`, `LanguageSpanish`, `LanguageFrench`, `LanguageRussian`, `LanguageSwedish`, `LanguageNorwegian`, `LanguageHungarian` |
| Modes | `AnalysisModeBasic`, `AnalysisModeFast`, `AnalysisModeSmart` |
| Defaults | `DefaultMinTokenLength`, `DefaultMaxTokenLength`, `DefaultPhoneticCodeLength` |

## Sub-packages

`linguistics_language_*` packages supply stemmers, phonetic encoders, and stop-word tables per language. Import the language packages the application needs so their factories register themselves on `init`.

## See also

- [Cache API reference](cache-api.md) for how search uses these analysers.
