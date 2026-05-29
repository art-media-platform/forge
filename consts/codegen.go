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

// constsInfix is the ".consts" segment shared by every generated filename
// (<stem>.consts.<ext>) and trimmed off the input stem.  Emitters carry only
// their language extension (".go", ".cs", …); this is added at the path site.
const constsInfix = ".consts"

// GenOpts controls code generation output.
type GenOpts struct {
	SourceName string // source filename for header comment
}

// declSet buckets a parsed file's declarations by kind.  Every emitter opens by
// calling categorize, so they all share one view of the file.
type declSet struct {
	Tags    []*TagsBlock  // tags { ... } blocks
	Scalars []*ConstDecl  // top-level non-uid const scalars
	UIDs    []*ConstDecl  // top-level const uid declarations
	Groups  []*ConstGroup // const Name { ... } groups
}

// categorize sorts a file's top-level declarations into a declSet.
func categorize(src *ConstFile) declSet {
	var decls declSet
	for _, decl := range src.Decls {
		switch {
		case decl.Tags != nil:
			decls.Tags = append(decls.Tags, decl.Tags)
		case decl.Const != nil:
			if decl.Const.Type == "uid" {
				decls.UIDs = append(decls.UIDs, decl.Const)
			} else {
				decls.Scalars = append(decls.Scalars, decl.Const)
			}
		case decl.ConstGrp != nil:
			decls.Groups = append(decls.Groups, decl.ConstGrp)
		}
	}
	return decls
}

// needsTagName reports whether the file references the TagName shape — true when
// any tags block exists.  Runtime-less targets (TS, Python, C) emit TagName
// inline only when this holds.
func (decls declSet) needsTagName() bool { return len(decls.Tags) > 0 }

// needsUID reports whether the file references the UID type anywhere: TagName
// embeds a UID, and uid consts (top-level or inside a group) are bare UIDs.
func (decls declSet) needsUID() bool {
	if decls.needsTagName() || len(decls.UIDs) > 0 {
		return true
	}
	for _, grp := range decls.Groups {
		for _, member := range grp.Members {
			if member.Type == "uid" {
				return true
			}
		}
	}
	return false
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

// sectionIsGroup reports whether a section reads as a visual group worth setting
// off with a blank line: it carries a header comment or spans a parent entry
// plus its nested children.  Runs of flat single entries pack together.
func sectionIsGroup(sec *tagSection) bool {
	return sec.comment != "" || len(sec.entries) > 1
}

// blankBetweenSections reports whether a blank line belongs before section i,
// given the section list.  The first section never gets a leading blank; later
// sections get one only when they or their predecessor read as a group.
func blankBetweenSections(sections []tagSection, i int) bool {
	return i > 0 && (sectionIsGroup(&sections[i-1]) || sectionIsGroup(&sections[i]))
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
	remaining := max(sectionRuleWidth-contentWidth, 3)
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

// padRight right-pads s with spaces to width, for column alignment.  A width
// smaller than len(s) leaves s unchanged.
func padRight(s string, width int) string {
	if pad := width - len(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// Generator parses one .consts.sdl file and writes output for each requested
// language target.  The caller supplies the emitter set explicitly (see
// cmd/forge), so adding a target is a one-line edit there rather than a registry.
type Generator struct {
	InputPath string            // path to the .consts.sdl source
	Emitters  []Emitter         // candidate targets
	OutDirs   map[string]string // emitter Info().Flag → output directory ("" = skip)
	Opts      *GenOpts          // optional codegen options
}

// Run parses the source and writes each emitter whose flag maps to a non-empty
// directory in OutDirs.
func (g Generator) Run() error {
	text, err := os.ReadFile(g.InputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", g.InputPath, err)
	}

	src, err := Parse(g.InputPath, text)
	if err != nil {
		return err
	}

	// Attach source comments to AST nodes (protoc-style position matching)
	comments := ExtractComments(text)
	comments.Attach(src)

	// Normalize UUID-string `uid` literals into canonical hex-pair form.
	if err := rewriteUUIDLiterals(src); err != nil {
		return err
	}

	baseName := filepath.Base(g.InputPath)
	stem := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	stem = strings.TrimSuffix(stem, constsInfix) // amp.std.consts.sdl → amp.std

	opts := g.Opts
	if opts == nil {
		opts = &GenOpts{}
	}
	if opts.SourceName == "" {
		opts.SourceName = baseName
	}

	for _, emitter := range g.Emitters {
		info := emitter.Info()
		dir := g.OutDirs[info.Flag]
		if dir == "" {
			continue // target not requested
		}
		generated, err := emitter.Generate(src, opts)
		if err != nil {
			return fmt.Errorf("generating %s: %w", info.Language, err)
		}
		outPath := filepath.Join(dir, stem+constsInfix+info.Extension)
		if err := os.WriteFile(outPath, generated, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
		fmt.Printf("=> %s\n", outPath)
	}

	return nil
}
