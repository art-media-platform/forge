package consts

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/art-media-platform/amp.SDK/stdlib/tag"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	// The attr type universe: message types these packages link in are what
	// an attr's trailing name word resolves against (ZO §4.8).  forge rides
	// the amp.SDK release train — a tail naming a type newer than forge's
	// pinned amp.SDK surfaces as a generation error until forge re-pins.
	_ "github.com/art-media-platform/amp.SDK/amp"
	_ "github.com/art-media-platform/amp.SDK/amp/std"
	_ "github.com/art-media-platform/amp.SDK/stdlib/safe"
	_ "github.com/art-media-platform/amp.SDK/stdlib/status"
)

// SDK anchors for the generated registration rails.  The Go rail calls the
// std registrar; the C# rail rides the SDK's root namespace types
// (Google.Protobuf parsers + the EditFlow enum).
const (
	sdkStdGoPkg = "github.com/art-media-platform/amp.SDK/amp/std"
	sdkAmpGoPkg = "github.com/art-media-platform/amp.SDK/amp"
	sdkPkgRoot  = "github.com/art-media-platform/amp.SDK/"
)

// Declared tag-entry flags (ZO §4.8 markup — astar 08-08: declared SDL
// markup over convention, "future proof and does not rely on conventions").
// `tape` declares EditFlow_Tape; `vocab` exempts a UID-vocabulary leaf or
// subtree (its UIDs are VALUES a Tag resolves to, never AttrIDs — e.g.
// `channel.type.Spreadsheet`).  Unmarked = fold, the universal default.
// A subtree root's flags inherit; a per-leaf declaration wins.
const (
	attrFlagTape  = "tape"
	attrFlagVocab = "vocab"
)

// msgType is one resolvable message type in the linked type universe.
type msgType struct {
	name        string // Go type name == C# class name == the §4.8 tail
	goPkgPath   string // Go import path
	goPkgName   string // Go package ident
	csNamespace string // csharp_namespace of the owning .proto
}

// attrReg is one attr registration the generator emits.
type attrReg struct {
	varName string  // consts ident, e.g. "AppState"
	text    string  // case-preserved canonic, e.g. "session.app.state.Tag"
	msg     msgType // resolved prototype message type
	isTape  bool    // EditFlow_Tape vs EditFlow_Fold (see attrIsTape)
}

// classifyAttrs walks every `tags Attr` block and returns the attrs the
// registration rails emit, in declaration order.  ZO §4.8 compiled:
//
//   - only leaves classify (a parent entry is a namespace node);
//   - the trailing name word must be PublicTitleCase to be a value-shape
//     tail; lowercase leaves are use-scope nodes, item keys, or unit tails;
//   - a TitleCase word BEFORE the tail marks a UID-vocabulary member or a
//     unit-schema path (`amp.law.MemberKind.Person`, `item.series.S2.UTC64`)
//     — never an attr;
//   - `.UID` is the valueless tail (the ItemID is the payload) — an attr
//     with no prototype registers nothing;
//   - a leaf whose effective flags carry `vocab` is UID vocabulary;
//   - what remains IS an attr: its tail must resolve to exactly one message
//     type in the linked universe, or generation fails; `tape` declares
//     EditFlow_Tape, unmarked folds.
//
// Flag validation is strict: an unknown flag, `tape, vocab` together, a
// flag outside a `tags Attr` block, or an explicit `tape` on a leaf that
// does not classify as an attr are all generation errors — dead markup
// cannot sit silently.
func classifyAttrs(src *ConstFile) ([]attrReg, error) {
	universe := typeUniverse()

	if err := validateAttrFlags(src); err != nil {
		return nil, err
	}

	var regs []attrReg
	for _, decl := range src.Decls {
		if decl.Tags == nil || decl.Tags.Name != "Attr" {
			continue
		}
		flat := resolveTagEntries(decl.Tags.Entries, tag.Name{}, nil)
		for _, entry := range flat {
			isTape := hasFlag(entry.flags, attrFlagTape)
			if entry.isParent {
				continue
			}
			if hasFlag(entry.flags, attrFlagVocab) {
				continue // declared UID vocabulary
			}
			words := strings.Split(entry.text, ".")
			tail := words[len(words)-1]
			isAttrShape := isTitleWord(tail) && tail != "UID" &&
				!anyTitleWord(words[:len(words)-1])
			if !isAttrShape {
				// An entry's OWN `: tape` on a non-attr leaf is dead markup;
				// a subtree-inherited tape passing over unit tails is not.
				if isTape && entry.flagsDeclared {
					return nil, fmt.Errorf(
						"entry %q (%s): `: tape` on a leaf that is not an attr "+
							"(no message-type tail) — dead markup (ZO §4.8)",
						entry.varName, entry.text)
				}
				continue // use-scope node, item key, unit tail, vocab member, or .UID
			}
			candidates := universe[tail]
			if len(candidates) == 0 {
				return nil, fmt.Errorf(
					"attr %q (%s): trailing word %q resolves to no message type (ZO §4.8); "+
						"mark the leaf or its subtree root `: %s` if it is UID vocabulary",
					entry.varName, entry.text, tail, attrFlagVocab)
			}
			if len(candidates) > 1 {
				var homes []string
				for _, c := range candidates {
					homes = append(homes, c.goPkgPath)
				}
				return nil, fmt.Errorf(
					"attr %q (%s): trailing word %q is ambiguous across %s",
					entry.varName, entry.text, tail, strings.Join(homes, ", "))
			}
			regs = append(regs, attrReg{
				varName: entry.varName,
				text:    entry.text,
				msg:     candidates[0],
				isTape:  isTape,
			})
		}
	}
	return regs, nil
}

