package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/clipperhouse/flag"
	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/filters/ascii"
	"github.com/clipperhouse/jargon/filters/contractions"
	"github.com/clipperhouse/jargon/filters/stackoverflow"
	"github.com/clipperhouse/jargon/filters/stemmer"
	"github.com/spf13/afero"
)

var version, commit, date string

func main() {
	// Local to prevent mistaken use in other funcs
	check := func(err error) {
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Stderr.WriteString("\n")
			os.Exit(1)
		}
	}

	//
	// Flags.
	//
	// We don't actually consume some of these flags, see setFilters below; declared here for Usage and errors
	flag.Bool("stack", false, "a filter to recognize tech terms as Stack Overflow tags, e.g. Ruby on Rails → ruby-on-rails")
	flag.Bool("contractions", false, "a filter to expand contractions, e.g. Would've → Would have")
	flag.Bool("ascii", false, "a filter to replace diacritics with ascii equivalents, e.g. café → cafe")
	flag.Bool("stem", false, "a filter to stem words using snowball stemmer, e.g. management|manager → manag")
	lang := flag.String("lang", "english", "language of input, relevant when used with -stem. options:\n"+strings.Join(langs, ", "))

	html := flag.Bool("html", false, "parse input as html (keep tags whole)")
	filein := flag.String("file", "", "input file path (if none, stdin is used as input)")
	fileout := flag.String("out", "", "output file path (if none, stdout is used as input)")
	flag.Bool("lemmas", false, "only return tokens that have been changed by a filter (lemmatized)")
	count := flag.Bool("count", false, "count the tokens")
	lines := flag.Bool("lines", false, "add a line break between tokens")
	flag.Bool("distinct", false, "only return unique tokens")
	v := flag.Bool("version", false, "display the version")

	flag.Parse()

	if *v {
		fmt.Println("Version: " + version)
		fmt.Println("Commit: " + commit)
		fmt.Println("Build date: " + date)
		return
	}

	c := config{
		Fs:    afero.NewOsFs(),
		HTML:  *html,
		Count: *count,
		Lines: *lines,
	}

	//
	// Input
	//
	fi, err := os.Stdin.Stat()
	check(err)
	mode := fi.Mode()

	err = setInput(&c, mode, *filein)
	if err == errNoInput {
		printUsage()
		return
	}
	check(err)
	if c.Filein != nil {
		defer c.Filein.Close()
	}

	//
	// Filters
	//
	err = setFilters(&c, os.Args[1:], *lang)
	check(err)

	//
	// Output
	//
	err = setOutput(&c, *fileout)
	check(err)
	if c.Fileout != nil {
		defer c.Fileout.Close()
	}

	//
	// Reader
	//
	err = setReader(&c)
	check(err)

	//
	// Writer
	//
	err = setWriter(&c)
	check(err)

	//
	// Execute filters
	//
	err = execute(&c)
	check(err)
}

type config struct {
	Fs afero.Fs

	HTML    bool
	Count   bool
	Lines   bool
	Filters []jargon.Filter

	Filein, Fileout   afero.File
	Pipedin, Pipedout bool

	Reader *bufio.Reader
	Writer *bufio.Writer
}

var errNoInput = fmt.Errorf("no input")
var errTwoInput = fmt.Errorf("choose *either* an input -file argument *or* piped input")

func printUsage() {
	// Display usage
	os.Stderr.WriteString(flag.CommandLine.Name() + " takes text from std input and processes it with one or more filters.\n\n")
	os.Stderr.WriteString("Example:\n\n  curl -s https://en.wikipedia.org/wiki/Computer_programming | jargon -html -stack -lemmas -lines\n\n")
	os.Stderr.WriteString("Flags:\n\n")
	flag.PrintDefaults()
}

func setInput(c *config, mode os.FileMode, filein string) error {
	if filein != "" {
		// Try to open it
		file, err := c.Fs.Open(filein)
		if err != nil {
			return err
		}

		c.Filein = file
	}

	c.Pipedin = (mode & os.ModeCharDevice) == 0 // https://filters/stackoverflow.com/a/43947435/70613

	// If no input, display usage
	input := c.Pipedin || c.Filein != nil
	if !input {
		return errNoInput
	}

	// Choose one input *or* the other
	if c.Pipedin && c.Filein != nil {
		return errTwoInput
	}

	return nil
}

