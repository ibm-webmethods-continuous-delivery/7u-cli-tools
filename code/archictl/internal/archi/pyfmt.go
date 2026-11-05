package archi

import (
	"strings"
	"unicode"
)

// PyRepr renders s like Python's repr(str), used to keep user-visible messages identical to the oracle.
func PyRepr(s string) string {
	q := '\''
	if strings.Contains(s, "'") && !strings.Contains(s, "\"") {
		q = '"'
	}
	var b strings.Builder
	b.WriteRune(q)
	for _, r := range s {
		switch {
		case r == q:
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			b.WriteString(hexEsc(r, 2, 'x'))
		case r < 0x7f || unicode.IsPrint(r):
			b.WriteRune(r)
		case r <= 0xff:
			b.WriteString(hexEsc(r, 2, 'x'))
		case r <= 0xffff:
			b.WriteString(hexEsc(r, 4, 'u'))
		default:
			b.WriteString(hexEsc(r, 8, 'U'))
		}
	}
	b.WriteRune(q)
	return b.String()
}

func hexEsc(r rune, width int, kind byte) string {
	const digits = "0123456789abcdef"
	buf := make([]byte, 0, width+2)
	buf = append(buf, '\\', kind)
	for i := width - 1; i >= 0; i-- {
		buf = append(buf, digits[(r>>(4*uint(i)))&0xf])
	}
	return string(buf)
}

// isPySpace mirrors Python's str.isspace for single characters.
func isPySpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// hasPySpace reports whether s contains a Python-whitespace character.
func hasPySpace(s string) bool {
	for _, r := range s {
		if isPySpace(r) {
			return true
		}
	}
	return false
}
