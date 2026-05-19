package sdl

import (
	"strconv"
	"strings"
)

func (sn *ScopedName) Export(out []byte) []byte {
	if sn == nil {
		return out
	}
	if len(sn.Namespace) > 0 {
		for i, ident := range sn.Namespace {
			if i > 0 {
				out = append(out, '.')
			}
			out = append(out, ident...)
		}
		out = append(out, ':')
	}
	if len(sn.Name) > 0 {
		for i, ident := range sn.Name {
			if i > 0 {
				out = append(out, '.')
			}
			out = append(out, ident...)
		}
	}
	return out
}

func (sn *ScopedName) IsSet() bool {
	if sn == nil {
		return false
	}
	for _, ident := range sn.Namespace {
		if len(ident) > 0 {
			return true
		}
	}
	for _, ident := range sn.Name {
		if len(ident) > 0 {
			return true
		}
	}
	return false

}

func (tag *Tag) Export(out []byte, opts ExportOpts) ([]byte, error) {

	out = append(out, opts.Indent...)
	out = tag.LHS.Export(out)

	for _, attr := range tag.Attributes {
		out = append(out, ' ')

		var err error
		out, err = attr.Export(out, opts)
		if err != nil {
			return out, err
		}
	}

	if tag.Children != nil {
		out = append(out, ' ', '{', '\n')
		opts.Indent = append(opts.Indent, ' ', ' ', ' ', ' ')

		for _, child := range tag.Children {
			var err error
			out, err = child.Export(out, opts)
			if err != nil {
				return out, err
			}
		}
		opts.Indent = opts.Indent[:len(opts.Indent)-4]
		out = append(out, opts.Indent...)
		out = append(out, '}')
	}

	out = append(out, '\n')
	return out, nil

}

func (attr *Attribute) Export(out []byte, opts ExportOpts) ([]byte, error) {
	if attr.LHS != nil {
		out = attr.LHS.Export(out)
		out = append(out, '=')
	}

	if attr.Value != nil {
		var err error
		out, err = attr.Value.Export(out, opts)
		if err != nil {
			return out, err
		}
	}

	return out, nil
}

func (val *Value) AsInt() int64 {
	if val.Int != nil {
		return *val.Int
	}

	if val.String != nil {
		value, err := parseInt(*val.String)
		if err == nil {
			return value
		}
	}

	f64 := val.AsFloat()
	return int64(f64)
}

func (val *Value) AsFloat() float64 {
	if val.Float != nil {
		return *val.Float
	}

	if val.String != nil {
		value, err := strconv.ParseFloat(*val.String, 64)
		if err == nil {
			return value
		}
	}

	// Try getting first float from nested tuple
	// if val.Tuple != nil && len(val.Tuple.Elements) > 0 {
	// 	return val.Tuple.Elements[0].AsFloat()
	// }

	return 0.0
}

func (val *Value) Export(out []byte, opts ExportOpts) ([]byte, error) {
	var err error

	switch {
	// case val.Bool != nil:
	// 	return fmt.Sprintf("%v", *l.Bool)
	case val.String != nil:
		quoted := strconv.Quote(*val.String)
		out = append(out, []byte(quoted)...)
	case val.Float != nil:
		text := strconv.FormatFloat(*val.Float, 'g', -1, 64)
		out = append(out, []byte(text)...)
		if strings.IndexByte(text, '.') < 0 {
			out = append(out, '.')
		}
	case val.Int != nil:
		out = append(out, []byte(strconv.FormatInt(*val.Int, 10))...)
	case val.List != nil:
		out, err = val.List.Export(out, opts)
	case val.DateTime != nil:
		out, err = val.DateTime.Export(out, opts)
	case val.Bool != nil:
		if *val.Bool {
			out = append(out, "true"...)
		} else {
			out = append(out, "false"...)
		}
	case val.Null:
		out = append(out, "null"...)
	default:
		err = ErrValueNotSet
	}

	return out, err
}

func (dt *DateTime) Export(out []byte, opts ExportOpts) ([]byte, error) {
	hasDate := false
	if dt.Date != nil && len(*dt.Date) > 0 {
		out = append(out, *dt.Date...)
		hasDate = true
	}
	if dt.Time != nil {
		if hasDate {
			out = append(out, ' ')
		}
		out = append(out, *dt.Time...)
	}
	return out, nil

}

func (list *List) Export(out []byte, opts ExportOpts) ([]byte, error) {
	out = append(out, '[')

	rowStart := len(out)
	i_last := len(list.Elements) - 1

	for i, val_i := range list.Elements {
		var err error
		out, err = val_i.Export(out, opts)
		if err != nil {
			return out, err
		}
		if i < i_last {
			out = append(out, ',')
			rowLen := len(out) - rowStart
			if rowLen > 60 {
				out = append(out, '\n')
				out = append(out, opts.Indent...)
			}
		}

	}

	out = append(out, ']')
	return out, nil
}

func (val *Value) GoString() string {
	text, _ := val.Export(make([]byte, 0, 127), ExportOpts{})
	return string(text)
}

// func (v *Value) String() string {
// 	line := v.MarshalOut(make([]byte, 0, 127), 0)
// 	return string(line)
// }

/*
	func (entry *Entry) MarshalOut(out []byte, opts MarshalOpts) []byte {
		if entry.LHS != nil {
			out = entry.LHS.MarshalOut(out, opts)
		}
		if entry.RHS != nil {
			out = append(out, ' ', '=', ' ')
			out = entry.RHS.MarshalOut(out, opts)
		}
		return out
	}

	func (entry *Entry) String() string {
		line := entry.MarshalOut(make([]byte, 0, 127), 0)
		return string(line)
	}

// MarshalINI appends a UTF8 text serialization of this INI AST.

	func (ini *INI) MarshalOut(opts MarshalOpts) (string, error) {
		out := make([]byte, 0, 4096)

		for _, entry := range ini.Properties {
			out = entry.MarshalOut(out, opts)
			// if entry.Comment != nil {
			// 	out = append(out, ' ', '#', ' ')
			// 	out = append(out, *entry.Comment...)
			// }
			out = append(out, '\n')
		}

		out = append(out, '\n')

		for _, sect := range ini.Sections {
			out = append(out, '\n')
			out = sect.MarshalOut(out, opts)
			out = append(out, '\n')
		}

		return string(out), nil
	}
*/
func parseInt(str string) (int64, error) {
	lower := strings.ToLower(strings.TrimSpace(str))

	if strings.HasPrefix(lower, "0x") {
		value, err := strconv.ParseInt(lower[2:], 16, 64)
		if err == nil {
			return value, nil
		}
	}
	if strings.HasPrefix(lower, "0b") {
		value, err := strconv.ParseInt(lower[2:], 2, 64)
		if err == nil {
			return value, nil
		}
	}
	if strings.HasPrefix(lower, "0o") {
		value, err := strconv.ParseInt(lower[2:], 8, 64)
		if err == nil {
			return value, nil
		}
	}

	return strconv.ParseInt(lower, 10, 64)
}
