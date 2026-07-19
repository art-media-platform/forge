package consts

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSourcePinnedOutDir verifies Generator.Run's source-pinned output
// routing: an `option ts_out = "<dir>";` in the .consts.sdl emits TypeScript
// into that directory RESOLVED RELATIVE TO THE SOURCE FILE, an explicit
// OutDirs entry (CLI flag) overrides the option, and a file with neither
// emits nothing for that target.
func TestSourcePinnedOutDir(t *testing.T) {
	const sdlBody = `
option ts_out = "gen/ts";

const string Greeting = "hello";
`
	writeSDL := func(t *testing.T, body string) (dir, sdlPath string) {
		t.Helper()
		dir = t.TempDir()
		sdlPath = filepath.Join(dir, "pin_test.consts.sdl")
		if err := os.WriteFile(sdlPath, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		return dir, sdlPath
	}

	t.Run("option routes relative to the source file", func(t *testing.T) {
		dir, sdlPath := writeSDL(t, sdlBody)
		if err := os.MkdirAll(filepath.Join(dir, "gen", "ts"), 0755); err != nil {
			t.Fatal(err)
		}
		err := (Generator{
			InputPath: sdlPath,
			Emitters:  []Emitter{TSEmitter{}},
			OutDirs:   map[string]string{"ts_out": ""},
		}).Run()
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		want := filepath.Join(dir, "gen", "ts", "pin_test.consts.ts")
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("option-routed output missing: %v", err)
		}
	})

	t.Run("explicit flag overrides the option", func(t *testing.T) {
		dir, sdlPath := writeSDL(t, sdlBody)
		flagDir := filepath.Join(dir, "flagged")
		if err := os.MkdirAll(flagDir, 0755); err != nil {
			t.Fatal(err)
		}
		err := (Generator{
			InputPath: sdlPath,
			Emitters:  []Emitter{TSEmitter{}},
			OutDirs:   map[string]string{"ts_out": flagDir},
		}).Run()
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if _, err := os.Stat(filepath.Join(flagDir, "pin_test.consts.ts")); err != nil {
			t.Fatalf("flag-routed output missing: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "gen", "ts", "pin_test.consts.ts")); !os.IsNotExist(err) {
			t.Fatalf("option path emitted despite explicit flag (err=%v)", err)
		}
	})

	t.Run("no option, no flag emits nothing", func(t *testing.T) {
		dir, sdlPath := writeSDL(t, "const string Greeting = \"hello\";\n")
		err := (Generator{
			InputPath: sdlPath,
			Emitters:  []Emitter{TSEmitter{}},
			OutDirs:   map[string]string{"ts_out": ""},
		}).Run()
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".ts" {
				t.Fatalf("unexpected output %s", entry.Name())
			}
		}
	})
}
