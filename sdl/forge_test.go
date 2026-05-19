package sdl_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/art-media-platform/forge/sdl"
)

func TestSDL(t *testing.T) {
	filePath := "golden/generic_test"
	filePath_in := filePath + ".in.sdl"
	src, err := os.ReadFile(filePath_in)
	if err != nil {
		t.Fatalf("Failed to read SDL file %q: %v", filePath_in, err)
	}

	ast, err := sdl.ParseBytes(filePath_in, src)
	if err != nil {
		t.Fatalf("Failed to parse SDL  %q: %v", filePath_in, err)
	}

	ast.EnumAttributes(func(attrID []byte, rhs *sdl.Value) bool {
		if rhs == nil {
			// TODO export as Tag UID?
		} else {
			rhs.Export(nil, sdl.ExportOpts{}) // export as text
		}

		fmt.Printf("%s = \n", attrID)
		return true
	})

	text, err := ast.Export(nil, sdl.ExportOpts{})
	if err != nil {
		t.Fatal(err)
	}

	filePath_out := filePath + ".out.sdl"
	os.WriteFile(filePath_out, text, 0777)

}
