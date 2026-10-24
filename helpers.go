package lzfse

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

// sliceData returns the address of the first byte of b, or 0 when b is empty.
func sliceData(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(b)))
}

// keepAlive pins a Go value across a call into native code that received the
// value's address as a uintptr.
func keepAlive(v any) {
	runtime.KeepAlive(v)
}

// encodeDestSize returns a destination buffer size that lzfse_encode_buffer
// can always fill successfully for an input of n bytes. The worst case is
// bounded by the largest magic the container can emit for a block plus block
// headers; n + n/2 + 1024 has ample headroom.
func encodeDestSize(n int) int {
	if n < 1024 {
		n = 1024
	}
	return n + n/2 + 1024
}

// decodeDestSize returns the initial destination buffer size DecodeBuffer
// tries for compressed input of n bytes; it grows geometrically when needed.
func decodeDestSize(n int) int {
	if n == 0 {
		return 1024
	}
	return n * 4
}

// shrink returns b with its backing array trimmed to b's length when the
// array is much larger. The output buffers are sized for the worst case, so
// returning them as they are would keep far more memory alive than the data
// they hold for as long as the caller keeps the result.
func shrink(b []byte) []byte {
	if cap(b)-len(b) <= len(b)/4 {
		return b
	}
	return append(make([]byte, 0, len(b)), b...)
}

// maxPeekRatio bounds how much larger than the compressed input a size read
// from a block header may be before it is distrusted. The header is
// attacker-controlled: without the bound, an 8 byte input could make
// DecodeBuffer allocate gigabytes. LZVN expands at most ~140x, so a larger
// claim is not worth an up-front allocation; DecodeBuffer's bounded growth
// path handles it instead.
const maxPeekRatio = 1 << 10

// peekDecodedSize returns the total decoded size of an LZFSE stream when it
// is readable from the first block header, or 0. Both the "bvx-" raw block
// and the "bvxn" LZVN block header store the block's decoded byte count at
// offset 4 (little endian uint32); a stream made of a single such block, as
// the encoder emits for inputs that fit one block, is fully sized this way.
func peekDecodedSize(src []byte) int {
	if len(src) < 8 {
		return 0
	}
	magic := binary.LittleEndian.Uint32(src)
	// 0x2d787662 = "bvx-" (raw block), 0x6e787662 = "bvxn" (LZVN block).
	if magic != 0x2d787662 && magic != 0x6e787662 {
		return 0
	}
	size := uint64(binary.LittleEndian.Uint32(src[4:8]))
	if size > uint64(len(src))*maxPeekRatio {
		return 0
	}
	return int(size)
}
