package goja

import (
	"testing"
)

func TestEncodeURI(t *testing.T) {
	const SCRIPT = `
	encodeURI('тест')
	`

	testScript1(SCRIPT, asciiString("%D1%82%D0%B5%D1%81%D1%82"), t)
}

func TestDecodeURI(t *testing.T) {
	const SCRIPT = `
	decodeURI("http://ru.wikipedia.org/wiki/%d0%ae%D0%bd%D0%B8%D0%BA%D0%BE%D0%B4")
	`

	testScript1(SCRIPT, newStringValue("http://ru.wikipedia.org/wiki/Юникод"), t)
}
