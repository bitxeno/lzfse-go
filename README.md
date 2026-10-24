# lzfse-go

Go bindings for [lzfse](https://github.com/lzfse/lzfse) (Apple's LZFSE compression) that
require **no cgo**. The native code ships as a small shared library loaded and called
through [purego](https://github.com/ebitengine/purego), so binaries build and run with
`CGO_ENABLED=0` — including fully static binaries on Linux.

The API mirrors [blacktop/lzfse-cgo](https://github.com/blacktop/lzfse-cgo):

```go
package main

import (
	"log"
	"os"

	"github.com/bitxeno/lzfse-go"
)

func main() {
	src, _ := os.ReadFile("file.raw")

	enc, err := lzfse.EncodeBuffer(src) // compress
	if err != nil {
		log.Fatal(err)
	}
	dec, err := lzfse.DecodeBuffer(enc) // decompress
	if err != nil {
		log.Fatal(err)
	}
	_ = dec
}
```

```bash
CGO_ENABLED=0 go build ./...   # works out of the box
```

## How it works

The C sources of lzfse (plus [libzbitmap](https://github.com/eafer/libzbitmap), as
vendored by lzfse-cgo) are compiled into a tiny shared library per platform (396 KB in
total, prebuilt ones are committed under `libs/`). The libraries are embedded in the Go package with `go:embed`; on first use
the copy for the current platform is extracted to the user cache directory and loaded
with `purego.Dlopen`, and all calls go through `purego.RegisterFunc`. No cgo is
involved anywhere: the Go compiler never sees C code, so the package builds with
`CGO_ENABLED=0`.

Rebuild the embedded libraries after changing anything under `native/`:

```bash
make build-libs        # host platform only, uses the system C compiler
make build-libs-all    # all six targets below, cross-compiled with zig
```
Supported without cgo: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`,
`windows/amd64`, `windows/arm64` (the same targets
[mygo's sqlite plugin](https://github.com/egoist/mygo/tree/main/plugins/sqlite) ships).
On other platforms the package compiles but returns `ErrUnsupported`.

## Library lookup

Package-level functions load the native library on first use:

1. `$LZFSE_LIBRARY_PATH`, when set, or the path given to `SetLibraryPath`;
2. otherwise the embedded copy for the current platform, extracted to the user cache
   directory (`~/Library/Caches/lzfse-go/<version>`, `$XDG_CACHE_HOME/lzfse-go/<version>`,
   or `%LocalAppData%\lzfse-go\<version>`).

`SetLibraryPath`/`LZFSE_LIBRARY_PATH` accept either a shared library file or a path as
produced by `go run ./internal/libbuild -out <dir> -all`.

For full control, load a library explicitly and use its methods — useful to pin the
location in embedded or multi-tenant processes:

```go
lib, err := lzfse.NewLibrary("/path/to/liblzfse-go.dylib")
if err != nil {
	log.Fatal(err)
}
dec, err := lib.DecodeBuffer(enc)
```

## API

The package offers the same surface as lzfse-cgo, plus the LZBITMAP container used by
lzfse-cgo's `libzbitmap`:

| Function | Purpose |
| --- | --- |
| `EncodeBuffer(src) ([]byte, error)` | Compress with LZFSE into a new buffer. |
| `EncodeBufferInto(src, dst) (int, error)` | Compress into `dst`; `ErrShortBuffer` when it cannot fit. |
| `DecodeBuffer(src) ([]byte, error)` | Decompress into a right-sized new buffer. |
| `DecodeBufferInto(src, dst) (int, error)` | Decompress into `dst`; `ErrShortBuffer` when it cannot fit. |
| `EncodeLZVNBuffer(src, dst) (int, error)` | Compress with the LZVN block format. |
| `DecodeLZVNBuffer(src, dst) (int, error)` | Decompress LZVN data. |
| `EncodeZBMBuffer(src, dst) (int, error)` | Compress with the LZBITMAP container format. |
| `DecodeZBMBuffer(src, dst) (int, error)` | Decompress LZBITMAP data. |
| `EncodeScratchSize() / DecodeScratchSize()` | Scratch buffer sizes, for allocation control. |

`EncodeBuffer`/`DecodeBuffer` allocate; the `*Into` variants write into caller-provided
buffers and return `ErrShortBuffer` when it does not fit. For reusable buffers see
`DecodeBufferInto` with a `sync.Pool`.

All functions are safe for concurrent use by multiple goroutines.

## Notes on fidelity

- The vendored C code is unmodified upstream lzfse; golden vectors from lzfse-cgo's test
  suite (`testdata/enc.bin`, `testdata/dec.bin`) are reproduced byte for byte, and the
  upstream CLI produces identical output for the same input.
- Unlike cgo builds, the shared library is loaded at run time: keep the embedded copy
  working (the default) or ship it alongside the binary and point `LZFSE_LIBRARY_PATH`
  at it.

## Development

```bash
make test          # CGO_ENABLED=0 go test ./...
make vet           # CGO_ENABLED=0 go vet ./...
make build-libs    # rebuild the embedded library for this platform
```

`internal/libbuild` compiles `native/` into `libs/<goos>-<goarch>/`, with zig as the
cross compiler for `-all` (the same approach mygo uses for its SQLite plugin) and the
host C compiler for `-host`.

## Credit

- <https://github.com/lzfse/lzfse> — Apple's LZFSE (Apache-2.0-licensed C sources in `native/lzfse`)
- <https://github.com/ebitengine/purego> — cgo-free FFI
- <https://github.com/blacktop/lzfse-cgo> — API reference and golden test vectors
- <https://github.com/eafer/libzbitmap> — libzbitmap LZBITMAP container
- <https://github.com/egoist/mygo> — the cgo-free shared-library pattern this repo follows

## License

MIT — see [LICENSE](LICENSE). The vendored C sources keep their upstream licenses
(`native/lzfse/LICENSE`).
