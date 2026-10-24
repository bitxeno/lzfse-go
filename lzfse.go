// Package lzfse provides Go bindings for the lzfse compression library
// (https://github.com/lzfse/lzfse) that require no cgo.
//
// The native code ships as a small shared library that this package loads
// and calls through github.com/ebitengine/purego, so binaries build and run
// with CGO_ENABLED=0. By default the package uses an embedded copy of the
// shared library for the current platform, extracted to the user cache
// directory on first use. SetLibraryPath or the LZFSE_LIBRARY_PATH
// environment variable can point at a different shared library instead.
//
// The API mirrors github.com/blacktop/lzfse-cgo:
//
//	out, err := lzfse.EncodeBuffer(src)
//	src2, err := lzfse.DecodeBuffer(out)
//
// All functions are safe for concurrent use.
package lzfse

import (
	"errors"
	"fmt"
	"sync"
)

// Version of the vendored lzfse C sources.
const Version = "1.0.0"

// ErrUnsupported is returned on platforms where the package cannot call the
// native library.
var ErrUnsupported = errors.New("lzfse: unsupported platform")

// Library is a loaded native lzfse library. The default Library is loaded
// lazily on first use; all package-level functions use it.
type Library struct {
	encodeScratchSize func() uintptr
	encodeBuffer      func(dst, dstSize, src, srcSize, scratch uintptr) uintptr
	decodeScratchSize func() uintptr
	decodeBuffer      func(dst, dstSize, src, srcSize, scratch uintptr) uintptr

	lzvnEncodeScratchSize func() uintptr
	lzvnEncodeBuffer      func(dst, dstSize, src, srcSize, scratch uintptr) uintptr
	lzvnDecodeBuffer      func(dst, dstSize, src, srcSize uintptr) uintptr

	zbmCompress   func(dst, dstSize, src, srcSize uintptr) uintptr
	zbmDecompress func(dst, dstSize, src, srcSize uintptr) uintptr
}

var (
	mu         sync.Mutex
	defaultLib *Library
	defaultErr error

	// Set by the platform loader in an init function; libOpener stays nil
	// where the package is unsupported.
	libOpener func(path string) (uintptr, error)
	symFinder func(handle uintptr, name string) (uintptr, error)
	libCloser func(handle uintptr)
)

// NewLibrary loads the native lzfse library from path. The returned library
// stays loaded for the lifetime of the process.
func NewLibrary(path string) (*Library, error) {
	if libOpener == nil {
		return nil, ErrUnsupported
	}
	handle, err := libOpener(path)
	if err != nil {
		return nil, fmt.Errorf("lzfse: loading %s: %w", path, err)
	}
	lib := &Library{}
	if err := loadSymbols(lib, func(name string) (uintptr, error) {
		return symFinder(handle, name)
	}); err != nil {
		libCloser(handle)
		return nil, fmt.Errorf("lzfse: %s is not the lzfse-go library: %w", path, err)
	}
	return lib, nil
}

// DefaultLibrary returns the process-wide library used by the package-level
// functions, loading it on first use. The location is decided by
// SetLibraryPath, then the search described in the package docs.
func DefaultLibrary() (*Library, error) {
	mu.Lock()
	defer mu.Unlock()
	if defaultLib != nil {
		return defaultLib, nil
	}
	if defaultErr != nil {
		return nil, defaultErr
	}
	defaultLib, defaultErr = loadDefaultLibrary()
	return defaultLib, defaultErr
}

// SetLibraryPath sets the file path of the native lzfse library used by the
// package-level functions, bypassing the default search. It must be called
// before the first compression or decompression call.
func SetLibraryPath(path string) error {
	mu.Lock()
	defer mu.Unlock()
	if defaultLib != nil {
		return errors.New("lzfse: native library already loaded")
	}
	lib, err := NewLibrary(path)
	if err != nil {
		return err
	}
	defaultLib, defaultErr = lib, nil
	return nil
}

