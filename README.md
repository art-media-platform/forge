# forge

A codegen tool for **cross-language string and numerical constants** — the kind that need to stay byte-identical between Go, C#, TypeScript, Python, C, and other languages, with deterministic IDs that never drift across rebuilds.  PRs for other languages happily accepted.

## The Problem

You have a server in your fleet, a vendor API key, a feature flag — things that need a stable identifier everywhere they show up. You want:

- The same **stable UUID** for each name, identical across Go service, C# client, log streams, on-disk records.
- A **typed constant** in every language so the compiler catches typos.
- A **human-readable label** alongside the UUID for debugging and CLI output.

Today's options each get this wrong:

- **`protoc` enums** — integer values only. No string consts, no hashed identifiers, no UUIDs.
- **Hand-written const files** invite maintenance — add a string in Go, forget to mirror it in C#, fix it months later.
- **Hash at runtime** — one language's hash function disagreeing with another's is a silent, late-discovered pain.

`protoc`, `flatbuffers`, Cap'n Proto, JSON-Schema codegen, and friends solve **wire format and types**. They don't solve the hard cases — hashed identifiers from names, UUID literals as first-class values, deterministic IDs that stay byte-identical across languages without you copy-pasting hex by hand. That's the gap forge fills.

## The Solution

One `.sdl` source file, multiple language outputs. IDs computed once at codegen time and embedded as literals.

```
// your-project.consts.sdl

// Go and C# need a package / namespace; TypeScript, Python, and C don't.
option go_package = "github.com/you/project/widget";
option csharp_namespace = "your.project";

// Stable, named identifiers — names get hashed to UIDs at codegen time.
tags ID {
    SiteDownloads "http://acme.com/downloads/"
    TestNet       "your-scheme://server.com:23382/path"
    BestShow      "Fraggle Rock"
}

// A pre-existing UUID you need to preserve verbatim — vendor key, database
// row, anything you already have written down somewhere.
const uid VendorAPIKey = "550e8400-e29b-41d4-a716-446655440000";

// Plain scalar constants.
const string APIVersion      = "v3.1.0";
const uint64 MaxItemsPerPage = 1000;
```

Run:

```
go run github.com/art-media-platform/forge/cmd/forge@latest consts your-project.consts.sdl \
    --go_out=./go --csharp_out=./csharp --ts_out=./ts --py_out=./py --c_out=./c
```

Generated **Go**:

```go
package widget

var ID = struct {
    SiteDownloads tag.Name
    TestNet       tag.Name
    BestShow      tag.Name
}{
    SiteDownloads: tag.Name{ID: tag.UID{0xB30..., 0x379...}, Canonic: "http://acme.com/downloads/"},
    TestNet:       tag.Name{ID: tag.UID{0x464..., 0xDC3...}, Canonic: "your-scheme://server.com:23382/path"},
    BestShow:      tag.Name{ID: tag.UID{0x57B..., 0x78F...}, Canonic: "fraggle.rock"},
}

var (
    VendorAPIKey = tag.UID{0x550..., 0xA71...}
)

const (
    APIVersion      = "v3.1.0"
    MaxItemsPerPage = uint64(1000)
)
```

Generated **C#** — file-scoped namespace, compile-time scalars as `const` (usable in attributes, `switch`, and other const contexts), UIDs as `static readonly` since a struct ctor isn't a constant expression:

```cs
namespace your.project;

public static partial class ID {
    public static readonly TagName SiteDownloads = new(new(0xB30..., 0x379...), "http://acme.com/downloads/");
    public static readonly TagName TestNet       = new(new(0x464..., 0xDC3...), "your-scheme://server.com:23382/path");
    public static readonly TagName BestShow      = new(new(0x57B..., 0x78F...), "fraggle.rock");
}

public static partial class Const {
    public const           string APIVersion      = "v3.1.0";
    public const           ulong  MaxItemsPerPage = 1000UL;
    public static readonly UID    VendorAPIKey    = new(0x550..., 0xA71...);
}
```

Generated **TypeScript** — zero runtime, self-contained types, `bigint` literals so 64-bit values never lose precision:

```ts
export type UID = readonly [bigint, bigint];

export interface TagName {
    readonly id:      UID;
    readonly canonic: string;
}

export const ID = {
    SiteDownloads: { id: [0xB30...n, 0x379...n], canonic: "http://acme.com/downloads/" },
    TestNet      : { id: [0x464...n, 0xDC3...n], canonic: "your-scheme://server.com:23382/path" },
    BestShow     : { id: [0x57B...n, 0x78F...n], canonic: "fraggle.rock" },
} satisfies Record<string, TagName>;

export const VendorAPIKey: UID = [0x550...n, 0xA71...n];

export const APIVersion:      string = "v3.1.0";
export const MaxItemsPerPage: bigint = 1000n;
```

Generated **Python** — zero dependencies, stdlib `NamedTuple` records that are typed, immutable, and hashable (usable as dict keys):

```python
from typing import NamedTuple

UID = tuple[int, int]


class TagName(NamedTuple):
    id:      UID
    canonic: str


class ID:
    SiteDownloads = TagName((0xB30..., 0x379...), "http://acme.com/downloads/")
    TestNet       = TagName((0x464..., 0xDC3...), "your-scheme://server.com:23382/path")
    BestShow      = TagName((0x57B..., 0x78F...), "fraggle.rock")


VendorAPIKey: UID = (0x550..., 0xA71...)


APIVersion:      str = "v3.1.0"
MaxItemsPerPage: int = 1000
```

