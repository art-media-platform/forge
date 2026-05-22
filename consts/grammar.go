// Package consts parses .consts.sdl files and generates Go, C#, TypeScript,
// Python, and C constant definitions from a single source of truth.
//
// The .consts.sdl format is a proto3-inspired DSL for declaring hierarchical tag
// names and scalar constants that compile to multiple languages.
//
// Like protoc, source comments are extracted by position and attached to AST nodes
// after parsing, then canonized into native-language doc strings.
package consts

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// ConstFile is the root AST node for a .consts.sdl file.
type ConstFile struct {
	Pos   lexer.Position
	Decls []*Decl `parser:"@@*"`
}

// Decl is a top-level declaration: option, tags block, const group, or const scalar.
type Decl struct {
	Pos      lexer.Position
	Option   *Option     `parser:"  @@"`
	Tags     *TagsBlock  `parser:"| @@"`
	ConstGrp *ConstGroup `parser:"| @@"`
	Const    *ConstDecl  `parser:"| @@"`
}

// Option is a proto-style file option: option go_package = "...";
type Option struct {
	Pos   lexer.Position
	Name  string `parser:"\"option\" @Ident"`
	Value string `parser:"\"=\" @String \";\""`
}

// TagsBlock declares a hierarchy of tag name constants.
// The block name becomes the C# outer class name (e.g. "ID").
type TagsBlock struct {
	Pos     lexer.Position
	Name    string      `parser:"\"tags\" @Ident"`
	Entries []*TagEntry `parser:"\"{\" @@* \"}\""`

	LeadComment string // populated post-parse by comment extraction
}

// TagEntry is a single tag name declaration, optionally with children.
// All entries emit as tag.Name / TagName; callers use .ID for the bare UID.
type TagEntry struct {
	Pos      lexer.Position
	IsName   bool        `parser:"@\"name\"?"`
	VarName  string      `parser:"@Ident"`
	Literal  string      `parser:"@String"`
	Children []*TagEntry `parser:"( \"{\" @@* \"}\" )?"`

	LeadComment  string // populated post-parse (lines above → doc comment)
	TrailComment string // populated post-parse (same-line → inline comment)
}

// ConstGroup is a named group of scalar constants: const GroupName { ... }
// C# emits as a nested static class; Go prefixes the group name.
type ConstGroup struct {
	Pos     lexer.Position
	Name    string       `parser:"\"const\" @Ident"`
	Members []*ConstDecl `parser:"\"{\" @@* \"}\""`

	LeadComment string // populated post-parse
}

// ConstDecl is a scalar constant: const string Foo = "bar";
// Inside a ConstGroup, the leading "const" keyword is omitted.
type ConstDecl struct {
	Pos     lexer.Position
	Type    string `parser:"\"const\"? @(\"string\" | \"int32\" | \"int64\" | \"uint32\" | \"uint64\" | \"float32\" | \"float64\" | \"float\" | \"double\" | \"fixed64\" | \"uid\")"`
	VarName string `parser:"@Ident"`
	Value   *Value `parser:"\"=\" @@ \";\""`

	LeadComment  string // populated post-parse
	TrailComment string // populated post-parse
}

// UIDLit is a UID literal pair: {hex, hex}
type UIDLit struct {
	Pos lexer.Position
	Hi  string `parser:"\"{\" @Hex"`
	Lo  string `parser:"\",\" @Hex \"}\""`
}

// Value holds a typed constant value (string, integer, float, hex, or UID pair).
type Value struct {
	Pos     lexer.Position
	UIDPair *UIDLit  `parser:"  @@"`
	String  *string  `parser:"| @String"`
	Float   *float64 `parser:"| @Float"`
	Hex     *string  `parser:"| @Hex"`
	Int     *int64   `parser:"| @Int"`
}

var constParser = participle.MustBuild[ConstFile](
	participle.UseLookahead(50),
	participle.Lexer(constLexer),
	participle.Unquote("String"),
)

var constLexer = lexer.Must(lexer.New(lexer.Rules{
	"Root": {
		{Name: "comment", Pattern: `(?://[^\n]*)|/\*.*?\*/`},
		{Name: "eol", Pattern: `[\n\r]+`},
		{Name: "whitespace", Pattern: `[ \t]+`},
		{Name: "Float", Pattern: `[-+]?(?:\d+\.\d*|\.\d+)([eE][-+]?\d+)?`},
		{Name: "Hex", Pattern: `0[xX][0-9a-fA-F]+`},
		{Name: "Int", Pattern: `[-+]?[0-9]+`},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
		{Name: "String", Pattern: `"(\\\d\d\d|\\.|[^"])*"|'(\\\d\d\d|\\.|[^'])*'`},
		{Name: "Punct", Pattern: `[{}=;,]`},
	},
}))

// Parse parses a .consts.sdl file from bytes.
func Parse(filename string, src []byte) (*ConstFile, error) {
	return constParser.ParseBytes(filename, src)
}

// GetOption returns the value for a named option, or "" if not found.
func (src *ConstFile) GetOption(name string) string {
	for _, decl := range src.Decls {
		if decl.Option != nil && decl.Option.Name == name {
			return decl.Option.Value
		}
	}
	return ""
}
