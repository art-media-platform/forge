package consts

import "strings"

// CommentMap maps source line numbers to comment text, following the protoc
// convention of extracting comments by position and attaching to AST nodes
// after parsing.
type CommentMap struct {
	// Full-line comments (lines where the only content is a // comment)
	leading map[int]string
	// Trailing comments (// comments at end of a code line)
	trailing map[int]string
}

// ExtractComments scans source lines and categorizes each comment by type.
func ExtractComments(src []byte) *CommentMap {
	cm := &CommentMap{
		leading:  make(map[int]string),
		trailing: make(map[int]string),
	}
	for idx, line := range strings.Split(string(src), "\n") {
		lineNum := idx + 1 // 1-based, matching lexer.Position
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") {
			text := strings.TrimPrefix(trimmed, "//")
			if len(text) > 0 && text[0] == ' ' {
				text = text[1:] // trim at most one leading space
			}
			cm.leading[lineNum] = text
		} else if trimmed != "" {
			if tc := findTrailingComment(line); tc != "" {
				cm.trailing[lineNum] = tc
			}
		}
	}
	return cm
}

// LeadingComment returns consecutive comment lines immediately above the given line.
// Stops at blank lines or code lines (protoc "attached comment" semantics).
func (cm *CommentMap) LeadingComment(line int) string {
	var lines []string
	for ln := line - 1; ln >= 1; ln-- {
		text, ok := cm.leading[ln]
		if !ok {
			break
		}
		lines = append([]string{text}, lines...)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// TrailingComment returns the trailing comment on the given line, if any.
func (cm *CommentMap) TrailingComment(line int) string {
	return cm.trailing[line]
}

// Attach populates LeadComment and TrailComment fields on AST nodes
// by matching source positions to nearby comments.
func (cm *CommentMap) Attach(cf *ConstFile) {
	for _, decl := range cf.Decls {
		if decl.Tags != nil {
			decl.Tags.LeadComment = cm.LeadingComment(decl.Tags.Pos.Line)
			cm.attachTagEntries(decl.Tags.Entries)
		}
		if decl.Const != nil {
			decl.Const.LeadComment = cm.LeadingComment(decl.Const.Pos.Line)
			decl.Const.TrailComment = cm.TrailingComment(decl.Const.Pos.Line)
		}
		if decl.ConstGrp != nil {
			decl.ConstGrp.LeadComment = cm.LeadingComment(decl.ConstGrp.Pos.Line)
			for _, member := range decl.ConstGrp.Members {
				member.LeadComment = cm.LeadingComment(member.Pos.Line)
				member.TrailComment = cm.TrailingComment(member.Pos.Line)
			}
		}
	}
}

func (cm *CommentMap) attachTagEntries(entries []*TagEntry) {
	for _, entry := range entries {
		entry.LeadComment = cm.LeadingComment(entry.Pos.Line)
		entry.TrailComment = cm.TrailingComment(entry.Pos.Line)
		if len(entry.Children) > 0 {
			cm.attachTagEntries(entry.Children)
		}
	}
}

// findTrailingComment extracts a // comment from the end of a code line,
// skipping // that appears inside a quoted string.
func findTrailingComment(line string) string {
	inString := false
	escaped := false
	for idx := 0; idx < len(line); idx++ {
		ch := line[idx]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' || ch == '\'' {
			inString = !inString
			continue
		}
		if !inString && idx+1 < len(line) && line[idx] == '/' && line[idx+1] == '/' {
			text := strings.TrimSpace(line[idx+2:])
			return text
		}
	}
	return ""
}
