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

func TestCodeStyles(t *testing.T) {
	input := "hello\n\n```go\nfunc(s string)\n```\n\ndoes it go?"
	output := "hello<br><pre><code><span class=kw>func</span><span class=op>(</span>s <span class=tp>string</span><span class=op>)</span></code></pre><p>does it go?"
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

func TestQuotedlink(t *testing.T) {
	input := `quoted "https://example.com/link" here`
	output := `quoted "<a class="mention u-url" href="https://example.com/link">https://example.com/link</a>" here`
	doonezerotest(t, input, output)
}

func TestHonklink(t *testing.T) {
	input := `https://en.wikipedia.org/wiki/Honk!`
	output := `<a class="mention u-url" href="https://en.wikipedia.org/wiki/Honk!">https://en.wikipedia.org/wiki/Honk!</a>`
	doonezerotest(t, input, output)
}

func TestImagelink(t *testing.T) {
	input := `an image <img alt="caption" src="https://example.com/wherever"> and linked [<img src="there">](example.com)`
	output := `an image <img alt="caption" src="https://example.com/wherever"> and linked <a class="mention u-url" href="example.com"><img src="there"></a>`
	doonezerotest(t, input, output)
}

