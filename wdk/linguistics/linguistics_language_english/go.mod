module piko.sh/piko/wdk/linguistics/linguistics_language_english

go 1.26.0

require piko.sh/piko v0.0.0 // indirect

require (
	piko.sh/piko/wdk/linguistics/linguistics_phonetic_english v0.0.0
	piko.sh/piko/wdk/linguistics/linguistics_stemmer_english v0.0.0
	piko.sh/piko/wdk/linguistics/linguistics_stopwords_english v0.0.0
)

require (
	github.com/kljensen/snowball v0.10.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace (
	piko.sh/piko v0.0.0 => ../../..
	piko.sh/piko/wdk/linguistics/linguistics_phonetic_english v0.0.0 => ../../../wdk/linguistics/linguistics_phonetic_english
	piko.sh/piko/wdk/linguistics/linguistics_stemmer_english v0.0.0 => ../../../wdk/linguistics/linguistics_stemmer_english
	piko.sh/piko/wdk/linguistics/linguistics_stopwords_english v0.0.0 => ../../../wdk/linguistics/linguistics_stopwords_english
)
