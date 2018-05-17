package jargon

import (
	"github.com/clipperhouse/jargon/stackexchange"
)

// Lemmatizer is the main structure for looking up canonical tags
type Lemmatizer struct {
	values        map[string]string
	maxGramLength int
	normalize     func(string) string
}

var stackExchange = NewLemmatizer(stackexchange.Dictionary)

// Lemmatize will process text using the Stack Exchange dictionary of tags & synonyms,
// replacing tech terms and phrases like Python and Ruby on Rails with canonical tags,
// like python and ruby-on-rails.
// It returns the original text, with white space preserved, differeing only by the above replacements.
func Lemmatize(text string) string {
	tokens := TechProse.Tokenize(text)
	lemmatized := stackExchange.LemmatizeTokens(tokens)
	return Join(lemmatized)
}

// LemmatizeHTML will process HTML text using the Stack Exchange dictionary of tags & synonyms,
// replacing tech terms and phrases like Python and Ruby on Rails with canonical tags,
// like python and ruby-on-rails.
// It returns the original HTML, with white space preserved, differeing only by the above replacements.
func LemmatizeHTML(text string) string {
	tokens := TechHTML.Tokenize(text)
	lemmatized := stackExchange.LemmatizeTokens(tokens)
	return Join(lemmatized)
}

// NewLemmatizer creates and populates a new Lemmatizer for the purpose of looking up canonical tags.
// Data and rules mostly live in the Dictionary interface, which is usually imported.
func NewLemmatizer(d Dictionary) *Lemmatizer {
	lem := &Lemmatizer{
		values:        make(map[string]string),
		maxGramLength: d.MaxGramLength(),
		normalize:     d.Normalize,
	}
	tags := d.GetTags()
	for _, tag := range tags {
		key := lem.normalize(tag)
		lem.values[key] = tag
	}
	synonyms := d.GetSynonyms()
	for synonym, canonical := range synonyms {
		key := lem.normalize(synonym)
		lem.values[key] = canonical
	}
	return lem
}

// LemmatizeTokens takes a slice of tokens and returns tokens with canonicalized terms.
// Terms (tokens) that are not canonicalized are returned as-is, e.g.
// ["I", " ", "think", " ", "Ruby", " ", "on", " ", "Rails", " ", "is", " ", "great"] →
//    ["I", " ", "think", " ", "ruby-on-rails", " ", "is", " ", "great"]
// Note that fewer tokens may be returned than were input.
// A lot depends on the original tokenization, so make sure that it's right!
func (lem *Lemmatizer) LemmatizeTokens(tokens []Token) []Token {
	lemmatized := make([]Token, 0)
	pos := 0

	for pos < len(tokens) {
		switch current := tokens[pos]; {
		case current.IsPunct() || current.IsSpace():
			// Emit it
			lemmatized = append(lemmatized, current)
			pos++
		default:
		Grams:
			// Else it's a word, try n-grams
			for take := lem.maxGramLength; take > 0; take-- {
				run, consumed, ok := wordrun(tokens, pos, take)
				if ok {
					gram := Join(run)
					key := lem.normalize(gram)
					canonical, found := lem.values[key]

					if found {
						// Emit token, replacing consumed tokens
						token := Token{
							value: canonical,
							space: false,
							punct: false,
						}
						lemmatized = append(lemmatized, token)
						pos += consumed
						break Grams
					}

					if take == 1 {
						// No n-grams, just emit
						token := tokens[pos]
						lemmatized = append(lemmatized, token)
						pos++
					}
				}
			}
		}
	}

	return lemmatized
}

// Analogous to tokens.Skip(skip).Take(take) in Linq
func wordrun(tokens []Token, skip, take int) ([]Token, int, bool) {
	taken := make([]Token, 0)
	consumed := 0 // tokens consumed, not necessarily equal to take

	for len(taken) < take {
		end := skip + consumed

		candidate := tokens[end]
		switch {
		// Note: test for punct before space; newlines and tabs can be
		// considered both punct and space (depending on the tokenizer!)
		// and we want to treat them as breaking word runs.
		case candidate.IsPunct():
			// Hard stop
			return nil, 0, false
		case candidate.IsSpace():
			// Ignore and continue
			consumed++
		default:
			// Found a word
			taken = append(taken, candidate)
			consumed++
		}
	}

	return taken, consumed, true
}
