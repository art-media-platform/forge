package consts

// EmitterInfo is an emitter's static metadata, returned by value from
// Emitter.Info.
type EmitterInfo struct {
	Language  string // language name, e.g. "Go"
	Extension string // language file extension, e.g. ".go" (the ".consts" infix is added at the path site)
	Flag      string // CLI output-directory flag, e.g. "go_out"
}

// Emitter renders a parsed .consts.sdl file into one target language.  Generate
// produces the source bytes; Info reports the language name, output-file suffix,
// and the CLI flag that selects this target.
//
// The heavy lifting for each language stays in its own gen_<lang>.go file; the
// Generate methods there delegate to the package-level Generate<Lang> functions,
// so callers can pick between the interface (enumerate/dispatch over targets)
// and the bare function (one-off, get bytes for a single language).  The set of
// emitters is assembled explicitly by the caller (see cmd/forge) — no registry.
type Emitter interface {
	Generate(src *ConstFile, opts *GenOpts) ([]byte, error)
	Info() EmitterInfo
}
