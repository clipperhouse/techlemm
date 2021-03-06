package synonyms

import (
	"reflect"
	"testing"

	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/tokenqueue"
)

func TestFill(t *testing.T) {
	type test struct {
		// input
		input    string
		maxWords int
		previous *tokenqueue.TokenQueue

		// expected
		buffer   *tokenqueue.TokenQueue
		outgoing *tokenqueue.TokenQueue
	}

	tests := []test{
		{
			input:    "test one",
			maxWords: 3,
			previous: tokenqueue.New(),
			buffer: tokenqueue.New(
				jargon.NewToken("test", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("one", false),
			),
			outgoing: tokenqueue.New(),
		},
		{
			input:    "test two",
			maxWords: 1,
			previous: tokenqueue.New(),
			buffer: tokenqueue.New(
				jargon.NewToken("test", false),
			),
			outgoing: tokenqueue.New(),
		},
		{
			input:    " test three",
			maxWords: 2,
			previous: tokenqueue.New(),

			buffer: tokenqueue.New(
				jargon.NewToken("test", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("three", false),
			),
			outgoing: tokenqueue.New(
				jargon.NewToken(" ", false),
			),
		},
		{
			input:    "test four, and five",
			maxWords: 4,
			previous: tokenqueue.New(),

			buffer: tokenqueue.New(
				jargon.NewToken("test", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("four", false),
				jargon.NewToken(",", false),
			),
			outgoing: tokenqueue.New(),
		},
		{
			input:    ", test six and seven",
			maxWords: 4,
			previous: tokenqueue.New(),

			buffer: tokenqueue.New(
				jargon.NewToken("test", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("six", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("and", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("seven", false),
			),
			outgoing: tokenqueue.New(
				jargon.NewToken(",", false),
				jargon.NewToken(" ", false),
			),
		},
		{
			input:    " test eight and nine",
			maxWords: 4,
			previous: tokenqueue.New(
				jargon.NewToken("previous", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("stuff", false),
			),

			buffer: tokenqueue.New(
				jargon.NewToken("previous", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("stuff", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("test", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("eight", false),
			),
			outgoing: tokenqueue.New(),
		},
		{
			input:    ". test ten and eleven",
			maxWords: 4,
			previous: tokenqueue.New(
				jargon.NewToken("previous", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("stuff", false),
			),
			buffer: tokenqueue.New(
				jargon.NewToken("previous", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("stuff", false),
				jargon.NewToken(".", false),
			),
			outgoing: tokenqueue.New(),
		},
		{
			input:    " test twelve and thirteen",
			maxWords: 3,
			previous: tokenqueue.New(
				jargon.NewToken(".", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("leftover", false),
			),
			buffer: tokenqueue.New(
				jargon.NewToken("leftover", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("test", false),
				jargon.NewToken(" ", false),
				jargon.NewToken("twelve", false),
			),
			outgoing: tokenqueue.New(
				jargon.NewToken(".", false),
				jargon.NewToken(" ", false),
			),
		},
	}

	for _, test := range tests {
		incoming := jargon.TokenizeString(test.input)
		tokens := &tokens{
			incoming: incoming,
			buffer:   test.previous,
			outgoing: tokenqueue.New(),
			filter: &filter{
				maxWords: test.maxWords,
			},
		}
		tokens.fill()

		expected := test.buffer.String()
		got := tokens.buffer.String()
		if expected != got {
			t.Errorf("expected %s, got %s", expected, got)
		}

		expected = test.outgoing.String()
		got = tokens.outgoing.String()
		if expected != got {
			t.Errorf("expected %s, got %s", expected, got)
		}
	}
}

func TestPassthrough(t *testing.T) {
	// If the filter doesn't do anything, the tokens should come back verbatim

	mappings := map[string]string{}
	ignore := []rune{}
	synonyms := NewFilter(mappings, false, ignore)

	text := "This is a test, with spaces and punctuation."

	original := jargon.TokenizeString(text)
	expected, err := original.ToSlice()
	if err != nil {
		t.Error(err)
	}

	filtered := jargon.TokenizeString(text)
	if err != nil {
		t.Error(err)
	}
	got, err := synonyms(filtered).ToSlice()
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestLazyLoad(t *testing.T) {
	mappings := map[string]string{
		"developer, engineer, programmer,": "boffin",
	}
	ignore := []rune{'-', ' ', '.', '/'}
	synonyms := &filter{
		config: &config{
			mappings:    mappings,
			ignoreCase:  true,
			ignoreRunes: ignore,
		},
	}

	if synonyms.trie != nil {
		t.Errorf("trie should be nil prior to first Filter() call")
	}

	original := `we are looking for a developer or engineer`
	tokens := jargon.TokenizeString(original)
	filtered := synonyms.Filter(tokens)

	if synonyms.trie == nil {
		t.Errorf("trie should not be nil after first Filter() call")
	}

	expected := `we are looking for a boffin or boffin`
	got, err := filtered.String()
	if err != nil {
		t.Error(err)
	}

	if expected != got {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFilter(t *testing.T) {
	mappings := map[string]string{
		"developer, engineer, programmer,": "boffin",
		"rock star, 10x developer":         "cliché",
		"Ruby on Rails, rails":             "ruby-on-rails",
		"nodeJS, iojs":                     "node.js",
	}

	ignore := []rune{'-', ' ', '.', '/'}
	synonyms := NewFilter(mappings, true, ignore)

	original := `we are looking for a rockstar, 10x developer, or engineer, for ruby on rails and Nodejs`
	tokens := jargon.TokenizeString(original)

	expected := `we are looking for a cliché, cliché, or boffin, for ruby-on-rails and node.js`

	got, err := synonyms(tokens).String()
	if err != nil {
		t.Error(err)
	}

	if expected != got {
		t.Errorf("given %q, expected %q, got %q", original, expected, got)
	}
}

func BenchmarkFilter(b *testing.B) {
	mappings := map[string]string{
		"developer, engineer, programmer,": "boffin",
		"rock star, 10x developer":         "cliché",
		"Ruby on Rails, rails":             "ruby-on-rails",
		"nodeJS, iojs":                     "node.js",
	}

	ignore := []rune{'-', ' ', '.', '/'}
	filter := NewFilter(mappings, true, ignore)

	original := `we are looking for a rockstar 10x developer or engineer for ruby on rails and Nodejs`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokens := jargon.TokenizeString(original)
		_, err := filter(tokens).Count()
		if err != nil {
			b.Error(err)
		}
	}
}