var filterMap = map[string]jargon.Filter{
	"-ascii":        ascii.Fold,
	"-contractions": contractions.Expand,
	"-lemmas":       (*jargon.TokenStream).Lemmas,
	"-distinct":     (*jargon.TokenStream).Distinct,
	"-stack":        stackoverflow.Tags,
	"-stem":         stemmer.English,
}

var langs = []string{"english", "french", "norwegian", "russian", "spanish", "swedish"}
var stemmerMap = map[string]jargon.Filter{
	"english":   stemmer.English,
	"french":    stemmer.French,
	"norwegian": stemmer.Norwegian,
	"russian":   stemmer.Russian,
	"spanish":   stemmer.Spanish,
	"swedish":   stemmer.Swedish,
}

func setFilters(c *config, args []string, lang string) error {
	// Loop through filters; order matters, so can't use flag package
	for _, arg := range args {
		filter, found := filterMap[arg]
		if found {
			if arg == "-stem" && lang != "" {
				// Look for a language specification
				stem, found := stemmerMap[lang]
				if found {
					filter = stem
				} else {
					err := fmt.Errorf("lang %q is not known by %s; options are %s", lang, flag.CommandLine.Name(), strings.Join(langs, ", "))
					return err
				}
			}
			c.Filters = append(c.Filters, filter)
		}
	}

	return nil
}

func setOutput(c *config, fileout string) error {
	if fileout != "" {
		file, err := c.Fs.Create(fileout)
		if err != nil {
			return err
		}

		c.Fileout = file
	}
	c.Pipedout = (c.Fileout == nil)

	return nil
}

func setReader(c *config) error {
	if c.Pipedin || c.Pipedout {
		// We're limited by the OS pipe buffer, typically 64K with back pressure
		// Using anything larger doesn't buy us anything
		size := 64 * 1024
		if c.Pipedin {
			c.Reader = bufio.NewReaderSize(os.Stdin, size)
		} else {
			c.Reader = bufio.NewReaderSize(c.Filein, size)
		}
	}

	if c.Filein != nil {
		fi, err := c.Filein.Stat()
		if err != nil {
			return err
		}

		size := fi.Size()
		switch {
		case size <= 4*1024:
			// Minimum of 4K
			c.Reader = bufio.NewReaderSize(c.Filein, 4*1024)
		case size <= 1024*1024:
			// Aim for a right-sized buffer (single read, perhaps) up to 1MB
			c.Reader = bufio.NewReaderSize(c.Filein, int(size))
		default:
			// Otherwise, use 1MB buffer size, better perf over default, but not huge
			c.Reader = bufio.NewReaderSize(c.Filein, 1024*1024)
		}
	}

	return nil
}

func setWriter(c *config) error {
	if c.Reader == nil {
		return fmt.Errorf("reader is required")
	}

	// Match the input buffer size; mismatch doesn't buy us anything
	size := c.Reader.Size()
	if c.Pipedout {
		c.Writer = bufio.NewWriterSize(os.Stdout, size)
	} else {
		c.Writer = bufio.NewWriterSize(c.Fileout, size)
	}

	return nil
}

func execute(c *config) error {
	if c.Reader == nil {
		return fmt.Errorf("reader is required")
	}
	if c.Writer == nil {
		return fmt.Errorf("writer is required")
	}

	var tokens *jargon.TokenStream
	if c.HTML {
		tokens = jargon.TokenizeHTML(c.Reader)
	} else {
		tokens = jargon.Tokenize(c.Reader)
	}

	for _, f := range c.Filters {
		tokens = f(tokens)
	}

	if c.Count {
		count, err := tokens.Count()
		if err != nil {
			return err
		}
		c.Writer.WriteString(strconv.Itoa(count) + "\n")
		err = c.Writer.Flush()
		if err != nil {
			return err
		}
		return nil
	}

	// Write all
	for tokens.Scan() {
		token := tokens.Token()
		_, err := c.Writer.WriteString(token.String())
		if err != nil {
			return err
		}

		if c.Lines {
			_, err := c.Writer.WriteRune('\n')
			if err != nil {
				return err
			}
		}
	}
	if err := tokens.Err(); err != nil {
		return err
	}

	if err := c.Writer.Flush(); err != nil {
		return err
	}

	return nil
}