// loadSymbols resolves every symbol the binding needs. All the wrapped C
// functions share one shape: buffer pointers and sizes in, a size_t out.
func loadSymbols(lib *Library, sym func(string) (uintptr, error)) error {
	for _, e := range []struct {
		name string
		fn   any
	}{
		{"lzfse_encode_scratch_size", &lib.encodeScratchSize},
		{"lzfse_encode_buffer", &lib.encodeBuffer},
		{"lzfse_decode_scratch_size", &lib.decodeScratchSize},
		{"lzfse_decode_buffer", &lib.decodeBuffer},
		{"lzvn_encode_scratch_size", &lib.lzvnEncodeScratchSize},
		{"lzvn_encode_buffer", &lib.lzvnEncodeBuffer},
		{"lzvn_decode_buffer", &lib.lzvnDecodeBuffer},
		{"lzfse_zbm_compress", &lib.zbmCompress},
		{"lzfse_zbm_decompress", &lib.zbmDecompress},
	} {
		addr, err := sym(e.name)
		if err != nil {
			return err
		}
		registerLibFunc(e.fn, addr)
	}
	return nil
}

// EncodeScratchSize returns the required scratch buffer size to compress
// using LZFSE. It is lzfse_encode_scratch_size.
func (lib *Library) EncodeScratchSize() int {
	return int(lib.encodeScratchSize())
}

// DecodeScratchSize returns the required scratch buffer size to decompress
// using LZFSE. It is lzfse_decode_scratch_size.
func (lib *Library) DecodeScratchSize() int {
	return int(lib.decodeScratchSize())
}

