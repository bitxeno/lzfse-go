package lzfse

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func goldenFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", name, err)
	}
	return b
}

func TestDecodeBufferGolden(t *testing.T) {
	got, err := DecodeBuffer(goldenFile(t, "enc.bin"))
	if err != nil {
		t.Fatal(err)
	}
	want := goldenFile(t, "dec.bin")
	if len(got) != len(want) || !bytesEqual(got, want) {
		t.Fatalf("DecodeBuffer mismatch: got %d bytes, want %d bytes", len(got), len(want))
	}
}

func TestEncodeBufferGolden(t *testing.T) {
	got, err := EncodeBuffer(goldenFile(t, "dec.bin"))
	if err != nil {
		t.Fatal(err)
	}
	want := goldenFile(t, "enc.bin")
	if !bytesEqual(got, want) {
		t.Fatalf("EncodeBuffer mismatch: got %d bytes, want %d bytes", len(got), len(want))
	}
}

func TestLZVNGolden(t *testing.T) {
	enc := goldenFile(t, "lzvn_enc.bin")
	dec := goldenFile(t, "lzvn_dec.bin")

	dst := make([]byte, 1024)
	n := DefaultLibraryOrSkip(t).DecodeLZVNBuffer(enc, dst)
	if n != len(dec) || !bytesEqual(dst[:n], dec) {
		t.Fatalf("DecodeLZVNBuffer mismatch: got %d bytes, want %d\ngot:  %x\nwant: %x", n, len(dec), dst[:n], dec)
	}

	out := make([]byte, encodeDestSize(len(dec)))
	n = DefaultLibraryOrSkip(t).EncodeLZVNBuffer(dec, out)
	if n != len(enc) || !bytesEqual(out[:n], enc) {
		t.Fatalf("EncodeLZVNBuffer mismatch: got %d bytes, want %d", n, len(enc))
	}
}

func TestZBMGolden(t *testing.T) {
	lib := DefaultLibraryOrSkip(t)

	enc := goldenFile(t, "zbm_enc.bin")
	dec := goldenFile(t, "dec.bin")
	dst := make([]byte, len(dec)+1024)
	n := lib.DecodeZBMBuffer(enc, dst)
	if n != len(dec) || !bytesEqual(dst[:n], dec) {
		t.Fatalf("DecodeZBMBuffer mismatch: got %d bytes, want %d", n, len(dec))
	}

	out := make([]byte, len(dec)+len(dec)/8+65536)
	n = lib.EncodeZBMBuffer(dec, out)
	if n <= 0 {
		t.Fatalf("EncodeZBMBuffer returned %d", n)
	}
	re := make([]byte, len(dec)+1024)
	n = lib.DecodeZBMBuffer(out[:n], re)
	if n != len(dec) || !bytesEqual(re[:n], dec) {
		t.Fatalf("ZBM roundtrip mismatch: got %d bytes, want %d", n, len(dec))
	}
}

func TestRoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 2, 63, 64, 100, 4096, 100_000} {
		for _, kind := range []string{"zeros", "text", "random"} {
			t.Run(fmt.Sprintf("%s-%d", kind, size), func(t *testing.T) {
				src := make([]byte, size)
				switch kind {
				case "text":
					for i := range src {
						src[i] = byte('a' + i%26)
					}
				case "random":
					rng := rand.New(rand.NewSource(1))
					rng.Read(src)
				}

				enc, err := EncodeBuffer(src)
				if err != nil {
					t.Fatal(err)
				}
				if size == 0 {
					if enc != nil {
						t.Fatalf("EncodeBuffer(nil) = %v, want nil", enc)
					}
					return
				}

				dec, err := DecodeBuffer(enc)
				if err != nil {
					t.Fatal(err)
				}
				if !bytesEqual(dec, src) {
					t.Fatalf("roundtrip mismatch: got %d bytes, want %d", len(dec), len(src))
				}
			})
		}
	}
}

