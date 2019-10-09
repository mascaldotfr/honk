package main

import (
	"testing"
)

func doonezerotest(t *testing.T, input, output string) {
	result := markitzero(input)
	if result != output {
		t.Errorf("\nexpected:\n%s\noutput:\n%s", output, result)
	}
}

func TestBasictest(t *testing.T) {
	input := `link to https://example.com/ with **bold** text`
	output := `link to <a class="mention u-url" href="https://example.com/">https://example.com/</a> with <b>bold</b> text`
	doonezerotest(t, input, output)
}

func TestLinebreak1(t *testing.T) {
	input := "hello\n> a quote\na comment"
	output := "hello<blockquote>a quote</blockquote><p>a comment"
	doonezerotest(t, input, output)
}

func TestLinebreak2(t *testing.T) {
	input := "hello\n\n> a quote\n\na comment"
	output := "hello<br><blockquote>a quote</blockquote><p>a comment"
	doonezerotest(t, input, output)
}

func TestLinebreak3(t *testing.T) {
	input := "hello\n\n```\nfunc(s string)\n```\n\ndoes it go?"
	output := "hello<br><pre><code>func(s string)</code></pre><p>does it go?"
	doonezerotest(t, input, output)
}

func TestSimplelink(t *testing.T) {
	input := "This is a [link](https://example.com)."
	output := `This is a <a class="mention u-url" href="https://example.com">link</a>.`
	doonezerotest(t, input, output)
}

func TestSimplelink2(t *testing.T) {
	input := "See (http://example.com) for examples."
	output := `See (<a class="mention u-url" href="http://example.com">http://example.com</a>) for examples.`
	doonezerotest(t, input, output)
}

func TestWikilink(t *testing.T) {
	input := "I watched [Hackers](https://en.wikipedia.org/wiki/Hackers_(film))"
	output := `I watched <a class="mention u-url" href="https://en.wikipedia.org/wiki/Hackers_(film)">Hackers</a>`
	doonezerotest(t, input, output)
}
