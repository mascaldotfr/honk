package main

import (
	"testing"
)

func onetest(t *testing.T, input, output string) {
	result := markitzero(input)
	if result != output {
		t.Errorf("\nexpected:\n%s\noutput:\n%s", output, result)
	}
}

func basictest(t *testing.T) {
	input := `link to https://example.com/ with **bold** text`
	output := `link to <a class="mention u-url" href="https://example.com/">https://example.com/</a> with <b>bold</b> text`
	onetest(t, input, output)
}

func linebreak1(t *testing.T) {
	input := "hello\n> a quote\na comment"
	output := "hello<blockquote>a quote</blockquote><p>a comment"
	onetest(t, input, output)
}

func linebreak2(t *testing.T) {
	input := "hello\n\n> a quote\n\na comment"
	output := "hello<br><blockquote>a quote</blockquote><p>a comment"
	onetest(t, input, output)
}

func linebreak3(t *testing.T) {
	input := "hello\n\n```\nfunc(s string)\n```\n\ndoes it go?"
	output := "hello<br><pre><code>func(s string)</code></pre><p>does it go?"
	onetest(t, input, output)
}

func TestMarkitzero(t *testing.T) {
	basictest(t)
	linebreak1(t)
	linebreak2(t)
	linebreak3(t)
}
