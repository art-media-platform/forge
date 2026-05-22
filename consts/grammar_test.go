package consts

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGolden regenerates Go, C#, TypeScript, Python, and C output from the
// golden .consts.sdl input and writes the results alongside it. A developer
// reviews any diff against what's committed to verify correctness.
//
// To accept changes after an intentional edit: commit the updated golden files.
func TestGolden(t *testing.T) {
	goldenDir := "golden"
	sdlPath := filepath.Join(goldenDir, "grammar_test.sdl")

	text, err := os.ReadFile(sdlPath)
	if err != nil {
		t.Fatalf("reading %s: %v", sdlPath, err)
	}

	src, err := Parse("grammar_test.sdl", text)
	if err != nil {
		t.Fatal(err)
	}
	cm := ExtractComments(text)
	cm.Attach(src)

	if err := rewriteUUIDLiterals(src); err != nil {
		t.Fatalf("rewriteUUIDLiterals: %v", err)
	}

	opts := &GenOpts{SourceName: "grammar_test.sdl"}

	emitters := []Emitter{GoEmitter{}, CSharpEmitter{}, TSEmitter{}, PyEmitter{}, CEmitter{}}
	for _, emitter := range emitters {
		info := emitter.Info()
		out, err := emitter.Generate(src, opts)
		if err != nil {
			t.Fatalf("Generate %s: %v", info.Language, err)
		}
		path := filepath.Join(goldenDir, "grammar_test"+constsInfix+info.Extension)
		if err := os.WriteFile(path, out, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
