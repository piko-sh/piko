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

package linguistics_domain

const (
	// DefaultMinTokenLength is the smallest token length in characters.
	DefaultMinTokenLength = 2

	// DefaultMaxTokenLength is the longest token length to process.
	DefaultMaxTokenLength = 50

	// DefaultPhoneticCodeLength is the standard length of codes that the phonetic
	// encoder produces.
	DefaultPhoneticCodeLength = 4
)

// Token represents a single analysed token from text with its various forms
// and position.
type Token struct {
	// Original is the token as it appeared in the source text.
	Original string

	// Normalised is the lowercase form with accents and special marks removed.
	Normalised string

	// Stemmed is the root form of the word (e.g., "running" becomes "run").
	// Only populated when using a stemmer.
	Stemmed string

	// Phonetic is the Double Metaphone encoding of the token.
	// Only set when phonetic encoding is enabled.
	Phonetic string

	// Position is the offset of this token in the original text, measured in
	// tokens rather than bytes.
	Position int

	// ByteOffset is the starting byte position in the original text.
	ByteOffset int

	// ByteLength is the length in bytes of the original token.
	ByteLength int
}

// AnalysisMode specifies which text analysis methods to use.
type AnalysisMode int

const (
	// AnalysisModeBasic performs only tokenisation and normalisation. This is the
	// fastest option, suitable for exact and prefix matching.
	AnalysisModeBasic AnalysisMode = iota

	// AnalysisModeFast performs tokenisation, normalisation, and stop word
	// filtering. Good balance of performance and quality for most use cases.
	AnalysisModeFast

	// AnalysisModeSmart performs full linguistic analysis including stemming and
	// phonetics. Best for fuzzy matching and handling misspellings.
	AnalysisModeSmart
)

// AnalyserConfig configures the text analysis pipeline.
// Fields ordered for optimal memory alignment (larger types first).
type AnalyserConfig struct {
	// StopWords is a set of words to filter out (e.g., "the", "and", "is").
	// Only used in Fast and Smart modes.
	StopWords map[string]bool

	// Language specifies the language for stemming, defaulting
	// to "english".
	Language string

	// MinTokenLength filters out tokens shorter than this value; 0 means no
	// minimum.
	MinTokenLength int

	// MaxTokenLength sets the maximum length for tokens; 0 means no limit.
	MaxTokenLength int

	// Mode specifies which analysis techniques to apply during text processing.
	Mode AnalysisMode

	// PreserveCase prevents lowercasing when true; useful for code identifiers.
	PreserveCase bool
}

// DefaultConfig returns a default configuration for general text analysis.
//
// Returns AnalyserConfig which provides sensible defaults for English text.
func DefaultConfig() AnalyserConfig {
	return DefaultConfigForLanguage(LanguageEnglish)
}

// DefaultConfigForLanguage returns a default configuration for a given
// language. The stop words are set based on the language code.
//
// Takes language (string) which specifies the language code for the analyser.
//
// Returns AnalyserConfig which contains the default settings for the given
// language.
func DefaultConfigForLanguage(language string) AnalyserConfig {
	language = ValidateLanguage(language)

	return AnalyserConfig{
		Mode:           AnalysisModeFast,
		Language:       language,
		StopWords:      DefaultStopWords(language),
		MinTokenLength: DefaultMinTokenLength,
		MaxTokenLength: DefaultMaxTokenLength,
		PreserveCase:   false,
	}
}

// DefaultStopWords returns the default stop words for the specified language.
// It first tries the builtin stop words registry, then falls back to hardcoded
// stop words for backward compatibility.
//
// Takes language (string) which specifies the language code
// (e.g. "english", "spanish", "french", "german").
//
// Returns map[string]bool which contains the stop words as keys with true
// values for quick lookup.
func DefaultStopWords(language string) map[string]bool {
	provider := CreateStopWordsProvider("builtin")
	stopWords := provider.GetStopWords(language)
	if len(stopWords) > 0 {
		return stopWords
	}

	return defaultStopWordsFallback(language)
}

