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

// attrVocabOption names the .consts.sdl option declaring UID-vocabulary
// subtrees (ZO §4.8): dotted canonic roots, semicolon-separated.  Leaves
// under a declared root mint UIDs used as VALUES (a Tag resolves to one),
// never as AttrIDs, so they are exempt from attr classification even when
// they fit the attr grammar (e.g. `channel.type.Spreadsheet`).
const attrVocabOption = "attr_vocab"

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

// attrIsTape decides which register call an attr compiles to.
//
// PROVISIONAL (TODO/000-backlog-astar #1, B-attr-autoreg §5, ruling (a)):
// the reserved `item.series.` tag literal IS the tape declaration — an attr
// whose canonic name carries it registers EditFlow_Tape; everything else
// folds.  This is the ONE site a ruling flip edits.
func attrIsTape(text string) bool {
	return strings.Contains(text, "item.series.")
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
//   - leaves under an `option attr_vocab` root are UID vocabulary;
//   - what remains IS an attr: its tail must resolve to exactly one message
//     type in the linked universe, or generation fails.
func classifyAttrs(src *ConstFile) ([]attrReg, error) {
	vocabRoots := attrVocabRoots(src)
	universe := typeUniverse()

	var regs []attrReg
	for _, decl := range src.Decls {
		if decl.Tags == nil || decl.Tags.Name != "Attr" {
			continue
		}
		flat := resolveTagEntries(decl.Tags.Entries, tag.Name{})
		for _, entry := range flat {
			if entry.isParent {
				continue
			}
			words := strings.Split(entry.text, ".")
			tail := words[len(words)-1]
			if !isTitleWord(tail) {
				continue
			}
			if tail == "UID" {
				continue // valueless attr: no prototype to register
			}
			if anyTitleWord(words[:len(words)-1]) {
				continue // vocabulary member or unit-schema path
			}
			if underVocabRoot(entry.text, vocabRoots) {
				continue // declared UID vocabulary
			}
			candidates := universe[tail]
			if len(candidates) == 0 {
				return nil, fmt.Errorf(
					"attr %q (%s): trailing word %q resolves to no message type (ZO §4.8); "+
						"declare a vocabulary root via `option %s` if this leaf is UID vocabulary",
					entry.varName, entry.text, tail, attrVocabOption)
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
				isTape:  attrIsTape(entry.text),
			})
		}
	}
	return regs, nil
}

// attrVocabRoots parses the attr_vocab option into canonic roots.
func attrVocabRoots(src *ConstFile) []string {
	var roots []string
	for _, part := range strings.Split(src.GetOption(attrVocabOption), ";") {
		if root := strings.TrimSpace(part); root != "" {
			roots = append(roots, root)
		}
	}
	return roots
}

func underVocabRoot(text string, roots []string) bool {
	for _, root := range roots {
		if text == root || strings.HasPrefix(text, root+".") {
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
