package sdl

import (
	"errors"

	"github.com/alecthomas/participle/v2"
)

var (
	ErrValueNotSet = errors.New("Value not set")
)

type ExportOpts struct {
	Indent []byte // prefix for each line
}

// ParseBytes parses SDL from UTF8 text into an AST.
func ParseBytes(fileDesc string, text []byte) (*AST, error) {
	ini, err := sdlParser.ParseBytes(fileDesc, text)
	return ini, err
}

// ParseString parses SDL from a string into an AST.
func ParseString(fileDesc, text string) (*AST, error) {
	return ParseBytes(fileDesc, []byte(text))
}

func Parser() *participle.Parser[AST] {
	return sdlParser
}

// Initiates enumeration of all attributes in the AST, calling kvOut for each key-value pair.
func (ast *AST) EnumAttributes(kvOut func(attrID []byte, rhs *Value) bool) {
	attrID := make([]byte, 0, 255) // push and pop the attribute ID as we recurse, starting with an nil ID
	EnumAttributes(attrID, ast.Tags, kvOut)
}

// High level call to export a generic sdl AST
func (ast *AST) Export(out []byte, opts ExportOpts) ([]byte, error) {
	var err error
	for _, ti := range ast.Tags {
		out, err = ti.Export(out, opts)
		if err != nil {
			return out, err
		}
		out = append(out, '\n')
	}
	return out, err
}

func Join(attr, suffix []byte) []byte {
	if len(suffix) == 0 {
		return attr
	}
	if len(attr) == 0 {
		return append(attr, suffix...)
	}

	L := attr[len(attr)-1]
	if L != '.' && suffix[0] != '.' {
		attr = append(attr, '.')
		return append(attr, suffix...)
	} else if L == '.' && suffix[0] == '.' {
		return append(attr, suffix[1:]...)
	} else {
		return append(attr, suffix...)
	}
}

// Enumerates all attributes in the given tags, calling kvOut for each key-value pair.
func EnumAttributes(baseAttrID []byte, tags []*Tag, kvOut func(attrID []byte, RHS *Value) bool) {
	tmp := make([]byte, 0, 255)

	for _, ti := range tags {
		if ti == nil {
			continue
		}

		suffix := ti.LHS.Export(tmp)
		tagAttrID := Join(baseAttrID, suffix)

		for _, ai := range ti.Attributes {
			suffix = ai.LHS.Export(tmp)
			ai_ID := Join(tagAttrID, suffix)
			if !kvOut(ai_ID, ai.Value) {
				return
			}
		}
		if len(ti.Children) > 0 { // recurse into children
			EnumAttributes(tagAttrID, ti.Children, kvOut)
		}
	}
}