// defaultStopWordsFallback provides hardcoded stop words for backward
// compatibility when no stop words adapter is imported.
//
// Takes language (string) which specifies the language for stop words.
//
// Returns map[string]bool which contains the stop words for the given
// language, defaulting to English if the language is not supported.
func defaultStopWordsFallback(language string) map[string]bool {
	switch language {
	case "spanish":
		return defaultSpanishStopWords()
	case "french":
		return defaultFrenchStopWords()
	case "german":
		return defaultGermanStopWords()
	case "dutch":
		return defaultDutchStopWords()
	case "russian":
		return defaultRussianStopWords()
	case "swedish":
		return defaultSwedishStopWords()
	case "norwegian":
		return defaultNorwegianStopWords()
	case "hungarian":
		return defaultHungarianStopWords()
	default:
		return defaultEnglishStopWords()
	}
}

// makeStopWordSet converts a slice of stop words into a set for fast lookup.
//
// Takes stopWords ([]string) which contains the words to include in the set.
//
// Returns map[string]bool which maps each stop word to true for quick lookups.
func makeStopWordSet(stopWords []string) map[string]bool {
	set := make(map[string]bool, len(stopWords))
	for _, word := range stopWords {
		set[word] = true
	}
	return set
}

// defaultEnglishStopWords returns hardcoded English stop words.
//
// Returns map[string]bool which contains common English stop words as keys.
func defaultEnglishStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"a", "an", "the",
		"and", "but", "or", "nor", "for", "yet", "so",
		"in", "on", "at", "to", "from", "of", "with", "about", "by",
		"i", "you", "he", "she", "it", "we", "they",
		"me", "him", "her", "us", "them",
		"my", "your", "his", "her", "its", "our", "their",
		"is", "am", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "do", "does", "did",
		"this", "that", "these", "those",
		"what", "which", "who", "when", "where", "why", "how",
		"not", "can", "will", "would", "should", "could",
	})
}

// defaultSpanishStopWords returns hardcoded Spanish stop words.
//
// Returns map[string]bool which contains the set of common Spanish words to
// filter from text.
func defaultSpanishStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"el", "la", "los", "las", "un", "una", "unos", "unas",
		"yo", "tú", "él", "ella", "nosotros", "vosotros", "ellos", "ellas",
		"me", "te", "se", "nos", "os",
		"de", "en", "a", "por", "para", "con", "sin", "sobre",
		"y", "o", "pero", "porque", "que", "si",
		"es", "son", "ser", "estar", "ha", "hay", "fue", "sido",
		"este", "esta", "estos", "estas", "ese", "esa", "esos", "esas",
		"como", "más", "su", "sus", "del", "al",
	})
}

// defaultFrenchStopWords returns hardcoded French stop words.
//
// Returns map[string]bool which contains common French stop words as keys.
func defaultFrenchStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"le", "la", "les", "un", "une", "des",
		"je", "tu", "il", "elle", "nous", "vous", "ils", "elles",
		"me", "te", "se", "lui", "leur",
		"de", "à", "en", "pour", "par", "avec", "dans", "sur", "sous",
		"et", "ou", "mais", "donc", "car", "que", "qui",
		"est", "sont", "être", "avoir", "a", "ont", "été",
		"ce", "ces", "cette", "cet",
		"plus", "pas", "très", "tout", "tous", "du", "au",
	})
}

// defaultRussianStopWords returns hardcoded Russian stop words.
//
// Returns map[string]bool which contains common Russian stop words as keys.
func defaultRussianStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"я", "ты", "он", "она", "мы", "вы", "они",
		"меня", "тебя", "его", "её", "нас", "вас", "их",
		"в", "на", "с", "к", "о", "от", "для", "по", "из", "за", "у",
		"и", "а", "но", "или", "что", "чтобы", "если",
		"быть", "есть", "был", "была", "были", "будет",
		"это", "этот", "эта", "эти", "тот", "та", "те",
		"не", "нет", "да", "как", "так", "ещё", "уже",
	})
}

