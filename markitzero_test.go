package main

import (
	"testing"
)

func TestMarkitzero(t *testing.T) {
	input := `link to https://example.com/ with **bold** text`
	output := `link to <a class="mention u-url" href="https://example.com/">https://example.com/</a> with <b>bold</b> text`

	result := markitzero(input)
	if result != output {
		t.Errorf("\nexpected:\n%s\noutput:\n%s", output, result)
	}
}
