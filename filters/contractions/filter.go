// Package contractions provides a filter to expand English contractions, such as "don't" → "does not", for use with jargon
package contractions

import (
	"strings"

	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/tokenqueue"
)

//go:generate go run generate/main.go

// Expand converts single-token contractions to non-contracted version. Examples:
// don't → does not
// We’ve → We have
// SHE'S -> SHE IS
func Expand(incoming *jargon.TokenStream) *jargon.TokenStream {
	t := &tokens{
		incoming: incoming,
		outgoing: tokenqueue.New(),
	}
	return jargon.NewTokenStream(t.next)
}

type tokens struct {
	incoming *jargon.TokenStream
	outgoing *tokenqueue.TokenQueue
}

func (t *tokens) next() (*jargon.Token, error) {
	if t.outgoing.Any() {
		return t.outgoing.Pop(), nil
	}

	token, err := t.incoming.Next()
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, nil
	}

	// Try case-sensitive
	found, err := t.tryExpansion(token, false)
	if err != nil {
		return nil, err
	}
	if !found {
		// Try case-insensitive
		_, err := t.tryExpansion(token, true)
		if err != nil {
			return nil, err
		}
	}

	if t.outgoing.Any() {
		return t.outgoing.Pop(), nil
	}

	return token, nil
}

func (t *tokens) tryExpansion(token *jargon.Token, ignoreCase bool) (bool, error) {
	key := token.String()
	if ignoreCase {
		key = strings.ToLower(key)
	}

	expansion, found := mappings[key]

	if found {
		tokens, err := jargon.TokenizeString(expansion).ToSlice()
		if err != nil {
			return found, err
		}
		t.outgoing.Push(tokens...)
	}

	return found, nil
}
