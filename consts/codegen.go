package consts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/art-media-platform/amp.SDK/stdlib/tag"
)

const sectionRuleWidth = 64 // total rune-width of ─── divider lines

// GenOpts controls code generation output.
type GenOpts struct {
	SourceName string // source filename for header comment
}

// resolvedEntry holds pre-computed data for one tag entry output line.
// All entries are tag.Name — callers use .ID when they need the UID.
type resolvedEntry struct {
	varName string
	literal string // the "literal" from the .consts.sdl source
	canonic string // dot-delimited canonic path
	base32  string // 27-char Crockford Base32 UID
	uidHi   uint64 // pre-computed UID[0]
	uidLo   uint64 // pre-computed UID[1]

	leadComment  string // doc comment from source (lines above)
	trailComment string // inline comment from source (same line)
	isParent     bool   // has children → visual break after
}

// tagSection groups entries under a section header for column alignment.
type tagSection struct {
	comment string          // section header from source comment on top-level entry
	entries []resolvedEntry // flattened tree of the top-level entry + children
}

// buildTagSections creates one alignment section per top-level entry in a tags block.
func buildTagSections(entries []*TagEntry) []tagSection {
	var sections []tagSection
	for _, entry := range entries {
		flat := resolveTagEntries([]*TagEntry{entry}, tag.Name{})
		sections = append(sections, tagSection{
			comment: entry.LeadComment,
			entries: flat,
		})
	}
	return sections
}

// resolveTagEntries recursively flattens tag entries into output lines,
// computing canonic paths and UIDs at codegen time.
// All entries emit as tag.Name — callers use .ID when they need the bare UID.
func resolveTagEntries(entries []*TagEntry, parentName tag.Name) []resolvedEntry {
	var result []resolvedEntry
	for _, entry := range entries {
		entryName := parentName.With(entry.Literal)
		uid := entryName.ID

		result = append(result, resolvedEntry{
			varName:      entry.VarName,
			literal:      entry.Literal,
			canonic:      entryName.Canonic,
			base32:       uid.Base32(),
			uidHi:        uid[0],
			uidLo:        uid[1],
			leadComment:  entry.LeadComment,
			trailComment: entry.TrailComment,
			isParent:     len(entry.Children) > 0,
		})

		if len(entry.Children) > 0 {
			result = append(result, resolveTagEntries(entry.Children, entryName)...)
		}
	}
	return result
}

// sectionDivider creates a ─── decorated section header from a single line.
//
//	─── Channel metadata and catalog ─────────────────────────
//
// Multi-line comments must be split by the caller (see emitSectionHeader);
// only the first line becomes the decorated bar.
func sectionDivider(text string) string {
	prefix := "─── "
	suffix := " "
	content := prefix + text + suffix
	contentWidth := utf8.RuneCountInString(content)
	remaining := sectionRuleWidth - contentWidth
	if remaining < 3 {
		remaining = 3
	}
	return content + strings.Repeat("─", remaining)
}

// emitSectionHeader writes a decorated section bar from the first line of
// comment, followed by any remaining lines as plain prose comments below.
// linePrefix is the per-line comment lead (e.g. "\t// " for Go, "    // " for C#).
func emitSectionHeader(buf *strings.Builder, comment, linePrefix string) {
	lines := strings.Split(comment, "\n")
	buf.WriteString(linePrefix + sectionDivider(lines[0]) + "\n")
	for _, line := range lines[1:] {
		buf.WriteString(linePrefix + line + "\n")
	}
}

// Generate parses a .consts.sdl file and writes Go and/or C# output.
func Generate(inputPath, goOutDir, csharpOutDir string, opts *GenOpts) error {
	src, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inputPath, err)
	}

	cf, err := Parse(inputPath, src)
	if err != nil {
		return err
	}

	// Attach source comments to AST nodes (protoc-style position matching)
	comments := ExtractComments(src)
	comments.Attach(cf)

	// Normalize UUID-string `uid` literals into canonical hex-pair form.
	if err := rewriteUUIDLiterals(cf); err != nil {
		return err
	}

	baseName := filepath.Base(inputPath)
	stem := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	stem = strings.TrimSuffix(stem, ".consts") // amp.std.consts.sdl → amp.std

	if opts == nil {
		opts = &GenOpts{}
	}
	if opts.SourceName == "" {
		opts.SourceName = baseName
	}

	if goOutDir != "" {
		out, err := GenerateGo(cf, opts)
		if err != nil {
			return fmt.Errorf("generating Go: %w", err)
		}
		outPath := filepath.Join(goOutDir, stem+".consts.go")
		if err := os.WriteFile(outPath, out, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
		fmt.Printf("=> %s\n", outPath)
	}

	if csharpOutDir != "" {
		out, err := GenerateCSharp(cf, opts)
		if err != nil {
			return fmt.Errorf("generating C#: %w", err)
		}
		outPath := filepath.Join(csharpOutDir, stem+".consts.cs")
		if err := os.WriteFile(outPath, out, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
		fmt.Printf("=> %s\n", outPath)
	}

	return nil
}