// validateAttrFlags walks every tags block's DECLARED (not inherited) flags:
// names must be known, `tape, vocab` cannot combine, and flags outside a
// `tags Attr` block have no meaning.
func validateAttrFlags(src *ConstFile) error {
	var walk func(blockName string, entries []*TagEntry) error
	walk = func(blockName string, entries []*TagEntry) error {
		for _, entry := range entries {
			seen := map[string]bool{}
			for _, flag := range entry.Flags {
				if flag != attrFlagTape && flag != attrFlagVocab {
					return fmt.Errorf("entry %q: unknown flag %q (known: %s, %s)",
						entry.VarName, flag, attrFlagTape, attrFlagVocab)
				}
				if blockName != "Attr" {
					return fmt.Errorf("entry %q: flag %q outside a `tags Attr` block",
						entry.VarName, flag)
				}
				seen[flag] = true
			}
			if seen[attrFlagTape] && seen[attrFlagVocab] {
				return fmt.Errorf("entry %q: `%s, %s` conflict — an entry is one or the other",
					entry.VarName, attrFlagTape, attrFlagVocab)
			}
			if err := walk(blockName, entry.Children); err != nil {
				return err
			}
		}
		return nil
	}
	for _, decl := range src.Decls {
		if decl.Tags == nil {
			continue
		}
		if err := walk(decl.Tags.Name, decl.Tags.Entries); err != nil {
			return err
		}
	}
	return nil
}

func hasFlag(flags []string, flag string) bool {
	for _, one := range flags {
		if one == flag {
			return true
		}
	}
	return false
}

func isTitleWord(word string) bool {
	first, _ := utf8.DecodeRuneInString(word)
	return unicode.IsUpper(first)
}

func anyTitleWord(words []string) bool {
	for _, word := range words {
		if isTitleWord(word) {
			return true
		}
	}
	return false
}

var (
	typeUniverseOnce sync.Once
	typeUniverseMap  map[string][]msgType
)

// typeUniverse indexes every amp.SDK message type linked into forge by its
// Go type name.  Built once from the protobuf global registry; the Go type,
// its import path, and the owning file's csharp_namespace all come from the
// generated descriptors — no hand mapping.
func typeUniverse() map[string][]msgType {
	typeUniverseOnce.Do(func() {
		typeUniverseMap = make(map[string][]msgType)
		protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
			goType := goTypeOf(mt)
			if goType.name == "" || !strings.HasPrefix(goType.goPkgPath, sdkPkgRoot) {
				return true
			}
			fileOpts, _ := mt.Descriptor().ParentFile().Options().(*descriptorpb.FileOptions)
			goType.csNamespace = fileOpts.GetCsharpNamespace()
			typeUniverseMap[goType.name] = append(typeUniverseMap[goType.name], goType)
			return true
		})
		for _, candidates := range typeUniverseMap {
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].goPkgPath < candidates[j].goPkgPath
			})
		}
	})
	return typeUniverseMap
}

// goTypeOf reflects a registered message type's concrete Go identity.
func goTypeOf(mt protoreflect.MessageType) msgType {
	goType := reflect.TypeOf(mt.Zero().Interface())
	if goType == nil {
		return msgType{}
	}
	if goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	return msgType{
		name:      goType.Name(),
		goPkgPath: goType.PkgPath(),
		goPkgName: path.Base(goType.PkgPath()),
	}
}
