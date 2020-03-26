package stackoverflow_test

import (
	"testing"

	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/stackoverflow"
)

func TestFilter(t *testing.T) {
	type test struct {
		input     string
		canonical string
	}
	expecteds := []test{
		{"Foo", "Foo"},    // nothing should happen
		{"c sharp", "c#"}, // should be replaced
		{"C#", "c#"},      // ditto
		// {"C+", "C+"},
		{"C++", "c++"},
		{"something about Ruby on Rails, and such.", "something about ruby-on-rails, and such."},
		{"more about Ruby stuff", "more about ruby stuff"},
		{"Rub", "Rub"}, // don't pick up 'R' substring
	}

	for _, expected := range expecteds {
		tokens := jargon.TokenizeString(expected.input)
		canonical, err := stackoverflow.Tags.Filter(tokens).String()
		if err != nil {
			t.Error(err)
		}
		if canonical != expected.canonical {
			t.Errorf("expected to find canonical %q, got %q", expected.canonical, canonical)
		}
	}
}
