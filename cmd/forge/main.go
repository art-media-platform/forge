package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/art-media-platform/forge/consts"
	"github.com/art-media-platform/forge/sdl"
)

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

var CLI struct {
	SDL    FilterSDL `cmd:"" help:"Process SDL files."`
	Consts ConstsGen `cmd:"" help:"Generate Go/C#/TypeScript/Python/C from .consts.sdl files."`
}

type ConstsGen struct {
	FileIn    string `arg:"" name:"file" type:"existingfile" help:".consts.sdl source file"`
	GoOut     string `name:"go_out"      help:"Go output directory"         optional:""`
	CSharpOut string `name:"csharp_out"  help:"C# output directory"         optional:""`
	TSOut     string `name:"ts_out"      help:"TypeScript output directory" optional:""`
	PyOut     string `name:"py_out"      help:"Python output directory"     optional:""`
	COut      string `name:"c_out"       help:"C output directory"          optional:""`
}

// emitters is the explicit set of code generators forge ships.  Adding a
// language is a one-line edit here plus its gen_<lang>.go file — no registry.
var emitters = []consts.Emitter{
	consts.GoEmitter{},
	consts.CSharpEmitter{},
	consts.TSEmitter{},
	consts.PyEmitter{},
	consts.CEmitter{},
}

func (cmd *ConstsGen) Run() error {
	// Each emitter selects its directory by its EmitterInfo.Flag; keys here match
	// the kong flag names above.
	return consts.Generator{
		InputPath: cmd.FileIn,
		Emitters:  emitters,
		OutDirs: map[string]string{
			"go_out":     cmd.GoOut,
			"csharp_out": cmd.CSharpOut,
			"ts_out":     cmd.TSOut,
			"py_out":     cmd.PyOut,
			"c_out":      cmd.COut,
		},
	}.Run()
}

type FilterSDL struct {
	FileIn  string `arg:"" name:"file_in"  type:"existingfile" help:"input SDL file"`
	FileOut string `arg:"" name:"file_out"                     help:"output SDL file"`
}

func (cmd *FilterSDL) Run() error {
	src, err := os.ReadFile(cmd.FileIn)
	if err != nil {
		return fmt.Errorf("reading SDL %q: %w", cmd.FileIn, err)
	}

	ast, err := sdl.ParseBytes(cmd.FileIn, src)
	if err != nil {
		return err
	}

	text, err := ast.Export(nil, sdl.ExportOpts{})
	if err != nil {
		return err
	}

	return os.WriteFile(cmd.FileOut, text, 0644)
}
