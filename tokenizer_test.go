package jargon

import (
	"strings"
	"testing"
)

// TODO: test ordering

func TestTokenizer(t *testing.T) {
	original := `Hi.    
	node.js, first_last, my.name@domain.com
	123.456, 789, .234, 1,000, a16z, 3G and $200.13.
	wishy-washy and C++ and F# and .net
	Let’s Let's possessive' possessive’
	ש״ח
	א"ב
	ב'
	"אא"בב"abc
	Then ウィキペディア and 象形.`
	original += "crlf is \r\n"

	tokens := Tokenize(strings.NewReader(original))

	// First, sanity check
	roundtrip, err := tokens.String()
	if err != nil {
		t.Error(err)
	}
	if roundtrip != original {
		t.Error("roundtrip should equal the original")
	}

	// Gotta re-tokenize
	tokens = Tokenize(strings.NewReader(original))

	got := map[string]bool{}

	for tokens.Scan() {
		token := tokens.Token()
		got[token.String()] = true
	}

	if err := tokens.Err(); err != nil {
		t.Error(err)
	}

	type test struct {
		value string
		found bool
	}

	expecteds := []test{
		{"Hi", true},
		{".", true},
		{"Hi.", false},

		{"node.js", true},
		{"node", false},
		{"js", false},

		// Note: uax29 package considers underscore a mid-word char
		{"first_last", true},
		{"first", false},
		{"_", false},
		{"last", false},

		{"my.name", true},
		{"my.name@", false},
		{"@", true},
		{"domain.com", true},
		{"@domain.com", false},

		{"123.456", true},
		{"123,", false},
		{"456", false},
		{"123.456,", false},

		{"789", true},
		{"789,", false},

		{".234", false},
		{"234", true},

		{"1,000", true},
		{"1,000,", false},

		{"wishy-washy", false},
		{"wishy", true},
		{"-", true},
		{"washy", true},

		{"C++", false},
		{"C", true},
		{"+", true},

		{"F#", false},
		{"F", true},
		{"#", true},

		{".net", false},
		{"net", true},

		{"Let's", true},
		{"Let’s", true},
		{"Let", false},
		{"s", false},

		{"possessive", true},
		{"'", true},
		{"’", true},
		{"possessive'", false},
		{"possessive’", false},

		{"a16z", true},

		{"3G", true},

		{"$", true},
		{"200.13", true},

		{"ש״ח", true},
		{`א"ב`, true},
		{"ב'", true},
		{"אא\"בב", true},
		{"abc", true},

		{"ウィキペディア", true},
		{"ウ", false},

		{"象", true},
		{"形", true},
		{"象形", false},

		{"\r\n", true},
		{"\r", false},
	}

	for _, expected := range expecteds {
		if got[expected.value] != expected.found {
			t.Errorf("expected %q to be %t", expected.value, expected.found)
		}
	}
}
