package consts

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// String literals are emitted per target language.  Centralizing the quoters
// here keeps the escaping policy in one place and lets each emitter read
// identically (`<lang>Quote(s)`).
//
// Go, TypeScript, and Python reuse Go's strconv.Quote:
//   - Go:     it *is* the Go quoter, so output is exact by construction.
//   - Python: every escape strconv.Quote emits (\a \b \f \n \r \t \v \xNN
//     \uNNNN \U00NNNNNN) is also a valid Python escape, so it round-trips.
//   - TypeScript: mostly compatible, with one known gap — JS has no \a escape
//     and would silently read it as a literal 'a'.  Harmless for the printable
//     ASCII / path strings forge sees in practice; revisit with a dedicated
//     tsQuote if non-printable string consts ever appear.
//
// C# and C get dedicated quoters because their \x escape is *greedy*
// (variable-length), so borrowing Go's \xNN form could silently change a value.

// goQuote returns a Go double-quoted string literal.
func goQuote(s string) string { return strconv.Quote(s) }

// tsQuote returns a TypeScript/JS double-quoted string literal.
func tsQuote(s string) string { return strconv.Quote(s) }

// pyQuote returns a Python double-quoted string literal.
func pyQuote(s string) string { return strconv.Quote(s) }

// csharpQuote returns a C# regular (non-verbatim) double-quoted string literal.
// Non-printables use the fixed-width \uNNNN / \UNNNNNNNN forms rather than C#'s
// greedy \x, which would absorb following hex digits.
func csharpQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			switch {
			case r < 0x20 || r == 0x7f:
				fmt.Fprintf(&b, `\u%04X`, r)
			case unicode.IsPrint(r):
				b.WriteRune(r)
			case r > 0xFFFF:
				fmt.Fprintf(&b, `\U%08X`, r)
			default:
				fmt.Fprintf(&b, `\u%04X`, r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// cQuote returns a C double-quoted string literal.  It works on raw bytes (C
// strings are byte arrays) and escapes non-printables as fixed 3-digit octal,
// avoiding C's greedy \x escape.
func cQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 || c >= 0x7f {
				fmt.Fprintf(&b, `\%03o`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
