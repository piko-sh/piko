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
func DefaultStopWords(language string) map[string]bool
func RegisteredStemmerFactories() []string
func RegisteredPhoneticEncoderFactories() []string
func RegisteredStopWordsProviderFactories() []string
```

`DefaultStopWords` returns the built-in stop word set for a language. `RegisteredStemmerFactories`, `RegisteredPhoneticEncoderFactories`, and `RegisteredStopWordsProviderFactories` enumerate the languages or names currently registered.

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

Per-language backends ship as four parallel series, one for each capability. Import only the series and languages your application actually uses. Each blank import registers its factories on `init`:

| Series | Purpose | Example import |
|---|---|---|
| `linguistics_language_<lang>` | Bundle that wires the language's stemmer, phonetic encoder, and stop-words for use with `DefaultConfigForLanguage`. | `_ "piko.sh/piko/wdk/linguistics/linguistics_language_english"` |
| `linguistics_stemmer_<lang>` | Standalone stemmer. | `_ "piko.sh/piko/wdk/linguistics/linguistics_stemmer_french"` |
| `linguistics_phonetic_<lang>` | Standalone phonetic encoder. | `_ "piko.sh/piko/wdk/linguistics/linguistics_phonetic_german"` |
| `linguistics_stopwords_<lang>` | Standalone stop-words provider. | `_ "piko.sh/piko/wdk/linguistics/linguistics_stopwords_spanish"` |

Each series ships ten variants. Dutch, English, French, German, Hungarian, Norwegian, Russian, Spanish, Swedish, plus a `_mock` variant for testing. The bundled `linguistics_language_<lang>` is a convenience that pulls in the matching stemmer, phonetic, and stop-words registrations together. Pick the standalone series instead when you need only one capability or want to swap implementations independently.

## See also

- [Cache API reference](cache-api.md) for how search uses these analysers.
