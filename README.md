# forge

Codegen toolchain for the [art.media.platform](https://github.com/art-media-platform) ecosystem.

## Subcommands

- `forge consts <file.consts.sdl> [--go_out=DIR] [--csharp_out=DIR]` — generate Go and/or C# constants from a `.consts.sdl` source.
- `forge sdl <in.sdl> <out.sdl>` — parse and re-export an SDL file (round-trip / normalization).

## Install

```
go install github.com/art-media-platform/forge/cmd/forge@latest
```

## Run without installing

```
go run github.com/art-media-platform/forge/cmd/forge consts path/to/file.consts.sdl --go_out=./pkg
```

## Packages

- `consts/` — `.consts.sdl` grammar, AST, and Go/C# emitters.
- `sdl/` — SDL grammar, AST, export.