func TestEncodeBufferInto(t *testing.T) {
	src := goldenFile(t, "dec.bin")

	enc, err := EncodeBuffer(src)
	if err != nil {
		t.Fatal(err)
	}

	dst := make([]byte, encodeDestSize(len(src)))
	n, err := EncodeBufferInto(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(dst[:n], enc) {
		t.Fatal("EncodeBufferInto result differs from EncodeBuffer")
	}

	// The encoder adapts to the destination size, so a slightly undersized
	// buffer still succeeds; only a dst that cannot hold the output at all
	// fails. Random data is incompressible, so it cannot fit a tiny buffer.
	incompressible := make([]byte, 64<<10)
	rng := rand.New(rand.NewSource(2))
	rng.Read(incompressible)
	if _, err := EncodeBufferInto(incompressible, make([]byte, 1024)); !errors.Is(err, ErrShortBuffer) {
		t.Fatalf("tiny dst for incompressible data: got %v, want ErrShortBuffer", err)
	}

	if n, err := EncodeBufferInto(nil, dst); err != nil || n != 0 {
		t.Fatalf("EncodeBufferInto(nil) = %d, %v; want 0, nil", n, err)
	}
}

func TestDecodeBufferInto(t *testing.T) {
	enc := goldenFile(t, "enc.bin")
	want := goldenFile(t, "dec.bin")

	dst := make([]byte, len(want))
	n, err := DecodeBufferInto(enc, dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(want) || !bytesEqual(dst[:n], want) {
		t.Fatalf("DecodeBufferInto mismatch: got %d bytes, want %d", n, len(want))
	}

	if _, err := DecodeBufferInto(enc, dst[:len(want)-1]); !errors.Is(err, ErrShortBuffer) {
		t.Fatalf("short dst: got %v, want ErrShortBuffer", err)
	}

	if n, err := DecodeBufferInto(nil, dst); err != nil || n != 0 {
		t.Fatalf("DecodeBufferInto(nil) = %d, %v; want 0, nil", n, err)
	}
}

func TestDecodeBufferCorruptInput(t *testing.T) {
	enc := goldenFile(t, "enc.bin")
	bad := append([]byte(nil), enc...)
	binary.LittleEndian.PutUint32(bad, 0xdeadbeef)
	if _, err := DecodeBuffer(bad); err == nil {
		t.Fatal("DecodeBuffer(corrupt) succeeded, want error")
	}
	if _, err := DecodeBuffer([]byte{1, 2, 3}); err == nil {
		t.Fatal("DecodeBuffer(garbage) succeeded, want error")
	}
}

// TestDecodeMultiBlockStream builds a two-block LZFSE stream by hand: a raw
// "bvx-" block followed by a raw block and the end-of-stream marker. The
// first block header alone does not give the stream size, so DecodeBuffer
// must fall back to growth and DecodeBufferInto must accept a dst that
// exactly fits.
func TestDecodeMultiBlockStream(t *testing.T) {
	first := []byte("block one data")
	second := []byte("block two data")
	stream := make([]byte, 0, 64)
	stream = binary.LittleEndian.AppendUint32(stream, 0x2d787662) // bvx-
	stream = binary.LittleEndian.AppendUint32(stream, uint32(len(first)))
	stream = append(stream, first...)
	stream = binary.LittleEndian.AppendUint32(stream, 0x2d787662) // bvx-
	stream = binary.LittleEndian.AppendUint32(stream, uint32(len(second)))
	stream = append(stream, second...)
	stream = binary.LittleEndian.AppendUint32(stream, 0x24787662) // bvx$
	want := append(append([]byte(nil), first...), second...)

	dec, err := DecodeBuffer(stream)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(dec, want) {
		t.Fatalf("multi-block decode: got %q, want %q", dec, want)
	}

	dst := make([]byte, len(want))
	n, err := DecodeBufferInto(stream, dst)
	if err != nil {
		t.Fatalf("DecodeBufferInto on exact dst: %v", err)
	}
	if n != len(want) || !bytesEqual(dst[:n], want) {
		t.Fatalf("multi-block DecodeBufferInto: got %d bytes", n)
	}
}

func TestScratchSizes(t *testing.T) {
	encSize, err := EncodeScratchSize()
	if err != nil || encSize <= 0 {
		t.Fatalf("EncodeScratchSize = %d, %v", encSize, err)
	}
	decSize, err := DecodeScratchSize()
	if err != nil || decSize <= 0 {
		t.Fatalf("DecodeScratchSize = %d, %v", decSize, err)
	}
}

func TestNewLibrary(t *testing.T) {
	dir, err := userLibraryDir()
	if err != nil {
		t.Skipf("no user cache dir: %v", err)
	}
	lib, err := NewLibrary(filepath.Join(dir, libraryFileName()))
	if err != nil {
		t.Fatal(err)
	}
	dec, err := lib.DecodeBuffer(goldenFile(t, "enc.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(dec, goldenFile(t, "dec.bin")) {
		t.Fatal("decoded data mismatch")
	}
}

func TestNewLibraryBadPath(t *testing.T) {
	if _, err := NewLibrary(filepath.Join(t.TempDir(), "missing.dylib")); err == nil {
		t.Fatal("NewLibrary(missing) succeeded, want error")
	}
	if _, err := NewLibrary("go.mod"); err == nil {
		t.Fatal("NewLibrary(non-library) succeeded, want error")
	}
}

func TestConcurrentRoundTrip(t *testing.T) {
	enc := goldenFile(t, "enc.bin")
	want := goldenFile(t, "dec.bin")
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dec, err := DecodeBuffer(enc)
			if err != nil {
				t.Errorf("decode: %v", err)
				return
			}
			if !bytesEqual(dec, want) {
				t.Errorf("decode mismatch")
				return
			}
			re, err := EncodeBuffer(dec)
			if err != nil {
				t.Errorf("encode: %v", err)
				return
			}
			if !bytesEqual(re, enc) {
				t.Errorf("encode mismatch")
			}
		}()
		_ = i
	}
	wg.Wait()
}

// TestLargerFiles exercises files big enough to force DecodeBuffer to grow
// its destination buffer, using generated data.
func TestDecodeBufferGrowsDestination(t *testing.T) {
	// Random data compresses poorly, so a small compressed input implies a
	// small output; use zeros, which compress extremely well, to force the
	// output to be much larger than the compressed size.
	src := make([]byte, 3<<20)
	enc, err := EncodeBuffer(src)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(enc)) > int64(len(src))/10 {
		t.Fatalf("zeros compress poorly: %d -> %d", len(src), len(enc))
	}
	dec, err := DecodeBuffer(enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(dec) != len(src) {
		t.Fatalf("decoded %d bytes, want %d", len(dec), len(src))
	}
}

// TestDecodeBufferForgedHeaderAllocation checks that a tiny input whose block
// header claims a huge decoded size does not make DecodeBuffer allocate that
// much memory up front.
func TestDecodeBufferForgedHeaderAllocation(t *testing.T) {
	DefaultLibraryOrSkip(t)
	src := make([]byte, 8)
	binary.LittleEndian.PutUint32(src, 0x2d787662) // "bvx-"
	binary.LittleEndian.PutUint32(src[4:], 0xF0000000)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if _, err := DecodeBuffer(src); err == nil {
		t.Fatal("DecodeBuffer accepted a forged header")
	}
	runtime.ReadMemStats(&after)
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 1<<30 {
		t.Fatalf("DecodeBuffer allocated %d MB for an %d byte input", grown>>20, len(src))
	}
}

// TestResultsDoNotRetainScratchSpace checks that the slices returned by
// EncodeBuffer and DecodeBuffer do not pin worst-case sized backing arrays.
func TestResultsDoNotRetainScratchSpace(t *testing.T) {
	DefaultLibraryOrSkip(t)
	src := make([]byte, 1<<20) // zeros: compress to almost nothing
	enc, err := EncodeBuffer(src)
	if err != nil {
		t.Fatal(err)
	}
	if cap(enc) > len(enc)+len(enc)/4 {
		t.Errorf("EncodeBuffer result retains %d bytes for %d bytes of data", cap(enc), len(enc))
	}
	dec, err := DecodeBuffer(enc)
	if err != nil {
		t.Fatal(err)
	}
	if cap(dec) > len(dec)+len(dec)/4 {
		t.Errorf("DecodeBuffer result retains %d bytes for %d bytes of data", cap(dec), len(dec))
	}
}

// DefaultLibraryOrSkip returns the default library for tests that need it.
func DefaultLibraryOrSkip(t *testing.T) *Library {
	t.Helper()
	lib, err := DefaultLibrary()
	if err != nil {
		t.Skipf("native library unavailable: %v", err)
	}
	return lib
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