Generated **C** — a single C99 header, zero dependencies, self-contained types, dotted `ID.SiteDownloads` access via designated initializers. The `FORGE_CONSTS_TYPES` guard lets several forge headers share one translation unit, and `FORGE_UNUSED` keeps it `-Wall -Wextra` clean even when a caller references only some of the constants:

```c
...

static const struct {
    TagName SiteDownloads;
    TagName TestNet;
    TagName BestShow;
} ID FORGE_UNUSED = {
    .SiteDownloads = { { 0xB30..., 0x379... }, "http://acme.com/downloads/" },
    .TestNet       = { { 0x464..., 0xDC3... }, "your-scheme://server.com:23382/path" },
    .BestShow      = { { 0x57B..., 0x78F... }, "fraggle.rock" },
};

static const struct {
    const char *APIVersion;
    uint64_t    MaxItemsPerPage;
    TagUID      VendorAPIKey;
} Const FORGE_UNUSED = {
    .APIVersion      = "v3.1.0",
    .MaxItemsPerPage = 1000ULL,
    .VendorAPIKey    = { 0x550..., 0xA71... },
};
```

## `tag.UID`

Every name compiles to a 128-bit identifier from the [`stdlib/tag`](https://github.com/art-media-platform/amp.SDK/blob/main/stdlib/tag/README.md) package of [amp.SDK](https://github.com/art-media-platform/amp.SDK) — a phonetic, search-friendly addressing system worth reading about in its own right. Forge emits these depending on the language:

| Language | `UID` | `TagName` |
|---|---|---|
| Go | `tag.UID` (two `uint64`) | `tag.Name` (struct) |
| C# | `UID` (two `ulong`) | `TagName` (readonly struct) |
| TypeScript | `readonly [bigint, bigint]` | `TagName` (interface) |
| Python | `tuple[int, int]` | `TagName` (NamedTuple) |
| C | `TagUID` (two `uint64_t`) | `TagName` (struct) |


Either way, three things matter to forge users:

- **Resilient short label** — `id.Base32()` gives a compact, human-safe string with no look-alike characters. Forge embeds it as a trailing comment on every generated line, so the readable label always sits beside the binary value.
- **No runtime parsing** — the value is already a literal in your generated code; forge parsed it at codegen time. Your binary never parses a UUID string at startup unless *you* hand it one — no startup hashing, no parse-error branch, no cross-language library-version skew.

For the canonicalization and parsing rules (URL-mode parsing, acronym preservation) and the Base32 alphabet rationale, see the [`tag` package README](https://github.com/art-media-platform/amp.SDK/blob/main/stdlib/tag/README.md).

## See It Yourself

The repo's golden test exercises every grammar feature — tag hierarchies, UUID variants, mixed-type groups, doc-comment preservation:

| File | Role |
|---|---|
| [`grammar_test.sdl`](consts/golden/grammar_test.sdl) | input |
| [`grammar_test.consts.go`](consts/golden/grammar_test.consts.go) | generated Go |
| [`grammar_test.consts.cs`](consts/golden/grammar_test.consts.cs) | generated C# |
| [`grammar_test.consts.ts`](consts/golden/grammar_test.consts.ts) | generated TypeScript |
| [`grammar_test.consts.py`](consts/golden/grammar_test.consts.py) | generated Python |
| [`grammar_test.consts.h`](consts/golden/grammar_test.consts.h) | generated C |

[`grammar_test.go`](consts/grammar_test.go) regenerates these on `go test ./...`.

## What You Get

- **Hierarchical namespaces** — names nest in the source, producing dot-paths like `server.region.east` that compile to typed constants with stable UIDs.
- **UUID literals** — paste an existing UUID (`"550e8400-..."`) as a `uid` value and forge bakes it into typed constants in every target language.
- **Mixed-type const groups** — strings, integers (`int32`/`int64`/`uint32`/`uint64`/`fixed64`), floats (`float32`/`float64`), hex literals, and explicit UID pairs in one file.
- **Comment preservation** — leading and trailing source comments carry through to every language with idiomatic formatting.
- **Column-aligned output** — generated files are diff-friendly and pleasant to read.

## Install

```
go install github.com/art-media-platform/forge/cmd/forge@latest
```

Or run pinned without installing:

```
go run github.com/art-media-platform/forge/cmd/forge@v0.3.0 consts ...
```

## CLI

- `forge consts <file.consts.sdl> [--go_out=DIR] [--csharp_out=DIR] [--ts_out=DIR] [--py_out=DIR] [--c_out=DIR]` — emit Go, C#, TypeScript, Python, and/or C from a `.consts.sdl` source.
- `forge sdl <in.sdl> <out.sdl>` — parse and re-export an SDL file (round-trip / normalization).


## Powered by Participle

Forge's `.sdl` grammar is parsed using [**participle**](https://github.com/alecthomas/participle) by [@alecthomas](https://github.com/alecthomas). Participle is one of those rare libraries that makes the hard thing — building a real parser, with a real lexer, that gives you a real AST with source positions and recoverable errors — feel like *describing* the grammar instead of *writing* a parser. The entire forge grammar is ~100 lines of annotated Go struct tags:

```go
type ConstDecl struct {
    Pos     lexer.Position
    Type    string `"const"? @("string" | "int32" | "int64" | "uint32" | ...)`
    VarName string `@Ident`
    Value   *Value `"=" @@ ";"`
}
```

That's the whole declaration. The struct *is* the parser. Source-position tracking, lookahead, error recovery — all included, no boilerplate. If you've ever hand-rolled a recursive-descent parser, or wrestled with yacc/ANTLR for a small DSL, give participle a serious look. It's the reason forge can exist.
