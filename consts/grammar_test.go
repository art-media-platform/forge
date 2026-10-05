package consts

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current render")

// TestGolden renders Go, C#, TypeScript, Python, and C output from the golden
// .consts.sdl input and compares each byte-for-byte against the committed
// golden file; any drift fails.
//
// After an intentional render change, rewrite the goldens and commit them:
//
//	go test ./consts/ -run TestGolden -update
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
		got, err := emitter.Generate(src, opts)
		if err != nil {
			t.Fatalf("Generate %s: %v", info.Language, err)
		}
		path := filepath.Join(goldenDir, "grammar_test"+constsInfix+info.Extension)
		if *update {
			if err := os.WriteFile(path, got, 0644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s: %v (run with -update to create it)", path, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: render differs from the committed golden (run with -update to accept)\n%s", path, firstDiff(want, got))
		}
	}
}

// firstDiff describes the first line where want and got differ.
func firstDiff(want, got []byte) string {
	wantLines := bytes.Split(want, []byte("\n"))
	gotLines := bytes.Split(got, []byte("\n"))
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		wantLine := lineAt(wantLines, i)
		gotLine := lineAt(gotLines, i)
		if wantLine != gotLine {
			return fmt.Sprintf("  line %d:\n  - %s\n  + %s\n  (golden %d lines, render %d lines)",
				i+1, wantLine, gotLine, len(wantLines), len(gotLines))
		}
	}
	return "  (no line differs; byte-level difference only)"
}

// lineAt returns lines[i] quoted, or "<EOF>" past the end.
func lineAt(lines [][]byte, i int) string {
	if i >= len(lines) {
		return "<EOF>"
	}
	return fmt.Sprintf("%q", lines[i])
}
