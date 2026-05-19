package sdl

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

/*
https://github.com/dlang-community/SDLang-D/wiki/Language-Guide

Tags are written using the form:

	Tag := Namespace:Name Attribute* {
	    ChildTag
	}

	Attribute := Namespace:Name = Value

	Value :=
		| Int
		| Float
		| String
		| List

	List := ( Value, Value,  ... )

https://github.com/SdlangInitiative/sdlanggo/blob/master/ast.go
*/
type AST struct {
	Pos lexer.Position

	Tags []*Tag `parser:"@@*"`
}

type Tag struct {
	Pos lexer.Position

	LHS        ScopedName   `parser:"( @@? "`
	Attributes []*Attribute `parser:"  @@* )!"`
	Children   []*Tag       `parser:"( \"{\" @@+ \"}\" )? \";\"*"`
}

type ScopedName struct {
	Namespace []string `parser:"( @Ident ( \".\"+ @Ident )*  \":\")?"`
	Name      []string `parser:"  @Ident ( \".\"+ @Ident )*"`
}

type Attribute struct {
	Pos lexer.Position

	LHS   *ScopedName `parser:"(@@ \"=\")?"`
	Value *Value      `parser:"@@"`
}

type Value struct {
	Pos lexer.Position

	Float    *float64  `parser:"@Float"`
	Int      *int64    `parser:"| @Int"`
	Null     bool      `parser:"| @Null"`
	Bool     *Bool     `parser:"| @Bool"`
	String   *string   `parser:"| @String"`
	DateTime *DateTime `parser:"| @@"`
	List     *List     `parser:"| @@"`
	// Bytes   *[]byte  // TODO
}

type Bool bool

func (b *Bool) Capture(v []string) error {
	*b = v[0] == "true"
	return nil
}

type DateTime struct {
	Pos lexer.Position

	Date *string `parser:"( @Date? "`
	Time *string `parser:"  @Time? )!"`
}

type List struct {
	Pos lexer.Position

	Elements []*Value `parser:"\"[\" ( @@ ( \",\"? @@ )* )? \"]\""`
}

/*
https://yourbasic.org/golang/regexp-cheat-sheet/
*/
var sdlParser = participle.MustBuild[AST](
	participle.UseLookahead(100),
	participle.Lexer(sdlLexer),
	participle.Unquote("String"),
)

var (
	sdlLexer = lexer.Must(lexer.New(lexer.Rules{
		"Root": {
			{
				Name:    "comment",
				Pattern: `(?:(?://|#|--)[^\n]*)|/\*.*?\*/`,
			}, {
				Name:    "wrap",
				Pattern: `\\\r?\n\r?`,
			}, {
				Name:    "eol",
				Pattern: `[\n\r]+`,
			}, {
				Name:    "whitespace",
				Pattern: `\s+`,
			}, {
				Name:    "Date", // 2023-01-30 | Thrs January 30, 2023
				Pattern: `(\d\d\d\d[-/]\d\d[-/]\d\d)|((\w+ )?\w+ \d\d?,? \d\d\d\d)`,
			}, {
				Name:    "Time", // T00:00:00Z | 2023-01-01 00:00:00Z
				Pattern: `[T\s]?\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d|([\.-]?(\w+)?))?`,
				// }, {
				// 	Name:    "DateTime", // Mon Dec 12 00:00:00 MST 2016
				// 	Pattern: `(\w+ )?\w+ \d\d?,? \d\d:\d\d:\d\d(?:\.\d+) `,
			}, {
				Name:    `Bool`,
				Pattern: `(true|false)`,
			}, {
				Name:    `Null`,
				Pattern: `null`,
			}, {
				Name:    "Float",
				Pattern: `[-+]?(?:\d+\.\d*|\.\d+)([eE][-+]?\d+)?`,
			}, {
				Name:    `Int`,
				Pattern: `[-+]?[0-9]+`,
			}, {
				Name: "Ident",
				//	Pattern: `\b[\pL\pN\w]*(-\w+)*\b`,
				//	Pattern: `[a-zA-Z_]\w*(-\w+)*`,
				Pattern: `[a-zA-Z_]\w*([\.-]\w+)*`,
				//Pattern: `[a-zA-Z_]*`,
				//Pattern: `\b[a-zA-Z_][\pL\pN]*\b`,
				//Pattern: `\b[[:alpha:]]\w*(-\w+)*\b`,
			}, {
				Name:    "String",
				Pattern: `"(\\\d\d\d|\\.|[^"])*"|'(\\\d\d\d|\\.|[^'])*'`,
			}, {
				Name:    "Punct",
				Pattern: `[][*?{}=:;,()|/\.]`,
			},

			// 	// //{Name: "EOL", Pattern: `[\n\r]+`},
			// 	// {Name: "whitespace", Pattern: `[\n\r\t\f ]+`},

		},
	}))
)

// type Date struct {
// 	Pos lexer.Position

// 	//Year *string `@Int`

// 	Year  *string `@(\d\d\d\d) [/-]`
// 	Month int     `@(\d\d) [/-]`
// 	Day   int     `@(\d\d)`
// }

// type Time struct {
// 	Pos lexer.Position

// 	Hour   int     `(?:[T ])@(\d\d) ":" `
// 	Minute int     `        @(\d\d) ":" `
// 	Second float64 `        @(\d\d(?:\.\d+))`
// 	Zone   string  `@("Z" | [+-] @(\d\d) ":" @(\d\d) | (-?GMT))?`
// }
