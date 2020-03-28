package synonyms

import (
	"bytes"
	"fmt"

	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/synonyms/trie"
)

// Filter is the data structure of a synonyms filter. Use NewFilter to create.
type Filter struct {
	trie     *trie.RuneTrie
	maxWords int
}

// NewFilter creates a new synonyms Filter
func NewFilter(mappings map[string]string, ignoreCase bool, ignoreRunes []rune) (*Filter, error) {
	trie := trie.New(ignoreCase, ignoreRunes)
	maxWords := 1
	for synonyms, canonical := range mappings {
		tokens, err := jargon.TokenizeString(synonyms).ToSlice()
		if err != nil {
			return nil, err
		}

		start := 0
		skipSpaces := true
		for i, token := range tokens {
			// Leading spaces, and spaces following commas, should be ignored
			if skipSpaces && token.IsSpace() {
				start++
				continue
			}
			skipSpaces = false

			if token.String() == "," {
				slice := tokens[start:i]
				trie.Add(slice, canonical)
				updateMaxWords(slice, &maxWords)

				start = i + 1 // ignore the comma
				skipSpaces = true
				continue
			}
		}

		// Remaining after the last comma
		slice := tokens[start:]
		trie.Add(slice, canonical)
		updateMaxWords(slice, &maxWords)
	}

	return &Filter{
		trie:     trie,
		maxWords: maxWords,
	}, nil
}

func updateMaxWords(tokens []*jargon.Token, maxWords *int) {
	words := 0
	for _, token := range tokens {
		if !token.IsSpace() && !token.IsPunct() {
			words++
		}
	}
	if words > *maxWords {
		*maxWords = words
	}
}

// Filter replaces tokens with their canonical terms, based on Stack Overflow tags & synonyms
func (f *Filter) Filter(incoming *jargon.Tokens) *jargon.Tokens {
	t := &tokens{
		incoming: incoming,
		buffer:   &jargon.TokenQueue{},
		outgoing: &jargon.TokenQueue{},
		filter:   f,
	}
	return &jargon.Tokens{
		Next: t.next,
	}
}

type tokens struct {
	// incoming stream of tokens from another source, such as a tokenizer
	incoming *jargon.Tokens
	// a 'lookahead' buffer for incoming tokens
	buffer *jargon.TokenQueue
	// outgoing queue of filtered tokens
	outgoing *jargon.TokenQueue
	filter   *Filter
}

// next returns the next token; nil indicates end of data
func (t *tokens) next() (*jargon.Token, error) {
	// Clear out any outgoing
	if t.outgoing.Any() {
		return t.outgoing.Pop(), nil
	}

	// Consume all the words
	for {
		err := t.fill()
		if err != nil {
			return nil, err
		}

		run := t.wordrun()
		if len(run) == 0 {
			// No more words
			break
		}

		// Try to lemmatize
		found, canonical, consumed := t.filter.trie.SearchCanonical(run...)
		if found {
			if canonical != "" {
				token := jargon.NewToken(canonical, true)
				t.outgoing.Push(token)
			}
			t.buffer.Drop(consumed)
			continue
		}

		t.buffer.PopTo(t.outgoing)
	}

	// Queue up the rest of the buffer to go out
	t.buffer.FlushTo(t.outgoing)

	if t.outgoing.Any() {
		return t.outgoing.Pop(), nil
	}

	return nil, nil
}

// fill the buffer until EOF, punctuation, or enough word tokens
func (t *tokens) fill() error {
	words := 0
	for _, token := range t.buffer.Tokens {
		if !token.IsPunct() && !token.IsSpace() {
			words++
		}
	}

	for words < t.filter.maxWords {
		token, err := t.incoming.Next()
		if err != nil {
			return err
		}
		if token == nil {
			// EOF
			break
		}

		t.buffer.Push(token)

		if token.IsPunct() {
			break
		}

		if token.IsSpace() {
			continue
		}

		// It's a word
		words++
	}

	return nil
}

// wordrun pulls the longest series of tokens comprised of words
func (t *tokens) wordrun() []*jargon.Token {
	spaces := 0
	for _, token := range t.buffer.Tokens {
		if !token.IsSpace() {
			break
		}
		t.outgoing.Push(token)
		spaces++
	}
	t.buffer.Drop(spaces)

	var (
		end      int
		words    int
		consumed int
	)

	for i, token := range t.buffer.Tokens {
		if token.IsPunct() {
			// fall through and send back word run we've gotten so far (if any)
			// don't consume this punct, leave it in the buffer
			break
		}

		// It's a word or space
		end = i + 1
		consumed++

		if !token.IsSpace() {
			// It's a word
			words++
		}

		if words >= t.filter.maxWords {
			break
		}
	}

	return t.buffer.Tokens[:end]
}

// String returns a Go source declaration of the Filter
func (f *Filter) String() string {
	return f.Decl()
}

// Decl returns a Go source declaration of the Filter
func (f *Filter) Decl() string {
	var b bytes.Buffer

	fmt.Fprintf(&b, "&synonyms.Filter{\n")
	if f.trie != nil {
		fmt.Fprintf(&b, "Trie: %s,\n", f.trie.String())
	}
	if f.maxWords > 0 {
		// default value does not need to be declared
		fmt.Fprintf(&b, "MaxWords: %d,\n", f.maxWords)
	}
	fmt.Fprintf(&b, "}")

	result := b.Bytes()
	// result, err := format.Source(result)
	// if err != nil {
	// 	panic(err)
	// }

	return string(result)
}