// defaultDutchStopWords returns hardcoded Dutch stop words.
//
// Returns map[string]bool which contains Dutch stop words as keys for fast
// lookup.
func defaultDutchStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"de", "het", "een",
		"ik", "je", "jij", "u", "hij", "zij", "ze", "wij", "we", "jullie",
		"mij", "me", "jou", "hem", "haar", "ons", "hen", "hun",
		"mijn", "jouw", "uw", "zijn", "haar", "ons", "onze", "hun",
		"in", "op", "aan", "van", "voor", "met", "naar", "om", "bij", "tot", "uit", "over", "door",
		"en", "of", "maar", "want", "dus", "dat", "die", "wie", "wat", "waar", "wanneer", "hoe",
		"is", "zijn", "was", "waren", "ben", "bent", "wordt", "worden", "werd", "werden",
		"heeft", "hebben", "had", "hadden", "kan", "kunnen", "kon", "konden",
		"zal", "zullen", "zou", "zouden", "moet", "moeten",
		"dit", "deze", "dat", "die",
		"niet", "geen", "wel", "ook", "nog", "al", "er", "hier", "daar",
		"nu", "dan", "zo", "als", "meer", "veel", "te", "zeer",
	})
}

// defaultGermanStopWords returns hardcoded German stop words.
//
// Returns map[string]bool which contains the stop words as keys set to true.
func defaultGermanStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"der", "die", "das", "den", "dem", "des", "ein", "eine", "einer", "einem", "einen", "eines",
		"ich", "du", "er", "sie", "es", "wir", "ihr", "sie",
		"mich", "dich", "ihn", "uns", "euch", "ihnen",
		"mir", "dir", "ihm", "ihr",
		"mein", "dein", "sein", "ihr", "unser", "euer",
		"meine", "deine", "seine", "ihre", "unsere", "eure",
		"in", "an", "auf", "aus", "bei", "mit", "nach", "von", "zu", "um", "für", "über", "unter",
		"vor", "hinter", "neben", "zwischen", "durch", "gegen", "ohne",
		"und", "oder", "aber", "denn", "weil", "dass", "ob", "wenn", "als", "wie",
		"ist", "sind", "war", "waren", "bin", "bist", "seid", "sein", "gewesen",
		"hat", "haben", "hatte", "hatten", "habe", "hast", "habt",
		"wird", "werden", "wurde", "wurden", "werde", "wirst", "werdet",
		"kann", "können", "konnte", "konnten",
		"muss", "müssen", "musste", "mussten",
		"soll", "sollen", "sollte", "sollten",
		"will", "wollen", "wollte", "wollten",
		"dieser", "diese", "dieses", "jener", "jene", "jenes",
		"nicht", "kein", "keine", "auch", "noch", "schon", "nur", "sehr", "mehr",
		"hier", "dort", "da", "wo", "was", "wer", "wann", "warum",
		"so", "dann", "doch", "also", "nun", "jetzt", "immer", "wieder",
	})
}