// EncodeBuffer compresses src using LZFSE into a newly allocated buffer,
// returning the compressed bytes.
func (lib *Library) EncodeBuffer(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	dst := make([]byte, encodeDestSize(len(src)))
	n, err := lib.EncodeBufferInto(src, dst)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

// EncodeBufferInto compresses src into dst and returns the number of bytes
// written. It returns ErrShortBuffer if dst is too small. The encoder
// adapts its block layout to the space available, so a slightly undersized
// dst may still yield a valid (different) encoding; only when the output
// cannot fit at all does it fail.
func (lib *Library) EncodeBufferInto(src, dst []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	scratch, freeScratch, err := lib.getScratch(lib.EncodeScratchSize())
	if err != nil {
		return 0, err
	}
	defer freeScratch()

	s := sliceData(src)
	d := sliceData(dst)
	n := lib.encodeBuffer(d, uintptr(len(dst)), s, uintptr(len(src)), scratch)
	keepAlive(src)
	keepAlive(dst)
	keepAlive(scratch)
	if n == 0 {
		return 0, ErrShortBuffer
	}
	return int(n), nil
}

// DecodeBuffer decompresses LZFSE-encoded src into a newly allocated buffer,
// returning the decompressed bytes.
func (lib *Library) DecodeBuffer(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	scratch, freeScratch, err := lib.getScratch(lib.DecodeScratchSize())
	if err != nil {
		return nil, err
	}
	defer freeScratch()

	s := sliceData(src)
	// Peek at the first block header: the LZFSE "bvx-" raw block and the
	// LZVN "bvxn" block header both store the block's decoded size at offset
	// 4, so a stream that consists of a single such block is fully sized up
	// front. Allocate one byte extra to tell a complete decode from a stream
	// that continues with more blocks, and fall back to growth for those.
	if size := peekDecodedSize(src); size > 0 {
		dst := make([]byte, size+1)
		n := lib.decodeBuffer(sliceData(dst), uintptr(size+1), s, uintptr(len(src)), scratch)
		keepAlive(dst)
		if n == uintptr(size) {
			return dst[:size], nil
		}
	}

	dst := make([]byte, decodeDestSize(len(src)))
	for range 16 {
		n := lib.decodeBuffer(sliceData(dst), uintptr(len(dst)), s, uintptr(len(src)), scratch)
		keepAlive(dst)
		// lzfse_decode_buffer returns dst_size when the output did not fit,
		// and 0 on malformed input; the only way to tell them apart is to
		// retry with a larger buffer, as the reference CLI does.
		if n != 0 && n != uintptr(len(dst)) {
			return dst[:n], nil
		}
		next := len(dst) * 2
		if n == 0 || next > maxDecodeSize(len(src)) {
			return nil, ErrDecompress
		}
		dst = make([]byte, next)
	}
	return nil, ErrDecompress
}

// DecodeBufferInto decompresses LZFSE-encoded src into dst, returning the
// number of bytes written. It returns ErrShortBuffer if dst is too small.
func (lib *Library) DecodeBufferInto(src, dst []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	if len(dst) == 0 {
		return 0, ErrShortBuffer
	}
	scratch, freeScratch, err := lib.getScratch(lib.DecodeScratchSize())
	if err != nil {
		return 0, err
	}
	defer freeScratch()

	return lib.decodeIntoChecked(src, dst, scratch)
}

// decodeIntoChecked implements DecodeBufferInto: decode into dst, then
// disambiguate a full dst by decoding one byte further — the extra byte
// cannot push a decoder over the next block boundary, because every block
// starts with its own header.
func (lib *Library) decodeIntoChecked(src, dst []byte, scratch uintptr) (int, error) {
	s := sliceData(src)
	n := lib.decodeBuffer(sliceData(dst), uintptr(len(dst)), s, uintptr(len(src)), scratch)
	keepAlive(dst)
	if n == 0 {
		return 0, ErrDecompress
	}
	if n < uintptr(len(dst)) {
		return int(n), nil
	}
	// The output filled dst exactly. Either it is complete, or there is
	// more: decode one byte further and see whether anything more arrives.
	probe := make([]byte, len(dst)+1)
	pn := lib.decodeBuffer(sliceData(probe), uintptr(len(probe)), s, uintptr(len(src)), scratch)
	keepAlive(probe)
	if pn > uintptr(len(dst)) {
		return 0, ErrShortBuffer
	}
	if pn == 0 {
		return 0, ErrDecompress
	}
	return int(n), nil
}

// EncodeLZVNBuffer compresses src using the LZVN block format into dst,
// returning the number of bytes written, or 0 if dst is too small.
func (lib *Library) EncodeLZVNBuffer(src, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	scratch, freeScratch, err := lib.getScratch(lib.LZVNEncodeScratchSize())
	if err != nil {
		return 0
	}
	defer freeScratch()

	n := lib.lzvnEncodeBuffer(sliceData(dst), uintptr(len(dst)), sliceData(src), uintptr(len(src)), scratch)
	keepAlive(src)
	keepAlive(dst)
	keepAlive(scratch)
	return int(n)
}

// DecodeLZVNBuffer decompresses LZVN-encoded src into dst, returning the
// number of bytes written, or 0 on error or if dst is too small.
func (lib *Library) DecodeLZVNBuffer(src, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	n := lib.lzvnDecodeBuffer(sliceData(dst), uintptr(len(dst)), sliceData(src), uintptr(len(src)))
	keepAlive(src)
	keepAlive(dst)
	return int(n)
}

// EncodeZBMBuffer compresses src using the LZBITMAP container format into
// dst, returning the number of bytes written, or 0 on error or if dst is too
// small.
func (lib *Library) EncodeZBMBuffer(src, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	n := lib.zbmCompress(sliceData(dst), uintptr(len(dst)), sliceData(src), uintptr(len(src)))
	keepAlive(src)
	keepAlive(dst)
	return int(n)
}

// DecodeZBMBuffer decompresses LZBITMAP-encoded src into dst, returning the
// number of bytes written, or 0 on error or if dst is too small.
func (lib *Library) DecodeZBMBuffer(src, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	n := lib.zbmDecompress(sliceData(dst), uintptr(len(dst)), sliceData(src), uintptr(len(src)))
	keepAlive(src)
	keepAlive(dst)
	return int(n)
}

// LZVNEncodeScratchSize returns the scratch size required by
// EncodeLZVNBuffer.
func (lib *Library) LZVNEncodeScratchSize() int {
	return int(lib.lzvnEncodeScratchSize())
}
