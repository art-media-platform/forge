package consts

import (
	"strings"
	"testing"
)

// parseSDL is a test helper: parse an inline .consts.sdl body.
func parseSDL(t *testing.T, body string) *ConstFile {
	t.Helper()
	src, err := Parse("attrs_test.sdl", []byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return src
}

// The classification contract (ZO §4.8 compiled): leaves with a TitleCase
// message-type tail register; parents, lowercase leaves, `.UID` tails,
// mid-path TitleCase (vocabulary members / unit schemas), and `: vocab`
// leaves/subtrees never do.  `: tape` declares EditFlow_Tape (a subtree
// root's flags inherit — SeriesLabels below — and inherited tape passing
// over a unit tail is inert); `: sealed` declares a SealedValue cell and
// folds; unmarked folds.
func TestClassifyAttrs(t *testing.T) {
	src := parseSDL(t, `
tags Attr {
    ItemAttr "item" {
        ItemLabels "Labels"
        ChildLink  "child.link.UID"
        TileAttr   "tile"
        ItemSeries "series" : tape {
            SeriesAssetTag "asset.Tag" : tape
            SeriesLabels   "Labels"
            SeriesS2T      "S2.UTC64"
        }
    }
    NodeAttr "node" {
        NodeCredentials "Labels" : sealed
    }
    ChannelAttr "channel" {
        ChannelType "type" : vocab {
            ChannelTypeSpreadsheet "Spreadsheet"
        }
    }
    LawAttr "amp.law" {
        LawMemberKind "MemberKind" {
            LawMemberKind_Person "Person"
        }
    }
}

tags Crypto {
    NotAnAttr "NotARealMessageXYZ"
}
`)
	regs, err := classifyAttrs(src)
	if err != nil {
		t.Fatalf("classifyAttrs: %v", err)
	}

	want := map[string]struct {
		text     string
		msg      string
		isTape   bool
		isSealed bool
	}{
		"ItemLabels":      {"item.Labels", "Labels", false, false},
		"SeriesAssetTag":  {"item.series.asset.Tag", "Tag", true, false},
		"SeriesLabels":    {"item.series.Labels", "Labels", true, false},
		"NodeCredentials": {"node.Labels", "Labels", false, true},
	}
	if len(regs) != len(want) {
		var got []string
		for _, reg := range regs {
			got = append(got, reg.varName)
		}
		t.Fatalf("classified %v, want exactly %d attrs", got, len(want))
	}
	for _, reg := range regs {
		expect, ok := want[reg.varName]
		if !ok {
			t.Errorf("unexpected attr %q (%s)", reg.varName, reg.text)
			continue
		}
		if reg.text != expect.text || reg.msg.name != expect.msg ||
			reg.isTape != expect.isTape || reg.isSealed != expect.isSealed {
			t.Errorf("attr %q: got (%s, %s, tape=%v, sealed=%v), want (%s, %s, tape=%v, sealed=%v)",
				reg.varName, reg.text, reg.msg.name, reg.isTape, reg.isSealed,
				expect.text, expect.msg, expect.isTape, expect.isSealed)
		}
	}
}

// A TitleCase tail resolving to no linked message type is a generation
// error — §4.8 has no silent skip for a typo'd attr.
func TestClassifyAttrsUnresolvedTailErrors(t *testing.T) {
	src := parseSDL(t, `
tags Attr {
    ItemAttr "item" {
        Typo "MediaInfoo"
    }
}
`)
	_, err := classifyAttrs(src)
	if err == nil {
		t.Fatal("classifyAttrs accepted an unresolvable tail")
	}
	if !strings.Contains(err.Error(), "MediaInfoo") || !strings.Contains(err.Error(), "§4.8") {
		t.Fatalf("error names neither the tail nor §4.8: %v", err)
	}
}

// Flag validation is strict: unknown names, `vocab` combined with either
// other flag, `sealed, tape` together, flags outside `tags Attr`, and an OWN
// `: tape` / `: sealed` on a non-attr leaf all fail generation — dead markup
// cannot sit silently.
func TestClassifyAttrsFlagValidation(t *testing.T) {
	cases := []struct {
		name string
		sdl  string
		want string
	}{
		{
			"unknown flag",
			`tags Attr { ItemAttr "item" { ItemLabels "Labels" : tap } }`,
			"unknown flag",
		},
		{
			"tape+vocab conflict",
			`tags Attr { ItemAttr "item" { ItemLabels "Labels" : tape, vocab } }`,
			"conflict",
		},
		{
			"flag outside tags Attr",
			`tags Crypto { Poly "amp.crypto.poly" : tape }`,
			"outside",
		},
		{
			"own tape on a non-attr leaf",
			`tags Attr { ItemAttr "item" { TileAttr "tile" : tape } }`,
			"dead markup",
		},
		{
			"sealed+tape conflict",
			`tags Attr { ItemAttr "item" { ItemLabels "Labels" : sealed, tape } }`,
			"conflict",
		},
		{
			"sealed+vocab conflict",
			`tags Attr { ItemAttr "item" { ItemLabels "Labels" : sealed, vocab } }`,
			"conflict",
		},
		{
			"sealed outside tags Attr",
			`tags Crypto { Poly "amp.crypto.poly" : sealed }`,
			"outside",
		},
		{
			"own sealed on a non-attr leaf",
			`tags Attr { ItemAttr "item" { TileAttr "tile" : sealed } }`,
			"dead markup",
		},
	}
	for _, c := range cases {
		src := parseSDL(t, c.sdl)
		_, err := classifyAttrs(src)
		if err == nil {
			t.Errorf("%s: accepted", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q lacks %q", c.name, err, c.want)
		}
	}
}

// The type universe resolves cross-package tails with their C# namespaces —
// the identity data both rails emit from.
func TestTypeUniverseResolution(t *testing.T) {
	universe := typeUniverse()
	cases := []struct {
		tail        string
		goPkgPath   string
		csNamespace string
	}{
		{"Tag", sdkAmpGoPkg, "art.media.platform"},
		{"Labels", sdkStdGoPkg, "art.media.platform.std"},
		{"Status", "github.com/art-media-platform/amp.SDK/stdlib/status", "art.media.platform.status"},
	}
	for _, c := range cases {
		candidates := universe[c.tail]
		if len(candidates) != 1 {
			t.Errorf("tail %q: %d candidates, want 1", c.tail, len(candidates))
			continue
		}
		if candidates[0].goPkgPath != c.goPkgPath || candidates[0].csNamespace != c.csNamespace {
			t.Errorf("tail %q: got (%s, %s), want (%s, %s)", c.tail,
				candidates[0].goPkgPath, candidates[0].csNamespace, c.goPkgPath, c.csNamespace)
		}
	}
}