// defaultSwedishStopWords returns hardcoded Swedish stop words.
//
// Returns map[string]bool which contains the stop words as keys for fast
// lookup.
func defaultSwedishStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"en", "ett", "den", "det", "de",
		"jag", "du", "han", "hon", "den", "det", "vi", "ni", "de",
		"mig", "dig", "honom", "henne", "oss", "er", "dem",
		"min", "din", "hans", "hennes", "sin", "vår", "er", "deras",
		"mitt", "ditt", "sitt", "vårt", "ert",
		"mina", "dina", "sina", "våra", "era",
		"i", "på", "till", "från", "med", "av", "för", "om", "vid", "efter", "under", "över", "mellan",
		"och", "eller", "men", "så", "att", "om", "när", "där", "hur", "vad", "vem", "vilken",
		"är", "var", "varit", "vara", "blir", "blev", "blivit", "bli",
		"har", "hade", "haft", "ha",
		"kan", "kunde", "kunnat", "kunna",
		"ska", "skulle", "skall",
		"vill", "ville", "velat", "vilja",
		"måste", "får", "fick", "fått",
		"denna", "dette", "dessa", "den", "det", "de",
		"inte", "ingen", "inget", "inga", "inte",
		"också", "bara", "redan", "än", "nu", "här", "där",
		"mycket", "mer", "mest", "alla", "allt", "annat",
		"sig", "som", "sedan", "dock", "ju", "nog",
	})
}

// defaultNorwegianStopWords returns hardcoded Norwegian stop words.
//
// Returns map[string]bool which contains the stop words as keys set to true.
func defaultNorwegianStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"en", "ei", "et", "den", "det", "de",
		"jeg", "du", "han", "hun", "den", "det", "vi", "dere", "de",
		"meg", "deg", "ham", "henne", "oss", "dere", "dem",
		"min", "din", "hans", "hennes", "sin", "vår", "deres",
		"mitt", "ditt", "sitt", "vårt",
		"mine", "dine", "sine", "våre",
		"i", "på", "til", "fra", "med", "av", "for", "om", "ved", "etter", "under", "over", "mellom",
		"og", "eller", "men", "så", "at", "om", "når", "hvor", "hvordan", "hva", "hvem", "hvilken",
		"er", "var", "vært", "være", "blir", "ble", "blitt", "bli",
		"har", "hadde", "hatt", "ha",
		"kan", "kunne", "kunnet",
		"skal", "skulle", "skullet",
		"vil", "ville", "villet",
		"må", "måtte", "måttet",
		"denne", "dette", "disse",
		"ikke", "ingen", "intet", "inga",
		"også", "bare", "allerede", "enn", "nå", "her", "der",
		"mye", "mer", "mest", "alle", "alt", "annet",
		"seg", "som", "siden", "da", "jo", "nok",
	})
}

// defaultHungarianStopWords returns hardcoded Hungarian stop words.
//
// Returns map[string]bool which contains the stop words as keys.
func defaultHungarianStopWords() map[string]bool {
	return makeStopWordSet([]string{
		"a", "az", "egy",
		"én", "te", "ő", "mi", "ti", "ők",
		"engem", "téged", "őt", "minket", "titeket", "őket",
		"nekem", "neked", "neki", "nekünk", "nektek", "nekik",
		"enyém", "tiéd", "övé", "miénk", "tiétek", "övék",
		"ban", "ben", "ból", "ből", "ba", "be",
		"on", "en", "ön", "ról", "ről", "ra", "re",
		"nál", "nél", "hoz", "hez", "höz",
		"tól", "től", "ig", "ért", "val", "vel",
		"alatt", "fölött", "között", "mellett", "mögött", "előtt", "után",
		"és", "vagy", "de", "hogy", "ha", "mint", "mert", "amikor", "ahol", "ami", "aki",
		"van", "volt", "lesz", "lett", "lenni",
		"vagyok", "vagy", "vagyunk", "vagytok", "vannak",
		"voltam", "voltál", "voltunk", "voltatok", "voltak",
		"ez", "az", "ezek", "azok", "itt", "ott",
		"mi", "ki", "hol", "mikor", "hogyan", "miért", "melyik", "mennyi",
		"nem", "is", "még", "már", "csak", "meg", "el", "ki", "be", "fel", "le",
		"igen", "sem", "minden", "más", "sok", "kevés", "nagy", "kicsi",
		"új", "régi", "jó", "rossz", "így", "úgy", "olyan", "ilyen",
		"majd", "most", "akkor", "tehát", "pedig", "hiszen", "persze",
	})
}
