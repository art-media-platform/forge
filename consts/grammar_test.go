package consts

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGolden regenerates Go, C#, and TypeScript output from the golden
// .consts.sdl input and writes the results alongside it. A developer reviews
// any diff against what's committed to verify correctness.
//
// To accept changes after an intentional edit: commit the updated golden files.
func TestGolden(t *testing.T) {
	goldenDir := "golden"
	sdlPath := filepath.Join(goldenDir, "grammar_test.sdl")

	src, err := os.ReadFile(sdlPath)
	if err != nil {
		t.Fatalf("reading %s: %v", sdlPath, err)
	}

	cf, err := Parse("grammar_test.sdl", src)
	if err != nil {
		t.Fatal(err)
	}
	cm := ExtractComments(src)
	cm.Attach(cf)

	if err := rewriteUUIDLiterals(cf); err != nil {
		t.Fatalf("rewriteUUIDLiterals: %v", err)
	}

	opts := &GenOpts{SourceName: "grammar_test.sdl"}

	goOut, err := GenerateGo(cf, opts)
	if err != nil {
		t.Fatalf("GenerateGo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goldenDir, "grammar_test.consts.go"), goOut, 0644); err != nil {
		t.Fatal(err)
	}

	csOut, err := GenerateCSharp(cf, opts)
	if err != nil {
		t.Fatalf("GenerateCSharp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goldenDir, "grammar_test.consts.cs"), csOut, 0644); err != nil {
		t.Fatal(err)
	}

	tsOut, err := GenerateTypeScript(cf, opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goldenDir, "grammar_test.consts.ts"), tsOut, 0644); err != nil {
		t.Fatal(err)
	}
}
